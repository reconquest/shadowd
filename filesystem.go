package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/seletskiy/hierr"
)

type filesystem struct {
	hashTablesDir string
	hashTTL       time.Duration
	sshKeysDir    string
	clients       map[string]time.Time
	clientsLock   *sync.Mutex
}

func (fs *filesystem) Init() error {
	stat, err := os.Stat(fs.hashTablesDir)
	if err != nil {
		return err
	}

	if stat.Mode()&0077 != 0 {
		return fmt.Errorf(
			"hash tables dir is too open: %s "+
				"(should be accessible only by owner)",
			stat.Mode())
	}

	go func() {
		for range time.Tick(time.Minute) {
			fs.cleanupRecentClients()
		}
	}()

	return nil
}

func (fs *filesystem) SetHashTable(token string, table []string) error {
	path := filepath.Join(fs.hashTablesDir, token)

	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			return hierr.Errorf(
				err, "can't create directory %s", dir,
			)
		}
	}

	err := ioutil.WriteFile(
		path,
		[]byte(strings.Join(table, "\n")+"\n"),
		0600,
	)
	if err != nil {
		return hierr.Errorf(
			err, "can't write file %s", path,
		)
	}

	return nil
}

func (fs *filesystem) AddPublicKey(
	token string, key []byte, truncate bool,
) error {
	path := filepath.Join(fs.sshKeysDir, token)

	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			return hierr.Errorf(
				err, "can't create directory %s", dir,
			)
		}
	}

	openFlags := os.O_CREATE | os.O_WRONLY

	if truncate {
		openFlags |= os.O_TRUNC
	} else {
		openFlags |= os.O_APPEND
	}

	keyFile, err := os.OpenFile(path, openFlags, 0600)
	if err != nil {
		return hierr.Errorf(
			err, "can't open file %s", path,
		)
	}

	defer keyFile.Close()

	_, err = keyFile.Write(append(key, []byte("\n")...))
	if err != nil {
		return hierr.Errorf(
			err, "can't write key file %s", path,
		)
	}

	return nil
}

func (fs *filesystem) GetPublicKeys(token string) (string, error) {
	path := filepath.Join(fs.sshKeysDir, token)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNotFound
		}

		return "", hierr.Errorf(
			err, "can't read key file %s", path,
		)
	}

	return string(data), nil
}

func (fs *filesystem) IsHashExists(token string, hash string) (bool, error) {
	table, err := openHashTable(filepath.Join(fs.hashTablesDir, token))
	if err != nil {
		return false, err
	}

	return table.hashExists(hash)
}

func (fs *filesystem) GetTableSize(token string) (int64, error) {
	table, err := openHashTable(filepath.Join(fs.hashTablesDir, token))
	if err != nil {
		return 0, err
	}

	return table.getSize()
}

func (fs *filesystem) IsRecentClient(identifier string) (bool, error) {
	fs.clientsLock.Lock()
	defer fs.clientsLock.Unlock()

	if fs.clients == nil {
		fs.clients = map[string]time.Time{}
		return false, nil
	}

	_, ok := fs.clients[identifier]
	return ok, nil
}

func (fs *filesystem) AddRecentClient(identifier string) error {
	fs.clientsLock.Lock()
	defer fs.clientsLock.Unlock()

	if fs.clients == nil {
		fs.clients = map[string]time.Time{}
	}

	fs.clients[identifier] = time.Now()

	return nil
}

func (fs *filesystem) GetHash(token string, number int64) (string, error) {
	table, err := openHashTable(filepath.Join(fs.hashTablesDir, token))
	if err != nil {
		return "", err
	}

	record, err := table.getRecord(number)
	if err != nil {
		return "", err
	}

	return string(record), nil
}

func (fs *filesystem) GetTokens(prefix string) ([]string, error) {
	directory := filepath.Join(fs.hashTablesDir, prefix)

	stat, err := os.Stat(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf(
			"%s is not a directory", directory,
		)
	}

	tokens := []string{}
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

			tokens = append(
				tokens,
				strings.TrimPrefix(strings.TrimPrefix(path, directory), "/"),
			)

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (fs *filesystem) cleanupRecentClients() {
	actual := map[string]time.Time{}

	for identifier, requestTime := range fs.clients {
		if time.Now().Sub(requestTime) > fs.hashTTL {
			continue
		}

		actual[identifier] = requestTime
	}

	fs.clients = actual
}
