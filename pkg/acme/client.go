package acme

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"

	"github.com/last-emo-boy/infra-core/pkg/config"
)

// Client represents an ACME client for automatic certificate management
type Client struct {
	config     *config.Config
	legoClient *lego.Client
	user       *User
	certDir    string
	mu         sync.RWMutex

	// Certificate cache
	certificates map[string]*tls.Certificate
	certFiles    map[string]*CertificateFiles
}

// User represents an ACME user
type User struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
	key          crypto.PrivateKey
}

// CertificateFiles represents certificate file paths
type CertificateFiles struct {
	Domain     string    `json:"domain"`
	CertPath   string    `json:"cert_path"`
	KeyPath    string    `json:"key_path"`
	IssuerPath string    `json:"issuer_path"`
	NotAfter   time.Time `json:"not_after"`
	NotBefore  time.Time `json:"not_before"`
	Created    time.Time `json:"created"`
	Updated    time.Time `json:"updated"`
}

// GetEmail returns user email
func (u *User) GetEmail() string {
	return u.Email
}

// GetRegistration returns user registration
func (u *User) GetRegistration() *registration.Resource {
	return u.Registration
}

// GetPrivateKey returns user private key
func (u *User) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// NewClient creates a new ACME client
func NewClient(cfg *config.Config) (*Client, error) {
	if cfg.Gate.ACME.Email == "" {
		return nil, fmt.Errorf("ACME email is required")
	}

	certDir := cfg.Gate.ACME.CacheDir
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cert directory: %w", err)
	}

	client := &Client{
		config:       cfg,
		certDir:      certDir,
		certificates: make(map[string]*tls.Certificate),
		certFiles:    make(map[string]*CertificateFiles),
	}

	// Load or create user
	user, err := client.loadOrCreateUser()
	if err != nil {
		return nil, fmt.Errorf("failed to load/create user: %w", err)
	}
	client.user = user

	// Create lego client
	legoConfig := lego.NewConfig(user)
	if cfg.Gate.ACME.DirectoryURL != "" {
		legoConfig.CADirURL = cfg.Gate.ACME.DirectoryURL
	} else {
		legoConfig.CADirURL = lego.LEDirectoryProduction
	}

	legoClient, err := lego.NewClient(legoConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create lego client: %w", err)
	}

	client.legoClient = legoClient

	// Setup HTTP-01 challenge
	err = legoClient.Challenge.SetHTTP01Provider(&HTTP01Provider{
		client: client,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to setup HTTP-01 provider: %w", err)
	}

	// Register user if needed
	if user.Registration == nil {
		reg, err := legoClient.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return nil, fmt.Errorf("failed to register user: %w", err)
		}
		user.Registration = reg

		// Save user registration
		if err := client.saveUser(user); err != nil {
			return nil, fmt.Errorf("failed to save user registration: %w", err)
		}
	}

	// Load existing certificates
	if err := client.loadCertificates(); err != nil {
		return nil, fmt.Errorf("failed to load certificates: %w", err)
	}

	return client, nil
}

// loadOrCreateUser loads existing user or creates a new one
func (c *Client) loadOrCreateUser() (*User, error) {
	userPath := filepath.Join(c.certDir, "user.json")
	keyPath := filepath.Join(c.certDir, "user.key")

	// Try to load existing user
	if fileExists(userPath) && fileExists(keyPath) {
		userData, err := os.ReadFile(userPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read user file: %w", err)
		}

		var user User
		if err := json.Unmarshal(userData, &user); err != nil {
			return nil, fmt.Errorf("failed to parse user file: %w", err)
		}

		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read user key: %w", err)
		}

		keyBlock, _ := pem.Decode(keyData)
		if keyBlock == nil {
			return nil, fmt.Errorf("failed to decode user key")
		}

		privateKey, err := x509.ParseECPrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse user key: %w", err)
		}

		user.key = privateKey
		return &user, nil
	}

	// Create new user
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	user := &User{
		Email: c.config.Gate.ACME.Email,
		key:   privateKey,
	}

	return user, c.saveUser(user)
}

// saveUser saves user data to disk
func (c *Client) saveUser(user *User) error {
	userPath := filepath.Join(c.certDir, "user.json")
	keyPath := filepath.Join(c.certDir, "user.key")

	// Save user data
	userData, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	if err := os.WriteFile(userPath, userData, 0600); err != nil {
		return fmt.Errorf("failed to write user file: %w", err)
	}

	// Save private key
	privateKey := user.key.(*ecdsa.PrivateKey)
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

// IssueCertificate issues a new certificate for the given domains
func (c *Client) IssueCertificate(domains []string) error {
	if len(domains) == 0 {
		return fmt.Errorf("no domains specified")
	}

	primaryDomain := domains[0]

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if certificate already exists and is valid
	if certFile, exists := c.certFiles[primaryDomain]; exists {
		if time.Now().Before(certFile.NotAfter.Add(-30 * 24 * time.Hour)) {
			return nil // Certificate is still valid (more than 30 days left)
		}
	}

	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}

	certificates, err := c.legoClient.Certificate.Obtain(request)
	if err != nil {
		return fmt.Errorf("failed to obtain certificate: %w", err)
	}

	// Save certificate files
	certFile, err := c.saveCertificate(primaryDomain, certificates)
	if err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}

	// Load certificate into memory
	cert, err := c.loadCertificate(certFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	c.certificates[primaryDomain] = cert
	c.certFiles[primaryDomain] = certFile

	return nil
}

// GetCertificate returns certificate for the given domain
func (c *Client) GetCertificate(domain string) (*tls.Certificate, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cert, exists := c.certificates[domain]
	if !exists {
		return nil, fmt.Errorf("certificate not found for domain: %s", domain)
	}

	return cert, nil
}

// ListCertificates returns all certificates
func (c *Client) ListCertificates() map[string]*CertificateFiles {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*CertificateFiles)
	for k, v := range c.certFiles {
		result[k] = v
	}
	return result
}

// RenewExpiring renews certificates that are expiring soon
func (c *Client) RenewExpiring() error {
	c.mu.RLock()
	expiring := make([]string, 0)
	for domain, certFile := range c.certFiles {
		if time.Now().After(certFile.NotAfter.Add(-30 * 24 * time.Hour)) {
			expiring = append(expiring, domain)
		}
	}
	c.mu.RUnlock()

	for _, domain := range expiring {
		if err := c.IssueCertificate([]string{domain}); err != nil {
			return fmt.Errorf("failed to renew certificate for %s: %w", domain, err)
		}
	}

	return nil
}

// saveCertificate saves certificate to disk
func (c *Client) saveCertificate(domain string, certificates *certificate.Resource) (*CertificateFiles, error) {
	certPath := filepath.Join(c.certDir, domain+".crt")
	keyPath := filepath.Join(c.certDir, domain+".key")
	issuerPath := filepath.Join(c.certDir, domain+".issuer.crt")

	// Write certificate
	if err := os.WriteFile(certPath, certificates.Certificate, 0644); err != nil {
		return nil, fmt.Errorf("failed to write certificate: %w", err)
	}

	// Write private key
	if err := os.WriteFile(keyPath, certificates.PrivateKey, 0600); err != nil {
		return nil, fmt.Errorf("failed to write private key: %w", err)
	}

	// Write issuer certificate
	if err := os.WriteFile(issuerPath, certificates.IssuerCertificate, 0644); err != nil {
		return nil, fmt.Errorf("failed to write issuer certificate: %w", err)
	}

	// Parse certificate to get validity dates
	certBlock, _ := pem.Decode(certificates.Certificate)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode certificate")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	certFile := &CertificateFiles{
		Domain:     domain,
		CertPath:   certPath,
		KeyPath:    keyPath,
		IssuerPath: issuerPath,
		NotAfter:   cert.NotAfter,
		NotBefore:  cert.NotBefore,
		Created:    time.Now(),
		Updated:    time.Now(),
	}

	return certFile, nil
}

// loadCertificates loads all certificates from disk
func (c *Client) loadCertificates() error {
	files, err := os.ReadDir(c.certDir)
	if err != nil {
		return nil // Directory doesn't exist or is empty
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".crt") && !strings.Contains(file.Name(), ".issuer.") {
			domain := strings.TrimSuffix(file.Name(), ".crt")
			certPath := filepath.Join(c.certDir, file.Name())
			keyPath := filepath.Join(c.certDir, domain+".key")

			if !fileExists(keyPath) {
				continue
			}

			certFile := &CertificateFiles{
				Domain:   domain,
				CertPath: certPath,
				KeyPath:  keyPath,
			}

			cert, err := c.loadCertificate(certFile)
			if err != nil {
				continue // Skip invalid certificates
			}

			// Get certificate validity dates
			if len(cert.Certificate) > 0 {
				x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
				if err == nil {
					certFile.NotAfter = x509Cert.NotAfter
					certFile.NotBefore = x509Cert.NotBefore
				}
			}

			c.certificates[domain] = cert
			c.certFiles[domain] = certFile
		}
	}

	return nil
}

// loadCertificate loads a certificate from files
func (c *Client) loadCertificate(certFile *CertificateFiles) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile.CertPath, certFile.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}
	return &cert, nil
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
