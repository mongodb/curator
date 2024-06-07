package rollups

func convertToFloats(ints []int64) []float64 {
	floats := []float64{}
	for i := range ints {
		floats = append(floats, float64(ints[i]))
	}

	return floats
}

// expects slice of cumulative values
func extractValues(vals []float64, lastValue float64) []float64 {
	extractedVals := make([]float64, len(vals))

	for i := range vals {
		extractedVals[i] = vals[i] - lastValue
		lastValue = vals[i]
	}

	return extractedVals
}
