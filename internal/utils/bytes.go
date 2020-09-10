package utils

import (
	"fmt"
)

var binaryAbbrs = []string{"", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi", "Yi"}

func getSizeAndUnit(size float64, base float64, _map []string) (float64, string) {
	i := 0
	unitsLimit := len(_map) - 1
	for size >= base && i < unitsLimit {
		size = size / base
		i++
	}
	return size, _map[i]
}

func CustomSize(format string, size float64, base float64, _map []string) string {
	size, unit := getSizeAndUnit(size, base, _map)
	return fmt.Sprintf(format, size, unit)
}

func BytesSize(size float64) string {
	return CustomSize("%.4g%s", size, 1024.0, binaryAbbrs)
}
