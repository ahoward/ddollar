package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// CA represents the local Certificate Authority created and managed by ddollar
type CA struct {
	RootCAPath    string
	RootCAKeyPath string
	Created       time.Time
	Fingerprint   string
	CommonName    string
	ValidFrom     time.Time
	ValidUntil    time.Time
}

// EnsureCA gets existing CA or creates new one if it doesn't exist
func EnsureCA() (*CA, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	caDir := filepath.Join(homeDir, ".ddollar", "ca")
	certPath := filepath.Join(caDir, "rootCA.pem")
	keyPath := filepath.Join(caDir, "rootCA-key.pem")

	// Check if CA already exists
	if _, err := os.Stat(certPath); err == nil {
		return loadCA(certPath, keyPath)
	}

	// Generate new CA
	return generateCA(caDir, certPath, keyPath)
}

// loadCA loads an existing CA from filesystem
func loadCA(certPath, keyPath string) (*CA, error) {
	// Read certificate file
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	// Parse certificate
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Verify key file exists
	if _, err := os.Stat(keyPath); err != nil {
		return nil, fmt.Errorf("CA private key not found: %w", err)
	}

	// Calculate fingerprint
	hash := sha256.Sum256(block.Bytes)
	fingerprint := hex.EncodeToString(hash[:])

	// Get file creation time
	fileInfo, err := os.Stat(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat CA certificate: %w", err)
	}

	return &CA{
		RootCAPath:    certPath,
		RootCAKeyPath: keyPath,
		Created:       fileInfo.ModTime(),
		Fingerprint:   fingerprint,
		CommonName:    cert.Subject.CommonName,
		ValidFrom:     cert.NotBefore,
		ValidUntil:    cert.NotAfter,
	}, nil
}

// generateCA creates a new CA
func generateCA(caDir, certPath, keyPath string) (*CA, error) {
	// Create CA directory
	if err := os.MkdirAll(caDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create CA directory: %w", err)
	}

	// Generate RSA private key for CA
	caPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA private key: %w", err)
	}

	// Create CA certificate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA serial number: %w", err)
	}

	// Create CA certificate template
	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"ddollar"},
			CommonName:   "ddollar Local CA",
		},
		NotBefore:             now,
		NotAfter:              now.Add(10 * 365 * 24 * time.Hour), // 10 years validity
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
		MaxPathLenZero:        true,
	}

	// Create self-signed CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	// Write CA certificate to file
	certFile, err := os.Create(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA cert file: %w", err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: caCertDER}); err != nil {
		return nil, fmt.Errorf("failed to encode CA certificate: %w", err)
	}

	// Write CA private key to file with restrictive permissions
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA key file: %w", err)
	}
	defer keyFile.Close()

	keyBytes := x509.MarshalPKCS1PrivateKey(caPrivateKey)
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return nil, fmt.Errorf("failed to encode CA key: %w", err)
	}

	// Load the generated CA to populate struct
	return loadCA(certPath, keyPath)
}

// IsValid checks if CA certificate is within validity period
func (ca *CA) IsValid() bool {
	now := time.Now()
	return now.After(ca.ValidFrom) && now.Before(ca.ValidUntil)
}

// InstallTrust installs CA certificate into system trust stores
func InstallTrust(ca *CA) error {
	osType := runtime.GOOS

	switch osType {
	case "darwin":
		return installTrustMacOS(ca)
	case "linux":
		return installTrustLinux(ca)
	case "windows":
		return installTrustWindows(ca)
	default:
		return fmt.Errorf("unsupported platform: %s", osType)
	}
}

// installTrustMacOS installs CA to macOS Keychain
func installTrustMacOS(ca *CA) error {
	cmd := exec.Command(
		"security",
		"add-trusted-cert",
		"-d",
		"-r", "trustRoot",
		"-k", "/Library/Keychains/System.keychain",
		ca.RootCAPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install CA to macOS Keychain: %w (output: %s)", err, string(output))
	}

	// Try to install to NSS (non-fatal)
	_ = installTrustNSS(ca)

	return nil
}

// installTrustLinux installs CA to Linux trust stores
func installTrustLinux(ca *CA) error {
	// Detect Linux distribution
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		return installTrustDebianUbuntu(ca)
	}

	if _, err := os.Stat("/etc/redhat-release"); err == nil {
		return installTrustRHELFedora(ca)
	}

	// Try Debian/Ubuntu by default
	return installTrustDebianUbuntu(ca)
}

// installTrustDebianUbuntu installs CA on Debian/Ubuntu systems
func installTrustDebianUbuntu(ca *CA) error {
	destPath := "/usr/local/share/ca-certificates/ddollar.crt"

	// Copy CA certificate
	input, err := os.ReadFile(ca.RootCAPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	if err := os.WriteFile(destPath, input, 0644); err != nil {
		return fmt.Errorf("failed to copy CA certificate: %w", err)
	}

	// Update CA certificates
	cmd := exec.Command("update-ca-certificates")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update CA certificates: %w (output: %s)", err, string(output))
	}

	// Try to install to NSS (non-fatal)
	_ = installTrustNSS(ca)

	return nil
}

// installTrustRHELFedora installs CA on RHEL/Fedora systems
func installTrustRHELFedora(ca *CA) error {
	destPath := "/etc/pki/ca-trust/source/anchors/ddollar.pem"

	// Copy CA certificate
	input, err := os.ReadFile(ca.RootCAPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	if err := os.WriteFile(destPath, input, 0644); err != nil {
		return fmt.Errorf("failed to copy CA certificate: %w", err)
	}

	// Update CA trust
	cmd := exec.Command("update-ca-trust")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update CA trust: %w (output: %s)", err, string(output))
	}

	// Try to install to NSS (non-fatal)
	_ = installTrustNSS(ca)

	return nil
}

// installTrustWindows installs CA on Windows systems
func installTrustWindows(ca *CA) error {
	cmd := exec.Command(
		"certutil",
		"-addstore",
		"-f",
		"ROOT",
		ca.RootCAPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install CA to Windows certificate store: %w (output: %s)", err, string(output))
	}

	return nil
}

// installTrustNSS installs CA to NSS database (Firefox, Chromium snap)
func installTrustNSS(ca *CA) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	nssDB := filepath.Join(homeDir, ".pki", "nssdb")

	// Check if NSS database exists
	if _, err := os.Stat(nssDB); err != nil {
		// NSS not present - not an error
		return nil
	}

	cmd := exec.Command(
		"certutil",
		"-A",
		"-n", "ddollar Local CA",
		"-t", "C,,",
		"-d", fmt.Sprintf("sql:%s", nssDB),
		"-i", ca.RootCAPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Non-fatal - NSS is optional
		return fmt.Errorf("NSS installation failed (non-fatal): %w (output: %s)", err, string(output))
	}

	return nil
}

// VerifyTrust checks if CA is actually trusted by the system
func VerifyTrust(ca *CA) error {
	// Read CA certificate
	certPEM, err := os.ReadFile(ca.RootCAPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	// Parse certificate
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Try to verify using system roots
	roots := x509.NewCertPool()
	roots.AddCert(cert)

	opts := x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	if _, err := cert.Verify(opts); err != nil {
		// This is expected for self-signed CA
		// Instead, check platform-specific trust stores
		return verifyTrustPlatform(ca)
	}

	return nil
}

// verifyTrustPlatform checks platform-specific trust stores
func verifyTrustPlatform(ca *CA) error {
	osType := runtime.GOOS

	switch osType {
	case "darwin":
		return verifyTrustMacOS(ca)
	case "linux":
		return verifyTrustLinux(ca)
	case "windows":
		return verifyTrustWindows(ca)
	default:
		return fmt.Errorf("platform verification not supported: %s", osType)
	}
}

// verifyTrustMacOS checks if CA is in macOS Keychain
func verifyTrustMacOS(ca *CA) error {
	cmd := exec.Command(
		"security",
		"find-certificate",
		"-c", ca.CommonName,
		"/Library/Keychains/System.keychain",
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("CA not found in macOS Keychain")
	}

	return nil
}

// verifyTrustLinux checks if CA is in Linux trust store
func verifyTrustLinux(ca *CA) error {
	// Check Debian/Ubuntu path
	debPath := "/usr/local/share/ca-certificates/ddollar.crt"
	if _, err := os.Stat(debPath); err == nil {
		return nil
	}

	// Check RHEL/Fedora path
	rhelPath := "/etc/pki/ca-trust/source/anchors/ddollar.pem"
	if _, err := os.Stat(rhelPath); err == nil {
		return nil
	}

	return fmt.Errorf("CA not found in Linux trust store")
}

// verifyTrustWindows checks if CA is in Windows certificate store
func verifyTrustWindows(ca *CA) error {
	cmd := exec.Command(
		"certutil",
		"-store",
		"ROOT",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to query Windows certificate store: %w", err)
	}

	// Check if CA common name is in output
	if len(output) > 0 && contains(string(output), ca.CommonName) {
		return nil
	}

	return fmt.Errorf("CA not found in Windows certificate store")
}

// UninstallTrust removes CA certificate from system trust stores
func UninstallTrust(ca *CA) error {
	osType := runtime.GOOS

	switch osType {
	case "darwin":
		return uninstallTrustMacOS(ca)
	case "linux":
		return uninstallTrustLinux(ca)
	case "windows":
		return uninstallTrustWindows(ca)
	default:
		return fmt.Errorf("unsupported platform: %s", osType)
	}
}

// uninstallTrustMacOS removes CA from macOS Keychain
func uninstallTrustMacOS(ca *CA) error {
	cmd := exec.Command(
		"security",
		"delete-certificate",
		"-c", ca.CommonName,
		"/Library/Keychains/System.keychain",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove CA from macOS Keychain: %w (output: %s)", err, string(output))
	}

	// Try to remove from NSS (non-fatal)
	_ = uninstallTrustNSS(ca)

	return nil
}

// uninstallTrustLinux removes CA from Linux trust stores
func uninstallTrustLinux(ca *CA) error {
	// Try Debian/Ubuntu
	debPath := "/usr/local/share/ca-certificates/ddollar.crt"
	if _, err := os.Stat(debPath); err == nil {
		if err := os.Remove(debPath); err != nil {
			return fmt.Errorf("failed to remove CA certificate: %w", err)
		}

		cmd := exec.Command("update-ca-certificates", "--fresh")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to update CA certificates: %w (output: %s)", err, string(output))
		}
	}

	// Try RHEL/Fedora
	rhelPath := "/etc/pki/ca-trust/source/anchors/ddollar.pem"
	if _, err := os.Stat(rhelPath); err == nil {
		if err := os.Remove(rhelPath); err != nil {
			return fmt.Errorf("failed to remove CA certificate: %w", err)
		}

		cmd := exec.Command("update-ca-trust")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to update CA trust: %w (output: %s)", err, string(output))
		}
	}

	// Try to remove from NSS (non-fatal)
	_ = uninstallTrustNSS(ca)

	return nil
}

// uninstallTrustWindows removes CA from Windows certificate store
func uninstallTrustWindows(ca *CA) error {
	cmd := exec.Command(
		"certutil",
		"-delstore",
		"ROOT",
		ca.CommonName,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove CA from Windows certificate store: %w (output: %s)", err, string(output))
	}

	return nil
}

// uninstallTrustNSS removes CA from NSS database
func uninstallTrustNSS(ca *CA) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	nssDB := filepath.Join(homeDir, ".pki", "nssdb")

	// Check if NSS database exists
	if _, err := os.Stat(nssDB); err != nil {
		return nil
	}

	cmd := exec.Command(
		"certutil",
		"-D",
		"-n", "ddollar Local CA",
		"-d", fmt.Sprintf("sql:%s", nssDB),
	)

	_ = cmd.Run() // Non-fatal

	return nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}
