package cmd

import (
	"fmt"
	"sync"

	"github.com/thedataflows/etcd2s3/pkg/appconfig"
	"github.com/thedataflows/etcd2s3/pkg/s3"
)

// CLIContext holds shared context for commands with S3 client caching
type CLIContext struct {
	Version   string
	Config    *appconfig.AppConfig
	s3Factory *s3.ClientFactory
	s3Client  *s3.Client
	s3Mutex   sync.Mutex
}

// NewCLIContext creates a new CLI context with S3 factory
func NewCLIContext(version string, config *appconfig.AppConfig) *CLIContext {
	return &CLIContext{
		Version:   version,
		Config:    config,
		s3Factory: s3.NewFactory(),
	}
}

// GetS3Client returns a cached S3 client or creates a new one
func (ctx *CLIContext) GetS3Client() (*s3.Client, error) {
	if ctx.Config.S3.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket name is required")
	}

	ctx.s3Mutex.Lock()
	defer ctx.s3Mutex.Unlock()

	if ctx.s3Client == nil {
		var err error
		ctx.s3Client, err = ctx.s3Factory.CreateClient(ctx.Config.S3)
		if err != nil {
			return nil, err
		}
	}
	return ctx.s3Client, nil
}

// GetS3ClientOrNil returns a cached S3 client or nil if creation fails
func (ctx *CLIContext) GetS3ClientOrNil() *s3.Client {
	client, _ := ctx.GetS3Client()
	return client
}

// GetS3Factory returns the S3 client factory
func (ctx *CLIContext) GetS3Factory() *s3.ClientFactory {
	return ctx.s3Factory
}
