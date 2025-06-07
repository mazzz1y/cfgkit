package server_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cfgkit/internal/server"
)

type mockLogger struct{}

func (mockLogger) Info(string, ...any)                                           {}
func (mockLogger) LogRequest(context.Context, *http.Request, int, string, error) {}

func prepareConfig(t *testing.T, dir, device, password string) {
	t.Helper()

	const configStr = `
devices:
  %s:
    template: default
    password: "%s"
    variables:
      key1: "foo"
      key2: "bar"
templates:
  default:
    type: json
    data: '{"field1":"{{ .Device.key1 }}","field2":"{{ .Device.key2 }}"}'
`
	config := fmt.Sprintf(configStr, device, password)
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}

func newTestServer(dir string) *server.Server {
	return server.New(dir, "0", mockLogger{})
}

func TestServeHTTP_Success(t *testing.T) {
	tmp := t.TempDir()
	device := "user1"
	pass := "pass1"
	prepareConfig(t, tmp, device, pass)

	srv := newTestServer(tmp)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth(device, pass)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200; got %d", res.StatusCode)
	}

	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json; got %s", ct)
	}

	bodyBytes, _ := io.ReadAll(res.Body)
	trimmedBody := strings.Join(strings.Fields(string(bodyBytes)), "")
	want := `{"field1":"foo","field2":"bar"}`
	if trimmedBody != want {
		t.Errorf("unexpected body; got: %s want: %s", trimmedBody, want)
	}
}

func TestServeHTTP_AuthFail(t *testing.T) {
	tmp := t.TempDir()
	device := "user1"
	pass := "pass1"
	prepareConfig(t, tmp, device, pass)

	srv := newTestServer(tmp)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth(device, "wrongpass")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403; got %d", rec.Code)
	}
}

func TestServeHTTP_ConfigError(t *testing.T) {
	srv := newTestServer("/tmp/no/such/dir")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500; got %d", rec.Code)
	}
}
