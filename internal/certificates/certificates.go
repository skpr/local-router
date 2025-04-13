package certificates

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const caCertFile = "ca-cert.pem"
const caKeyFile = "ca-key.pem"

type Manager struct {
	lock   sync.Mutex
	caCert *x509.Certificate
	caKey  *rsa.PrivateKey
	path   string
	cache  map[string]*tls.Certificate
}

func NewManager(path string) (*Manager, error) {
	caCert, caKey, err := loadOrCreateCA(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load or create CA: %w", err)
	}

	manager := &Manager{
		cache:  make(map[string]*tls.Certificate),
		caCert: caCert,
		caKey:  caKey,
		path:   path,
	}

	return manager, nil
}

func (m *Manager) GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	domain := clientHello.ServerName
	if domain == "" {
		domain = "localhost"
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	if cert, ok := m.cache[domain]; ok {
		return cert, nil
	}

	var (
		certFile = filepath.Join(m.path, fmt.Sprintf("%s-cert.pem", domain))
		keyFile  = filepath.Join(m.path, fmt.Sprintf("%s-key.pem", domain))
	)

	if _, err := os.Stat(certFile); err == nil {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err == nil {
			m.cache[domain] = &cert
			return &cert, nil
		}
		log.Printf("Failed to load cert from disk, regenerating: %v", err)
	}

	log.Printf("Generating certificate for %s signed by local CA...\n", domain)

	cert, err := m.generateCertSignedByCA(domain)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0644)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(cert.PrivateKey.(*rsa.PrivateKey))}), 0600)
	if err != nil {
		return nil, err
	}

	m.cache[domain] = cert

	return cert, nil
}

func loadOrCreateCA(path string) (*x509.Certificate, *rsa.PrivateKey, error) {
	var (
		certPath = filepath.Join(path, caCertFile)
		keyPath  = filepath.Join(path, caKeyFile)
	)

	// Load existing CA
	if certPEM, err := os.ReadFile(certPath); err == nil {
		if keyPEM, err := os.ReadFile(keyPath); err == nil {
			block, _ := pem.Decode(certPEM)
			if block == nil {
				return nil, nil, fmt.Errorf("failed to decode CA cert PEM")
			}

			caCert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, err
			}

			block, _ = pem.Decode(keyPEM)
			if block == nil {
				return nil, nil, fmt.Errorf("failed to decode CA key PEM")
			}

			caKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, err
			}

			return caCert, caKey, nil
		}
	}

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{
				"Local Dev CA",
			},
			CommonName: "Local Dev CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}

	caCert := caTemplate

	err = os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes}), 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write CA file: %w", err)
	}

	err = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)}), 0600)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write private file: %w", err)
	}

	return caCert, caKey, err
}

func (m *Manager) generateCertSignedByCA(domain string) (*tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: domain,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{domain},
		IsCA:        false,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, m.caCert, &priv.PublicKey, m.caKey)
	if err != nil {
		return nil, err
	}

	var (
		certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
		keyPEM  = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	)

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tlsCert, nil
}
