package main

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type HashValidatorHandler struct {
	Dir string
}

func (handler *HashValidatorHandler) ServeHTTP(
	response http.ResponseWriter, request *http.Request,
) {
	path := strings.TrimPrefix(request.URL.Path, "/v/")
	path = strings.TrimRight(path, "/")

	pathParts := strings.Split(strings.TrimRight(path, "/"), "/")
	if len(pathParts) < 2 {
		log.Printf(
			"got bad request to hash table validator: %s", request.URL.Path,
		)
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	hash := pathParts[len(pathParts)-1]
	token := strings.TrimSuffix(path, "/"+hash)

	log.Printf(
		"got request to hash table validator, hash: '%s', token: '%s'",
		hash, token,
	)

	table, err := OpenHashTable(filepath.Join(handler.Dir, token))
	if err != nil {
		log.Println(err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	exists, err := table.HashExists(hash)
	if err != nil {
		log.Println(err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !exists {
		log.Printf("hash '%s' does not exists for '%s' token", hash, token)
		response.WriteHeader(http.StatusNotFound)
		return
	}
}
