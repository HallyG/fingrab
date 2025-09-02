package testutil

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func LoadTestDataFile(t *testing.T, filename string) []byte {
	t.Helper()

	fullPath := filepath.Join(filepath.Join("testdata", "api"), filename)
	path := filepath.Clean(fullPath)

	_, err := os.Stat(path)
	require.NoError(t, err, "test data file %s must exist", filename)

	b, err := os.ReadFile(path)
	assert.NoError(t, err)

	return b
}

// Validates that an HTTP request matches the expected method, headers, and query parameters.
func AssertRequest(t *testing.T, r *http.Request, method string, expectedHeaders map[string]string, expectedQueryParams map[string]string) {
	t.Helper()

	assert.Equal(t, method, r.Method, "HTTP method should match")

	for header, expected := range expectedHeaders {
		actual := r.Header.Get(header)
		assert.Equal(t, expected, actual, "header %s should match", header)
	}

	query := r.URL.Query()
	for key, expected := range expectedQueryParams {
		actual := query.Get(key)
		assert.Equal(t, expected, actual, "query param %s should match", key)
	}
}
