package trending

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"addon-radar/internal/testutil"
)

// seedAddonWithSnapshots creates an addon and its snapshots for testing
func seedAddonWithSnapshots(t *testing.T, tdb *testutil.TestDB, id int32, slug string, downloads int64, thumbsUp int32, snapshotCount int) {
	ctx := context.Background()

	// Insert addon with required fields
	_, err := tdb.Pool.Exec(ctx, `
		INSERT INTO addons (id, slug, name, status, download_count, thumbs_up_count, latest_file_date)
		VALUES ($1, $2, $3, 'active', $4, $5, NOW() - INTERVAL '2 days')
	`, id, slug, "Test "+slug, downloads, thumbsUp)
	require.NoError(t, err)

	// Insert snapshots with varying download counts to create velocity
	for i := 0; i < snapshotCount; i++ {
		recordedAt := time.Now().Add(-time.Duration(i) * time.Hour)
		downloadAtSnapshot := downloads - int64(i*100)                       // Each hour 100 fewer downloads
		thumbsAtSnapshot := thumbsUp - int32(int64(i)*2)                     //nolint:gosec // Test data with small known values

		_, err := tdb.Pool.Exec(ctx, `
			INSERT INTO snapshots (addon_id, recorded_at, download_count, thumbs_up_count)
			VALUES ($1, $2, $3, $4)
		`, id, recordedAt, downloadAtSnapshot, thumbsAtSnapshot)
		require.NoError(t, err)
	}
}

func TestCalculatorCalculateAll(t *testing.T) {
	t.Run("calculates scores for addons with snapshots", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		// Seed addon with enough downloads for "hot" threshold (>=500)
		seedAddonWithSnapshots(t, tdb, 1, "hot-addon", 5000, 100, 10)

		// Seed addon with downloads in "rising" range (50-10000)
		seedAddonWithSnapshots(t, tdb, 2, "rising-addon", 500, 20, 10)

		calc := NewCalculator(tdb.Queries)
		err := calc.CalculateAll(ctx)
		require.NoError(t, err)

		// Verify scores were created
		var count int
		err = tdb.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM trending_scores`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)

		// Check hot addon has a positive hot score
		var hotScore float64
		err = tdb.Pool.QueryRow(ctx, `
			SELECT COALESCE(hot_score, 0) FROM trending_scores WHERE addon_id = $1
		`, 1).Scan(&hotScore)
		require.NoError(t, err)
		assert.Greater(t, hotScore, 0.0, "hot addon should have positive hot score")

		// Check rising addon has a positive rising score
		var risingScore float64
		err = tdb.Pool.QueryRow(ctx, `
			SELECT COALESCE(rising_score, 0) FROM trending_scores WHERE addon_id = $1
		`, 2).Scan(&risingScore)
		require.NoError(t, err)
		assert.Greater(t, risingScore, 0.0, "rising addon should have positive rising score")
	})

	t.Run("handles empty database", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		calc := NewCalculator(tdb.Queries)
		err := calc.CalculateAll(ctx)
		require.NoError(t, err)

		// No scores should be created
		var count int
		err = tdb.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM trending_scores`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("handles addons without snapshots", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		// Insert addon without snapshots
		_, err := tdb.Pool.Exec(ctx, `
			INSERT INTO addons (id, slug, name, status, download_count)
			VALUES ($1, $2, $3, 'active', 1000)
		`, 1, "no-snapshots", "No Snapshots Addon")
		require.NoError(t, err)

		calc := NewCalculator(tdb.Queries)
		err = calc.CalculateAll(ctx)
		require.NoError(t, err)

		// Score might be 0 but shouldn't error
	})

	t.Run("updates existing scores", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		// Seed addon
		seedAddonWithSnapshots(t, tdb, 1, "update-test", 5000, 100, 10)

		calc := NewCalculator(tdb.Queries)

		// First calculation
		err := calc.CalculateAll(ctx)
		require.NoError(t, err)

		var firstScore float64
		err = tdb.Pool.QueryRow(ctx, `
			SELECT COALESCE(hot_score, 0) FROM trending_scores WHERE addon_id = 1
		`).Scan(&firstScore)
		require.NoError(t, err)

		// Add more snapshots to change velocity
		for i := 0; i < 5; i++ {
			recordedAt := time.Now().Add(-time.Duration(i) * time.Minute)
			_, err := tdb.Pool.Exec(ctx, `
				INSERT INTO snapshots (addon_id, recorded_at, download_count, thumbs_up_count)
				VALUES ($1, $2, $3, $4)
			`, 1, recordedAt, 6000+i*200, 110+i*5)
			require.NoError(t, err)
		}

		// Second calculation
		err = calc.CalculateAll(ctx)
		require.NoError(t, err)

		var secondScore float64
		err = tdb.Pool.QueryRow(ctx, `
			SELECT COALESCE(hot_score, 0) FROM trending_scores WHERE addon_id = 1
		`).Scan(&secondScore)
		require.NoError(t, err)

		// Score should have changed
		assert.NotEqual(t, firstScore, secondScore, "score should update on recalculation")
	})

	t.Run("respects download thresholds", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		// Addon with downloads below hot threshold (< 500)
		seedAddonWithSnapshots(t, tdb, 1, "low-downloads", 100, 10, 10)

		// Addon with downloads above rising max (> 10000)
		seedAddonWithSnapshots(t, tdb, 2, "high-downloads", 50000, 1000, 10)

		calc := NewCalculator(tdb.Queries)
		err := calc.CalculateAll(ctx)
		require.NoError(t, err)

		// Low downloads addon shouldn't have hot score
		var lowHotScore float64
		err = tdb.Pool.QueryRow(ctx, `
			SELECT COALESCE(hot_score, 0) FROM trending_scores WHERE addon_id = 1
		`).Scan(&lowHotScore)
		require.NoError(t, err)
		assert.Equal(t, 0.0, lowHotScore, "low download addon should not have hot score")

		// High downloads addon shouldn't have rising score (above 10000 threshold)
		var highRisingScore float64
		err = tdb.Pool.QueryRow(ctx, `
			SELECT COALESCE(rising_score, 0) FROM trending_scores WHERE addon_id = 2
		`).Scan(&highRisingScore)
		require.NoError(t, err)
		assert.Equal(t, 0.0, highRisingScore, "high download addon should not have rising score")
	})

	t.Run("tracks first_hot_at and first_rising_at timestamps", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		// Seed addon eligible for hot
		seedAddonWithSnapshots(t, tdb, 1, "timestamp-test", 5000, 100, 10)

		calc := NewCalculator(tdb.Queries)

		// First calculation
		err := calc.CalculateAll(ctx)
		require.NoError(t, err)

		var firstHotAt time.Time
		var hotScore float64
		err = tdb.Pool.QueryRow(ctx, `
			SELECT COALESCE(hot_score, 0), first_hot_at FROM trending_scores WHERE addon_id = 1
		`).Scan(&hotScore, &firstHotAt)
		require.NoError(t, err)

		if hotScore > 0 {
			assert.False(t, firstHotAt.IsZero(), "first_hot_at should be set for hot addons")

			// Second calculation shouldn't change first_hot_at
			time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamp if incorrectly updated
			err = calc.CalculateAll(ctx)
			require.NoError(t, err)

			var secondFirstHotAt time.Time
			err = tdb.Pool.QueryRow(ctx, `
				SELECT first_hot_at FROM trending_scores WHERE addon_id = 1
			`).Scan(&secondFirstHotAt)
			require.NoError(t, err)

			assert.Equal(t, firstHotAt.Unix(), secondFirstHotAt.Unix(), "first_hot_at should persist across calculations")
		}
	})

	t.Run("calculates multipliers correctly", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		// Seed addon
		seedAddonWithSnapshots(t, tdb, 1, "multiplier-test", 5000, 100, 10)

		calc := NewCalculator(tdb.Queries)
		err := calc.CalculateAll(ctx)
		require.NoError(t, err)

		var sizeMultiplier, maintenanceMultiplier float64
		err = tdb.Pool.QueryRow(ctx, `
			SELECT COALESCE(size_multiplier, 0), COALESCE(maintenance_multiplier, 0)
			FROM trending_scores WHERE addon_id = 1
		`).Scan(&sizeMultiplier, &maintenanceMultiplier)
		require.NoError(t, err)

		// Size multiplier should be between 0.1 and 1.0
		assert.GreaterOrEqual(t, sizeMultiplier, 0.1)
		assert.LessOrEqual(t, sizeMultiplier, 1.0)

		// Maintenance multiplier should be between 0.95 and 1.15
		assert.GreaterOrEqual(t, maintenanceMultiplier, 0.95)
		assert.LessOrEqual(t, maintenanceMultiplier, 1.15)
	})
}

func TestCalculatorPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	t.Run("handles many addons efficiently", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		// Insert 100 addons with snapshots
		for i := 1; i <= 100; i++ {
			//nolint:gosec // Test data with small known values
			seedAddonWithSnapshots(t, tdb, int32(int64(i)), "addon-"+string(rune('a'+i)), int64(1000+i*100), int32(int64(10+i)), 5)
		}

		calc := NewCalculator(tdb.Queries)

		start := time.Now()
		err := calc.CalculateAll(ctx)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, duration, 30*time.Second, "calculation should complete in reasonable time")

		// Verify all scores were created
		var count int
		err = tdb.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM trending_scores`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 100, count)
	})
}
