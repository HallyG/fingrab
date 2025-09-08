package testhelper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

type HTTPTestRoute struct {
	Method  string
	URL     string // URL pattern (e.g., "/api/v1"). Must not be empty.
	Handler http.HandlerFunc
}

func NewHTTPTestServer(t *testing.T, routes []HTTPTestRoute) *httptest.Server {
	t.Helper()

	router := http.NewServeMux()

	for _, route := range routes {
		if route.URL == "" {
			t.Fatalf("HTTPTestRoute.URL must not be empty")
		}

		method := strings.ToUpper(strings.TrimSpace(route.Method))

		if route.Method == "" {
			t.Fatalf("HTTPTestRoute.Method must not be empty")
		}

		if route.Handler == nil {
			t.Fatalf("HTTPTestRoute.Handler must not be nil for route %s", route.URL)
		}

		pattern := fmt.Sprintf("%s %s", method, route.URL)
		router.HandleFunc(pattern, route.Handler)
	}

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server
}

func ServeJSONTestDataHandler(t *testing.T, statusCode int, filename string) http.HandlerFunc {
	t.Helper()

	data := LoadTestDataFile(t, filename)

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write(data)
	}
}
