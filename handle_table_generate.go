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

var (
	saltSymbols = []rune(
		"qwertyuiopasdfghjklzxcvbnm" +
			"QWERTYUIOPASDFGHJKLZXCVBNM" +
			"0123456789" +
			"./",
	)
	saltLength = 16
)

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
		return generateSHA256
	case "sha512":
		return generateSHA512
	}

	return nil
}

func generateSHA256(password string) string {
	salt := fmt.Sprintf("$5$%s", generateSHASalt())
	return C.GoString(C.crypt(C.CString(password), C.CString(salt)))
}

func generateSHA512(password string) string {
	salt := fmt.Sprintf("$6$%s", generateSHASalt())
	return C.GoString(C.crypt(C.CString(password), C.CString(salt)))
}

func generateSHASalt() string {
	salt := make([]rune, saltLength)
	for i := 0; i < saltLength; i++ {
		salt[i] = saltSymbols[rand.Intn(len(saltSymbols))]
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
		return "", hierr.Errorf(
			err, "%q", sttyEchoDisable.Args,
		)
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
