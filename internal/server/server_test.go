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
	"time"

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
	return server.New(dir, "", "0", 1, mockLogger{})
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

func TestServeHTTP_CheckBusy(t *testing.T) {
	tmp := t.TempDir()
	const configStr = `
devices:
  user1:
    template: default
    password: pass1
templates:
  default:
    type: text
    data: hello
    check: "sleep 1"
    check_timeout: 500ms
`
	if err := os.WriteFile(filepath.Join(tmp, "config.yaml"), []byte(configStr), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	srv := newTestServer(tmp)
	codes := make(chan int, 2)
	for range 2 {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.SetBasicAuth("user1", "pass1")
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			codes <- rec.Code
		}()
		time.Sleep(100 * time.Millisecond)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("user1", "pass1")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503; got %d", rec.Code)
	}
	for range 2 {
		if code := <-codes; code != http.StatusInternalServerError {
			t.Errorf("expected queued request to time out with 500; got %d", code)
		}
	}
}
