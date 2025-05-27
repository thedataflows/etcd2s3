package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	s5cmdlog "github.com/peak/s5cmd/v2/log"
	"github.com/peak/s5cmd/v2/storage"
	"github.com/peak/s5cmd/v2/storage/url"
	"github.com/thedataflows/etcd2s3/pkg/appconfig"
	"github.com/thedataflows/etcd2s3/pkg/compression"
)

// Client wraps s5cmd library functionality for S3 operations
type Client struct {
	bucket   string
	prefix   string
	s3Client *storage.S3
}

// Object represents an S3 object
type Object struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
}

// NewClient creates a new S3 client using s5cmd library
func NewClient(cfg appconfig.S3Config) (*Client, error) {
	// Initialize s5cmd logger to prevent nil pointer dereference
	s5cmdlog.Init("error", false) // Set to error level to minimize noise

	// Create storage options for s5cmd
	opts := storage.Options{
		Endpoint:      cfg.EndpointURL,
		NoVerifySSL:   false,
		DryRun:        false,
		NoSignRequest: false,
	}

	// Set region if provided
	if cfg.Region != "" {
		opts.SetRegion(cfg.Region)
	}

	// Create a dummy URL to get the remote client
	bucketURL, err := url.New(fmt.Sprintf("s3://%s", cfg.Bucket))
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket URL: %w", err)
	}

	// Set environment variables for AWS credentials if missing
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" && cfg.AccessKeyID != "" {
		_ = os.Setenv("AWS_ACCESS_KEY_ID", cfg.AccessKeyID)
	}
	if os.Getenv("AWS_SECRET_ACCESS_KEY") == "" && cfg.SecretAccessKey != "" {
		_ = os.Setenv("AWS_SECRET_ACCESS_KEY", cfg.SecretAccessKey)
	}
	if os.Getenv("AWS_SESSION_TOKEN") == "" && cfg.SessionToken != "" {
		_ = os.Setenv("AWS_SESSION_TOKEN", cfg.SessionToken)
	}
	if os.Getenv("AWS_REGION") == "" && cfg.Region != "" {
		_ = os.Setenv("AWS_REGION", cfg.Region)
	}
	if os.Getenv("AWS_ENDPOINT_URL") == "" && cfg.EndpointURL != "" {
		_ = os.Setenv("AWS_ENDPOINT_URL", cfg.EndpointURL)
	}

	// Create S3 client specifically
	s3Client, err := storage.NewRemoteClient(context.Background(), bucketURL, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &Client{
		bucket:   cfg.Bucket,
		prefix:   cfg.Prefix,
		s3Client: s3Client,
	}, nil
}

// buildKey constructs the full S3 key by applying the prefix
func (c *Client) buildKey(key string) string {
	if c.prefix == "" {
		return key
	}
	return filepath.Join(c.prefix, key)
}

// Upload uploads a file to S3
func (c *Client) Upload(ctx context.Context, filePath, key string) error {
	// Apply prefix to the key
	fullKey := c.buildKey(key)

	// Create destination URL for S3
	dstURL, err := url.New(fmt.Sprintf("s3://%s/%s", c.bucket, fullKey))
	if err != nil {
		return fmt.Errorf("failed to create destination URL: %w", err)
	}

	// Open source file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	// Upload using s5cmd Put method
	metadata := storage.Metadata{}
	err = c.s3Client.Put(ctx, file, dstURL, metadata, 5, 64*1024*1024) // 5 concurrent, 64MB parts
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// Download downloads a file from S3
func (c *Client) Download(ctx context.Context, key, filePath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Apply prefix to the key
	fullKey := c.buildKey(key)

	// Create source URL for S3
	srcURL, err := url.New(fmt.Sprintf("s3://%s/%s", c.bucket, fullKey))
	if err != nil {
		return fmt.Errorf("failed to create source URL: %w", err)
	}

	// Create destination file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer file.Close()

	// Download using s5cmd Get method
	_, err = c.s3Client.Get(ctx, srcURL, file, 5, 64*1024*1024) // 5 concurrent, 64MB parts
	if err != nil {
		return fmt.Errorf("failed to download from S3: %w", err)
	}

	return nil
}

// List lists objects in S3 with the given prefix
func (c *Client) List(ctx context.Context, prefix string) ([]Object, error) {
	// Apply client prefix to the search prefix
	fullPrefix := c.buildKey(prefix)

	// Create URL for listing
	var listURL *url.URL
	var err error

	if fullPrefix != "" {
		listURL, err = url.New(fmt.Sprintf("s3://%s/%s*", c.bucket, fullPrefix))
	} else {
		listURL, err = url.New(fmt.Sprintf("s3://%s/*", c.bucket))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create list URL: %w", err)
	}

	// List objects using s5cmd List method
	objectChan := c.s3Client.List(ctx, listURL, false)

	var objects []Object
	for obj := range objectChan {
		if obj.Err != nil {
			// Check if it's a "no object found" error which is not really an error
			if strings.Contains(obj.Err.Error(), "no object found") {
				continue
			}
			return nil, fmt.Errorf("error listing objects: %w", obj.Err)
		}

		// Skip directories
		if obj.Type.IsDir() {
			continue
		}

		// Extract key from URL path
		key := obj.URL.Path
		if key == "" {
			continue
		}

		// Strip client prefix from the key to maintain relative perspective
		if c.prefix != "" && strings.HasPrefix(key, c.prefix+"/") {
			key = key[len(c.prefix)+1:]
		} else if c.prefix != "" && key == c.prefix {
			key = ""
		}

		// Get last modified time
		lastModified := time.Now()
		if obj.ModTime != nil {
			lastModified = *obj.ModTime
		}

		objects = append(objects, Object{
			Key:          key,
			Size:         obj.Size,
			LastModified: lastModified,
		})
	}

	return objects, nil
}

// Delete deletes an object from S3
func (c *Client) Delete(ctx context.Context, key string) error {
	// Apply prefix to the key
	fullKey := c.buildKey(key)

	// Create URL for the object to delete
	deleteURL, err := url.New(fmt.Sprintf("s3://%s/%s", c.bucket, fullKey))
	if err != nil {
		return fmt.Errorf("failed to create delete URL: %w", err)
	}

	// Delete using s5cmd Delete method
	err = c.s3Client.Delete(ctx, deleteURL)
	if err != nil {
		return fmt.Errorf("failed to delete S3 object: %w", err)
	}

	return nil
}

// DeleteMultiple deletes multiple objects from S3
func (c *Client) DeleteMultiple(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	// Create a channel of URLs for deletion
	urlChan := make(chan *url.URL, len(keys))
	go func() {
		defer close(urlChan)
		for _, key := range keys {
			// Apply prefix to the key
			fullKey := c.buildKey(key)
			deleteURL, err := url.New(fmt.Sprintf("s3://%s/%s", c.bucket, fullKey))
			if err != nil {
				// Log error but continue with other deletions
				continue
			}
			urlChan <- deleteURL
		}
	}()

	// Delete using s5cmd MultiDelete method
	resultChan := c.s3Client.MultiDelete(ctx, urlChan)

	// Process results and check for errors
	for result := range resultChan {
		if result.Err != nil {
			return fmt.Errorf("failed to delete S3 object %s: %w", result.URL.Path, result.Err)
		}
	}

	return nil
}

// Exists checks if an object exists in S3
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	// Apply prefix to the key
	fullKey := c.buildKey(key)

	// Create URL for the object
	objURL, err := url.New(fmt.Sprintf("s3://%s/%s", c.bucket, fullKey))
	if err != nil {
		return false, fmt.Errorf("failed to create object URL: %w", err)
	}

	// Use s5cmd Stat method to check existence
	_, err = c.s3Client.Stat(ctx, objURL)
	if err != nil {
		// If error contains "not found" or "NoSuchKey", object doesn't exist
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		// Other errors should be returned
		return false, err
	}

	return true, nil
}

// ResolveCompressedKey attempts to find the best available version of a snapshot file.
// If the key ends with .db, it checks for compressed versions first, then falls back to uncompressed.
// Returns the actual key found and whether it was found.
func (c *Client) ResolveCompressedKey(ctx context.Context, key string) (string, bool, error) {
	// Import compression package functions
	candidates := compression.ResolveCompressedFilename(key)

	// Try each candidate in order of preference
	for _, candidate := range candidates {
		exists, err := c.Exists(ctx, candidate)
		if err != nil {
			return "", false, fmt.Errorf("failed to check existence of %s: %w", candidate, err)
		}
		if exists {
			return candidate, true, nil
		}
	}

	// No file found
	return key, false, nil
}
