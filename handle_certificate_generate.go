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
	"strconv"
	"time"
)

func handleCertificateGenerate(args map[string]interface{}) error {
	rsaBlockSize, err := strconv.Atoi(args["-s"].(string))
	if err != nil {
		return err
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, rsaBlockSize)
	if err != nil {
		return fmt.Errorf("Failed to generate private key: %s", err)
	}

	validDuration, err := time.ParseDuration(args["-d"].(string))
	if err != nil {
		return err
	}

	validNotBefore := time.Now()
	validNotAfter := validNotBefore.Add(validDuration)

	serialNumberBlockSize := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberBlockSize)
	if err != nil {
		return fmt.Errorf("Failed to generate serial number: %s", err)
	}

	certicate := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{},
		NotBefore:             validNotBefore,
		NotAfter:              validNotAfter,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := args["-h"].([]string)
	certicate.DNSNames = hosts

	addrs := args["-i"].([]string)
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil {
			certicate.IPAddresses = append(certicate.IPAddresses, ip)
		}

	}

	keyUsage := x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	if args["-ca"].(bool) {
		certicate.IsCA = true
		keyUsage |= x509.KeyUsageCertSign
	}
	certicate.KeyUsage = keyUsage

	certBytes, err := x509.CreateCertificate(
		rand.Reader, &certicate, &certicate, &privateKey.PublicKey, privateKey,
	)
	if err != nil {
		return fmt.Errorf("Failed to create certificate: %s", err)
	}

	certOutFd, err := os.Create("cert.pem")
	if err != nil {
		return fmt.Errorf("Failed to create cert file: %s", err)
	}

	err = pem.Encode(
		certOutFd,
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certBytes,
		},
	)
	if err != nil {
		return fmt.Errorf("Failed to write to cert file: %s", err)
	}

	err = certOutFd.Close()
	if err != nil {
		return fmt.Errorf("Failed to close cert file: %s", err)
	}

	keyOutFd, err := os.OpenFile(
		"key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600,
	)
	if err != nil {
		return fmt.Errorf("Failed to create key file: %s", err)
	}

	err = pem.Encode(
		keyOutFd,
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		},
	)
	if err != nil {
		return fmt.Errorf("Failed to write to key file: %s", err)
	}

	err = keyOutFd.Close()
	if err != nil {
		return fmt.Errorf("Failed to close key file: %s", err)
	}

	return nil
}
