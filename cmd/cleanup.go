package cmd

import (
	"context"

	log "github.com/rs/zerolog/log"
	"github.com/thedataflows/etcd2s3/pkg/retention"
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
		log.Info().Str(KEY_PKG, PKG_CMD).Msg("Starting cleanup operation (DRY RUN)")
	} else {
		log.Info().Str(KEY_PKG, PKG_CMD).Msg("Starting cleanup operation")
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
	log.Info().Str(KEY_PKG, PKG_CMD).Msg("Using unified retention evaluation")

	// Create S3 client if needed using factory
	s3Client := ctx.GetS3ClientOrNil()
	if s3Client == nil {
		log.Warn().Str(KEY_PKG, PKG_CMD).Msg("S3 client unavailable, will only clean local snapshots")
	}

	// Apply unified retention policy
	if err := retentionManager.ApplyUnified(context.Background(), ctx.Config.Etcd.SnapshotDir, s3Client, c.DryRun); err != nil {
		log.Error().Str(KEY_PKG, PKG_CMD).Err(err).Msgf("Failed to apply unified retention policy")
		return err
	}

	log.Info().Str(KEY_PKG, PKG_CMD).Msg("Unified cleanup operation completed")
	return nil
}

func (c *CleanupCmd) runSeparateCleanup(ctx *CLIContext, retentionManager *retention.Manager) error {
	log.Info().Str(KEY_PKG, PKG_CMD).Msg("Using separate retention evaluation for each storage type")

	// Clean local snapshots
	if !c.Remote {
		log.Info().Str(KEY_PKG, PKG_CMD).Msg("Cleaning local snapshots")
		if err := retentionManager.ApplyLocal(ctx.Config.Etcd.SnapshotDir, c.DryRun); err != nil {
			log.Error().Str(KEY_PKG, PKG_CMD).Err(err).Msgf("Failed to clean local snapshots")
		} else {
			log.Info().Str(KEY_PKG, PKG_CMD).Msg("Local snapshot cleanup completed")
		}
	}

	// Clean S3 snapshots
	if !c.Local {
		log.Info().Str(KEY_PKG, PKG_CMD).Msg("Cleaning S3 snapshots")
		s3Client, err := ctx.GetS3Client()
		if err != nil {
			log.Error().Str(KEY_PKG, PKG_CMD).Err(err).Msgf("Failed to create S3 client")
		} else {
			if err := retentionManager.ApplyS3(context.Background(), s3Client, c.DryRun); err != nil {
				log.Error().Str(KEY_PKG, PKG_CMD).Err(err).Msgf("Failed to clean S3 snapshots")
			} else {
				log.Info().Str(KEY_PKG, PKG_CMD).Msg("S3 snapshot cleanup completed")
			}
		}
	}

	log.Info().Str(KEY_PKG, PKG_CMD).Msg("Cleanup operation completed")
	return nil
}
