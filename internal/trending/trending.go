package trending

import "math"

// CalculateSizeMultiplier returns a value between 0.1 and 1.0
// based on logarithmic scaling of downloads against the 95th percentile.
func CalculateSizeMultiplier(downloads, percentile95 float64) float64 {
	if percentile95 <= 0 {
		return 1.0
	}
	if downloads <= 0 {
		return 0.1
	}

	multiplier := math.Log10(downloads+1) / math.Log10(percentile95+1)
	return clamp(multiplier, 0.1, 1.0)
}

// CalculateMaintenanceMultiplier returns a multiplier (0.95-1.15)
// based on update frequency in the last 90 days.
func CalculateMaintenanceMultiplier(updatesIn90Days int) float64 {
	if updatesIn90Days == 0 {
		return 0.95 // Stale/abandoned
	}

	avgDaysBetweenUpdates := 90.0 / float64(updatesIn90Days)

	switch {
	case avgDaysBetweenUpdates <= 14:
		return 1.15 // Very active
	case avgDaysBetweenUpdates <= 30:
		return 1.10 // Regular
	case avgDaysBetweenUpdates <= 60:
		return 1.05 // Occasional
	default:
		return 1.00 // Baseline
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
