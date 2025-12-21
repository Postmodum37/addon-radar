-- name: UpsertAddon :exec
INSERT INTO addons (
    id, name, slug, summary, author_name, author_id, logo_url,
    primary_category_id, categories, game_versions,
    created_at, last_updated_at, last_synced_at,
    download_count, thumbs_up_count, popularity_rank, rating, latest_file_date
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), $13, $14, $15, $16, $17
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    slug = EXCLUDED.slug,
    summary = EXCLUDED.summary,
    author_name = EXCLUDED.author_name,
    author_id = EXCLUDED.author_id,
    logo_url = EXCLUDED.logo_url,
    primary_category_id = EXCLUDED.primary_category_id,
    categories = EXCLUDED.categories,
    game_versions = EXCLUDED.game_versions,
    last_updated_at = EXCLUDED.last_updated_at,
    last_synced_at = NOW(),
    download_count = EXCLUDED.download_count,
    thumbs_up_count = EXCLUDED.thumbs_up_count,
    popularity_rank = EXCLUDED.popularity_rank,
    rating = EXCLUDED.rating,
    latest_file_date = EXCLUDED.latest_file_date,
    status = 'active';

-- name: CreateSnapshot :exec
INSERT INTO snapshots (addon_id, recorded_at, download_count, thumbs_up_count, popularity_rank, rating, latest_file_date)
VALUES ($1, NOW(), $2, $3, $4, $5, $6);

-- name: GetAddonByID :one
SELECT * FROM addons WHERE id = $1;

-- name: GetAddonBySlug :one
SELECT * FROM addons WHERE slug = $1 AND status = 'active';

-- name: GetAllAddonIDs :many
SELECT id FROM addons WHERE status = 'active';

-- name: GetHotAddonIDs :many
SELECT id FROM addons WHERE is_hot = TRUE AND status = 'active';

-- name: CountAddons :one
SELECT COUNT(*) FROM addons WHERE status = 'active';

-- name: CountSnapshots :one
SELECT COUNT(*) FROM snapshots;

-- name: UpsertCategory :exec
INSERT INTO categories (id, name, slug, parent_id, icon_url)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    slug = EXCLUDED.slug,
    parent_id = EXCLUDED.parent_id,
    icon_url = EXCLUDED.icon_url;

-- name: ListAddons :many
SELECT * FROM addons
WHERE status = 'active'
ORDER BY download_count DESC
LIMIT $1 OFFSET $2;

-- name: CountActiveAddons :one
SELECT COUNT(*) FROM addons WHERE status = 'active';

-- name: ListAddonsByCategory :many
SELECT a.* FROM addons a
WHERE a.status = 'active'
  AND $3 = ANY(a.categories)
ORDER BY a.download_count DESC
LIMIT $1 OFFSET $2;

-- name: CountAddonsByCategory :one
SELECT COUNT(*) FROM addons
WHERE status = 'active'
  AND $1 = ANY(categories);

-- name: SearchAddons :many
SELECT * FROM addons
WHERE status = 'active'
  AND (name ILIKE '%' || $3 || '%' OR summary ILIKE '%' || $3 || '%')
ORDER BY download_count DESC
LIMIT $1 OFFSET $2;

-- name: CountSearchAddons :one
SELECT COUNT(*) FROM addons
WHERE status = 'active'
  AND (name ILIKE '%' || $1 || '%' OR summary ILIKE '%' || $1 || '%');

-- name: GetAddonSnapshots :many
SELECT recorded_at, download_count, thumbs_up_count, popularity_rank
FROM snapshots
WHERE addon_id = $1
ORDER BY recorded_at DESC
LIMIT $2;

-- name: ListCategories :many
SELECT * FROM categories ORDER BY name;

-- name: GetCategoryBySlug :one
SELECT * FROM categories WHERE slug = $1;

-- name: GetSnapshotStats :one
-- Gets download/thumbs changes for velocity calculation
SELECT
    COALESCE(MAX(download_count) - MIN(download_count), 0) AS download_change,
    COALESCE(MAX(thumbs_up_count) - MIN(thumbs_up_count), 0) AS thumbs_change,
    COUNT(*) AS snapshot_count,
    MIN(download_count) AS min_downloads,
    MAX(download_count) AS max_downloads
FROM snapshots
WHERE addon_id = $1
  AND recorded_at >= NOW() - ($2 || ' hours')::INTERVAL;

-- name: GetDownloadPercentile :one
-- Gets the Nth percentile of total downloads for size multiplier calculation
SELECT COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY download_count), 500000)::FLOAT8 AS percentile_95
FROM addons
WHERE status = 'active' AND download_count > 0;

-- name: GetAddonLatestFileDate :one
-- Gets the latest file date for maintenance multiplier
SELECT latest_file_date FROM addons WHERE id = $1;

-- name: CountRecentFileUpdates :one
-- Counts file updates in last N days (approximated by comparing latest_file_date changes in snapshots)
SELECT COUNT(DISTINCT DATE(latest_file_date))
FROM snapshots
WHERE addon_id = $1
  AND recorded_at >= NOW() - ($2 || ' days')::INTERVAL
  AND latest_file_date IS NOT NULL;

-- name: UpsertTrendingScore :exec
INSERT INTO trending_scores (
    addon_id, hot_score, rising_score,
    download_velocity, thumbs_velocity,
    download_growth_pct, thumbs_growth_pct,
    size_multiplier, maintenance_multiplier,
    first_hot_at, first_rising_at, calculated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
ON CONFLICT (addon_id) DO UPDATE SET
    hot_score = EXCLUDED.hot_score,
    rising_score = EXCLUDED.rising_score,
    download_velocity = EXCLUDED.download_velocity,
    thumbs_velocity = EXCLUDED.thumbs_velocity,
    download_growth_pct = EXCLUDED.download_growth_pct,
    thumbs_growth_pct = EXCLUDED.thumbs_growth_pct,
    size_multiplier = EXCLUDED.size_multiplier,
    maintenance_multiplier = EXCLUDED.maintenance_multiplier,
    first_hot_at = COALESCE(EXCLUDED.first_hot_at, trending_scores.first_hot_at),
    first_rising_at = COALESCE(EXCLUDED.first_rising_at, trending_scores.first_rising_at),
    calculated_at = NOW();

-- name: GetTrendingScore :one
SELECT * FROM trending_scores WHERE addon_id = $1;

-- name: ListHotAddons :many
SELECT a.*, t.hot_score
FROM addons a
JOIN trending_scores t ON a.id = t.addon_id
WHERE a.status = 'active'
  AND a.download_count >= 500
  AND t.hot_score > 0
ORDER BY t.hot_score DESC
LIMIT $1;

-- name: ListRisingAddons :many
SELECT a.*, t.rising_score
FROM addons a
JOIN trending_scores t ON a.id = t.addon_id
WHERE a.status = 'active'
  AND a.download_count >= 50
  AND a.download_count <= 10000
  AND t.rising_score > 0
  AND a.id NOT IN (
      SELECT addon_id FROM trending_scores
      WHERE hot_score > 0
      ORDER BY hot_score DESC
      LIMIT 20
  )
ORDER BY t.rising_score DESC
LIMIT $1;

-- name: ClearTrendingAgeForDroppedAddons :exec
-- Reset first_hot_at for addons that dropped out of hot list
UPDATE trending_scores
SET first_hot_at = NULL
WHERE addon_id NOT IN (
    SELECT addon_id FROM trending_scores
    WHERE hot_score > 0
    ORDER BY hot_score DESC
    LIMIT 20
);

-- name: ClearRisingAgeForDroppedAddons :exec
-- Reset first_rising_at for addons that dropped out of rising list
UPDATE trending_scores
SET first_rising_at = NULL
WHERE addon_id NOT IN (
    SELECT addon_id FROM trending_scores
    WHERE rising_score > 0
    ORDER BY rising_score DESC
    LIMIT 20
);

-- name: ListAddonsForTrendingCalc :many
-- Get addons with basic info needed for trending calculation
SELECT id, download_count, thumbs_up_count, latest_file_date, created_at
FROM addons
WHERE status = 'active';

-- name: GetAllSnapshotStats :many
-- Bulk fetch snapshot stats for all addons in both time windows
WITH stats_24h AS (
    SELECT
        addon_id,
        COALESCE(MAX(download_count) - MIN(download_count), 0)::bigint AS download_change,
        COALESCE(MAX(thumbs_up_count) - MIN(thumbs_up_count), 0)::int AS thumbs_change,
        COUNT(*)::int AS snapshot_count,
        MIN(download_count)::bigint AS min_downloads
    FROM snapshots
    WHERE recorded_at >= NOW() - INTERVAL '24 hours'
    GROUP BY addon_id
),
stats_7d AS (
    SELECT
        addon_id,
        COALESCE(MAX(download_count) - MIN(download_count), 0)::bigint AS download_change,
        COALESCE(MAX(thumbs_up_count) - MIN(thumbs_up_count), 0)::int AS thumbs_change,
        MIN(download_count)::bigint AS min_downloads
    FROM snapshots
    WHERE recorded_at >= NOW() - INTERVAL '7 days'
    GROUP BY addon_id
)
SELECT
    a.id AS addon_id,
    a.download_count,
    a.thumbs_up_count,
    a.latest_file_date,
    a.created_at,
    COALESCE(s24.download_change, 0) AS download_change_24h,
    COALESCE(s24.thumbs_change, 0) AS thumbs_change_24h,
    COALESCE(s24.snapshot_count, 0) AS snapshot_count_24h,
    COALESCE(s7.download_change, 0) AS download_change_7d,
    COALESCE(s7.thumbs_change, 0) AS thumbs_change_7d,
    COALESCE(s7.min_downloads, a.download_count) AS min_downloads_7d
FROM addons a
LEFT JOIN stats_24h s24 ON a.id = s24.addon_id
LEFT JOIN stats_7d s7 ON a.id = s7.addon_id
WHERE a.status = 'active';

-- name: GetAllTrendingScores :many
-- Bulk fetch all existing trending scores
SELECT addon_id, first_hot_at, first_rising_at
FROM trending_scores;

-- name: CountAllRecentFileUpdates :many
-- Bulk count file updates for all addons
SELECT
    addon_id,
    COUNT(DISTINCT DATE(latest_file_date))::int AS update_count
FROM snapshots
WHERE recorded_at >= NOW() - INTERVAL '90 days'
  AND latest_file_date IS NOT NULL
GROUP BY addon_id;

-- name: DeleteOldSnapshotsBatch :execrows
-- Delete snapshots older than 95 days in batches to avoid long-running transactions
-- Use ORDER BY id for consistent batching (faster than ORDER BY recorded_at)
DELETE FROM snapshots
WHERE id IN (
    SELECT id FROM snapshots
    WHERE recorded_at < NOW() - INTERVAL '95 days'
    ORDER BY id
    LIMIT $1
);

-- name: CountOldSnapshots :one
-- Count snapshots older than 95 days (for progress logging)
SELECT COUNT(*) FROM snapshots
WHERE recorded_at < NOW() - INTERVAL '95 days';

-- name: MarkMissingAddonsInactive :execrows
-- Mark addons as inactive if they no longer appear in CurseForge API response
WITH synced_ids AS (SELECT unnest($1::integer[]) AS id)
UPDATE addons
SET status = 'inactive', last_synced_at = NOW()
WHERE status = 'active'
  AND NOT EXISTS (SELECT 1 FROM synced_ids WHERE synced_ids.id = addons.id);

-- name: InsertRankHistory :exec
-- Record current rank for an addon in a category
INSERT INTO trending_rank_history (addon_id, category, rank, score, recorded_at)
VALUES ($1, $2, $3, $4, NOW());

-- name: GetRankAt :one
-- Get the rank of an addon at a specific time (closest record before that time)
SELECT rank FROM trending_rank_history
WHERE addon_id = $1
  AND category = $2
  AND recorded_at <= $3
ORDER BY recorded_at DESC
LIMIT 1;

-- name: DeleteOldRankHistory :execrows
-- Delete rank history older than 7 days
DELETE FROM trending_rank_history
WHERE recorded_at < NOW() - INTERVAL '7 days';

-- name: GetRankChanges :many
-- Get rank changes for top addons (24h and 7d ago)
WITH current_ranks AS (
    SELECT addon_id, category, rank, score
    FROM trending_rank_history
    WHERE recorded_at = (
        SELECT MAX(recorded_at) FROM trending_rank_history
    )
),
ranks_24h AS (
    SELECT DISTINCT ON (addon_id, category) addon_id, category, rank
    FROM trending_rank_history
    WHERE recorded_at <= NOW() - INTERVAL '24 hours'
    ORDER BY addon_id, category, recorded_at DESC
),
ranks_7d AS (
    SELECT DISTINCT ON (addon_id, category) addon_id, category, rank
    FROM trending_rank_history
    WHERE recorded_at <= NOW() - INTERVAL '7 days'
    ORDER BY addon_id, category, recorded_at DESC
)
SELECT
    c.addon_id,
    c.category,
    c.rank AS current_rank,
    c.score,
    r24.rank AS rank_24h_ago,
    r7.rank AS rank_7d_ago
FROM current_ranks c
LEFT JOIN ranks_24h r24 ON c.addon_id = r24.addon_id AND c.category = r24.category
LEFT JOIN ranks_7d r7 ON c.addon_id = r7.addon_id AND c.category = r7.category;
