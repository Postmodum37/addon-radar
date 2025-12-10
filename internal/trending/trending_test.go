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
