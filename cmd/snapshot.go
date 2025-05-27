package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/thedataflows/etcd2s3/pkg/compression"
	"github.com/thedataflows/etcd2s3/pkg/etcd"
	"github.com/thedataflows/etcd2s3/pkg/retention"
	log "github.com/thedataflows/go-lib-log"
)

// SnapshotCmd takes a snapshot of etcd and uploads to S3
type SnapshotCmd struct {
	Name           string `kong:"help='Custom snapshot name',default=''"`
	UploadToS3     bool   `kong:"help='Upload snapshot to S3',default=true,name='upload-to-s3'"`
	RemoveLocal    bool   `kong:"help='Remove local snapshot after S3 upload'"`
	ApplyRetention bool   `kong:"help='Apply retention policies after snapshot',default=true"`
	Unified        bool   `kong:"help='Use unified retention evaluation across local and S3',default=true"`
	Compression    string `kong:"help='Compression algorithm for snapshot',default='zstd',enum='none,bzip2,gzip,lz4,zstd'"`
}

func (s *SnapshotCmd) Run(ctx *CLIContext) error {
	log.Info(PKG_CMD, "Starting snapshot operation")

	// Create etcd client
	etcdClient, err := etcd.NewClient(ctx.Config.Etcd)
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer etcdClient.Close()

	// Generate snapshot name if not provided
	snapshotName := s.Name
	if len(snapshotName) == 0 {
		snapshotName = fmt.Sprintf("etcd-snapshot-%s.db", time.Now().Format("20060102-150405"))
	}
	if filepath.Ext(snapshotName) != ".db" {
		snapshotName = fmt.Sprintf("%s.db", snapshotName)
	}

	// Take snapshot with timeout
	snapshotPath := filepath.Join(ctx.Config.Etcd.SnapshotDir, snapshotName)

	// Create context with timeout for snapshot operation
	snapshotCtx, cancel := context.WithTimeout(context.Background(), ctx.Config.Etcd.SnapshotTimeout)
	defer cancel()

	if err := etcdClient.Snapshot(snapshotCtx, snapshotPath); err != nil {
		return fmt.Errorf("failed to take etcd snapshot: %w", err)
	}

	log.Logger.Info().Str(log.KEY_PKG, PKG_CMD).Str("file", snapshotPath).Msg("Snapshot saved")

	// Apply compression if specified
	finalSnapshotPath := snapshotPath
	if strings.ToLower(s.Compression) != "none" && s.Compression != "" {
		compressedPath := snapshotPath + compression.GetCompressionExt(s.Compression)

		// Time the compression operation
		compressionStart := time.Now()
		if err := compression.CompressFile(snapshotPath, compressedPath, s.Compression); err != nil {
			return fmt.Errorf("failed to compress snapshot: %w", err)
		}

		log.Logger.Info().Str(log.KEY_PKG, PKG_CMD).Str("algorithm", s.Compression).Str("file", compressedPath).Str("duration", fmt.Sprintf("%s", time.Since(compressionStart))).Msg("Snapshot compressed")

		// Remove original uncompressed file
		if err := etcdClient.RemoveSnapshot(snapshotPath); err != nil {
			log.Logger.Error().Err(err).Str(log.KEY_PKG, PKG_CMD).Str("file", snapshotPath).Msg("Failed to remove original snapshot")
		}

		finalSnapshotPath = compressedPath
		// Update snapshot name for S3 upload
		snapshotName = filepath.Base(compressedPath)
	}

	if s.UploadToS3 {
		// Create S3 client
		s3Client, err := ctx.GetS3Client()
		if err != nil {
			return err
		}

		// Upload the new snapshot to S3
		s3Key := snapshotName

		if err := s3Client.Upload(context.Background(), finalSnapshotPath, s3Key); err != nil {
			return fmt.Errorf("failed to upload snapshot to S3: %w", err)
		}

		log.Infof(PKG_CMD, "Snapshot uploaded to S3: s3://%s/%s", ctx.Config.S3.Bucket, s3Key)

		// Upload any other local snapshots that should be kept but are missing from S3
		if err := s.uploadMissingSnapshots(ctx); err != nil {
			log.Warnf(PKG_CMD, "Failed to upload missing local snapshots: %v", err)
		}

		// Remove local file if requested
		if s.RemoveLocal || ctx.Config.Policy.RemoveLocal {
			if err := etcdClient.RemoveSnapshot(finalSnapshotPath); err != nil {
				log.Warnf(PKG_CMD, "Failed to remove local snapshot %s: %v", finalSnapshotPath, err)
			} else {
				log.Infof(PKG_CMD, "Local snapshot removed: %s", finalSnapshotPath)
			}
		}
	}

	if s.ApplyRetention {
		// Apply retention policies
		retentionManager := retention.NewManager(ctx.Config.Policy)

		if s.Unified && s.UploadToS3 {
			// Use unified approach when both local and S3 are involved
			s3Client := ctx.GetS3ClientOrNil()
			if s3Client == nil {
				log.Warn(PKG_CMD, "S3 client unavailable for unified retention, falling back to local-only")
				// Fall back to local-only retention
				if err := retentionManager.ApplyLocal(ctx.Config.Etcd.SnapshotDir, false); err != nil {
					log.Warnf(PKG_CMD, "Failed to apply local retention policy: %v", err)
				}
			} else {
				if err := retentionManager.ApplyUnified(context.Background(), ctx.Config.Etcd.SnapshotDir, s3Client, false); err != nil {
					log.Warnf(PKG_CMD, "Failed to apply unified retention policy: %v", err)
				}
			}
		} else {
			// Use separate approach for individual storage types
			if err := retentionManager.ApplyLocal(ctx.Config.Etcd.SnapshotDir, false); err != nil {
				log.Warnf(PKG_CMD, "Failed to apply local retention policy: %v", err)
			}

			if s.UploadToS3 {
				s3Client := ctx.GetS3ClientOrNil()
				if s3Client == nil {
					log.Warn(PKG_CMD, "S3 client unavailable for S3 retention")
				} else {
					if err := retentionManager.ApplyS3(context.Background(), s3Client, false); err != nil {
						log.Warnf(PKG_CMD, "Failed to apply S3 retention policy: %v", err)
					}
				}
			}
		}
	}

	log.Info(PKG_CMD, "Snapshot operation completed successfully")
	return nil
}

// uploadMissingSnapshots uploads local snapshots that should be kept according to retention policy
// but are missing from S3
func (s *SnapshotCmd) uploadMissingSnapshots(ctx *CLIContext) error {
	log.Info(PKG_CMD, "Checking for local snapshots that need to be uploaded to S3")

	// Get S3 client from context
	s3Client, err := ctx.GetS3Client()
	if err != nil {
		return err
	}

	// Create retention manager to determine which snapshots should be kept
	retentionManager := retention.NewManager(ctx.Config.Policy)

	// Get local snapshots
	localSnapshots, err := retentionManager.GetLocalSnapshots(ctx.Config.Etcd.SnapshotDir)
	if err != nil {
		return fmt.Errorf("failed to get local snapshots: %w", err)
	}

	// Get S3 snapshots to see what's already there
	s3Snapshots, err := retentionManager.GetS3Snapshots(context.Background(), s3Client)
	if err != nil {
		return fmt.Errorf("failed to get S3 snapshots: %w", err)
	}

	// Create a map of S3 snapshot names for quick lookup
	s3SnapshotNames := make(map[string]bool)
	for _, s3Snap := range s3Snapshots {
		s3SnapshotNames[s3Snap.Name] = true
	}

	// Use unified retention to determine which snapshots should be kept
	var retentionDecisions map[string]bool
	if s.Unified {
		retentionDecisions = retentionManager.GetUnifiedRetentionStatus(localSnapshots, s3Snapshots)
	} else {
		retentionDecisions = retentionManager.GetRetentionStatus(localSnapshots)
	}

	// Find local snapshots that should be kept but are missing from S3
	var toUpload []retention.SnapshotFile
	for _, localSnap := range localSnapshots {
		// Check if this snapshot should be kept and is missing from S3
		if retentionDecisions[localSnap.Name] && !s3SnapshotNames[localSnap.Name] {
			toUpload = append(toUpload, localSnap)
		}
	}

	if len(toUpload) == 0 {
		log.Info(PKG_CMD, "All local snapshots that should be kept are already present in S3")
		return nil
	}

	log.Infof(PKG_CMD, "Found %d local snapshots to upload to S3", len(toUpload))

	// Upload missing snapshots
	for _, snapshot := range toUpload {
		s3Key := snapshot.Name

		log.Infof(PKG_CMD, "Uploading local snapshot to S3: %s", snapshot.Name)
		if err := s3Client.Upload(context.Background(), snapshot.Path, s3Key); err != nil {
			log.Warnf(PKG_CMD, "Failed to upload snapshot %s to S3: %v", snapshot.Name, err)
			continue
		}

		log.Infof(PKG_CMD, "Successfully uploaded: s3://%s/%s", ctx.Config.S3.Bucket, s3Key)
	}

	return nil
}
