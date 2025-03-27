package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		data := LoadTestDataFile(t, filename)
		_, err := w.Write(data)
		assert.NoError(t, err, "failed to write test response")
	}
}
