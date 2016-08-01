package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/seletskiy/hierr"

	"golang.org/x/crypto/ssh"
)

type SSHKeysHandler struct {
	backend Backend
}

func (handler *SSHKeysHandler) ServeHTTP(
	writer http.ResponseWriter, request *http.Request,
) {
	token := strings.TrimPrefix(request.URL.Path, "/ssh/")

	keys, err := handler.backend.GetPublicKeys(token)
	if err != nil {
		if err == ErrNotFound {
			writer.WriteHeader(http.StatusNotFound)
			return
		}

		log.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	writer.Write([]byte(keys))
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

	_, comment, _, _, err := ssh.ParseAuthorizedKey(key)
	if err != nil {
		return hierr.Errorf(
			err, "can't parse key",
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
