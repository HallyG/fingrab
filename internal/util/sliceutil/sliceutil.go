package sliceutil

import (
	"fmt"
	"strings"
)

func Filter[T any](list []T, filter func(T) bool) []T {
	filtered := make([]T, 0)

	for _, element := range list {
		if filter(element) {
			filtered = append(filtered, element)
		}
	}

	return filtered
}

func ToDelimitedString[T any](list []T) string {
	strTypes := make([]string, len(list))

	for i, t := range list {
		strTypes[i] = fmt.Sprintf("%v", t)
	}

	return strings.Join(strTypes, ", ")
}
