package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const (
	certFileName = "cert.pem"
	keyFileName  = "key.pem"
)

// CertPaths returns the paths to the certificate and key files
func CertPaths() (string, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to get home directory: %w", err)
	}

	ddollarDir := filepath.Join(homeDir, ".ddollar")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(ddollarDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create .ddollar directory: %w", err)
	}

	certPath := filepath.Join(ddollarDir, certFileName)
	keyPath := filepath.Join(ddollarDir, keyFileName)

	return certPath, keyPath, nil
}

// GenerateCert generates a self-signed certificate for HTTPS interception
func GenerateCert() (certPath, keyPath string, err error) {
	certPath, keyPath, err = CertPaths()
	if err != nil {
		return "", "", err
	}

	// Check if certificate already exists
	if _, err := os.Stat(certPath); err == nil {
		return certPath, keyPath, nil
	}

	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"ddollar"},
			CommonName:   "ddollar proxy",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames: []string{
			"api.openai.com",
			"api.anthropic.com",
			"api.cohere.ai",
			"generativelanguage.googleapis.com",
			"localhost",
		},
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	// Write certificate to file
	certFile, err := os.Create(certPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create cert file: %w", err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return "", "", fmt.Errorf("failed to encode certificate: %w", err)
	}

	// Write private key to file
	keyFile, err := os.Create(keyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		return "", "", fmt.Errorf("failed to encode private key: %w", err)
	}

	return certPath, keyPath, nil
}

// HasCert checks if a certificate already exists
func HasCert() bool {
	certPath, keyPath, err := CertPaths()
	if err != nil {
		return false
	}

	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	return certErr == nil && keyErr == nil
}
