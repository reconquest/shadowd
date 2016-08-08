package main

import (
	"bufio"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/kovetskiy/spinner-go"
	"github.com/seletskiy/hierr"
)

// #cgo LDFLAGS: -lcrypt
// #include <unistd.h>
// #include <crypt.h>
import "C"

type AlgorithmImplementation func(token string) string

func handleTableGenerate(backend Backend, args map[string]interface{}) error {
	var (
		token     = args["<token>"].(string)
		lengthRaw = args["--length"].(string)
		algorithm = args["--algorithm"].(string)
		quiet     = args["--quiet"].(bool)
		noconfirm = args["--no-confirm"].(bool)
	)

	err := validateToken(token)
	if err != nil {
		return err
	}

	length, err := strconv.Atoi(lengthRaw)
	if err != nil {
		return err
	}

	password, err := getPassword("Enter password: ")
	if err != nil {
		return hierr.Errorf(
			err, "can't get password",
		)
	}

	if !noconfirm {
		proofPassword, err := getPassword("Retype password: ")
		if err != nil {
			return hierr.Errorf(
				err, "can't get password confirmation",
			)
		}

		if password != proofPassword {
			return fmt.Errorf("specified passwords do not match")
		}
	}

	implementation := getAlgorithmImplementation(algorithm)
	if implementation == nil {
		return errors.New("specified algorithm is not available")
	}

	if !quiet {
		spinner.Start()
		spinner.SetInterval(time.Millisecond * 100)
	}

	table := []string{}
	for i := 0; i < length; i++ {
		if !quiet {
			spinner.SetStatus(
				fmt.Sprintf(
					"Generating hash table... %d%% ",
					(i+1)*100/length,
				),
			)
		}

		table = append(table, implementation(password))
	}

	if !quiet {
		spinner.Stop()
	}

	err = backend.SetHashTable(token, table)
	if err != nil {
		return hierr.Errorf(
			err, "can't save generated hash table",
		)
	}

	fmt.Printf(
		"Hash table %s with %d items successfully created.\n",
		token, length,
	)

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
		return "", hierr.Errorf(
			err, "can't read stdin for password",
		)
	}

	password = strings.TrimRight(password, "\n")

	return password, nil
}
