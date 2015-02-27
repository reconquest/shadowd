package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
)

// #cgo LDFLAGS: -lcrypt
// #include <unistd.h>
// #include <crypt.h>
import "C"

type AlgorithmImplementation func(token, password string) string

func handleTableGenerate(args map[string]interface{}) error {
	var (
		token         = args["<token>"].(string)
		password      = args["<password>"].(string)
		amountString  = args["-n"].(string)
		algorithm     = args["-a"].(string)
		hashTablesDir = args["-d"].(string)
	)

	amount, err := strconv.Atoi(amountString)
	if err != nil {
		return err
	}

	implementation := getAlgorithmImplementation(algorithm)
	if implementation == nil {
		return errors.New("specified algorithm is not available")
	}

	file, err := os.Create(filepath.Join(hashTablesDir, token))
	if err != nil {
		return err
	}

	defer file.Close()

	for i := 0; i < amount; i++ {
		fmt.Fprintln(file, implementation(token, password))
	}

	return nil
}

func getAlgorithmImplementation(algorithm string) AlgorithmImplementation {
	switch algorithm {
	case "sha256":
		return generateSha256
	case "sha512":
		return generateSha512
	}

	return nil
}

func makeShadowFileRecord(salt, hash string, algorithmId int) string {
	return fmt.Sprintf("$%d$%s$%s", algorithmId, salt, hash)
}

func generateSha256(token, password string) string {
	shadowRecord := fmt.Sprintf("$5$%s", generateShaSalt())
	return C.GoString(C.crypt(C.CString(password), C.CString(shadowRecord)))
}

func generateSha512(token, password string) string {
	shadowRecord := fmt.Sprintf("$6$%s", generateShaSalt())
	return C.GoString(C.crypt(C.CString(password), C.CString(shadowRecord)))
}

func generateShaSalt() string {
	size := 16
	letters := []rune("qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM")

	salt := make([]rune, size)
	for i := 0; i < size; i++ {
		salt[i] = letters[rand.Intn(len(letters))]
	}

	return string(salt)
}
