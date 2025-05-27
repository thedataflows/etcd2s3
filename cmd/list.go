package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/thedataflows/etcd2s3/pkg/retention"
	log "github.com/thedataflows/go-lib-log"
)

// ListCmd lists snapshots
type ListCmd struct {
	Local   bool   `kong:"help='List local snapshots only'"`
	Remote  bool   `kong:"help='List S3 snapshots only'"`
	Format  string `kong:"help='Output format (table,json,yaml)',default='table'"`
	Unified bool   `kong:"help='Use unified retention evaluation across local and S3',default=true"`
}

type SnapshotInfo struct {
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	Size      int64     `json:"size"`
	Modified  time.Time `json:"modified"`
	Retention string    `json:"retention"` // "keep" or "delete"
}

func (l *ListCmd) Run(ctx *CLIContext) error {
	log.Info(PKG_CMD, "Listing snapshots")

	// Create retention manager
	retentionMgr := retention.NewManager(ctx.Config.Policy)

	// Use unified approach if both local and remote snapshots are being listed
	if l.Unified && !l.Local && !l.Remote {
		return l.runUnifiedList(ctx, retentionMgr)
	}

	// Use separate approach for individual storage types
	return l.runSeparateList(ctx, retentionMgr)
}

func (l *ListCmd) runUnifiedList(ctx *CLIContext, retentionMgr *retention.Manager) error {
	// Get snapshots from both locations
	localRetentionSnapshots, err := l.getLocalRetentionSnapshots(ctx.Config.Etcd.SnapshotDir)
	if err != nil {
		log.Logger.Error().Err(err).Str(log.KEY_PKG, PKG_CMD).Msg("Failed to get local snapshots")
		localRetentionSnapshots = nil
	}

	s3RetentionSnapshots, err := l.getS3RetentionSnapshots(ctx)
	if err != nil {
		log.Logger.Error().Err(err).Str(log.KEY_PKG, PKG_CMD).Str("url", ctx.Config.S3.EndpointURL).Str("bucket", ctx.Config.S3.Bucket).Msg("Failed to get S3 snapshots")
		s3RetentionSnapshots = nil
	}

	// Get unified retention decisions
	retentionDecisions := retentionMgr.GetUnifiedRetentionStatus(localRetentionSnapshots, s3RetentionSnapshots)

	var snapshots []SnapshotInfo

	// Build final snapshot list with unified retention status
	for _, retSnap := range localRetentionSnapshots {
		retentionStatus := "delete"
		if retentionDecisions[retSnap.Name] {
			retentionStatus = "keep"
		}

		snapshots = append(snapshots, SnapshotInfo{
			Name:      retSnap.Name,
			Location:  "local",
			Size:      retSnap.Size,
			Modified:  retSnap.ModTime,
			Retention: retentionStatus,
		})
	}

	for _, retSnap := range s3RetentionSnapshots {
		retentionStatus := "delete"
		if retentionDecisions[retSnap.Name] {
			retentionStatus = "keep"
		}

		snapshots = append(snapshots, SnapshotInfo{
			Name:      retSnap.Name,
			Location:  "s3",
			Size:      retSnap.Size,
			Modified:  retSnap.ModTime,
			Retention: retentionStatus,
		})
	}

	// Sort by modified time (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Modified.After(snapshots[j].Modified)
	})

	return l.outputSnapshots(snapshots)
}

func (l *ListCmd) runSeparateList(ctx *CLIContext, retentionMgr *retention.Manager) error {
	var snapshots []SnapshotInfo

	// List local snapshots
	if !l.Remote {
		localSnapshots, err := l.listLocal(ctx.Config.Etcd.SnapshotDir, retentionMgr)
		if err != nil {
			log.Logger.Error().Err(err).Str(log.KEY_PKG, PKG_CMD).Msg("Failed to list local snapshots")
		} else {
			snapshots = append(snapshots, localSnapshots...)
		}
	}

	// List S3 snapshots
	if !l.Local {
		s3Snapshots, err := l.listS3(ctx, retentionMgr)
		if err != nil {
			log.Logger.Error().Err(err).Str(log.KEY_PKG, PKG_CMD).Msg("Failed to list S3 snapshots")
		} else {
			snapshots = append(snapshots, s3Snapshots...)
		}
	}

	// Sort by modified time (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Modified.After(snapshots[j].Modified)
	})

	return l.outputSnapshots(snapshots)
}

func (l *ListCmd) outputSnapshots(snapshots []SnapshotInfo) error {
	switch l.Format {
	case "json":
		return l.outputJSON(snapshots)
	case "yaml":
		return l.outputYAML(snapshots)
	default:
		return l.outputTable(snapshots)
	}
}

func (l *ListCmd) listLocal(snapshotDir string, retentionMgr *retention.Manager) ([]SnapshotInfo, error) {
	var snapshots []SnapshotInfo

	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		return snapshots, nil
	}

	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	// Build retention snapshots for analysis
	var retentionSnapshots []retention.SnapshotFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !retention.IsSnapshotFile(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		retentionSnapshots = append(retentionSnapshots, retention.SnapshotFile{
			Name:     entry.Name(),
			Path:     filepath.Join(snapshotDir, entry.Name()),
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			IsRemote: false,
		})
	}

	// Determine which snapshots to keep according to retention policy
	toKeep := retentionMgr.GetRetentionStatus(retentionSnapshots)

	// Build final snapshot list with retention status
	for _, retSnap := range retentionSnapshots {
		retentionStatus := "delete"
		if toKeep[retSnap.Name] {
			retentionStatus = "keep"
		}

		snapshots = append(snapshots, SnapshotInfo{
			Name:      retSnap.Name,
			Location:  "local",
			Size:      retSnap.Size,
			Modified:  retSnap.ModTime,
			Retention: retentionStatus,
		})
	}

	return snapshots, nil
}

func (l *ListCmd) listS3(ctx *CLIContext, retentionMgr *retention.Manager) ([]SnapshotInfo, error) {
	s3Client, err := ctx.GetS3Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	objects, err := s3Client.List(context.Background(), "")
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	// Build retention snapshots for analysis
	var retentionSnapshots []retention.SnapshotFile
	for _, obj := range objects {
		retentionSnapshots = append(retentionSnapshots, retention.SnapshotFile{
			Name:     filepath.Base(obj.Key),
			Path:     obj.Key,
			Size:     obj.Size,
			ModTime:  obj.LastModified,
			IsRemote: true,
		})
	}

	// Determine which snapshots to keep according to retention policy
	toKeep := retentionMgr.GetRetentionStatus(retentionSnapshots)

	// Build final snapshot list with retention status
	var snapshots []SnapshotInfo
	for _, retSnap := range retentionSnapshots {
		retentionStatus := "delete"
		if toKeep[retSnap.Name] {
			retentionStatus = "keep"
		}

		snapshots = append(snapshots, SnapshotInfo{
			Name:      retSnap.Name,
			Location:  "s3",
			Size:      retSnap.Size,
			Modified:  retSnap.ModTime,
			Retention: retentionStatus,
		})
	}

	return snapshots, nil
}

func (l *ListCmd) outputTable(snapshots []SnapshotInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tLOCATION\tSIZE\tMODIFIED\tRETENTION")

	for _, snapshot := range snapshots {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			snapshot.Name,
			snapshot.Location,
			formatSize(snapshot.Size),
			snapshot.Modified.Format("2006-01-02 15:04:05"),
			snapshot.Retention,
		)
	}

	return w.Flush()
}

func (l *ListCmd) outputJSON(snapshots []SnapshotInfo) error {
	out, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshots to JSON: %w", err)
	}
	fmt.Print(string(out))
	return nil
}

func (l *ListCmd) outputYAML(snapshots []SnapshotInfo) error {
	out, err := yaml.MarshalWithOptions(snapshots, yaml.Indent(4))
	if err != nil {
		return fmt.Errorf("failed to marshal snapshots to YAML: %w", err)
	}
	fmt.Print(string(out))
	return nil
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// getLocalRetentionSnapshots returns snapshots from local directory for unified retention evaluation
func (l *ListCmd) getLocalRetentionSnapshots(snapshotDir string) ([]retention.SnapshotFile, error) {
	var snapshots []retention.SnapshotFile

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

		if !retention.IsSnapshotFile(entry.Name()) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		snapshots = append(snapshots, retention.SnapshotFile{
			Name:     entry.Name(),
			Path:     filepath.Join(snapshotDir, entry.Name()),
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			IsRemote: false,
		})
	}

	return snapshots, nil
}

// getS3RetentionSnapshots returns snapshots from S3 for unified retention evaluation
func (l *ListCmd) getS3RetentionSnapshots(ctx *CLIContext) ([]retention.SnapshotFile, error) {
	s3Client, err := ctx.GetS3Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	objects, err := s3Client.List(context.Background(), "")
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	var snapshots []retention.SnapshotFile
	for _, obj := range objects {
		snapshots = append(snapshots, retention.SnapshotFile{
			Name:     filepath.Base(obj.Key),
			Path:     obj.Key,
			Size:     obj.Size,
			ModTime:  obj.LastModified,
			IsRemote: true,
		})
	}

	return snapshots, nil
}
