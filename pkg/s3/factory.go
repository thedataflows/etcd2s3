package s3

import (
	"fmt"

	"github.com/thedataflows/etcd2s3/pkg/appconfig"
	log "github.com/thedataflows/go-lib-log"
)

const PKG_S3_FACTORY = "s3.factory"

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
		log.Errorf(PKG_S3_FACTORY, err, "Failed to create S3 client for bucket '%s' at endpoint '%s'", config.Bucket, config.EndpointURL)
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	log.Debugf(PKG_S3_FACTORY, "Successfully created S3 client for bucket '%s' at endpoint '%s'", config.Bucket, config.EndpointURL)
	return client, nil
}

// CreateClientOrNil creates an S3 client but returns nil instead of error if creation fails
// Useful for optional S3 operations where fallback behavior is acceptable
func (f *ClientFactory) CreateClientOrNil(config appconfig.S3Config) *Client {
	client, err := f.CreateClient(config)
	if err != nil {
		log.Warnf(PKG_S3_FACTORY, "Failed to create S3 client, operations will be limited to local only: %v", err)
		return nil
	}
	return client
}

// MustCreateClient creates an S3 client and panics if creation fails
// Use sparingly, only when S3 client is absolutely required
func (f *ClientFactory) MustCreateClient(config appconfig.S3Config) *Client {
	client, err := f.CreateClient(config)
	if err != nil {
		log.Fatalf(PKG_S3_FACTORY, err, "Critical failure: could not create required S3 client")
	}
	return client
}
