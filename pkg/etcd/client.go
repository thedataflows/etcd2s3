package etcd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/thedataflows/etcd2s3/pkg/appconfig"
	log "github.com/thedataflows/go-lib-log"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/snapshot"
	etcdutlSnapshot "go.etcd.io/etcd/etcdutl/v3/snapshot"
	"go.uber.org/zap"
)

const PKG_ETCD = "etcd"

// Client wraps etcd client functionality
type Client struct {
	client *clientv3.Client
	config clientv3.Config
}

// RestoreOptions holds options for etcd restore
type RestoreOptions struct {
	SnapshotPath             string
	DataDir                  string
	Name                     string
	InitialCluster           string
	InitialAdvertisePeerURLs string
	SkipHashCheck            bool
}

// NewClient creates a new etcd client
func NewClient(cfg appconfig.EtcdConfig) (*Client, error) {
	log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Strs("endpoints", cfg.Endpoints).Msg("Creating new etcd client")

	clientConfig := clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: 5 * time.Second,
	}

	// Set up authentication
	if cfg.Username != "" {
		log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Str("username", cfg.Username).Msg("Setting up username/password authentication")
		clientConfig.Username = cfg.Username
		clientConfig.Password = cfg.Password
	}

	// Set up TLS
	if cfg.CertFile != "" && cfg.KeyFile != "" || cfg.CaFile != "" {
		log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Msg("Setting up TLS configuration")
		tlsConfig := &tls.Config{}

		// Load client certificate if both cert and key files are provided
		if cfg.CertFile != "" && cfg.KeyFile != "" {
			log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Str("cert_file", cfg.CertFile).Str("key_file", cfg.KeyFile).Msg("Loading client certificate")
			cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
			log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Msg("Client certificate loaded successfully")
		}

		if cfg.CaFile != "" {
			log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Str("ca_file", cfg.CaFile).Msg("Loading CA certificate")
			// Load CA certificate if provided
			caCert, err := os.ReadFile(cfg.CaFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate file: %w", err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}

			tlsConfig.RootCAs = caCertPool
			tlsConfig.InsecureSkipVerify = false
			log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Msg("CA certificate loaded, TLS verification enabled")
		} else {
			// If no CA file is provided but we're using TLS, skip verification
			tlsConfig.InsecureSkipVerify = true
			log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Msg("No CA certificate provided, TLS verification disabled")
		}

		clientConfig.TLS = tlsConfig
	} else {
		log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Msg("No TLS configuration provided")
	}

	log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Msg("Attempting to connect to etcd")
	client, err := clientv3.New(clientConfig)
	if err != nil {
		log.Logger.Error().Str(log.KEY_PKG, PKG_ETCD).Err(err).Msg("Failed to create etcd client")
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Msg("etcd client created successfully")

	// Test the connection with a quick status check
	log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Msg("Testing connection with status check")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Status(ctx, clientConfig.Endpoints[0])
	if err != nil {
		log.Logger.Error().Str(log.KEY_PKG, PKG_ETCD).Err(err).Msg("Connection test failed")
		client.Close()
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}
	log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Msg("Connection test successful")

	return &Client{client: client, config: clientConfig}, nil
}

// Close closes the etcd client
func (c *Client) Close() error {
	return c.client.Close()
}

// Snapshot takes a snapshot of etcd and saves it to the specified path
func (c *Client) Snapshot(ctx context.Context, snapshotPath string) error {
	log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Str("snapshot_path", snapshotPath).Msg("Starting snapshot operation")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}
	log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Msg("Snapshot directory created/verified")

	// Get endpoints from existing client
	endpoints := c.client.Endpoints()
	if len(endpoints) == 0 {
		return fmt.Errorf("no endpoints configured")
	}
	log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Str("endpoint", endpoints[0]).Msg("Using endpoint for snapshot")

	// Create config for snapshot based on original config
	// snapshot must use single endpoint
	snapshotConfig := clientv3.Config{
		Endpoints:   []string{endpoints[0]},
		DialTimeout: c.config.DialTimeout,
		Username:    c.config.Username,
		Password:    c.config.Password,
		TLS:         c.config.TLS, // Preserve TLS configuration
	}

	hasTLS := snapshotConfig.TLS != nil
	log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Bool("has_tls", hasTLS).Msg("Snapshot config prepared")
	if hasTLS {
		log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).
			Bool("insecure_skip_verify", snapshotConfig.TLS.InsecureSkipVerify).
			Bool("has_root_cas", snapshotConfig.TLS.RootCAs != nil).
			Int("client_cert_count", len(snapshotConfig.TLS.Certificates)).
			Msg("TLS configuration details")
	}

	// Use the new snapshot API
	logger := zap.NewNop()
	log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Msg("Calling snapshot.SaveWithVersion")
	_, err := snapshot.SaveWithVersion(ctx, logger, snapshotConfig, snapshotPath)
	if err != nil {
		log.Logger.Error().Str(log.KEY_PKG, PKG_ETCD).Err(err).Msg("Snapshot failed")
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	log.Logger.Debug().Str(log.KEY_PKG, PKG_ETCD).Msg("Snapshot completed successfully")
	return nil
}

// RestoreSnapshot restores etcd from a snapshot using etcdutl library without requiring a client connection
func RestoreSnapshot(ctx context.Context, opts RestoreOptions) error {
	// Convert snapshot path to absolute path to handle working directory changes
	snapshotPath := opts.SnapshotPath
	if !filepath.IsAbs(snapshotPath) {
		absPath, err := filepath.Abs(snapshotPath)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path for snapshot: %w", err)
		}
		snapshotPath = absPath
	}

	// Convert data directory to absolute path as well
	dataDir := opts.DataDir
	if !filepath.IsAbs(dataDir) {
		absPath, err := filepath.Abs(dataDir)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path for data directory: %w", err)
		}
		dataDir = absPath
	}

	// Create snapshot manager
	logger := zap.NewNop()
	manager := etcdutlSnapshot.NewV3(logger)

	// Parse peer URLs from string
	var peerURLs []string
	if opts.InitialAdvertisePeerURLs != "" {
		peerURLs = []string{opts.InitialAdvertisePeerURLs}
	}

	// Configure restore options
	restoreConfig := etcdutlSnapshot.RestoreConfig{
		SnapshotPath:        snapshotPath,
		Name:                opts.Name,
		OutputDataDir:       dataDir,
		PeerURLs:            peerURLs,
		InitialCluster:      opts.InitialCluster,
		InitialClusterToken: "etcd-cluster",
		SkipHashCheck:       opts.SkipHashCheck,
	}

	// Set default name if not provided
	if restoreConfig.Name == "" {
		restoreConfig.Name = "default"
	}

	// Set default initial cluster if not provided
	if restoreConfig.InitialCluster == "" {
		restoreConfig.InitialCluster = fmt.Sprintf("%s=http://localhost:2380", restoreConfig.Name)
	}

	// Perform restore
	if err := manager.Restore(restoreConfig); err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}

	return nil
}

// RemoveSnapshot removes a local snapshot file
func (c *Client) RemoveSnapshot(snapshotPath string) error {
	return os.Remove(snapshotPath)
}
