-- 007_snapshot_index.up.sql
-- Iteration 4: index for snapshot listing

CREATE INDEX IF NOT EXISTS idx_page_snapshots_page_created
    ON page_snapshots(page_id, created_at DESC);
