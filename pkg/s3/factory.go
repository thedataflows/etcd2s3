package s3

import (
	"fmt"

	log "github.com/rs/zerolog/log"
	"github.com/thedataflows/etcd2s3/pkg/appconfig"
)

const PKG_S3_FACTORY = "s3.factory"

const KEY_PKG = "pkg"

// ClientFactory provides methods for creating S3 clients with proper error handling
type ClientFactory struct{}

// NewFactory creates a new S3 client factory
func NewFactory() *ClientFactory {
	return &ClientFactory{}
}

// CreateClient creates an S3 client with standardized error handling and logging
func (f *ClientFactory) CreateClient(config appconfig.S3Config) (*Client, error) {
	client, err := NewClient(config)
	if err != nil {
		log.Error().Str(KEY_PKG, PKG_S3_FACTORY).Err(err).Msgf("Failed to create S3 client for bucket '%s' at endpoint '%s'", config.Bucket, config.EndpointURL)
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	log.Debug().Str(KEY_PKG, PKG_S3_FACTORY).Msgf("Successfully created S3 client for bucket '%s' at endpoint '%s'", config.Bucket, config.EndpointURL)
	return client, nil
}

// CreateClientOrNil creates an S3 client but returns nil instead of error if creation fails
// Useful for optional S3 operations where fallback behavior is acceptable
func (f *ClientFactory) CreateClientOrNil(config appconfig.S3Config) *Client {
	client, err := f.CreateClient(config)
	if err != nil {
		log.Warn().Str(KEY_PKG, PKG_S3_FACTORY).Msgf("Failed to create S3 client, operations will be limited to local only: %v", err)
		return nil
	}
	return client
}

// MustCreateClient creates an S3 client and panics if creation fails
// Use sparingly, only when S3 client is absolutely required
func (f *ClientFactory) MustCreateClient(config appconfig.S3Config) *Client {
	client, err := f.CreateClient(config)
	if err != nil {
		log.Fatal().Str(KEY_PKG, PKG_S3_FACTORY).Err(err).Msgf("Critical failure: could not create required S3 client")
	}
	return client
}
