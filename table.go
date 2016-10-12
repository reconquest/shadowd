package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/reconquest/hierr-go"
)

type hashTable struct {
	size       int64
	recordSize int
	file       *os.File
	path       string
}

func openHashTable(path string) (*hashTable, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	table := &hashTable{
		file: file,
		path: path,
	}

	return table, nil
}

func (table *hashTable) getRecord(number int64) ([]byte, error) {
	recordSize, err := table.getRecordSize()
	if err != nil {
		return nil, hierr.Errorf(
			err, "can't get table record size",
		)
	}

	tableSize, err := table.getSize()
	if err != nil {
		return nil, hierr.Errorf(
			err, "can't get table size",
		)
	}

	if number >= tableSize {
		return nil, errors.New("record number is out of range")
	}

	var (
		// +1 for new line
		offset = number * int64(recordSize+1)
		record = make([]byte, recordSize)
	)

	readLength, err := table.file.ReadAt(record, offset)
	if err != nil {
		return nil, err
	}

	if readLength != recordSize {
		return nil, errors.New("read bytes are less than required record size")
	}

	return record, nil
}

func (table *hashTable) hashExists(hash string) (bool, error) {
	defer table.file.Seek(0, 0)

	scanner := bufio.NewScanner(table.file)
	for scanner.Scan() {
		if scanner.Text() == hash {
			return true, nil
		}
	}

	return false, scanner.Err()
}

func (table *hashTable) getRecordSize() (int, error) {
	if table.recordSize != 0 {
		return table.recordSize, nil
	}

	var line string
	_, err := fmt.Fscanln(table.file, &line)
	if err != nil {
		return 0, err
	}

	_, err = table.file.Seek(0, 0)
	if err != nil {
		return 0, err
	}

	table.recordSize = len(line)

	return table.recordSize, nil
}

func (table *hashTable) getSize() (int64, error) {
	if table.size != 0 {
		return table.size, nil
	}

	recordSize, err := table.getRecordSize()
	if err != nil {
		return 0, hierr.Errorf(
			err, "can't get table record size",
		)
	}

	stat, err := os.Stat(table.path)
	if err != nil {
		return 0, hierr.Errorf(
			err, "can't stat table file",
		)
	}

	table.size = stat.Size() / int64(recordSize)

	return table.size, nil
}
