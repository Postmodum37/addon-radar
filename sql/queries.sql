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
    latest_file_date = EXCLUDED.latest_file_date;

-- name: CreateSnapshot :exec
INSERT INTO snapshots (addon_id, recorded_at, download_count, thumbs_up_count, popularity_rank, rating, latest_file_date)
VALUES ($1, NOW(), $2, $3, $4, $5, $6);

-- name: GetAddonByID :one
SELECT * FROM addons WHERE id = $1;

-- name: GetAddonBySlug :one
SELECT * FROM addons WHERE slug = $1;

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
