# Trending Algorithm Design

> **Status: ✅ IMPLEMENTED**
>
> This is the main differentiator for Addon Radar. Implemented and deployed to production.
> See `/docs/plans/2025-12-10-trending-algorithm-implementation.md` for implementation details.

## Overview

This document describes Addon Radar's trending algorithm for surfacing popular and rising WoW addons. The algorithm uses multiple signals, adaptive time windows, and category-specific decay rates to provide meaningful trending lists.

## Two Trending Categories

### Hot Right Now
Established addons with high absolute download velocity.
- Shows proven addons with significant recent activity
- Moderate decay (gravity 1.5) - stable presence on the list
- Minimum 500 total downloads required

### Rising Stars
Smaller addons gaining traction quickly.
- Surfaces new discoveries and emerging addons
- Aggressive decay (gravity 1.8) - cycles through discoveries quickly
- Between 50 and 10,000 total downloads
- Excludes addons already in Hot Right Now

## Core Formula

### Hot Right Now Score
```
score = (weighted_velocity * size_multiplier * maintenance_multiplier) / (age_hours + 2)^1.5
```

### Rising Stars Score
```
score = (weighted_growth_pct * size_multiplier * maintenance_multiplier) / (age_hours + 2)^1.8
```

Where:
- `weighted_velocity` = blended signal from multiple metrics
- `size_multiplier` = logarithmic scale based on total downloads
- `maintenance_multiplier` = reward for consistent update cadence
- `age_hours` = hours since addon first appeared on trending (resets when it drops off)

## Weighted Velocity Calculation

### Signal Blend

**Hot Right Now:**
```
weighted_velocity = (0.7 * download_velocity) + (0.2 * thumbs_velocity) + (0.1 * update_boost)
```

**Rising Stars:**
```
weighted_growth = (0.7 * download_growth_pct) + (0.2 * thumbs_growth_pct) + (0.1 * update_boost)
```

### Confidence-Based Adaptive Windows

The algorithm adapts time windows based on data quality:

```python
def calculate_velocity(addon, metric):
    velocity_24h = calculate_velocity_for_window(addon, metric, hours=24)
    velocity_7d = calculate_velocity_for_window(addon, metric, hours=168)

    data_points_24h = count_snapshots(addon, hours=24)
    change_24h = abs(get_change(addon, metric, hours=24))

    confident_24h = (data_points_24h >= 5) and (change_24h >= 10)

    if confident_24h:
        # Weight toward fresh data when statistically meaningful
        return (0.8 * velocity_24h) + (0.2 * velocity_7d)
    else:
        # Fall back to longer window when 24h data is sparse
        return (0.3 * velocity_24h) + (0.7 * velocity_7d)
```

This ensures:
- Hot addons with hourly snapshots use responsive 24h data
- Stable addons with daily snapshots use reliable 7d data
- New addons with limited history get appropriate treatment

## Size Multiplier (Logarithmic Scale)

Prevents tiny addons from dominating while avoiding arbitrary tier boundaries.

### Formula
```python
def calculate_size_multiplier(downloads, percentile_95):
    multiplier = log10(downloads + 1) / log10(percentile_95 + 1)
    return clamp(multiplier, 0.1, 1.0)
```

### Example Values
Assuming 95th percentile = 500,000 downloads:

| Total Downloads | Multiplier |
|----------------|------------|
| 10 | 0.18 |
| 100 | 0.35 |
| 1,000 | 0.53 |
| 10,000 | 0.70 |
| 100,000 | 0.88 |
| 500,000+ | 1.00 |

### Implementation Notes
- Recalculate 95th percentile daily during full sync
- Smooth curve with no arbitrary tier jumps
- Adapts automatically as the addon ecosystem grows

## Maintenance Multiplier (Update Frequency Reward)

Rewards authors who actively maintain their addons.

### Formula
```python
def calculate_maintenance_multiplier(addon):
    updates_90d = count_file_releases(addon, days=90)

    if updates_90d == 0:
        avg_days = float('inf')
    else:
        avg_days = 90 / updates_90d

    if avg_days <= 14:
        return 1.15  # Very active (updates every 1-2 weeks)
    elif avg_days <= 30:
        return 1.10  # Regularly maintained (monthly)
    elif avg_days <= 60:
        return 1.05  # Occasionally maintained (bi-monthly)
    elif avg_days <= 90:
        return 1.00  # Baseline
    else:
        return 0.95  # Stale/abandoned penalty
```

### Special Cases
- New addons (< 90 days old): Use actual history, minimum 1.0x multiplier
- Stable "just works" addons: Not over-penalized, just slight disadvantage

## Eligibility Thresholds

### Hot Right Now
- Minimum 500 total downloads
- Must have positive download velocity
- Display top 20 by score

### Rising Stars
- Between 50 and 10,000 total downloads
- Must have positive growth percentage
- Exclude addons already in Hot Right Now
- Display top 20 by score

## Trending Age Reset Logic

Each addon tracks `first_trending_at` timestamp per category:

1. When addon first qualifies for trending → set `first_trending_at = now`
2. Age used in formula = `now - first_trending_at`
3. When addon drops off the list → clear `first_trending_at`
4. When addon re-enters → fresh timestamp, age starts at 0

This prevents addons from being "stuck" with accumulated gravity decay and gives returning addons a fair chance.

## Recalculation Schedule

| Calculation | Frequency |
|-------------|-----------|
| Trending scores | Hourly |
| 95th percentile (size multiplier) | Daily |
| Maintenance multiplier | Daily |

## Complete Reference

| Component | Hot Right Now | Rising Stars |
|-----------|---------------|--------------|
| Primary metric | Download velocity | Growth percentage |
| Signal blend | 70% downloads, 20% thumbs, 10% update | Same |
| Time window | Adaptive (24h/7d confidence-based) | Same |
| Gravity (decay) | 1.5 (moderate) | 1.8 (aggressive) |
| Size multiplier | Log scale, 95th percentile ceiling | Same |
| Maintenance boost | 0.95x - 1.15x | Same |
| Min downloads | 500 | 50 |
| Max downloads | None | 10,000 |
| List size | Top 20 | Top 20 |

## Key Differentiators

1. **Multi-signal blend** - Not just downloads; incorporates thumbs up and update activity
2. **Confidence-based adaptive windows** - Uses fresh data when reliable, longer windows otherwise
3. **Continuous logarithmic size scaling** - Smooth curve, no arbitrary tier boundaries
4. **Maintenance reward** - Active authors get visibility boost
5. **Category-specific decay** - Hot stays stable, Rising Stars cycles quickly
6. **Age reset on re-entry** - Fair chance for returning addons

## Future Enhancements

Potential improvements to explore with collected data:
- Category-relative scoring (compare addons within their category)
- Seasonal adjustment (expansion launches, patch days)
- Author reputation factor
- Dependency graph influence (addons required by popular addons)
- User engagement signals (if available via API in future)
