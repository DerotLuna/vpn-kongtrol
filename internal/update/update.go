// Package update is Kongtrol's opt-in update notice (ADR-0034). It asks
// GitHub Releases for the latest published, non-prerelease version at most
// once every 24 hours and only when the user turned it on
// (updates.check_enabled). It never downloads or installs anything: callers
// show a notice that tells the user how to update.
//
// The only request is GET https://api.github.com/repos/<repo>/releases/latest
// with a generic User-Agent "kongtrol/<version>" and no cookies, tokens or
// user data.
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"
)

const (
	// DefaultRepo is the GitHub repository whose releases are checked.
	DefaultRepo = "DerotLuna/vpn-kongtrol"
	// DefaultAPIBase is the GitHub REST API root (https only).
	DefaultAPIBase = "https://api.github.com"
	// Interval is the minimum time between two requests.
	Interval = 24 * time.Hour
	// Timeout bounds one request.
	Timeout = 5 * time.Second
	// MaxBody caps the bytes read from the answer (release notes and the
	// asset list make it a few KB; 1 MiB is far above any real release).
	MaxBody = 1 << 20
	// DownloadPage is where the notice sends people who installed by hand.
	DownloadPage = "https://kongtrol.app/#descargar"
)

// ErrDisabled: the user did not opt in; no request was made.
var ErrDisabled = errors.New("update check disabled")

// State is what the checker remembers between runs (~/.kongtrol/update-check.json).
type State struct {
	LastCheck time.Time `json:"last_check"`
	Latest    string    `json:"latest,omitempty"`
	URL       string    `json:"url,omitempty"`
	// Notified is the latest version the user was already told about, so
	// each new version is announced once.
	Notified string `json:"notified,omitempty"`
}

// Result is the outcome of a check (or of the remembered last check).
type Result struct {
	Current   string    `json:"current"`
	Latest    string    `json:"latest,omitempty"`
	URL       string    `json:"url,omitempty"`
	Newer     bool      `json:"newer"`
	CheckedAt time.Time `json:"checked_at,omitempty"`
}

// Checker runs the opt-in check. The zero value is not usable; use New.
type Checker struct {
	Current   string // running version, e.g. v1.2.3
	APIBase   string
	Repo      string
	StatePath string
	Client    *http.Client
	Now       func() time.Time
	// Enabled reports the user's opt-in; nil or false means no request.
	Enabled func() bool

	mu sync.Mutex
}

// DefaultStatePath is ~/.kongtrol/update-check.json.
func DefaultStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kongtrol", "update-check.json")
}

// New returns a checker for the official repository.
func New(current string, enabled func() bool) *Checker {
	return &Checker{
		Current:   current,
		APIBase:   DefaultAPIBase,
		Repo:      DefaultRepo,
		StatePath: DefaultStatePath(),
		Client:    &http.Client{Timeout: Timeout},
		Now:       time.Now,
		Enabled:   enabled,
	}
}

func (c *Checker) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// LoadState reads the remembered state; a missing or broken file is empty.
func LoadState(path string) State {
	var st State
	if path == "" {
		return st
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return st
	}
	_ = json.Unmarshal(data, &st)
	return st
}

func saveState(path string, st State) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Last returns the remembered result without any request (for `kongtrol
// version`, `status` and the dashboards).
func (c *Checker) Last() Result {
	return resultFrom(c.Current, LoadState(c.StatePath))
}

// LastFrom is Last for callers without a Checker (the CLI).
func LastFrom(current, statePath string) Result {
	return resultFrom(current, LoadState(statePath))
}

func resultFrom(current string, st State) Result {
	return Result{
		Current:   current,
		Latest:    st.Latest,
		URL:       st.URL,
		Newer:     Newer(current, st.Latest),
		CheckedAt: st.LastCheck,
	}
}

// Check asks GitHub for the latest release when the user opted in and the
// last request is older than Interval. Otherwise it returns the remembered
// result and makes no request. The attempt time is recorded even when the
// request fails, so a failing network never causes more than one request
// per day.
func (c *Checker) Check(ctx context.Context) (Result, error) {
	if c.Enabled == nil || !c.Enabled() {
		return Result{Current: c.Current}, ErrDisabled
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	st := LoadState(c.StatePath)
	now := c.now()
	if d := now.Sub(st.LastCheck); !st.LastCheck.IsZero() && d >= 0 && d < Interval {
		return resultFrom(c.Current, st), nil
	}
	st.LastCheck = now
	tag, page, err := c.fetchLatest(ctx)
	if err == nil {
		st.Latest, st.URL = tag, page
	}
	if serr := saveState(c.StatePath, st); serr != nil && err == nil {
		err = serr
	}
	return resultFrom(c.Current, st), err
}

// TakeNotice returns the result when there is a newer version the user was
// not told about yet, and remembers that it was announced.
func (c *Checker) TakeNotice(r Result) (Result, bool) {
	if !r.Newer {
		return r, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	st := LoadState(c.StatePath)
	if st.Notified == r.Latest {
		return r, false
	}
	st.Notified = r.Latest
	_ = saveState(c.StatePath, st)
	return r, true
}

type releaseDTO struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

func (c *Checker) fetchLatest(ctx context.Context) (tag, page string, err error) {
	base, err := url.Parse(c.APIBase)
	if err != nil || base.Host == "" {
		return "", "", fmt.Errorf("update: bad API base %q", c.APIBase)
	}
	if !strings.EqualFold(base.Scheme, "https") && !testAllowHTTP {
		return "", "", fmt.Errorf("update: API base must be https")
	}
	endpoint := strings.TrimRight(c.APIBase, "/") + "/repos/" + c.Repo + "/releases/latest"
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}
	// Only what GitHub needs: a generic User-Agent (required by its API) and
	// the documented Accept header. No token, no cookies, no user data.
	req.Header.Set("User-Agent", "kongtrol/"+c.Current)
	req.Header.Set("Accept", "application/vnd.github+json")
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: Timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("update: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return "", "", fmt.Errorf("update: GitHub answered %d", resp.StatusCode)
	}
	var rel releaseDTO
	if err := json.NewDecoder(io.LimitReader(resp.Body, MaxBody)).Decode(&rel); err != nil {
		return "", "", fmt.Errorf("update: bad answer: %w", err)
	}
	tag = canonical(rel.TagName)
	if rel.Draft || rel.Prerelease || tag == "" || semver.Prerelease(tag) != "" {
		// "latest" never returns these (GitHub docs); ignore them anyway.
		return "", "", fmt.Errorf("update: no stable release")
	}
	page = rel.HTMLURL
	if u, perr := url.Parse(page); perr != nil || !strings.EqualFold(u.Scheme, "https") {
		page = ""
	}
	return tag, page, nil
}

// testAllowHTTP lets tests use an httptest (http://) server.
var testAllowHTTP = false

func canonical(v string) string {
	v = strings.TrimSpace(v)
	if v != "" && !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return ""
	}
	return semver.Canonical(v)
}

// Newer reports whether latest is a newer stable version than current.
// Unparseable versions (local builds) never get a notice.
func Newer(current, latest string) bool {
	cur, lat := canonical(current), canonical(latest)
	if cur == "" || lat == "" || semver.Prerelease(lat) != "" {
		return false
	}
	return semver.Compare(lat, cur) > 0
}

// Run checks once at start and then every hour (the 24h limit decides
// whether a request goes out), calling notify once per newer version. A
// value on kick (the user just turned the notice on) checks right away,
// still within the 24h limit.
func (c *Checker) Run(ctx context.Context, notify func(Result), kick <-chan struct{}) {
	tick := func() {
		r, err := c.Check(ctx)
		if err != nil && !errors.Is(err, ErrDisabled) {
			return
		}
		if r, ok := c.TakeNotice(r); ok && notify != nil {
			notify(r)
		}
	}
	tick()
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			tick()
		case <-kick:
			tick()
		}
	}
}
