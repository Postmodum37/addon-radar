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
