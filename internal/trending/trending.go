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

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
