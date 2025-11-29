package poml

import (
	"strconv"
)

func isPositiveNumber(v string) bool {
	if v == "" {
		return false
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return false
	}
	return f > 0
}
