package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/docopt/docopt-go"
)

const usage = `shadowd, secure login distribution service

Usage:
  shadowd [options] [-L <listen>] [-a <hash_ttl>]
  shadowd [options] -G <token> <password> [-n <amount>] [-a <algo>]
  shadowd [options] -C [-h <host>...] [-i <address>...] [-d <till>] [-b <bytes>]
  shadowd -h | --help

Options:
  -G  Generate and store hash-table for specified <token> and <password>.
      -n <amount>    Generate hash-table of specified length [default: 2048].
      -a <algo>      Use specified algorithm [default: sha256].
  -C  Generate certificate pair for authenticating via HTTPS.
      -b <bytes>     Generate rsa key of specified length [default: 2048].
      -h <host>      Set specified host as trusted [default: $CERT_HOST].
      -i <address>   Set specified ip address as trusted [default: $CERT_ADDR].
      -d <till>      Set time certificate valid till [default: $CERT_VALID].
  -t <table_dir>     Use specified dir for storing and reading hash-tables
                     [default: /var/shadowd/ht/].
  -c <cert_dir>      Use specified dir for storing and reading certificates
                     [default: /var/shadowd/cert/].
  -L <listen>        Listen specified IP and port [default: :8080].
      -a <hash_ttl>  Use specified time duration as hash TTL [default: 24h].
`

func main() {
	args, _ := docopt.Parse(
		replaceDefaults(usage), nil, true, "shadowd 1.0", false,
	)

	var err error
	switch {
	case args["-G"]:
		err = handleTableGenerate(args)
	case args["-C"]:
		err = handleCertificateGenerate(args)
	default:
		err = handleListen(args)
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
	return strings.Replace(usage, "$CERT_ADDR", getLocalIpAddress(), -1)
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

func getLocalIpAddress() string {
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
			switch ipAddress := address.(type) {
			case *net.IPNet:
				ipString := fmt.Sprint(ipAddress.IP)
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
