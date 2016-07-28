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
	"time"

	"github.com/kovetskiy/spinner-go"
)

// #cgo LDFLAGS: -lcrypt
// #include <unistd.h>
// #include <crypt.h>
import "C"

type AlgorithmImplementation func(token string) string

func handleTableGenerate(args map[string]interface{}) error {
	var (
		token         = args["<token>"].(string)
		amountString  = args["--length"].(string)
		algorithm     = args["--algorithm"].(string)
		hashTablesDir = args["--tables"].(string)
		quiet         = args["--quiet"].(bool)
	)

	err := validateToken(token)
	if err != nil {
		return err
	}

	password, err := getPassword("Enter password: ")
	if err != nil {
		return err
	}

	proofPassword, err := getPassword("Retype password: ")
	if err != nil {
		return err
	}

	if password != proofPassword {
		return fmt.Errorf("specified passwords do not match")
	}

	amount, err := strconv.Atoi(amountString)
	if err != nil {
		return err
	}

	implementation := getAlgorithmImplementation(algorithm)
	if implementation == nil {
		return errors.New("specified algorithm is not available")
	}

	hashTablePath := filepath.Join(hashTablesDir, token)
	hashTableDir := filepath.Dir(hashTablePath)
	if _, err := os.Stat(hashTableDir); os.IsNotExist(err) {
		err = os.MkdirAll(hashTableDir, 0700)
		if err != nil {
			return err
		}
	}

	err = validateTablesDirPermissions(hashTablesDir)
	if err != nil {
		return err
	}

	file, err := os.Create(hashTablePath)
	if err != nil {
		return err
	}

	defer file.Close()

	if !quiet {
		spinner.Start()
		spinner.SetInterval(time.Millisecond * 100)
	}

	for i := 1; i <= amount; i++ {
		if !quiet {
			spinner.SetStatus(
				fmt.Sprintf(
					"Generating hash table... %d%% ",
					i*100/amount,
				),
			)
		}

		fmt.Fprintln(file, implementation(password))
	}

	if !quiet {
		spinner.Stop()
	}

	fmt.Printf(
		"Hash table %s with %d items successfully created.\n",
		hashTablePath,
		amount,
	)

	return file.Close()
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

func generateSha256(password string) string {
	shadowRecord := fmt.Sprintf("$5$%s", generateShaSalt())
	return C.GoString(C.crypt(C.CString(password), C.CString(shadowRecord)))
}

func generateSha512(password string) string {
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

func validateToken(token string) error {
	if strings.Contains(token, "../") {
		return fmt.Errorf(
			"specified token is not available, do not use '../' in token",
		)
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
