# Trending Algorithm V2 Design

**Date**: 2025-12-22
**Status**: Approved
**Author**: Brainstorming session

---

## Summary

Improve the trending algorithm to better surface lesser-known addons and add position tracking. Three main changes:

1. **Position tracking** - Show rank movement (24h and 7d)
2. **Signal cleanup** - Remove useless thumbs_up signal (20% of algorithm was wasted)
3. **Rising Stars rework** - Use relative growth, remove size penalty

---

## Research Findings

Analyzed trending algorithms from Hacker News, Reddit, YouTube, Steam, and 2025 social media trends.

| Platform | Key Insight |
|----------|-------------|
| Hacker News | Time decay with gravity: `score / (age + 2)^G` |
| Reddit | Wilson score for small samples, log scaling |
| YouTube | Relative performance (CTR vs expectations), not absolute |
| Steam | Follows player interest, wishlist velocity > total wishlists |
| Instagram/Twitter 2025 | Explicitly favoring smaller creators for discovery |

**Key principle**: Modern discovery algorithms favor **relative performance** over absolute numbers to give smaller creators a fair chance.

---

## Database Changes

### New Table: Position History

```sql
CREATE TABLE trending_rank_history (
    addon_id INTEGER NOT NULL REFERENCES addons(id) ON DELETE CASCADE,
    category TEXT NOT NULL CHECK (category IN ('hot', 'rising')),
    rank SMALLINT NOT NULL,
    score DECIMAL(20,10) NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (addon_id, category, recorded_at)
);

CREATE INDEX idx_rank_history_time
    ON trending_rank_history(category, recorded_at DESC);

CREATE INDEX idx_rank_history_recorded
    ON trending_rank_history(recorded_at);
```

### Retention

- Keep 7 days of history
- Delete older rows during each sync job
- ~6,700 rows max (40 addons x 2 categories x 168 hours)

---

## Algorithm Changes

### Signal Cleanup

**Removed**: `thumbs_up` signal (20% weight was wasted - most addons have 0-1 thumbs)

### Hot Right Now (Established Addons)

**Eligibility**: >=500 total downloads, positive velocity

```
velocity = confidence_weighted(velocity_24h, velocity_7d)
signal = (0.85 x velocity) + (0.15 x update_boost)
score = (signal x size_multiplier x maintenance_multiplier) / (age + 2)^1.5
```

| Component | Value |
|-----------|-------|
| Download weight | 0.85 |
| Update boost weight | 0.15 |
| Size multiplier | Kept (log scale, 0.1-1.0) |
| Gravity | 1.5 |

### Rising Stars (Discovery)

**Eligibility**: 50-10,000 total downloads, positive growth, not in Hot

```
relative_growth = downloads_gained_24h / total_downloads
signal = (0.70 x relative_growth) + (0.30 x maintenance_multiplier)
score = signal / (age + 2)^1.8
```

| Component | Value |
|-----------|-------|
| Relative growth weight | 0.70 |
| Maintenance weight | 0.30 |
| Size multiplier | **Removed** |
| Gravity | 1.8 |

**Key change**: Relative growth naturally favors small addons. An addon going 100->200 (100% growth) beats 100k->101k (1% growth).

---

## API Changes

### Response Format

Current:
```json
{"id": 123, "name": "Details", "score": 1245.29}
```

New:
```json
{
  "id": 123,
  "name": "Details",
  "score": 1245.29,
  "rank": 1,
  "rank_change_24h": 0,
  "rank_change_7d": 2
}
```

- `rank_change_24h`: positive = moved up, negative = moved down, null = new to list
- `rank_change_7d`: same logic for weekly movement

---

## Implementation Plan

### Files to Modify

| File | Changes |
|------|---------|
| `sql/schema.sql` | Add `trending_rank_history` table |
| `sql/queries.sql` | Add queries for rank history insert, lookup, cleanup |
| `internal/database/*.go` | Regenerate sqlc |
| `internal/trending/trending.go` | New constants, remove thumbs entirely, add relative growth |
| `internal/trending/calculator.go` | Calculate ranks, store history, cleanup old rows |
| `internal/trending/trending_test.go` | Update tests for new formulas |
| `internal/api/handlers.go` | Add rank fields to response |
| `docs/ALGORITHM.md` | Update documentation |

### Implementation Order

1. **Database first** - Add table, regenerate sqlc
2. **Calculator changes** - Rank history storage and cleanup
3. **Algorithm changes** - New formulas, remove all thumbs-related code
4. **API changes** - Add rank/change fields to response
5. **Tests** - Update existing, add new for rank tracking
6. **Documentation** - Update ALGORITHM.md

### Migration Strategy

- New table is additive, no existing data affected
- Algorithm changes take effect immediately on next sync
- Rank history starts empty, 24h/7d changes show as `null` until data accumulates
- No breaking API changes (new fields are additive)

### Cleanup

- Remove `ThumbsWeight` constant entirely
- Remove `thumbs_velocity` and `thumbs_growth_pct` from calculator logic
- Keep DB columns for now (historical data), can drop in future migration

---

## Comparison: Before vs After

| Aspect | V1 (Current) | V2 (New) |
|--------|--------------|----------|
| Thumbs weight | 20% | 0% |
| Rising size penalty | Yes (0.1-1.0x) | No |
| Rising growth metric | Absolute % | Relative to size |
| Maintenance in Rising | Separate multiplier | Part of signal (30%) |
| Position tracking | None | 24h and 7d rank changes |
| Small addon discovery | Penalized | Favored |

---

## Future Considerations

Not in scope for V2, but worth exploring later:

- Category-relative ranking (e.g., #1 in "Roleplay" gets visibility)
- New addon grace period for initial discovery
- Popularity rank velocity from CurseForge as additional signal
