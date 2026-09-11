package goals

import "math"

func roundTo(value, unit float64) float64 {
	factor := 1 / unit
	return math.Round(value*factor) / factor
}
