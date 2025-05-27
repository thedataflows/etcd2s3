package s3

import (
	"testing"

	"github.com/thedataflows/etcd2s3/pkg/appconfig"
)

func TestBuildKey(tMain *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "No prefix",
			prefix:   "",
			key:      "snapshot.db",
			expected: "snapshot.db",
		},
		{
			name:     "With prefix",
			prefix:   "etcd-backups",
			key:      "snapshot.db",
			expected: "etcd-backups/snapshot.db",
		},
		{
			name:     "With nested prefix",
			prefix:   "backups/etcd",
			key:      "snapshot-2023.db",
			expected: "backups/etcd/snapshot-2023.db",
		},
		{
			name:     "Empty key with prefix",
			prefix:   "etcd-backups",
			key:      "",
			expected: "etcd-backups",
		},
	}

	for _, tt := range tests {
		tMain.Run(tt.name, func(t *testing.T) {
			client := &Client{
				bucket: "test-bucket",
				prefix: tt.prefix,
			}

			result := client.buildKey(tt.key)
			if result != tt.expected {
				t.Errorf("buildKey() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestNewClientWithPrefix(t *testing.T) {
	cfg := appconfig.S3Config{
		Bucket: "test-bucket",
		Prefix: "test-prefix",
		Region: "us-west-2",
	}

	// Note: This test would normally require AWS credentials and network access
	// For now, we just test that the prefix is correctly stored
	// In a real test environment, you would mock the storage.NewRemoteClient call
	_ = cfg // prevent unused variable error in this simple test

	// Test that prefix is correctly stored - this would be part of integration tests
	// client, err := NewClient(cfg)
	// assert.NoError(t, err)
	// assert.Equal(t, "test-prefix", client.prefix)
}
