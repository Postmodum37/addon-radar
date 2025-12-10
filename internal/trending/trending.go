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

// CalculateVelocity uses confidence-based adaptive windows.
// Returns (isConfident24h, blendedVelocity).
func CalculateVelocity(velocity24h, velocity7d float64, dataPoints24h int, change24h int64) (bool, float64) {
	// Confident if we have enough data points AND meaningful change
	confident := dataPoints24h >= 5 && change24h >= 10

	if confident {
		// Weight toward fresh data
		return true, (0.8 * velocity24h) + (0.2 * velocity7d)
	}
	// Fall back to longer window
	return false, (0.3 * velocity24h) + (0.7 * velocity7d)
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
