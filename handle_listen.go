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
	w http.ResponseWriter, r *http.Request,
) {
	// no need to validate token because 'http' package will validate request
	// uri and remove '../' partitions.
	token := strings.TrimPrefix(r.URL.Path, "/t/")

	table, err := OpenHashTable(
		filepath.Join(handler.Dir, token),
		handler.HashTTL,
	)

	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	handler.CleanupRecentClients()

	clientIp := r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]
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
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(record)
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
