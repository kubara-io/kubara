package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckFetchesLatestReleaseAndCachesResult(t *testing.T) {
	now := time.Date(2026, 3, 19, 10, 0, 0, 0, time.UTC)
	cacheFile := filepath.Join(t.TempDir(), "update-check.json")

	client := mockHTTPClient(func(_ *http.Request) (*http.Response, error) {
		return newResponse(
			http.StatusOK,
			`{"tag_name":"v1.4.0"}`,
		), nil
	})

	result, err := runCheck(context.Background(), "v1.3.0", true, true, checkDeps{
		now:           func() time.Time { return now },
		cacheFilePath: cacheFile,
		httpClient:    client,
	})
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected update result, got nil")
	}
	if result.CurrentVersion != "v1.3.0" {
		t.Fatalf("unexpected current version: %s", result.CurrentVersion)
	}
	if result.LatestVersion != "v1.4.0" {
		t.Fatalf("unexpected latest version: %s", result.LatestVersion)
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("failed to read cache file: %v", err)
	}
	var cached cacheEntry
	if err := json.Unmarshal(data, &cached); err != nil {
		t.Fatalf("failed to parse cache: %v", err)
	}
	if cached.LatestVersion != "v1.4.0" {
		t.Fatalf("unexpected cached latest version: %s", cached.LatestVersion)
	}
}

func TestCheckUsesFreshCacheWithoutNetworkCall(t *testing.T) {
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	cacheFile := filepath.Join(t.TempDir(), "update-check.json")

	if err := writeCache(cacheFile, cacheEntry{
		CheckedAt:     now.Add(-1 * time.Hour),
		LatestVersion: "v2.0.0",
	}); err != nil {
		t.Fatalf("failed to seed cache: %v", err)
	}

	requestCount := 0
	client := mockHTTPClient(func(_ *http.Request) (*http.Response, error) {
		requestCount++
		return newResponse(http.StatusInternalServerError, ``), nil
	})

	result, err := runCheck(context.Background(), "v1.9.0", true, true, checkDeps{
		now:           func() time.Time { return now },
		cacheFilePath: cacheFile,
		httpClient:    client,
	})
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected update result from cache, got nil")
	}
	if requestCount != 0 {
		t.Fatalf("expected 0 network calls, got %d", requestCount)
	}
}

func TestCheckFallsBackToStaleCacheWhenRemoteFails(t *testing.T) {
	now := time.Date(2026, 3, 19, 14, 0, 0, 0, time.UTC)
	cacheFile := filepath.Join(t.TempDir(), "update-check.json")

	if err := writeCache(cacheFile, cacheEntry{
		CheckedAt:     now.Add(-48 * time.Hour),
		LatestVersion: "v3.0.0",
	}); err != nil {
		t.Fatalf("failed to seed cache: %v", err)
	}

	client := mockHTTPClient(func(_ *http.Request) (*http.Response, error) {
		return newResponse(http.StatusServiceUnavailable, ``), nil
	})

	result, err := runCheck(context.Background(), "v2.9.0", true, true, checkDeps{
		now:           func() time.Time { return now },
		cacheFilePath: cacheFile,
		httpClient:    client,
	})
	if err != nil {
		t.Fatalf("check returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected stale-cache fallback result, got nil")
	}
	if result.LatestVersion != "v3.0.0" {
		t.Fatalf("unexpected fallback version: %s", result.LatestVersion)
	}
}

func TestShouldSkipUpdateCheckUsesSimpleEnvFlag(t *testing.T) {
	t.Setenv(UpdateCheckEnvVar, "0")
	if !shouldSkipUpdateCheck() {
		t.Fatal("expected update check to be skipped when env is 0")
	}

	t.Setenv(UpdateCheckEnvVar, "1")
	if shouldSkipUpdateCheck() {
		t.Fatal("expected update check to run when env is 1")
	}

	t.Setenv(UpdateCheckEnvVar, "")
	if shouldSkipUpdateCheck() {
		t.Fatal("expected update check to run when env is empty")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func mockHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func newResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
