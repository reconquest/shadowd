package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type HashTable struct {
	Count      int64
	RecordSize int
	File       *os.File
	HashTTL    time.Duration
}

func (table HashTable) GetRecord(number int64) ([]byte, error) {
	if number >= table.Count {
		return nil, errors.New("record number is out of range")
	}

	seekOffset := number * int64(table.RecordSize)

	data := make([]byte, table.RecordSize)
	readBytesCount, err := table.File.ReadAt(data, seekOffset)
	if err != nil {
		return nil, err
	}

	if readBytesCount != table.RecordSize {
		return nil, errors.New("read bytes are less than required record size")
	}

	return data, nil
}

func (table HashTable) GetRecordByHashedString(input string) ([]byte, error) {
	hash := sha256.Sum256([]byte(
		fmt.Sprintf("%s%d", input, table.getTimeHashPart())),
	)

	hashMaxLength := int64(1)
	index := int64(0)

	for _, hashByte := range hash {
		if hashMaxLength > table.Count {
			break
		}

		hashMaxLength <<= 8
		index += hashMaxLength * int64(hashByte)
	}

	remainder := big.NewInt(0).Mod(
		big.NewInt(index), big.NewInt(table.Count),
	).Int64()

	return table.GetRecord(remainder)
}

func (table HashTable) getTimeHashPart() int64 {
	return time.Now().Unix() / int64(table.HashTTL/time.Second)
}

type HashTableHandler struct {
	Dir           string
	RecentClients map[string]time.Time
	HashTTL       time.Duration
}

func OpenHashTable(path string, hashTTL time.Duration) (*HashTable, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var line string
	_, err = fmt.Fscanln(file, &line)
	if err != nil {
		return nil, err
	}

	recordSize := len(line) + 1

	count := stat.Size() / int64(recordSize)

	table := &HashTable{
		Count:      count,
		RecordSize: recordSize,
		File:       file,
		HashTTL:    hashTTL,
	}

	return table, nil
}

func handleListen(args map[string]interface{}) error {
	var (
		hashTablesDir = args["-t"].(string)
		sshKeysDir    = args["-k"].(string)
		certDir       = strings.TrimRight(args["-c"].(string), "/") + "/"
	)

	hashTTL, err := time.ParseDuration(args["-s"].(string))
	if err != nil {
		return err
	}

	http.Handle("/t/", &HashTableHandler{
		Dir:           hashTablesDir,
		RecentClients: map[string]time.Time{},
		HashTTL:       hashTTL,
	})

	http.Handle("/ssh/", &SSHKeysHandler{
		Dir: sshKeysDir,
	})

	var (
		certFile = filepath.Join(certDir, "cert.pem")
		keyFile  = filepath.Join(certDir, "key.pem")
	)

	certExist := true
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		certExist = false
	}

	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		certExist = false
	}

	if !certExist {
		log.Println("no certificate found, generating with default settings")
		err := handleCertificateGenerate(args)
		if err != nil {
			return err
		}
	}

	log.Println("starting listening on", args["-L"].(string))
	return http.ListenAndServeTLS(args["-L"].(string), certFile, keyFile, nil)
}

func (handler *HashTableHandler) ServeHTTP(
	writer http.ResponseWriter, request *http.Request,
) {
	// no need to validate token because 'http' package will validate request
	// uri and remove '../' partitions.
	token := strings.TrimPrefix(request.URL.Path, "/t/")

	if strings.HasSuffix(token, "/") || token == "" {
		listing, err := getFilesList(filepath.Join(handler.Dir, token))
		if err != nil {
			log.Println(err)

			if os.IsNotExist(err) {
				writer.WriteHeader(http.StatusNotFound)
			} else {
				writer.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		if len(listing) == 0 {
			writer.WriteHeader(http.StatusNoContent)
			return
		}

		_, err = writer.Write([]byte(strings.Join(listing, "\n")))
		if err != nil {
			log.Println(err)
		}

		return
	}

	table, err := OpenHashTable(
		filepath.Join(handler.Dir, token),
		handler.HashTTL,
	)

	if err != nil {
		log.Println(err)
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	handler.CleanupRecentClients()

	clientIp := request.RemoteAddr[:strings.LastIndex(request.RemoteAddr, ":")]
	clientCredentials := clientIp + "-" + token

	// in case of client requested shadow entry not too long ago,
	// we should send different entry on further invocations
	if _, ok := handler.RecentClients[clientCredentials]; ok {
		clientCredentials += "-next"
	} else {
		handler.RecentClients[clientCredentials] = time.Now()
	}

	record, err := table.GetRecordByHashedString(clientCredentials)

	if err != nil {
		writer.Write([]byte(err.Error()))
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Write(record)
}

func (handler *HashTableHandler) CleanupRecentClients() {
	actualClients := map[string]time.Time{}

	for ip, requestTime := range handler.RecentClients {
		if time.Now().Sub(requestTime) > handler.HashTTL {
			continue
		}

		actualClients[ip] = requestTime
	}

	handler.RecentClients = actualClients
}

func getFilesList(directory string) (files []string, err error) {
	files = []string{}

	directory = filepath.Clean(directory)

	if stat, err := os.Stat(directory); err != nil {
		return nil, err
	} else {
		if !stat.IsDir() {
			return nil, fmt.Errorf(
				"speficied path '%s' is not a directory", directory,
			)
		}
	}

	err = filepath.Walk(
		directory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// skip root dir
			if info.IsDir() && directory == path {
				return nil
			}

			if info.IsDir() {
				return filepath.SkipDir
			}

			files = append(
				files,
				strings.TrimPrefix(strings.TrimPrefix(path, directory), "/"),
			)

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return files, nil
}
