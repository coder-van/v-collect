package util

import (
	"hash/fnv"
)

// Contains reports whether an item is within the slice.
func Contains(slice []int32, item int32) bool {
	for _, val := range slice {
		if val == item {
			return true
		}
	}
	return false
}

// Hash generates hash number of a string.
func Hash(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}
