package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/seletskiy/hierr"

	"golang.org/x/crypto/ssh"
)

func (server *Server) HandleSSH(
	writer http.ResponseWriter, request *http.Request,
) {
	token := strings.TrimPrefix(request.URL.Path, "/ssh/")

	keys, err := server.backend.GetPublicKeys(token)
	if err != nil {
		if err == ErrNotFound {
			writer.WriteHeader(http.StatusNotFound)
			return
		}

		log.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = writer.Write([]byte(keys))
	if err != nil {
		log.Println(err)
	}
}

func handleSSHKeyAppend(backend Backend, args map[string]interface{}) error {
	var (
		token    = args["<token>"].(string)
		truncate = args["--truncate"].(bool)
	)

	key, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return hierr.Errorf(
			err, "can't read stdin",
		)
	}

	key = bytes.TrimSpace(key)

	_, comment, _, _, err := ssh.ParseAuthorizedKey(key)
	if err != nil {
		return hierr.Errorf(
			err, "can't parse public ssh key",
		)
	}

	err = backend.AddPublicKey(token, key, truncate)
	if err != nil {
		return hierr.Errorf(
			err, "can't add public key for %s", token,
		)
	}

	fmt.Println("Added new key with comment:", comment)

	return nil
}
