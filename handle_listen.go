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

const (
	passwordChangeSaltAmount = 10
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

	switch request.Method {
	case "GET":
		server.handleHashRetrieve(writer, request, token)
	case "PUT":
		server.handlePasswordChange(writer, request, token)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (server *Server) handleHashRetrieve(
	writer http.ResponseWriter,
	request *http.Request,
	token string,
) {
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
		if err == ErrNotFound {
			writer.WriteHeader(http.StatusNotFound)
		} else {
			log.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
		}
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

	modifier := 1
	if !recent {
		modifier = 0
		err = server.backend.AddRecentClient(remote)
		if err != nil {
			log.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	record, err := server.backend.GetHash(
		token,
		hashNumber(remote, tableSize, server.hashTTL, modifier),
	)
	if err != nil {
		writer.Write([]byte(err.Error()))
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Write([]byte(record))
}

func (server *Server) handlePasswordChange(
	writer http.ResponseWriter,
	request *http.Request,
	token string,
) {
	tableSize, err := server.backend.GetTableSize(token)
	if err != nil {
		if err == ErrNotFound {
			writer.WriteHeader(http.StatusNotFound)
		} else {
			log.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	remote := request.RemoteAddr[:strings.LastIndex(request.RemoteAddr, ":")]
	remote += "-" + token + "-salt-"

	salts := []string{}
	hashes := []string{}
	for i := 0; i < passwordChangeSaltAmount; i++ {
		hash, err := server.backend.GetHash(
			token,
			hashNumber(remote, tableSize, server.hashTTL, i),
		)
		if err != nil {
			log.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		parts := strings.Split(hash, "$")
		if len(parts) < 4 {
			log.Printf("invalid hash for %s found: '%s'", token, hash)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		salts = append(salts, "$"+parts[1]+"$"+parts[2])
		hashes = append(hashes, hash)
	}

	err = request.ParseForm()
	if err != nil {
		log.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	proofs, ok := request.Form["hash"]
	if !ok || len(proofs) != passwordChangeSaltAmount {
		fmt.Fprintln(writer, strings.Join(salts, "\n"))
		return
	}

	password := request.FormValue("password")
	if password == "" {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	for index, _ := range hashes {
		if proofs[index] != hashes[index] {
			log.Printf(
				"password change declined for %s, wrong hash: '%s'",
				token, proofs[index],
			)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	log.Printf(
		"password change for %s accepted, generating new hash table...",
	)

	table := []string{}
	for i := 0; i < int(tableSize); i++ {
		table = append(table, generateSHA512(password))
	}

	err = server.backend.SetHashTable(token, table)
	if err != nil {
		log.Println(
			hierr.Errorf(
				err, "can't save generated hash table for %s", token,
			),
		)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf(
		"hash table %s with %d items successfully created",
		token, tableSize,
	)
}

func hashNumber(
	source string, max int64, ttl time.Duration, modifier int,
) int64 {
	hash := sha256.Sum256([]byte(
		fmt.Sprintf(
			"%s%d",
			source, time.Now().Unix()/int64(ttl/time.Second),
		),
	))

	var (
		hashMaxLength int64 = 1
		hashIndex     int64 = 0
	)

	for _, hashByte := range hash {
		if hashMaxLength > max {
			break
		}

		hashMaxLength <<= 8
		hashIndex += hashMaxLength * int64(hashByte)
	}

	hashIndex += int64(modifier)

	modMax := max
	if modMax%10 == 0 {
		modMax = max - 1
	}

	number := big.NewInt(0).Mod(
		big.NewInt(hashIndex), big.NewInt(modMax),
	).Int64()

	return number
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
