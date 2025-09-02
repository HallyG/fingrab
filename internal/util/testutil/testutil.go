package testutil

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func LoadTestDataFile(t *testing.T, filename string) []byte {
	t.Helper()

	fullPath := filepath.Join(filepath.Join("testdata", "api"), filename)
	path := filepath.Clean(fullPath)

	_, err := os.Stat(path)
	require.NoError(t, err, "test data file %s must exist", filename)

	b, err := os.ReadFile(path)
	require.NoError(t, err)

	return b
}

func MarshalTestDataFile[T any](t *testing.T, filename string) *T {
	t.Helper()

	bytes := LoadTestDataFile(t, filename)

	var result T
	err := json.Unmarshal(bytes, &result)
	require.NoError(t, err)

	return &result
}

// Validates that an HTTP request matches the expected method, headers, and query parameters.
func AssertRequest(t *testing.T, r *http.Request, method string, expectedHeaders http.Header, expectedQueryParams url.Values) {
	t.Helper()

	require.Equal(t, method, r.Method, "HTTP method should match")

	for header, expected := range expectedHeaders {
		actual := r.Header.Values(header)
		require.Equal(t, expected, actual, "header %s should match", header)
	}

	query := r.URL.Query()
	for key, expected := range expectedQueryParams {
		actual := query[key]
		require.Equal(t, expected, actual, "query param %s should match", key)
	}
}

func MustParse[T any](t *testing.T, input string, fn func(string) (T, error)) T {
	t.Helper()

	result, err := fn(input)
	require.NoError(t, err)
	return result
}
