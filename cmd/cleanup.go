package cmd

import (
	"context"

	"github.com/thedataflows/etcd2s3/pkg/retention"
	log "github.com/thedataflows/go-lib-log"
)

// CleanupCmd deletes snapshots based on retention policies
type CleanupCmd struct {
	Local   bool `kong:"help='Clean local snapshots only'"`
	Remote  bool `kong:"help='Clean S3 snapshots only'"`
	DryRun  bool `kong:"help='Show what would be deleted without actually deleting'"`
	Unified bool `kong:"help='Use unified retention evaluation across local and S3',default=true"`
}

func (c *CleanupCmd) Run(ctx *CLIContext) error {
	if c.DryRun {
		log.Info(PKG_CMD, "Starting cleanup operation (DRY RUN)")
	} else {
		log.Info(PKG_CMD, "Starting cleanup operation")
	}

	retentionManager := retention.NewManager(ctx.Config.Policy)

	// Use unified approach if both local and S3 are being cleaned
	if c.Unified && !c.Local && !c.Remote {
		return c.runUnifiedCleanup(ctx, retentionManager)
	}

	// Use separate approach for individual storage types
	return c.runSeparateCleanup(ctx, retentionManager)
}

func (c *CleanupCmd) runUnifiedCleanup(ctx *CLIContext, retentionManager *retention.Manager) error {
	log.Info(PKG_CMD, "Using unified retention evaluation")

	// Create S3 client if needed using factory
	s3Client := ctx.GetS3ClientOrNil()
	if s3Client == nil {
		log.Warn(PKG_CMD, "S3 client unavailable, will only clean local snapshots")
	}

	// Apply unified retention policy
	if err := retentionManager.ApplyUnified(context.Background(), ctx.Config.Etcd.SnapshotDir, s3Client, c.DryRun); err != nil {
		log.Errorf(PKG_CMD, err, "Failed to apply unified retention policy")
		return err
	}

	log.Info(PKG_CMD, "Unified cleanup operation completed")
	return nil
}

func (c *CleanupCmd) runSeparateCleanup(ctx *CLIContext, retentionManager *retention.Manager) error {
	log.Info(PKG_CMD, "Using separate retention evaluation for each storage type")

	// Clean local snapshots
	if !c.Remote {
		log.Info(PKG_CMD, "Cleaning local snapshots")
		if err := retentionManager.ApplyLocal(ctx.Config.Etcd.SnapshotDir, c.DryRun); err != nil {
			log.Errorf(PKG_CMD, err, "Failed to clean local snapshots")
		} else {
			log.Info(PKG_CMD, "Local snapshot cleanup completed")
		}
	}

	// Clean S3 snapshots
	if !c.Local {
		log.Info(PKG_CMD, "Cleaning S3 snapshots")
		s3Client, err := ctx.GetS3Client()
		if err != nil {
			log.Errorf(PKG_CMD, err, "Failed to create S3 client")
		} else {
			if err := retentionManager.ApplyS3(context.Background(), s3Client, c.DryRun); err != nil {
				log.Errorf(PKG_CMD, err, "Failed to clean S3 snapshots")
			} else {
				log.Info(PKG_CMD, "S3 snapshot cleanup completed")
			}
		}
	}

	log.Info(PKG_CMD, "Cleanup operation completed")
	return nil
}
