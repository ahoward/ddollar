package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	certFileName = "cert.pem"
	keyFileName  = "key.pem"
)

// CertInfo holds certificate metadata for display
type CertInfo struct {
	Domains       []string
	ValidFrom     time.Time
	ValidUntil    time.Time
	Issuer        string
	Fingerprint   string
	DaysRemaining int
}

// CertPaths returns the paths to the certificate and key files
func CertPaths() (string, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to get home directory: %w", err)
	}

	certsDir := filepath.Join(homeDir, ".ddollar", "certs")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create certs directory: %w", err)
	}

	certPath := filepath.Join(certsDir, certFileName)
	keyPath := filepath.Join(certsDir, keyFileName)

	return certPath, keyPath, nil
}

// GenerateCert generates SSL certificate for AI provider domains, signed by ddollar's CA
func GenerateCert() (certPath, keyPath string, err error) {
	certPath, keyPath, err = CertPaths()
	if err != nil {
		return "", "", err
	}

	// Check if certificate already exists and is valid
	if _, err := os.Stat(certPath); err == nil {
		if err := ValidateCert(certPath); err == nil {
			// Certificate exists and is valid
			return certPath, keyPath, nil
		}
		// Certificate exists but is invalid - regenerate
	}

	// Get or create CA
	ca, err := EnsureCA()
	if err != nil {
		return "", "", fmt.Errorf("failed to initialize CA: %w", err)
	}

	// Define domains to cover
	domains := []string{
		"api.openai.com",
		"api.anthropic.com",
		"api.cohere.ai",
		"generativelanguage.googleapis.com",
		"localhost",
	}

	// Generate certificate using mkcert
	certPEM, keyPEM, err := generateCertFromCA(ca, domains)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate certificate: %w", err)
	}

	// Write certificate file
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return "", "", fmt.Errorf("failed to write certificate file: %w", err)
	}

	// Write key file with restrictive permissions
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return "", "", fmt.Errorf("failed to write key file: %w", err)
	}

	return certPath, keyPath, nil
}

// generateCertFromCA generates a certificate signed by the CA
func generateCertFromCA(ca *CA, domains []string) (certPEM, keyPEM []byte, err error) {
	// Load CA certificate and key
	caCertPEM, err := os.ReadFile(ca.RootCAPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caKeyPEM, err := os.ReadFile(ca.RootCAKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CA key: %w", err)
	}

	// Parse CA certificate
	caCertBlock, _ := pem.Decode(caCertPEM)
	if caCertBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA certificate PEM")
	}

	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Parse CA private key
	caKeyBlock, _ := pem.Decode(caKeyPEM)
	if caKeyBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA key PEM")
	}

	caKey, err := x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CA key: %w", err)
	}

	// Use mkcert to generate the certificate
	// Note: mkcert library doesn't export MakeCert directly in a way we can use
	// So we'll use the standard crypto approach but signed by our CA
	return generateLeafCert(caCert, caKey, domains)
}

// generateLeafCert creates a leaf certificate signed by the CA
func generateLeafCert(caCert *x509.Certificate, caKey interface{}, domains []string) (certPEM, keyPEM []byte, err error) {
	// Import required packages for cert generation
	// This uses the same approach as mkcert internally

	// Generate RSA private key for the leaf certificate
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"ddollar"},
			CommonName:   domains[0], // Use first domain as CN
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year validity
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              domains,
	}

	// Create certificate signed by CA
	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &privateKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM, keyPEM, nil
}

// RegenerateCert forces regeneration of certificate
func RegenerateCert() error {
	certPath, keyPath, err := CertPaths()
	if err != nil {
		return err
	}

	// Remove existing certificates
	_ = os.Remove(certPath)
	_ = os.Remove(keyPath)

	// Generate new certificate
	_, _, err = GenerateCert()
	return err
}

// ValidateCert verifies existing certificate is valid and covers required domains
func ValidateCert(certPath string) error {
	// Load certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("certificate file not found: %w", err)
	}

	// Parse certificate
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("certificate is not valid PEM format")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Check validity period
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate not yet valid (valid from: %s)", cert.NotBefore)
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate expired on %s", cert.NotAfter)
	}

	// Check required domains
	requiredDomains := []string{
		"api.openai.com",
		"api.anthropic.com",
		"api.cohere.ai",
		"generativelanguage.googleapis.com",
		"localhost",
	}

	for _, required := range requiredDomains {
		found := false
		for _, domain := range cert.DNSNames {
			if domain == required {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("certificate does not cover required domain: %s", required)
		}
	}

	return nil
}

// GetCertInfo extracts certificate metadata for display
func GetCertInfo(certPath string) (*CertInfo, error) {
	// Load certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	// Parse certificate
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Calculate days remaining
	daysRemaining := int(time.Until(cert.NotAfter).Hours() / 24)

	// Calculate fingerprint
	hash := sha256.Sum256(block.Bytes)
	fingerprint := hex.EncodeToString(hash[:])

	return &CertInfo{
		Domains:       cert.DNSNames,
		ValidFrom:     cert.NotBefore,
		ValidUntil:    cert.NotAfter,
		Issuer:        cert.Issuer.CommonName,
		Fingerprint:   fingerprint,
		DaysRemaining: daysRemaining,
	}, nil
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

// LoadCertificate loads the certificate for TLS configuration
func LoadCertificate() (tls.Certificate, error) {
	certPath, keyPath, err := CertPaths()
	if err != nil {
		return tls.Certificate{}, err
	}

	// Validate certificate before loading
	if err := ValidateCert(certPath); err != nil {
		return tls.Certificate{}, fmt.Errorf("certificate validation failed: %w", err)
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load certificate: %w", err)
	}

	return cert, nil
}

// PrintManualInstructions prints platform-specific manual trust instructions
func PrintManualInstructions() {
	homeDir, _ := os.UserHomeDir()
	caPath := filepath.Join(homeDir, ".ddollar", "ca", "rootCA.pem")

	fmt.Println("\nManual trust instructions:")
	fmt.Printf("  Certificate location: %s\n\n", caPath)

	// Detect platform and print specific instructions
	switch {
	case fileExists("/Library/Keychains/System.keychain"):
		// macOS
		fmt.Println("macOS:")
		fmt.Printf("  sudo security add-trusted-cert -d -r trustRoot \\\n")
		fmt.Printf("      -k /Library/Keychains/System.keychain \\\n")
		fmt.Printf("      %s\n", caPath)

	case fileExists("/etc/debian_version"):
		// Debian/Ubuntu
		fmt.Println("Debian/Ubuntu:")
		fmt.Printf("  sudo cp %s \\\n", caPath)
		fmt.Printf("      /usr/local/share/ca-certificates/ddollar.crt\n")
		fmt.Println("  sudo update-ca-certificates")

	case fileExists("/etc/redhat-release"):
		// RHEL/Fedora
		fmt.Println("RHEL/Fedora:")
		fmt.Printf("  sudo cp %s \\\n", caPath)
		fmt.Printf("      /etc/pki/ca-trust/source/anchors/ddollar.pem\n")
		fmt.Println("  sudo update-ca-trust")

	default:
		// Generic Linux or other
		fmt.Println("Linux:")
		fmt.Println("  1. Import the CA certificate to your system trust store")
		fmt.Println("  2. Run the appropriate trust update command for your distribution")
	}

	// Firefox/NSS instructions
	nssDB := filepath.Join(homeDir, ".pki", "nssdb")
	if fileExists(nssDB) {
		fmt.Println("\nFirefox (NSS):")
		fmt.Printf("  certutil -A -n \"ddollar Local CA\" -t \"C,,\" \\\n")
		fmt.Printf("      -d sql:%s \\\n", nssDB)
		fmt.Printf("      -i %s\n", caPath)
	}

	fmt.Println("\nVerify with:")
	fmt.Println("  ddollar status")
}

// fileExists checks if a file or directory exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// FormatDomains formats a list of domains for display
func FormatDomains(domains []string) string {
	return strings.Join(domains, ", ")
}
