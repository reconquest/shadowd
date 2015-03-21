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
	Count           int64
	RecordSize      int
	File            *os.File
	TimeOffsetGrain time.Duration
}

func (table HashTable) GetRecord(number int64) ([]byte, error) {
	if number > table.Count {
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
		fmt.Sprintf("%s%d", input, table.getTimeOffset())),
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

func (table HashTable) getTimeOffset() int64 {
	return time.Now().Unix() / int64(table.TimeOffsetGrain/time.Second)
}

type HashTableHandler struct {
	Dir              string
	RecentClients    map[string]time.Time
	RecentClientsTTL time.Duration
}

func OpenHashTable(path string, grain time.Duration) (*HashTable, error) {
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
		Count:           count,
		RecordSize:      recordSize,
		File:            file,
		TimeOffsetGrain: grain,
	}

	return table, nil
}

func handleListen(args map[string]interface{}) error {
	var (
		hashTablesDir = args["-t"].(string)
		certDir       = strings.TrimRight(args["-c"].(string), "/") + "/"
	)

	http.Handle("/t/", &HashTableHandler{
		Dir:              hashTablesDir,
		RecentClients:    map[string]time.Time{},
		RecentClientsTTL: time.Minute,
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

	log.Println("starting listening on", args["-l"].(string))
	return http.ListenAndServeTLS(args["-l"].(string), certFile, keyFile, nil)
}

func (handler *HashTableHandler) ServeHTTP(
	w http.ResponseWriter, r *http.Request,
) {
	prefix := strings.TrimPrefix(r.URL.Path, "/t/")
	table, err := OpenHashTable(
		filepath.Join(handler.Dir, prefix),
		handler.RecentClientsTTL,
	)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	handler.CleanupRecentClients()

	clientIp := r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]

	// in case of client requested shadow entry not to long ago,
	// we should send different entry on further invokations
	if _, ok := handler.RecentClients[clientIp]; ok {
		clientIp += "-next"
	} else {
		handler.RecentClients[clientIp] = time.Now()
	}

	record, err := table.GetRecordByHashedString(clientIp)

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
		if time.Now().Sub(requestTime) > handler.RecentClientsTTL {
			continue
		}

		actualClients[ip] = requestTime
	}

	handler.RecentClients = actualClients
}
