package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func handleCertificateGenerate(
	backend Backend, args map[string]interface{},
) error {
	var (
		certsDir        = args["--certs"].(string)
		rsaBlockSizeRaw = args["--bytes"].(string)
		validTill       = args["--till"].(string)
		hosts           = args["--host"].([]string)
		addresses       = args["--address"].([]string)
	)

	rsaBlockSize, err := strconv.Atoi(rsaBlockSizeRaw)
	if err != nil {
		return err
	}

	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		err = os.MkdirAll(certsDir, 0700)
		if err != nil {
			return err
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, rsaBlockSize)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %s", err)
	}

	invalidAfter, err := time.Parse("2006-02-01", validTill)
	if err != nil {
		return err
	}

	invalidBefore := time.Now()

	serialNumberBlockSize := big.NewInt(0).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberBlockSize)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %s", err)
	}

	cert := x509.Certificate{
		IsCA: true,

		Subject:      pkix.Name{},
		SerialNumber: serialNumber,

		NotBefore: invalidBefore,
		NotAfter:  invalidAfter,

		BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},

		DNSNames: hosts,
	}

	for _, address := range addresses {
		if addr := net.ParseIP(address); addr != nil {
			cert.IPAddresses = append(cert.IPAddresses, addr)
		}

	}

	certData, err := x509.CreateCertificate(
		rand.Reader, &cert, &cert, &privateKey.PublicKey, privateKey,
	)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %s", err)
	}

	certOutFd, err := os.Create(filepath.Join(certsDir, "cert.pem"))
	if err != nil {
		return fmt.Errorf("failed to create cert file: %s", err)
	}

	err = pem.Encode(
		certOutFd,
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certData,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write to cert file: %s", err)
	}

	err = certOutFd.Close()
	if err != nil {
		return fmt.Errorf("failed to close cert file: %s", err)
	}

	keyOutFd, err := os.OpenFile(
		filepath.Join(certsDir, "key.pem"),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0600,
	)
	if err != nil {
		return fmt.Errorf("failed to create key file: %s", err)
	}

	err = pem.Encode(
		keyOutFd,
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write to key file: %s", err)
	}

	err = keyOutFd.Close()
	if err != nil {
		return fmt.Errorf("failed to close key file: %s", err)
	}

	return nil
}
