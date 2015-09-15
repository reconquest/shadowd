package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
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

	_, err = io.Copy(writer, keyFile)
	if err != nil {
		log.Println(err)
	}

	err = keyFile.Close()
	if err != nil {
		log.Println(err)
	}
}

func handleSSHKeyAppend(args map[string]interface{}) error {
	var (
		token      = args["<token>"].(string)
		truncate   = args["-r"].(bool)
		sshKeysDir = args["-k"].(string)
	)

	sshKeyPath := filepath.Join(sshKeysDir, token)
	sshKeyDir := filepath.Dir(sshKeyPath)
	if _, err := os.Stat(sshKeyDir); os.IsNotExist(err) {
		err = os.MkdirAll(sshKeyDir, 0700)
		if err != nil {
			return err
		}
	}

	openFlags := os.O_CREATE | os.O_WRONLY

	if truncate {
		openFlags |= os.O_TRUNC
	} else {
		openFlags |= os.O_APPEND
	}

	sshKeyBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	_, comment, _, _, err := ssh.ParseAuthorizedKey(sshKeyBytes)
	if err != nil {
		return fmt.Errorf("can't parse key: %s", err)
	}

	keyFile, err := os.OpenFile(sshKeyPath, openFlags, 0644)

	if err != nil {
		return err
	}

	defer keyFile.Close()

	_, err = keyFile.Write(sshKeyBytes)
	if err != nil {
		return err
	}

	fmt.Println("Added new key with comment:", comment)

	return nil
}
