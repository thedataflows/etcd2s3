package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thedataflows/etcd2s3/pkg/compression"
	"github.com/thedataflows/etcd2s3/pkg/etcd"
	log "github.com/thedataflows/go-lib-log"
)

// RestoreCmd restores etcd from a snapshot
type RestoreCmd struct {
	Source                   string `kong:"arg,required,help='Snapshot source (local path or S3 key)'"`
	DataDir                  string `kong:"help='etcd data directory for restore',default='/var/lib/etcd'"`
	Name                     string `kong:"help='etcd member name',default='default'"`
	InitialCluster           string `kong:"help='Initial cluster configuration',default='default=http://localhost:2380'"`
	InitialAdvertisePeerURLs string `kong:"help='Initial advertise peer URLs',default='http://localhost:2380'"`
	SkipHashCheck            bool   `kong:"help='Skip hash check during restore'"`
}

func (r *RestoreCmd) Run(ctx *CLIContext) error {
	log.Info(PKG_CMD, "Starting restore operation")

	var snapshotPath string
	var err error

	// Determine snapshot source: s3:// URL, local file, or S3 key
	if strings.HasPrefix(r.Source, "s3://") {
		snapshotPath, err = r.downloadFromS3URL(ctx, r.Source)
	} else {
		// Check if local file exists (with compression resolution)
		resolvedPath, found := compression.ResolveCompressedFile(r.Source)
		if found {
			// Local file exists and has content (relative or absolute path)
			snapshotPath = resolvedPath
			log.Infof(PKG_CMD, "Using local snapshot: %s", snapshotPath)
		} else {
			// Local file missing/empty - attempt S3 download
			log.Warnf(PKG_CMD, "Local file '%s' not found or empty, attempting to download", r.Source)
			snapshotPath, err = r.downloadFromS3Key(ctx, r.Source)
		}
	}

	if err != nil {
		return err
	}

	// Handle decompression if the snapshot is compressed
	finalSnapshotPath := snapshotPath
	if compression.IsCompressed(snapshotPath) {
		// Generate decompressed filename
		decompressedPath := strings.TrimSuffix(snapshotPath, filepath.Ext(snapshotPath))
		if !strings.HasSuffix(decompressedPath, ".db") {
			decompressedPath += ".db"
		}

		compressionStart := time.Now()
		if err := compression.DecompressFile(snapshotPath, decompressedPath); err != nil {
			return fmt.Errorf("failed to decompress snapshot: %w", err)
		}

		finalSnapshotPath = decompressedPath
		log.Logger.Info().Str(log.KEY_PKG, PKG_CMD).Str("algorithm", compression.GetCompressionAlgorithmFromExt(snapshotPath)).Str("file", snapshotPath).Str("duration", fmt.Sprintf("%s", time.Since(compressionStart))).Msg("Snapshot decompressed")

	}

	// Restore snapshot using etcdutl (offline operation - no client connection needed)
	restoreOpts := etcd.RestoreOptions{
		SnapshotPath:             finalSnapshotPath,
		DataDir:                  r.DataDir,
		Name:                     r.Name,
		InitialCluster:           r.InitialCluster,
		InitialAdvertisePeerURLs: r.InitialAdvertisePeerURLs,
		SkipHashCheck:            r.SkipHashCheck,
	}

	if err := etcd.RestoreSnapshot(context.Background(), restoreOpts); err != nil {
		return fmt.Errorf("failed to restore etcd: %w", err)
	}

	log.Infof(PKG_CMD, "Restore completed successfully to %s", r.DataDir)
	return nil
}

// downloadFromS3URL downloads a snapshot from an s3:// URL
func (r *RestoreCmd) downloadFromS3URL(ctx *CLIContext, s3URL string) (string, error) {
	// Extract S3 key from s3:// URL
	s3Key := s3URL[5:] // Remove "s3://" prefix
	if idx := strings.Index(s3Key, "/"); idx > 0 {
		s3Key = s3Key[idx+1:] // Remove bucket name
	}
	return r.downloadSnapshot(ctx, s3Key)
}

// downloadFromS3Key downloads a snapshot using an S3 key
func (r *RestoreCmd) downloadFromS3Key(ctx *CLIContext, source string) (string, error) {
	s3Key := filepath.Base(source)
	return r.downloadSnapshot(ctx, s3Key)
}

// downloadSnapshot downloads a snapshot from S3 with validation and cleanup
func (r *RestoreCmd) downloadSnapshot(ctx *CLIContext, s3Key string) (string, error) {
	s3Client, err := ctx.GetS3Client()
	if err != nil {
		return "", err
	}

	// Resolve compressed file name - check for compressed versions first
	resolvedKey, found, err := s3Client.ResolveCompressedKey(context.Background(), s3Key)
	if err != nil {
		return "", fmt.Errorf("failed to resolve compressed snapshot: %w", err)
	}
	if !found {
		return "", fmt.Errorf("snapshot not found in S3: %s (checked compressed and uncompressed versions)", s3Key)
	}

	// Update the key to the resolved version
	actualKey := resolvedKey
	snapshotPath := filepath.Join(ctx.Config.Etcd.SnapshotDir, filepath.Base(actualKey))

	// Build display URL for logging - show what the user would see with prefix
	displayKey := actualKey
	if ctx.Config.S3.Prefix != "" {
		displayKey = filepath.Join(ctx.Config.S3.Prefix, actualKey)
	}
	displayURL := fmt.Sprintf("s3://%s/%s", ctx.Config.S3.Bucket, displayKey)

	log.Logger.Info().Str(log.KEY_PKG, PKG_CMD).Str("endpoint", ctx.Config.S3.EndpointURL).Str("url", displayURL).Msg("Downloading snapshot")

	if err := s3Client.Download(context.Background(), actualKey, snapshotPath); err != nil {
		// Clean up any partially created file on failure
		_ = os.Remove(snapshotPath)
		return "", fmt.Errorf("failed to download snapshot from S3: %w", err)
	}

	// Verify downloaded file has content
	if fileInfo, err := os.Stat(snapshotPath); err != nil || fileInfo.Size() == 0 {
		_ = os.Remove(snapshotPath)
		return "", fmt.Errorf("downloaded snapshot file is empty or invalid")
	}

	log.Infof(PKG_CMD, "Snapshot downloaded to: %s", snapshotPath)
	return snapshotPath, nil
}
