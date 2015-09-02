package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"code.google.com/p/go.crypto/ssh"
)

type SSHKeysHandler struct {
	Dir string
}

func (handler *SSHKeysHandler) ServeHTTP(
	writer http.ResponseWriter, request *http.Request,
) {
	token := strings.TrimPrefix(request.URL.Path, "/ssh/")

	keyPath := path.Join(handler.Dir, token)

	keyFile, err := os.Open(keyPath)
	if err != nil {
		log.Println(err)
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	defer keyFile.Close()

	io.Copy(writer, keyFile)
}

func handleSSHKeyAppend(args map[string]interface{}) error {
	var (
		token      = args["<token>"].(string)
		truncate   = args["-r"].(bool)
		sshKeysDir = args["-k"].(string)
	)

	sshKeyPath := filepath.Join(sshKeysDir, token)
	sshKeyDir := filepath.Dir(sshKeyPath)
	if _, err := os.Stat(sshKeyDir); err != nil && os.IsNotExist(err) {
		err = os.MkdirAll(sshKeyDir, 0700)
		if err != nil {
			return err
		}
	}

	var keyFile *os.File
	var err error

	if _, err := os.Stat(sshKeyPath); err != nil && os.IsNotExist(err) {
		truncate = true
	}

	if truncate {
		keyFile, err = os.Create(sshKeyPath)
	} else {
		keyFile, err = os.OpenFile(sshKeyPath, os.O_WRONLY|os.O_APPEND, 0644)
	}

	if err != nil {
		return err
	}

	defer keyFile.Close()

	sshKeyBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	_, comment, _, _, err := ssh.ParseAuthorizedKey(sshKeyBytes)
	if err != nil {
		return err
	}

	_, err = keyFile.Write(sshKeyBytes)
	if err != nil {
		return err
	}

	log.Println("added new key with comment:", comment)

	return nil
}
