package trending

import (
	"math"
	"testing"
)

func TestCalculateSizeMultiplier(t *testing.T) {
	percentile95 := float64(500000)

	tests := []struct {
		downloads float64
		want      float64
	}{
		{10, 0.18},
		{100, 0.35},
		{1000, 0.53},
		{10000, 0.70},
		{100000, 0.88},
		{500000, 1.0},
		{1000000, 1.0}, // Capped at 1.0
		{0, 0.1},       // Minimum 0.1
	}

	for _, tt := range tests {
		got := CalculateSizeMultiplier(tt.downloads, percentile95)
		if math.Abs(got-tt.want) > 0.02 { // Allow 2% tolerance
			t.Errorf("CalculateSizeMultiplier(%v) = %v, want %v", tt.downloads, got, tt.want)
		}
	}
}

func TestCalculateMaintenanceMultiplier(t *testing.T) {
	tests := []struct {
		updatesIn90Days int
		want            float64
	}{
		{12, 1.15}, // ~7 days avg = very active
		{6, 1.10},  // 15 days avg = regular (not <= 14)
		{4, 1.10},  // ~22 days avg = regular
		{3, 1.10},  // 30 days avg = regular
		{2, 1.05},  // 45 days avg = occasional
		{1, 1.00},  // 90 days avg = baseline
		{0, 0.95},  // No updates = stale
	}

	for _, tt := range tests {
		got := CalculateMaintenanceMultiplier(tt.updatesIn90Days)
		if got != tt.want {
			t.Errorf("CalculateMaintenanceMultiplier(%d) = %v, want %v", tt.updatesIn90Days, got, tt.want)
		}
	}
}
