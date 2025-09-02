package sliceutil_test

import (
	"testing"

	"github.com/HallyG/fingrab/internal/util/sliceutil"
	"github.com/stretchr/testify/require"
)

func TestFilter(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    []int
		expected []int
		filterFn func(x int) bool
	}{
		"filter even numbers": {
			input:    []int{1, 2, 3, 4, 5, 6},
			expected: []int{2, 4, 6},
			filterFn: func(x int) bool {
				return x%2 == 0
			},
		},
		"filter empty list": {
			input:    []int{},
			expected: []int{},
			filterFn: func(x int) bool {
				return x%2 == 0
			},
		},
		"filter all false": {
			input:    []int{1, 3, 5},
			expected: []int{},
			filterFn: func(x int) bool {
				return x%2 == 0
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := sliceutil.Filter(test.input, test.filterFn)
			require.Equal(t, test.expected, result)
		})
	}
}

func TestMap(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    []int
		expected []int
		mapperFn func(int) int
	}{
		"double numbers": {
			input:    []int{1, 2, 3, 4},
			expected: []int{2, 4, 6, 8},
			mapperFn: func(x int) int {
				return x * 2
			},
		},
		"empty input": {
			input:    []int{},
			expected: []int{},
			mapperFn: func(x int) int {
				return x * 2
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := sliceutil.Map(test.input, test.mapperFn)
			require.Equal(t, test.expected, result)
		})
	}
}

func TestToDelimitedString(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    []int
		expected string
	}{
		"double numbers": {
			input:    []int{1, 2, 3, 4},
			expected: "1, 2, 3, 4",
		},
		"empty input": {
			input: []int{},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := sliceutil.ToDelimitedString(test.input)
			require.Equal(t, test.expected, result)
		})
	}
}
