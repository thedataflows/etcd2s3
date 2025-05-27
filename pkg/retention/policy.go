package retention

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/thedataflows/etcd2s3/pkg/appconfig"
	"github.com/thedataflows/etcd2s3/pkg/compression"
	"github.com/thedataflows/etcd2s3/pkg/s3"
	log "github.com/thedataflows/go-lib-log"
)

const PKG_RETENTION = "retention"

// Manager handles retention policies for snapshots
type Manager struct {
	policy appconfig.RetentionPolicy
}

// SnapshotFile represents a snapshot file with metadata
type SnapshotFile struct {
	Name     string
	Path     string
	Size     int64
	ModTime  time.Time
	IsRemote bool
}

// NewManager creates a new retention manager
func NewManager(policy appconfig.RetentionPolicy) *Manager {
	return &Manager{
		policy: policy,
	}
}

// ApplyLocal applies retention policies to local snapshots
func (m *Manager) ApplyLocal(snapshotDir string, dryRun bool) error {
	log.Info(PKG_RETENTION, "Applying local retention policies")

	// Get all local snapshots
	snapshots, err := m.GetLocalSnapshots(snapshotDir)
	if err != nil {
		return fmt.Errorf("failed to get local snapshots: %w", err)
	}

	// Determine which snapshots to keep
	toKeep := m.determineSnapshotsToKeep(snapshots)
	toDelete := m.findSnapshotsToDelete(snapshots, toKeep)

	// Delete snapshots
	for _, snapshot := range toDelete {
		if dryRun {
			log.Warnf(PKG_RETENTION, "[DRY RUN] Would delete local snapshot: %s", snapshot.Name)
		} else {
			log.Warnf(PKG_RETENTION, "Deleting local snapshot: %s", snapshot.Name)
			if err := os.Remove(snapshot.Path); err != nil {
				log.Errorf(PKG_RETENTION, err, "Failed to delete local snapshot '%s'", snapshot.Path)
			}
		}
	}

	if dryRun {
		log.Infof(PKG_RETENTION, "Local retention dry run complete: %d snapshots would be kept, %d would be deleted", len(toKeep), len(toDelete))
	} else {
		log.Infof(PKG_RETENTION, "Local retention complete: %d snapshots kept, %d deleted", len(toKeep), len(toDelete))
	}
	return nil
}

// ApplyS3 applies retention policies to S3 snapshots
func (m *Manager) ApplyS3(ctx context.Context, s3Client *s3.Client, dryRun bool) error {
	log.Info(PKG_RETENTION, "Applying S3 retention policies")

	// Get all S3 snapshots
	snapshots, err := m.GetS3Snapshots(ctx, s3Client)
	if err != nil {
		return fmt.Errorf("failed to get S3 snapshots: %w", err)
	}

	// Determine which snapshots to keep
	toKeep := m.determineSnapshotsToKeep(snapshots)
	toDelete := m.findSnapshotsToDelete(snapshots, toKeep)

	// Delete snapshots
	var keys []string
	for _, snapshot := range toDelete {
		keys = append(keys, snapshot.Path) // For S3, Path contains the key
		if dryRun {
			log.Warnf(PKG_RETENTION, "[DRY RUN] Would delete S3 snapshot: %s", snapshot.Name)
		}
	}

	if len(keys) > 0 && !dryRun {
		log.Warnf(PKG_RETENTION, "Deleting %d S3 snapshots", len(keys))
		if err := s3Client.DeleteMultiple(ctx, keys); err != nil {
			return fmt.Errorf("failed to delete S3 snapshots: %w", err)
		}
	}

	if dryRun {
		log.Infof(PKG_RETENTION, "S3 retention dry run complete: %d snapshots would be kept, %d would be deleted", len(toKeep), len(toDelete))
	} else {
		log.Infof(PKG_RETENTION, "S3 retention complete: %d snapshots kept, %d deleted", len(toKeep), len(toDelete))
	}
	return nil
}

// GetRetentionStatus returns a map indicating which snapshots should be kept
func (m *Manager) GetRetentionStatus(snapshots []SnapshotFile) map[string]bool {
	return m.determineSnapshotsToKeep(snapshots)
}

// GetUnifiedRetentionStatus evaluates retention across both local and S3 snapshots
// This ensures consistent retention decisions for snapshots that exist in multiple locations
func (m *Manager) GetUnifiedRetentionStatus(localSnapshots, s3Snapshots []SnapshotFile) map[string]bool {
	// Create a unified list of unique snapshots by name, preferring the most recent version
	unifiedSnapshots := m.createUnifiedSnapshotList(localSnapshots, s3Snapshots)

	// Apply retention policy to the unified list
	return m.determineSnapshotsToKeep(unifiedSnapshots)
}

// GetLocalSnapshots gets all local snapshot files
func (m *Manager) GetLocalSnapshots(snapshotDir string) ([]SnapshotFile, error) {
	var snapshots []SnapshotFile

	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		return snapshots, nil
	}

	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !IsSnapshotFile(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		snapshots = append(snapshots, SnapshotFile{
			Name:     entry.Name(),
			Path:     filepath.Join(snapshotDir, entry.Name()),
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			IsRemote: false,
		})
	}

	return snapshots, nil
}

// GetS3Snapshots gets all S3 snapshot objects
func (m *Manager) GetS3Snapshots(ctx context.Context, s3Client *s3.Client) ([]SnapshotFile, error) {
	var snapshots []SnapshotFile

	objects, err := s3Client.List(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	for _, obj := range objects {
		if !IsSnapshotFile(obj.Key) {
			continue
		}

		snapshots = append(snapshots, SnapshotFile{
			Name:     filepath.Base(obj.Key),
			Path:     obj.Key, // For S3, store the full key as path
			Size:     obj.Size,
			ModTime:  obj.LastModified,
			IsRemote: true,
		})
	}

	return snapshots, nil
}

// determineSnapshotsToKeep determines which snapshots should be kept based on retention policies
func (m *Manager) determineSnapshotsToKeep(snapshots []SnapshotFile) map[string]bool {
	toKeep := make(map[string]bool)
	now := time.Now()

	// Sort snapshots by modification time (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].ModTime.After(snapshots[j].ModTime)
	})

	// Keep last N snapshots
	if m.policy.KeepLast > 0 {
		for i, snapshot := range snapshots {
			if i < m.policy.KeepLast {
				toKeep[snapshot.Name] = true
			}
		}
	}

	// Keep snapshots within time periods
	for _, snapshot := range snapshots {
		age := now.Sub(snapshot.ModTime)

		if m.policy.KeepLastHours > 0 && age <= time.Duration(m.policy.KeepLastHours)*time.Hour {
			toKeep[snapshot.Name] = true
		}

		if m.policy.KeepLastDays > 0 && age <= time.Duration(m.policy.KeepLastDays)*24*time.Hour {
			toKeep[snapshot.Name] = true
		}

		if m.policy.KeepLastWeeks > 0 && age <= time.Duration(m.policy.KeepLastWeeks)*7*24*time.Hour {
			toKeep[snapshot.Name] = true
		}

		if m.policy.KeepLastMonths > 0 && age <= time.Duration(m.policy.KeepLastMonths)*30*24*time.Hour {
			toKeep[snapshot.Name] = true
		}

		if m.policy.KeepLastYears > 0 && age <= time.Duration(m.policy.KeepLastYears)*365*24*time.Hour {
			toKeep[snapshot.Name] = true
		}
	}

	return toKeep
}

// findSnapshotsToDelete finds snapshots that should be deleted
func (m *Manager) findSnapshotsToDelete(snapshots []SnapshotFile, toKeep map[string]bool) []SnapshotFile {
	var toDelete []SnapshotFile

	for _, snapshot := range snapshots {
		if !toKeep[snapshot.Name] {
			toDelete = append(toDelete, snapshot)
		}
	}

	return toDelete
}

// IsSnapshotFile determines if a filename represents a snapshot file
func IsSnapshotFile(filename string) bool {
	ext := filepath.Ext(filename)
	if ext == ".db" || slices.Contains(compression.AllCompressionExts(), ext) {
		return true
	}

	// Accept files that look like snapshot names (test-snapshot-*, etcd-snapshot-*, etc.)
	// This allows for flexibility in snapshot naming conventions
	if strings.Contains(filename, "snapshot") {
		return true
	}
	return false
}

// createUnifiedSnapshotList combines local and S3 snapshots into a unified list
// For snapshots that exist in both locations, it uses the most recent timestamp
func (m *Manager) createUnifiedSnapshotList(localSnapshots, s3Snapshots []SnapshotFile) []SnapshotFile {
	snapshotMap := make(map[string]SnapshotFile)

	// Add all local snapshots
	for _, snapshot := range localSnapshots {
		snapshotMap[snapshot.Name] = snapshot
	}

	// Add S3 snapshots, keeping the most recent version if it exists in both places
	for _, s3Snapshot := range s3Snapshots {
		if existing, exists := snapshotMap[s3Snapshot.Name]; exists {
			// Keep the more recent version
			if s3Snapshot.ModTime.After(existing.ModTime) {
				snapshotMap[s3Snapshot.Name] = s3Snapshot
			}
		} else {
			snapshotMap[s3Snapshot.Name] = s3Snapshot
		}
	}

	// Convert map back to slice
	var unified []SnapshotFile
	for _, snapshot := range snapshotMap {
		unified = append(unified, snapshot)
	}

	return unified
}

// ApplyUnified applies retention policies considering both local and S3 snapshots together
// This ensures consistent retention decisions across storage locations
func (m *Manager) ApplyUnified(ctx context.Context, snapshotDir string, s3Client *s3.Client, dryRun bool) error {
	log.Info(PKG_RETENTION, "Applying unified retention policies")

	// Get snapshots from both locations
	localSnapshots, err := m.GetLocalSnapshots(snapshotDir)
	if err != nil {
		return fmt.Errorf("failed to get local snapshots: %w", err)
	}

	var s3Snapshots []SnapshotFile
	if s3Client != nil {
		s3Snapshots, err = m.GetS3Snapshots(ctx, s3Client)
		if err != nil {
			return fmt.Errorf("failed to get S3 snapshots: %w", err)
		}
	}

	// Get unified retention decisions
	retentionDecisions := m.GetUnifiedRetentionStatus(localSnapshots, s3Snapshots)

	// Apply decisions to local snapshots
	localKept, localDeleted := m.applyRetentionToLocal(localSnapshots, retentionDecisions, dryRun)

	// Apply decisions to S3 snapshots
	var s3Kept, s3Deleted int
	if s3Client != nil {
		s3Kept, s3Deleted = m.applyRetentionToS3(ctx, s3Client, s3Snapshots, retentionDecisions, dryRun)
	}

	if dryRun {
		log.Infof(PKG_RETENTION, "Unified retention dry run complete: Local (%d kept, %d deleted), S3 (%d kept, %d deleted)",
			localKept, localDeleted, s3Kept, s3Deleted)
	} else {
		log.Infof(PKG_RETENTION, "Unified retention complete: Local (%d kept, %d deleted), S3 (%d kept, %d deleted)",
			localKept, localDeleted, s3Kept, s3Deleted)
	}

	return nil
}

// applyRetentionToLocal applies retention decisions to local snapshots
func (m *Manager) applyRetentionToLocal(snapshots []SnapshotFile, retentionDecisions map[string]bool, dryRun bool) (kept, deleted int) {
	for _, snapshot := range snapshots {
		if retentionDecisions[snapshot.Name] {
			kept++
		} else {
			deleted++
			if dryRun {
				log.Warnf(PKG_RETENTION, "[DRY RUN] Would delete local snapshot: %s", snapshot.Name)
			} else {
				log.Warnf(PKG_RETENTION, "Deleting local snapshot: %s", snapshot.Name)
				if err := os.Remove(snapshot.Path); err != nil {
					log.Errorf(PKG_RETENTION, err, "Failed to delete local snapshot '%s'", snapshot.Path)
				}
			}
		}
	}
	return kept, deleted
}

// applyRetentionToS3 applies retention decisions to S3 snapshots
func (m *Manager) applyRetentionToS3(ctx context.Context, s3Client *s3.Client, snapshots []SnapshotFile, retentionDecisions map[string]bool, dryRun bool) (kept, deleted int) {
	var keysToDelete []string

	for _, snapshot := range snapshots {
		if retentionDecisions[snapshot.Name] {
			kept++
		} else {
			deleted++
			keysToDelete = append(keysToDelete, snapshot.Path)
			if dryRun {
				log.Warnf(PKG_RETENTION, "[DRY RUN] Would delete S3 snapshot: %s", snapshot.Name)
			}
		}
	}

	if len(keysToDelete) > 0 && !dryRun {
		log.Warnf(PKG_RETENTION, "Deleting %d S3 snapshots", len(keysToDelete))
		if err := s3Client.DeleteMultiple(ctx, keysToDelete); err != nil {
			log.Errorf(PKG_RETENTION, err, "Failed to delete S3 snapshots")
		}
	}

	return kept, deleted
}
