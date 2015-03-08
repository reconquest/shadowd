package main

import (
	"log"

	"github.com/docopt/docopt-go"
)

const usage = `shadowd, secure login distribution service

Usage:
  shadowd [options]
  shadowd [options] -G <token> <password> [-n <amount>] [-a <algo>]
  shadowd [options] -C [-f] (-h <host>... | -i <addr>...) [-s <amount>] [-d <duration>]
  shadowd -h | --help

Options:
  -G  Generate and store hash-table for specified <token> and <password>.
      -n <amount>  Generate hash-table of specified length [default: 2048].
      -a <algo>    Use specified algorithm [default: sha256].
  -C  Generate certificate pair for authenticating via HTTPS.
      -f  Do not ask, overwrite existing certificate.
	  -s <amount>   Generate rsa key of specified length
				    [default: 2048].
	  -h <host>     Set specified host as verified verified.
	  -i <addr>     Set specified ip address as verified.
	  -d <duration> Set specified valid duration (hours).

  -d <table_dir>  Use specified dir for storing and reading hash-tables
                  [default: /var/shadowd/ht/].
  -c <cert_dir>   Use specified dir for storing and reading certificates
                  [default: /var/shadowd/cert/].`

func main() {
	args, _ := docopt.Parse(usage, nil, true, "shadowd 1.0", false)

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
