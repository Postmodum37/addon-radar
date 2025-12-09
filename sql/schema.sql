-- Addons table: stores addon metadata
CREATE TABLE addons (
    id INTEGER PRIMARY KEY,  -- CurseForge addon ID
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    summary TEXT,
    author_name TEXT,
    author_id INTEGER,
    logo_url TEXT,
    primary_category_id INTEGER,
    categories INTEGER[] DEFAULT '{}',
    game_versions TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ,
    last_updated_at TIMESTAMPTZ,
    last_synced_at TIMESTAMPTZ DEFAULT NOW(),
    is_hot BOOLEAN DEFAULT FALSE,
    hot_until TIMESTAMPTZ,
    status TEXT DEFAULT 'active',

    -- Current metrics (updated each sync)
    download_count BIGINT DEFAULT 0,
    thumbs_up_count INTEGER DEFAULT 0,
    popularity_rank INTEGER,
    rating DECIMAL(3,2),
    latest_file_date TIMESTAMPTZ
);

CREATE INDEX idx_addons_slug ON addons(slug);
CREATE INDEX idx_addons_is_hot ON addons(is_hot) WHERE is_hot = TRUE;
CREATE INDEX idx_addons_status ON addons(status);

-- Snapshots table: time-series metrics
CREATE TABLE snapshots (
    id BIGSERIAL PRIMARY KEY,
    addon_id INTEGER NOT NULL REFERENCES addons(id) ON DELETE CASCADE,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    download_count BIGINT NOT NULL,
    thumbs_up_count INTEGER,
    popularity_rank INTEGER,
    rating DECIMAL(3,2),
    latest_file_date TIMESTAMPTZ
);

CREATE INDEX idx_snapshots_addon_time ON snapshots(addon_id, recorded_at DESC);
CREATE INDEX idx_snapshots_recorded_at ON snapshots(recorded_at DESC);

-- Categories table: reference data
CREATE TABLE categories (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    parent_id INTEGER REFERENCES categories(id),
    icon_url TEXT
);
