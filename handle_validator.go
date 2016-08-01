package main

import (
	"log"
	"net/http"
	"strings"
)

type HashValidatorHandler struct {
	backend Backend
}

func (handler *HashValidatorHandler) ServeHTTP(
	response http.ResponseWriter, request *http.Request,
) {
	path := strings.TrimPrefix(request.URL.Path, "/v/")
	path = strings.TrimRight(path, "/")

	slash := strings.LastIndex(path, "/")
	if slash == -1 {
		log.Printf(
			"got bad request to hash table validator: %s", request.URL.Path,
		)
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	token, hash := path[:slash], path[slash+1:]

	log.Printf(
		"got request to hash table validator, hash: '%s', token: '%s'",
		hash, token,
	)

	exists, err := handler.backend.IsHashExists(token, hash)
	if err != nil {
		log.Println(err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	if exists {
		response.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("hash '%s' does not exists for '%s' token", hash, token)
	response.WriteHeader(http.StatusNotFound)
}
