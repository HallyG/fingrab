package sliceutil_test

import (
	"testing"

	"github.com/HallyG/fingrab/internal/util/sliceutil"
	"github.com/stretchr/testify/require"
)

func TestFilter(t *testing.T) {
	t.Parallel()

	filterFn := func(x int) bool { return x%2 == 0 }

	tests := map[string]struct {
		input    []int
		expected []int
	}{
		"filter even numbers": {
			input:    []int{1, 2, 3, 4, 5, 6},
			expected: []int{2, 4, 6},
		},
		"filter empty list": {
			input:    []int{},
			expected: []int{},
		},
		"filter all false": {
			input:    []int{1, 3, 5},
			expected: []int{},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := sliceutil.Filter(test.input, filterFn)
			require.Equal(t, test.expected, result)
		})
	}
}
