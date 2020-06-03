package utils

// Maps a and b have intersection
func MapIntersects(a map[string]string, b map[string]string) bool {
	for aKey, aVal := range a {
		for bKey, bVal := range b {
			if aKey == bKey && aVal == bVal {
				return true
			}
		}
	}
	return false
}
