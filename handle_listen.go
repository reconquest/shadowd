package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/seletskiy/hierr"
)

type Server struct {
	backend Backend
	hashTTL time.Duration
}

func (server *Server) HandleTokens(
	writer http.ResponseWriter, request *http.Request,
) {
	// no need to validate token because net/http package will validate request
	// uri and remove '../' statements.
	token := strings.TrimPrefix(request.URL.Path, "/t/")

	if strings.HasSuffix(token, "/") || token == "" {
		tokens, err := server.backend.GetTokens(token)
		if err != nil {
			log.Println(
				hierr.Errorf(
					err,
					"can't get tokens with prefix '%s'", token,
				),
			)

			if err == ErrNotFound {
				writer.WriteHeader(http.StatusNotFound)
			} else {
				writer.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		if len(tokens) == 0 {
			writer.WriteHeader(http.StatusNoContent)
			return
		}

		_, err = writer.Write([]byte(strings.Join(tokens, "\n")))
		if err != nil {
			log.Println(err)
		}

		return
	}

	tableSize, err := server.backend.GetTableSize(token)
	if err != nil {
		log.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	remote := request.RemoteAddr[:strings.LastIndex(request.RemoteAddr, ":")]
	remote = remote + "-" + token

	// in case of client requested shadow entry not too long ago,
	// we should send different entry on further invocations
	recent, err := server.backend.IsRecentClient(remote)
	if err != nil {
		log.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	if recent {
		remote += "-next"
	} else {
		err = server.backend.AddRecentClient(remote)
		if err != nil {
			log.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	hash := sha256.Sum256([]byte(
		fmt.Sprintf(
			"%s%d",
			remote, time.Now().Unix()/int64(server.hashTTL/time.Second),
		),
	))

	var (
		hashMaxLength int64 = 1
		hashIndex     int64 = 0
	)

	for _, hashByte := range hash {
		if hashMaxLength > tableSize {
			break
		}

		hashMaxLength <<= 8
		hashIndex += hashMaxLength * int64(hashByte)
	}

	remainder := big.NewInt(0).Mod(
		big.NewInt(hashIndex), big.NewInt(tableSize),
	).Int64()

	record, err := server.backend.GetHash(token, remainder)
	if err != nil {
		writer.Write([]byte(err.Error()))
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Write([]byte(record))
}

func handleListen(
	backend Backend,
	args map[string]interface{},
	hashTTL time.Duration,
) error {
	wood := &Server{
		backend: backend,
		hashTTL: hashTTL,
	}

	http.HandleFunc("/v/", wood.HandleValidate)
	http.HandleFunc("/t/", wood.HandleTokens)
	http.HandleFunc("/ssh/", wood.HandleSSH)

	var (
		certFile = filepath.Join(args["--certs"].(string), "cert.pem")
		keyFile  = filepath.Join(args["--certs"].(string), "key.pem")
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

		err := handleCertificateGenerate(backend, args)
		if err != nil {
			return err
		}
	}

	log.Println("starting listening on", args["--listen"].(string))

	return http.ListenAndServeTLS(
		args["--listen"].(string), certFile, keyFile, nil,
	)
}
