package main

import (
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/seletskiy/hierr"
)

var version = `2.2`
var usage = `shadowd, secure login distribution service

Usage:
  shadowd [options] -L <address> [-s <time>]
  shadowd [options] -G <token> [-n <size>] [-a <algo>]
  shadowd [options] -C [-h <host>...] [-i <ip>...] [-d <date>] [-b <length>]
  shadowd [options] -K <token> [-r]
  shadowd --help
  shadowd --version

Options:
  -G --generate            Generate and store hash-table for specified <token>.
                            Password will be read from stdin.
    -n --length <size>     Generate hash-table of specified length [default: 2048].
    -a --algorithm <algo>  Use specified algorithm [default: sha256].
    --no-confirm           Do not prompt confirmation for password.
  -C --certificate         Generate certificate pair for authenticating via HTTPS.
    -b --bytes <length>    Generate rsa key of specified length [default: 2048].
    -h --host <host>       Set specified host as trusted [default: $CERT_HOST].
    -i --address <ip>      Set specified ip address as trusted [default: $CERT_ADDR].
    -d --till <date>       Set time certificate valid till [default: $CERT_VALID].
  -L --listen <address>    Listen specified IP and port [default: :443].
    -s --ttl <time>        Use specified time duration as hash TTL [default: 24h].
  -K --key                 Wait for SSH-key to be entered on stdin and append it to file,
                            determined from <token>.
    -r --truncate          Truncate file for specified token, do not append.
  -t --tables <dir>        Use specified dir for storing and reading hash-tables
                            [default: /var/shadowd/ht/].
  -c --certs <dir>         Use specified dir for storing and reading certificates
                            [default: /var/shadowd/cert/].
  -k --keys <dir>          Use specified dir for reading public SSH keys.
                            [default: /var/shadowd/ssh/].
  -c --config <path>       Use specified configuration file.
  -q --quiet               Quiet mode, be less chatty.
  --help                   Show this screen.
  --version                Show program version.
`

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	args, _ := docopt.Parse(
		replaceDefaults(usage), nil, true, "shadowd "+version, false,
	)

	hashTTL, err := time.ParseDuration(args["--ttl"].(string))
	if err != nil {
		hierr.Fatalf(
			err, "can't parse ttl time",
		)
	}

	var backend Backend

	if path, ok := args["--config"].(string); ok {
		config, err := getConfig(path)
		if err != nil {
			hierr.Fatalf(
				err, "can't parse configuration file",
			)
		}

		if config.Backend.Use == "mongodb" {
			backend, err = newMongoDB(config.Backend.Path)
			if err != nil {
				hierr.Fatalf(
					err, "can't initialize mongodb backend",
				)
			}
		}
	}

	if backend == nil {
		backend = &filesystem{
			hashTablesDir: args["--tables"].(string),
			sshKeysDir:    args["--keys"].(string),
			hashTTL:       hashTTL,
		}
	}

	err = backend.Init()
	if err != nil {
		hierr.Fatalf(
			err, "can't initialize shadowd backend",
		)
	}

	switch {
	case args["--generate"]:
		err = handleTableGenerate(backend, args)

	case args["--key"]:
		err = handleSSHKeyAppend(backend, args)

	case args["--certificate"]:
		err = handleCertificateGenerate(backend, args)

	default:
		err = handleListen(backend, args, hashTTL)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func replaceDefaultCertHost(usage string) string {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	return strings.Replace(usage, "$CERT_HOST", hostname, -1)
}

func replaceDefaultCertAddr(usage string) string {
	return strings.Replace(usage, "$CERT_ADDR", getLocalIP(), -1)
}

func replaceDefaultCertValidTill(usage string) string {
	return strings.Replace(
		usage, "$CERT_VALID",
		time.Now().AddDate(1, 0, 0).Format("2006-02-01"),
		-1,
	)
}

func replaceDefaults(usage string) string {
	usage = replaceDefaultCertHost(usage)
	usage = replaceDefaultCertAddr(usage)
	usage = replaceDefaultCertValidTill(usage)

	return usage
}

func getLocalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, netInterface := range interfaces {
		adresses, err := netInterface.Addrs()
		if err != nil {
			panic(err)
		}

		for _, address := range adresses {
			switch addr := address.(type) {
			case *net.IPNet:
				ipString := addr.IP.String()
				if strings.HasPrefix(ipString, "127.") || ipString == "::1" {
					continue
				} else {
					return ipString
				}
			}
		}

	}

	return "127.0.0.1"
}
