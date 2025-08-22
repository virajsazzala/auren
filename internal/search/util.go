package search

func FlattenVectors(vectors [][]float32) []float32 {
	if len(vectors) == 0 {
		return nil
	}
	dim := len(vectors[0])
	flat := make([]float32, 0, len(vectors)*dim)
	for _, vec := range vectors {
		flat = append(flat, vec...)
	}
	return flat
}