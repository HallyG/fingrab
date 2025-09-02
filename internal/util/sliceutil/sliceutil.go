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

func Map[T any, R any](items []T, mapper func(T) R) []R {
	results := make([]R, 0, len(items))
	for _, item := range items {
		results = append(results, mapper(item))
	}

	return results
}

func ToDelimitedString[T any](list []T) string {
	strTypes := make([]string, len(list))

	for i, t := range list {
		strTypes[i] = fmt.Sprintf("%v", t)
	}

	return strings.Join(strTypes, ", ")
}
