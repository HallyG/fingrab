package sliceutil_test

import (
	"testing"

	"github.com/HallyG/fingrab/internal/util/sliceutil"
	"github.com/stretchr/testify/require"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "filter even numbers",
			input:    []int{1, 2, 3, 4, 5, 6},
			expected: []int{2, 4, 6},
		},
		{
			name:     "filter empty list",
			input:    []int{},
			expected: []int{},
		},
		{
			name:     "filter all false",
			input:    []int{1, 3, 5},
			expected: []int{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			filter := func(x int) bool { return x%2 == 0 }

			result := sliceutil.Filter(test.input, filter)

			require.Equal(t, test.expected, result)
		})
	}
}
