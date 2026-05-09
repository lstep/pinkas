package pages

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// SpaceRetention holds snapshot retention settings for a space.
type SpaceRetention struct {
	ID            string
	RetentionDays int64
}

// defaultRetentionDays is used when a space has no retention configured.
const defaultRetentionDays int64 = 90

// RunSnapshotRetention deletes snapshots older than each space's retention period.
func RunSnapshotRetention(ctx context.Context, pageRepo *Repository, spaces []SpaceRetention, logger *slog.Logger) {
	totalDeleted := int64(0)
	for _, sp := range spaces {
		retentionDays := sp.RetentionDays
		if retentionDays <= 0 {
			retentionDays = defaultRetentionDays
		}
		cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).Unix()

		count, err := pageRepo.DeleteSnapshotsOlderThan(ctx, cutoff)
		if err != nil {
			logger.Error("[maintenance] snapshot retention failed",
				"space", sp.ID,
				"error", err,
			)
			continue
		}
		if count > 0 {
			totalDeleted += count
			logger.Info("[maintenance] deleted old snapshots",
				"space", sp.ID,
				"retention_days", retentionDays,
				"deleted", count,
			)
		}
	}

	if totalDeleted > 0 {
		logger.Info("[maintenance] snapshot retention complete", "total_deleted", totalDeleted)
	}
}

// RunCompaction finds pages with large Yjs snapshots and compacts them via the sidecar.
func RunCompaction(ctx context.Context, pageRepo *Repository, collabURL string, logger *slog.Logger, thresholdBytes int) {
	if thresholdBytes <= 0 {
		thresholdBytes = 500 * 1024 // 500KB default
	}

	pageIDs, err := pageRepo.ListPagesWithLargeSnapshots(ctx, thresholdBytes)
	if err != nil {
		logger.Error("[maintenance] failed to list large snapshots", "error", err)
		return
	}

	if len(pageIDs) == 0 {
		return
	}

	logger.Info("[maintenance] compacting large Yjs snapshots",
		"count", len(pageIDs),
		"threshold", thresholdBytes,
	)

	for _, docID := range pageIDs {
		if err := compactViaSidecar(ctx, collabURL, docID, logger); err != nil {
			logger.Error("[maintenance] compaction failed",
				"doc_id", docID,
				"error", err,
			)
		}
	}
}

func compactViaSidecar(ctx context.Context, collabURL, docID string, logger *slog.Logger) error {
	body, _ := json.Marshal(map[string]string{"docId": docID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, collabURL+"/internal/compact", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	logger.Info("[maintenance] compacted snapshot", "doc_id", docID)
	return nil
}
