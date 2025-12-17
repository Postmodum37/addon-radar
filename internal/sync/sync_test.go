package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"addon-radar/internal/curseforge"
	"addon-radar/internal/database"
	"addon-radar/internal/testutil"
)

// mockCurseForgeClient implements CurseForgeClient for testing
type mockCurseForgeClient struct {
	addons        []curseforge.Mod
	categories    []curseforge.Category
	addonsErr     error
	categoriesErr error
}

func (m *mockCurseForgeClient) GetAllWoWAddons(ctx context.Context) ([]curseforge.Mod, error) {
	if m.addonsErr != nil {
		return nil, m.addonsErr
	}
	return m.addons, nil
}

func (m *mockCurseForgeClient) GetCategories(ctx context.Context, gameID int) ([]curseforge.Category, error) {
	if m.categoriesErr != nil {
		return nil, m.categoriesErr
	}
	return m.categories, nil
}

// createTestMod creates a test Mod with sensible defaults
func createTestMod(id int, slug, name string) curseforge.Mod {
	return curseforge.Mod{
		ID:             id,
		Slug:           slug,
		Name:           name,
		Summary:        "Test summary for " + name,
		DownloadCount:  int64(1000 * id),
		ThumbsUpCount:  id * 10,
		PopularityRank: id,
		DateCreated:    time.Now().Add(-time.Hour * 24 * 30),
		DateModified:   time.Now().Add(-time.Hour),
		Authors: []curseforge.Author{
			{ID: 100 + id, Name: "Author " + name},
		},
		Logo: &curseforge.Logo{
			ThumbnailURL: "https://example.com/logo/" + slug + ".png",
		},
		Categories: []curseforge.Category{
			{ID: 1001, Name: "Category 1", Slug: "category-1"},
		},
		LatestFiles: []curseforge.File{
			{
				ID:           id * 1000,
				FileDate:     time.Now().Add(-time.Hour * 2),
				GameVersions: []string{"11.2.7", "11.2.5"},
			},
		},
	}
}

func TestRunFullSync(t *testing.T) {
	t.Run("success with addons", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		mockClient := &mockCurseForgeClient{
			addons: []curseforge.Mod{
				createTestMod(1, "addon-one", "Addon One"),
				createTestMod(2, "addon-two", "Addon Two"),
				createTestMod(3, "addon-three", "Addon Three"),
			},
			categories: []curseforge.Category{
				{ID: 1001, Name: "Category 1", Slug: "category-1"},
			},
		}

		service := NewServiceWithClient(tdb.Pool, tdb.Queries, mockClient)
		syncedIDs, err := service.RunFullSync(ctx)

		require.NoError(t, err)
		assert.Len(t, syncedIDs, 3)

		// Verify addons were created
		addons, err := tdb.Queries.ListAddons(ctx, database.ListAddonsParams{Limit: 10, Offset: 0})
		require.NoError(t, err)
		assert.Len(t, addons, 3)

		// Verify addon details
		addon, err := tdb.Queries.GetAddonBySlug(ctx, "addon-one")
		require.NoError(t, err)
		assert.Equal(t, "Addon One", addon.Name)
		assert.Equal(t, int64(1000), addon.DownloadCount.Int64)

		// Verify snapshots were created
		snapshots, err := tdb.Queries.GetAddonSnapshots(ctx, database.GetAddonSnapshotsParams{
			AddonID: addon.ID,
			Limit:   10,
		})
		require.NoError(t, err)
		assert.Len(t, snapshots, 1)
		assert.Equal(t, int64(1000), snapshots[0].DownloadCount)

		// Verify categories were created
		categories, err := tdb.Queries.ListCategories(ctx)
		require.NoError(t, err)
		assert.Len(t, categories, 1)
	})

	t.Run("handles empty addons", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		mockClient := &mockCurseForgeClient{
			addons:     []curseforge.Mod{},
			categories: []curseforge.Category{},
		}

		service := NewServiceWithClient(tdb.Pool, tdb.Queries, mockClient)
		syncedIDs, err := service.RunFullSync(ctx)

		require.NoError(t, err)
		assert.Empty(t, syncedIDs)

		addons, err := tdb.Queries.ListAddons(ctx, database.ListAddonsParams{Limit: 10, Offset: 0})
		require.NoError(t, err)
		assert.Empty(t, addons)
	})

	t.Run("returns error on client failure", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		mockClient := &mockCurseForgeClient{
			addonsErr: errors.New("API connection failed"),
		}

		service := NewServiceWithClient(tdb.Pool, tdb.Queries, mockClient)
		syncedIDs, err := service.RunFullSync(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch addons")
		assert.Nil(t, syncedIDs)
	})

	t.Run("continues on category sync failure", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		mockClient := &mockCurseForgeClient{
			addons: []curseforge.Mod{
				createTestMod(1, "addon-one", "Addon One"),
			},
			categoriesErr: errors.New("categories API failed"),
		}

		service := NewServiceWithClient(tdb.Pool, tdb.Queries, mockClient)
		syncedIDs, err := service.RunFullSync(ctx)

		// Should not return error - category sync failure is non-critical
		require.NoError(t, err)
		assert.Len(t, syncedIDs, 1)

		// Addon should still be synced
		addons, err := tdb.Queries.ListAddons(ctx, database.ListAddonsParams{Limit: 10, Offset: 0})
		require.NoError(t, err)
		assert.Len(t, addons, 1)
	})

	t.Run("deduplication - same addon synced twice", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		mockClient := &mockCurseForgeClient{
			addons: []curseforge.Mod{
				createTestMod(1, "addon-one", "Addon One"),
			},
			categories: []curseforge.Category{},
		}

		service := NewServiceWithClient(tdb.Pool, tdb.Queries, mockClient)

		// First sync
		_, err := service.RunFullSync(ctx)
		require.NoError(t, err)

		// Update the mock data
		mockClient.addons[0].DownloadCount = 2000
		mockClient.addons[0].Name = "Addon One Updated"

		// Second sync
		_, err = service.RunFullSync(ctx)
		require.NoError(t, err)

		// Should still have 1 addon (upserted, not duplicated)
		addons, err := tdb.Queries.ListAddons(ctx, database.ListAddonsParams{Limit: 10, Offset: 0})
		require.NoError(t, err)
		assert.Len(t, addons, 1)

		// Should have updated values
		addon, err := tdb.Queries.GetAddonBySlug(ctx, "addon-one")
		require.NoError(t, err)
		assert.Equal(t, "Addon One Updated", addon.Name)
		assert.Equal(t, int64(2000), addon.DownloadCount.Int64)

		// Should have 2 snapshots (one from each sync)
		snapshots, err := tdb.Queries.GetAddonSnapshots(ctx, database.GetAddonSnapshotsParams{
			AddonID: addon.ID,
			Limit:   10,
		})
		require.NoError(t, err)
		assert.Len(t, snapshots, 2)
	})
}

func TestSyncCategories(t *testing.T) {
	t.Run("syncs categories with parent hierarchy", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		mockClient := &mockCurseForgeClient{
			categories: []curseforge.Category{
				{ID: 1, Name: "Parent Category", Slug: "parent", ParentID: 0},
				{ID: 2, Name: "Child Category", Slug: "child", ParentID: 1},
			},
		}

		service := NewServiceWithClient(tdb.Pool, tdb.Queries, mockClient)
		err := service.syncCategories(ctx)

		require.NoError(t, err)

		categories, err := tdb.Queries.ListCategories(ctx)
		require.NoError(t, err)
		assert.Len(t, categories, 2)

		// Find child category and verify parent
		var childCat database.Category
		for _, c := range categories {
			if c.Slug == "child" {
				childCat = c
				break
			}
		}
		assert.True(t, childCat.ParentID.Valid)
		assert.Equal(t, int32(1), childCat.ParentID.Int32)
	})
}

func TestUpsertAddon(t *testing.T) {
	t.Run("handles addon without author", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		mod := curseforge.Mod{
			ID:            1,
			Slug:          "no-author",
			Name:          "Addon Without Author",
			Authors:       []curseforge.Author{}, // Empty authors
			DownloadCount: 100,
		}

		mockClient := &mockCurseForgeClient{}
		service := NewServiceWithClient(tdb.Pool, tdb.Queries, mockClient)

		err := service.upsertAddon(ctx, mod)
		require.NoError(t, err)

		addon, err := tdb.Queries.GetAddonBySlug(ctx, "no-author")
		require.NoError(t, err)
		assert.False(t, addon.AuthorName.Valid) // Should be NULL
	})

	t.Run("handles addon without logo", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		mod := curseforge.Mod{
			ID:            2,
			Slug:          "no-logo",
			Name:          "Addon Without Logo",
			Logo:          nil, // No logo
			DownloadCount: 100,
		}

		mockClient := &mockCurseForgeClient{}
		service := NewServiceWithClient(tdb.Pool, tdb.Queries, mockClient)

		err := service.upsertAddon(ctx, mod)
		require.NoError(t, err)

		addon, err := tdb.Queries.GetAddonBySlug(ctx, "no-logo")
		require.NoError(t, err)
		assert.False(t, addon.LogoUrl.Valid) // Should be NULL
	})

	t.Run("handles addon with multiple categories", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		mod := curseforge.Mod{
			ID:   3,
			Slug: "multi-cat",
			Name: "Multi Category Addon",
			Categories: []curseforge.Category{
				{ID: 1001, Name: "Cat 1"},
				{ID: 1002, Name: "Cat 2"},
				{ID: 1003, Name: "Cat 3"},
			},
			DownloadCount: 100,
		}

		mockClient := &mockCurseForgeClient{}
		service := NewServiceWithClient(tdb.Pool, tdb.Queries, mockClient)

		err := service.upsertAddon(ctx, mod)
		require.NoError(t, err)

		addon, err := tdb.Queries.GetAddonBySlug(ctx, "multi-cat")
		require.NoError(t, err)
		assert.Len(t, addon.Categories, 3)
		assert.Equal(t, int32(1001), addon.PrimaryCategoryID.Int32)
	})

	t.Run("handles addon with rating", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		mod := curseforge.Mod{
			ID:            4,
			Slug:          "rated-addon",
			Name:          "Rated Addon",
			Rating:        4.75,
			DownloadCount: 100,
		}

		mockClient := &mockCurseForgeClient{}
		service := NewServiceWithClient(tdb.Pool, tdb.Queries, mockClient)

		err := service.upsertAddon(ctx, mod)
		require.NoError(t, err)

		addon, err := tdb.Queries.GetAddonBySlug(ctx, "rated-addon")
		require.NoError(t, err)
		assert.True(t, addon.Rating.Valid)
	})
}

func TestCreateSnapshot(t *testing.T) {
	t.Run("creates snapshot with all fields", func(t *testing.T) {
		tdb := testutil.SetupTestDB(t)
		ctx := context.Background()

		// First create an addon
		mod := createTestMod(1, "snapshot-test", "Snapshot Test")

		mockClient := &mockCurseForgeClient{}
		service := NewServiceWithClient(tdb.Pool, tdb.Queries, mockClient)

		err := service.upsertAddon(ctx, mod)
		require.NoError(t, err)

		err = service.createSnapshot(ctx, mod)
		require.NoError(t, err)

		addon, err := tdb.Queries.GetAddonBySlug(ctx, "snapshot-test")
		require.NoError(t, err)
		snapshots, err := tdb.Queries.GetAddonSnapshots(ctx, database.GetAddonSnapshotsParams{
			AddonID: addon.ID,
			Limit:   10,
		})
		require.NoError(t, err)
		assert.Len(t, snapshots, 1)
		assert.Equal(t, int64(1000), snapshots[0].DownloadCount)
		assert.True(t, snapshots[0].ThumbsUpCount.Valid)
		assert.Equal(t, int32(10), snapshots[0].ThumbsUpCount.Int32)
	})
}

func TestExtractGameVersions(t *testing.T) {
	t.Run("extracts unique versions", func(t *testing.T) {
		mod := curseforge.Mod{
			LatestFiles: []curseforge.File{
				{GameVersions: []string{"11.2.7", "11.2.5"}},
				{GameVersions: []string{"11.2.7", "11.1.0"}}, // 11.2.7 is duplicate
			},
		}

		versions := extractGameVersions(mod)

		// Should have unique versions only
		assert.Len(t, versions, 3)

		versionSet := make(map[string]bool)
		for _, v := range versions {
			versionSet[v] = true
		}
		assert.True(t, versionSet["11.2.7"])
		assert.True(t, versionSet["11.2.5"])
		assert.True(t, versionSet["11.1.0"])
	})

	t.Run("handles empty files", func(t *testing.T) {
		mod := curseforge.Mod{
			LatestFiles: []curseforge.File{},
		}

		versions := extractGameVersions(mod)
		assert.Empty(t, versions)
	})
}
