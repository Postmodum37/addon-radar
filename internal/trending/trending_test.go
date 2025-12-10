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

func TestCalculateVelocity(t *testing.T) {
	tests := []struct {
		name           string
		velocity24h    float64
		velocity7d     float64
		dataPoints24h  int
		change24h      int64
		wantConfident  bool
		wantVelocity   float64
	}{
		{
			name:          "confident 24h - enough data and change",
			velocity24h:   100.0,
			velocity7d:    50.0,
			dataPoints24h: 10,
			change24h:     500,
			wantConfident: true,
			wantVelocity:  90.0, // 0.8 * 100 + 0.2 * 50
		},
		{
			name:          "not confident - few data points",
			velocity24h:   100.0,
			velocity7d:    50.0,
			dataPoints24h: 3,
			change24h:     500,
			wantConfident: false,
			wantVelocity:  65.0, // 0.3 * 100 + 0.7 * 50
		},
		{
			name:          "not confident - small change",
			velocity24h:   100.0,
			velocity7d:    50.0,
			dataPoints24h: 10,
			change24h:     5,
			wantConfident: false,
			wantVelocity:  65.0, // 0.3 * 100 + 0.7 * 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confident, velocity := CalculateVelocity(tt.velocity24h, tt.velocity7d, tt.dataPoints24h, tt.change24h)
			if confident != tt.wantConfident {
				t.Errorf("confident = %v, want %v", confident, tt.wantConfident)
			}
			if math.Abs(velocity-tt.wantVelocity) > 0.01 {
				t.Errorf("velocity = %v, want %v", velocity, tt.wantVelocity)
			}
		})
	}
}

func TestCalculateWeightedSignal(t *testing.T) {
	tests := []struct {
		name           string
		downloadSignal float64
		thumbsSignal   float64
		hasUpdate      bool
		want           float64
	}{
		{
			name:           "all signals with update",
			downloadSignal: 100.0,
			thumbsSignal:   50.0,
			hasUpdate:      true,
			want:           80.0 + 1.0, // 0.7*100 + 0.2*50 + 0.1*10
		},
		{
			name:           "all signals without update",
			downloadSignal: 100.0,
			thumbsSignal:   50.0,
			hasUpdate:      false,
			want:           80.0, // 0.7*100 + 0.2*50 + 0
		},
		{
			name:           "zero values",
			downloadSignal: 0,
			thumbsSignal:   0,
			hasUpdate:      false,
			want:           0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateWeightedSignal(tt.downloadSignal, tt.thumbsSignal, tt.hasUpdate)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("CalculateWeightedSignal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateHotScore(t *testing.T) {
	tests := []struct {
		name                  string
		weightedVelocity      float64
		sizeMultiplier        float64
		maintenanceMultiplier float64
		ageHours              float64
		want                  float64
	}{
		{
			name:                  "new addon",
			weightedVelocity:      100.0,
			sizeMultiplier:        0.5,
			maintenanceMultiplier: 1.1,
			ageHours:              0,
			want:                  19.45, // (100 * 0.5 * 1.1) / (0+2)^1.5
		},
		{
			name:                  "24h old addon",
			weightedVelocity:      100.0,
			sizeMultiplier:        0.5,
			maintenanceMultiplier: 1.1,
			ageHours:              24,
			want:                  0.41, // (100 * 0.5 * 1.1) / (24+2)^1.5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateHotScore(tt.weightedVelocity, tt.sizeMultiplier, tt.maintenanceMultiplier, tt.ageHours)
			if math.Abs(got-tt.want) > 0.1 {
				t.Errorf("CalculateHotScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateRisingScore(t *testing.T) {
	tests := []struct {
		name                  string
		weightedGrowthPct     float64
		sizeMultiplier        float64
		maintenanceMultiplier float64
		ageHours              float64
		want                  float64
	}{
		{
			name:                  "new addon",
			weightedGrowthPct:     50.0,
			sizeMultiplier:        0.3,
			maintenanceMultiplier: 1.0,
			ageHours:              0,
			want:                  4.31, // (50 * 0.3 * 1.0) / (0+2)^1.8 = 15 / 3.482 = 4.308
		},
		{
			name:                  "48h old addon",
			weightedGrowthPct:     50.0,
			sizeMultiplier:        0.3,
			maintenanceMultiplier: 1.0,
			ageHours:              48,
			want:                  0.016, // (50 * 0.3 * 1.0) / (48+2)^1.8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateRisingScore(tt.weightedGrowthPct, tt.sizeMultiplier, tt.maintenanceMultiplier, tt.ageHours)
			if math.Abs(got-tt.want) > 0.1 {
				t.Errorf("CalculateRisingScore() = %v, want %v", got, tt.want)
			}
		})
	}
}
