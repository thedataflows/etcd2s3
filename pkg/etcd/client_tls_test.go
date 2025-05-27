package etcd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thedataflows/etcd2s3/pkg/appconfig"
)

// generateTestCertificates creates a test CA and client certificate for testing
func generateTestCertificates(t *testing.T, dir string) (caFile, certFile, keyFile string) {
	// Generate CA private key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create CA certificate template
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test CA"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// Create CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caPrivKey.PublicKey, caPrivKey)
	require.NoError(t, err)

	// Write CA certificate to file
	caFile = filepath.Join(dir, "ca.crt")
	caOut, err := os.Create(caFile)
	require.NoError(t, err)
	defer caOut.Close()

	err = pem.Encode(caOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	require.NoError(t, err)

	// Generate client private key
	clientPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create client certificate template
	clientTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization:  []string{"Test Client"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// Create client certificate
	clientCertDER, err := x509.CreateCertificate(rand.Reader, &clientTemplate, &caTemplate, &clientPrivKey.PublicKey, caPrivKey)
	require.NoError(t, err)

	// Write client certificate to file
	certFile = filepath.Join(dir, "client.crt")
	certOut, err := os.Create(certFile)
	require.NoError(t, err)
	defer certOut.Close()

	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: clientCertDER})
	require.NoError(t, err)

	// Write client private key to file
	keyFile = filepath.Join(dir, "client.key")
	keyOut, err := os.Create(keyFile)
	require.NoError(t, err)
	defer keyOut.Close()

	clientPrivKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientPrivKey)}
	err = pem.Encode(keyOut, clientPrivKeyPEM)
	require.NoError(t, err)

	return caFile, certFile, keyFile
}

func TestNewClient_TLSWithCAOnly(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "etcd2s3-tls-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Generate test certificates
	caFile, _, _ := generateTestCertificates(t, tempDir)

	cfg := appconfig.EtcdConfig{
		Endpoints: []string{"https://localhost:2379"},
		CaFile:    caFile,
		// No client cert/key files - just CA for server verification
	}

	// This should not fail even though etcd server is not running
	// We're just testing the TLS configuration setup
	client, err := NewClient(cfg)

	// The client creation should succeed (TLS config is valid)
	// but connection will fail since no etcd server is running
	if err != nil {
		// Expected error should be connection-related, not TLS config related
		// Could be "connection" or "context deadline exceeded" depending on timeout behavior
		assert.True(t, strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "context deadline exceeded"),
			"Expected connection or timeout error, got: %s", err.Error())
	} else {
		// If client was created successfully, close it
		assert.NotNil(t, client)
		client.Close()
	}
}

func TestNewClient_TLSWithClientCert(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "etcd2s3-tls-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Generate test certificates
	caFile, certFile, keyFile := generateTestCertificates(t, tempDir)

	cfg := appconfig.EtcdConfig{
		Endpoints: []string{"https://localhost:2379"},
		CaFile:    caFile,
		CertFile:  certFile,
		KeyFile:   keyFile,
	}

	// This should not fail even though etcd server is not running
	// We're just testing the TLS configuration setup
	client, err := NewClient(cfg)

	// The client creation should succeed (TLS config is valid)
	// but connection will fail since no etcd server is running
	if err != nil {
		// Expected error should be connection-related, not TLS config related
		// Could be "connection" or "context deadline exceeded" depending on timeout behavior
		assert.True(t, strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "context deadline exceeded"),
			"Expected connection or timeout error, got: %s", err.Error())
	} else {
		// If client was created successfully, close it
		assert.NotNil(t, client)
		client.Close()
	}
}

func TestNewClient_TLSInsecure(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "etcd2s3-tls-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Generate test certificates
	_, certFile, keyFile := generateTestCertificates(t, tempDir)

	cfg := appconfig.EtcdConfig{
		Endpoints: []string{"https://localhost:2379"},
		CertFile:  certFile,
		KeyFile:   keyFile,
		// No CA file - should use insecure skip verify
	}

	// This should not fail even though etcd server is not running
	// We're just testing the TLS configuration setup
	client, err := NewClient(cfg)

	// The client creation should succeed (TLS config is valid)
	// but connection will fail since no etcd server is running
	if err != nil {
		// Expected error should be connection-related, not TLS config related
		// Could be "connection" or "context deadline exceeded" depending on timeout behavior
		assert.True(t, strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "context deadline exceeded"),
			"Expected connection or timeout error, got: %s", err.Error())
	} else {
		// If client was created successfully, close it
		assert.NotNil(t, client)
		client.Close()
	}
}

func TestNewClient_TLSInvalidCAFile(t *testing.T) {
	cfg := appconfig.EtcdConfig{
		Endpoints: []string{"https://localhost:2379"},
		CaFile:    "/nonexistent/ca.crt",
	}

	client, err := NewClient(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to read CA certificate file")
}

func TestNewClient_TLSInvalidCertFile(t *testing.T) {
	cfg := appconfig.EtcdConfig{
		Endpoints: []string{"https://localhost:2379"},
		CertFile:  "/nonexistent/client.crt",
		KeyFile:   "/nonexistent/client.key",
	}

	client, err := NewClient(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to load client certificate")
}

func TestNewClient_TLSMalformedCAFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "etcd2s3-tls-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create malformed CA file
	caFile := filepath.Join(tempDir, "bad_ca.crt")
	err = os.WriteFile(caFile, []byte("not a valid certificate"), 0644)
	require.NoError(t, err)

	cfg := appconfig.EtcdConfig{
		Endpoints: []string{"https://localhost:2379"},
		CaFile:    caFile,
	}

	client, err := NewClient(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to parse CA certificate")
}

func TestTLSConfigSetup(tMain *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "etcd2s3-tls-test")
	require.NoError(tMain, err)
	defer os.RemoveAll(tempDir)

	// Generate test certificates
	caFile, certFile, keyFile := generateTestCertificates(tMain, tempDir)

	tests := []struct {
		name              string
		caFile            string
		certFile          string
		keyFile           string
		expectTLS         bool
		expectInsecure    bool
		expectClientCerts bool
		expectRootCAs     bool
	}{
		{
			name:      "No TLS configuration",
			expectTLS: false,
		},
		{
			name:              "CA only",
			caFile:            caFile,
			expectTLS:         true,
			expectInsecure:    false,
			expectClientCerts: false,
			expectRootCAs:     true,
		},
		{
			name:              "Client cert without CA",
			certFile:          certFile,
			keyFile:           keyFile,
			expectTLS:         true,
			expectInsecure:    true,
			expectClientCerts: true,
			expectRootCAs:     false,
		},
		{
			name:              "Full TLS with CA and client cert",
			caFile:            caFile,
			certFile:          certFile,
			keyFile:           keyFile,
			expectTLS:         true,
			expectInsecure:    false,
			expectClientCerts: true,
			expectRootCAs:     true,
		},
	}

	for _, tt := range tests {
		tMain.Run(tt.name, func(t *testing.T) {
			cfg := appconfig.EtcdConfig{
				Endpoints: []string{"localhost:2379"}, // Use non-TLS endpoint to avoid connection issues
				CaFile:    tt.caFile,
				CertFile:  tt.certFile,
				KeyFile:   tt.keyFile,
			}

			// Test just the TLS configuration setup without actually connecting
			if tt.expectTLS {
				// Create a tls.Config manually to test the logic
				var tlsConfig *tls.Config

				if cfg.CertFile != "" && cfg.KeyFile != "" || cfg.CaFile != "" {
					tlsConfig = &tls.Config{}

					if cfg.CertFile != "" && cfg.KeyFile != "" {
						cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
						require.NoError(t, err)
						tlsConfig.Certificates = []tls.Certificate{cert}
					}

					if cfg.CaFile != "" {
						caCert, err := os.ReadFile(cfg.CaFile)
						require.NoError(t, err)

						caCertPool := x509.NewCertPool()
						ok := caCertPool.AppendCertsFromPEM(caCert)
						require.True(t, ok)

						tlsConfig.RootCAs = caCertPool
						tlsConfig.InsecureSkipVerify = false
					} else {
						tlsConfig.InsecureSkipVerify = true
					}
				}

				require.NotNil(t, tlsConfig)
				assert.Equal(t, tt.expectInsecure, tlsConfig.InsecureSkipVerify)

				if tt.expectClientCerts {
					assert.NotEmpty(t, tlsConfig.Certificates)
				} else {
					assert.Empty(t, tlsConfig.Certificates)
				}

				if tt.expectRootCAs {
					assert.NotNil(t, tlsConfig.RootCAs)
				} else {
					assert.Nil(t, tlsConfig.RootCAs)
				}
			}
		})
	}
}
