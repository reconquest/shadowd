package main

import (
	"bufio"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// #cgo LDFLAGS: -lcrypt
// #include <unistd.h>
// #include <crypt.h>
import "C"

type AlgorithmImplementation func(login, password string) string

func handleTableGenerate(args map[string]interface{}) error {
	var (
		login         = args["<login>"].(string)
		amountString  = args["-n"].(string)
		algorithm     = args["-a"].(string)
		hashTablesDir = args["-t"].(string)
	)

	password, err := getPassword("Enter password: ")
	if err != nil {
		return err
	}

	err = validateTablesDirPermissions(hashTablesDir)
	if err != nil {
		return err
	}

	amount, err := strconv.Atoi(amountString)
	if err != nil {
		return err
	}

	implementation := getAlgorithmImplementation(algorithm)
	if implementation == nil {
		return errors.New("specified algorithm is not available")
	}

	file, err := os.Create(filepath.Join(hashTablesDir, login))
	if err != nil {
		return err
	}

	defer file.Close()

	for i := 0; i < amount; i++ {
		fmt.Fprintln(file, implementation(login, password))
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

func generateSha256(login, password string) string {
	shadowRecord := fmt.Sprintf("$5$%s", generateShaSalt())
	return C.GoString(C.crypt(C.CString(password), C.CString(shadowRecord)))
}

func generateSha512(login, password string) string {
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

func validateTablesDirPermissions(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	if stat.Mode()&0077 != 0 {
		return fmt.Errorf(
			"hash tables dir is too open: %s "+
				"(should be accessible only by owner)",
			stat.Mode())
	}

	return nil
}

func getPassword(prompt string) (string, error) {
	var (
		sttyEchoDisable = exec.Command("stty", "-F", "/dev/tty", "-echo")
		sttyEchoEnable  = exec.Command("stty", "-F", "/dev/tty", "echo")
	)

	fmt.Print(prompt)

	err := sttyEchoDisable.Run()
	if err != nil {
		return "", err
	}

	defer func() {
		sttyEchoEnable.Run()
		fmt.Println()
	}()

	stdin := bufio.NewReader(os.Stdin)
	password, err := stdin.ReadString('\n')
	if err != nil {
		return "", err
	}

	password = strings.TrimRight(password, "\n")

	return password, nil
}
