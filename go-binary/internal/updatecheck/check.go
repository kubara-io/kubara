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
	"time"

	semver "github.com/Masterminds/semver/v3"
	"github.com/fatih/color"
)

const (
	githubLatestReleaseAPI = "https://api.github.com/repos/kubara-io/kubara/releases/latest"
	defaultCacheTTL        = 24 * time.Hour
	defaultHTTPTimeout     = 1500 * time.Millisecond
	// UpdateCheckEnvVar controls the startup update hint (`0` disables it).
	UpdateCheckEnvVar = "KUBARA_UPDATE_CHECK"
)

// Result represents an available update.
type Result struct {
	CurrentVersion string
	LatestVersion  string
}

var releaseHintStyle = color.New(color.FgYellow, color.Bold).SprintFunc()

type cacheEntry struct {
	CheckedAt     time.Time `json:"checkedAt"`
	LatestVersion string    `json:"latestVersion"`
}

type checkDeps struct {
	now           func() time.Time
	cacheFilePath string
	httpClient    *http.Client
}

// NotifyIfNewReleaseAvailable checks for updates and writes a hint if a newer
// release exists.
func NotifyIfNewReleaseAvailable(currentVersion string, output io.Writer) {
	if output == nil {
		output = os.Stderr
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()

	result, err := Check(ctx, currentVersion)
	// Fail-open: update checks must never block or fail normal CLI usage.
	if err != nil || result == nil {
		return
	}

	_, _ = fmt.Fprintf(output, "\n%s\n\n", formatReleaseHint(result))
}

// PrintLiveCheck runs an explicit online update check and writes a concise
// status message.
func PrintLiveCheck(currentVersion string, output io.Writer) error {
	if output == nil {
		output = os.Stdout
	}

	if strings.EqualFold(strings.TrimSpace(currentVersion), "dev") {
		_, _ = fmt.Fprintln(output, "Update check is not available for dev builds.")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := runCheck(ctx, currentVersion, false, false, defaultCheckDeps())
	if err != nil {
		return err
	}

	if result == nil {
		_, _ = fmt.Fprintf(output, "kubara %s is up to date.\n", currentVersion)
		return nil
	}

	_, _ = fmt.Fprintf(output, "%s\n", formatReleaseHint(result))
	return nil
}

// Check compares currentVersion with the latest GitHub release and returns
// update information when a newer version is available.
func Check(ctx context.Context, currentVersion string) (*Result, error) {
	return runCheck(ctx, currentVersion, true, true, defaultCheckDeps())
}

func runCheck(ctx context.Context, currentVersion string, useCache bool, respectDisable bool, deps checkDeps) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if respectDisable && shouldSkipUpdateCheck() {
		return nil, nil
	}

	if deps.now == nil {
		deps.now = time.Now
	}
	if deps.httpClient == nil {
		deps.httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultHTTPTimeout)
		defer cancel()
	}

	current, currentDisplay, ok := parseVersion(currentVersion)
	// Skip checks for non-release builds (e.g. "dev").
	if !ok {
		return nil, nil
	}

	var cached *cacheEntry
	if useCache {
		cached = readCache(deps.cacheFilePath)
		if isFresh(cached, deps.now()) {
			return compareVersions(current, currentDisplay, cached.LatestVersion), nil
		}
	}

	latestVersion, err := fetchLatestVersion(ctx, currentDisplay, deps.httpClient)
	if err != nil {
		// Fall back to stale cache if network access fails.
		if useCache && cached != nil {
			return compareVersions(current, currentDisplay, cached.LatestVersion), nil
		}
		return nil, err
	}

	if useCache && strings.TrimSpace(deps.cacheFilePath) != "" {
		_ = writeCache(deps.cacheFilePath, cacheEntry{
			CheckedAt:     deps.now().UTC(),
			LatestVersion: latestVersion,
		})
	}

	return compareVersions(current, currentDisplay, latestVersion), nil
}

func defaultCheckDeps() checkDeps {
	deps := checkDeps{
		now:        time.Now,
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
	}

	if path, err := defaultCacheFilePath(); err == nil {
		deps.cacheFilePath = path
	}

	return deps
}

func shouldSkipUpdateCheck() bool {
	return strings.TrimSpace(os.Getenv(UpdateCheckEnvVar)) == "0"
}

func defaultCacheFilePath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "kubara", "update-check.json"), nil
}

func readCache(cachePath string) *cacheEntry {
	if strings.TrimSpace(cachePath) == "" {
		return nil
	}
	data, err := os.ReadFile(cachePath)
	// Missing or unreadable cache is fine; we simply treat it as a cache miss.
	if err != nil {
		return nil
	}

	var cached cacheEntry
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil
	}
	if cached.CheckedAt.IsZero() || strings.TrimSpace(cached.LatestVersion) == "" {
		return nil
	}

	return &cached
}

func writeCache(cachePath string, cached cacheEntry) error {
	if strings.TrimSpace(cachePath) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath, data, 0o600)
}

func isFresh(cached *cacheEntry, now time.Time) bool {
	if cached == nil {
		return false
	}
	return now.Sub(cached.CheckedAt) < defaultCacheTTL
}

func compareVersions(current *semver.Version, currentDisplay, latestRaw string) *Result {
	latest, latestDisplay, ok := parseVersion(latestRaw)
	// No message when latest tag is invalid or not newer than current.
	if !ok || !latest.GreaterThan(current) {
		return nil
	}

	return &Result{
		CurrentVersion: currentDisplay,
		LatestVersion:  latestDisplay,
	}
}

func parseVersion(raw string) (*semver.Version, string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" || strings.EqualFold(value, "dev") {
		return nil, "", false
	}
	value = strings.TrimPrefix(value, "v")
	if value == "" {
		return nil, "", false
	}

	parsed, err := semver.NewVersion(value)
	if err != nil {
		return nil, "", false
	}

	return parsed, "v" + parsed.String(), true
}

func fetchLatestVersion(ctx context.Context, currentVersion string, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubLatestReleaseAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "github.com/kubara-io/kubara/"+currentVersion)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("github release check failed with status %s", resp.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.TagName) == "" {
		return "", fmt.Errorf("github release response did not contain tag_name")
	}

	return payload.TagName, nil
}

func formatReleaseHint(result *Result) string {
	return releaseHintStyle(fmt.Sprintf("A new kubara release is available: %s (current: %s)", result.LatestVersion, result.CurrentVersion))
}
