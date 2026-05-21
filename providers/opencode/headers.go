package opencode

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	HeaderProject = "x-opencode-project"
	HeaderSession = "x-opencode-session"
	HeaderRequest = "x-opencode-request"
	HeaderClient  = "x-opencode-client"
	HeaderUA      = "User-Agent"

	DefaultProjectID = "global"
	DefaultClient    = "cli"
	DefaultVersion   = "local"

	idLength    = 26
	idRandom    = idLength - 12
	base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// Header is one generated HTTP header.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HeaderOptions controls OpenCode-compatible header generation.
type HeaderOptions struct {
	// Dir is the repository or working directory used to resolve x-opencode-project.
	// If empty, the current working directory is used.
	Dir string
	// Client overrides x-opencode-client. If empty, OPENCODE_CLIENT is used,
	// falling back to "cli".
	Client string
	// Version overrides the User-Agent version. If empty, OPENCODE_VERSION is
	// used, falling back to "local".
	Version string
	// Time controls the timestamp encoded into generated IDs. If zero, time.Now()
	// is used.
	Time time.Time
}

// GenerateHeaders returns the headers OpenCode sends for opencode-hosted models.
func GenerateHeaders(opts HeaderOptions) ([]Header, error) {
	dir, err := resolveDir(opts.Dir)
	if err != nil {
		return nil, err
	}

	projectID, err := ResolveProjectID(dir)
	if err != nil {
		return nil, err
	}

	now := opts.Time
	if now.IsZero() {
		now = time.Now()
	}
	sessionID, err := NewSessionID(now)
	if err != nil {
		return nil, err
	}
	requestID, err := NewRequestID(now)
	if err != nil {
		return nil, err
	}

	return []Header{
		{Name: HeaderProject, Value: projectID},
		{Name: HeaderSession, Value: sessionID},
		{Name: HeaderRequest, Value: requestID},
		{Name: HeaderClient, Value: clientValue(opts.Client)},
		{Name: HeaderUA, Value: "opencode/" + versionValue(opts.Version)},
	}, nil
}

// ResolveProjectID approximates OpenCode's project ID resolution for a directory.
func ResolveProjectID(dir string) (string, error) {
	dir, err := resolveDir(dir)
	if err != nil {
		return "", err
	}

	commonDir, ok := gitOutput(dir, "rev-parse", "--git-common-dir")
	if !ok {
		return DefaultProjectID, nil
	}
	commonDir = resolveGitPath(dir, commonDir)

	if cached, err := os.ReadFile(filepath.Join(commonDir, "opencode")); err == nil {
		if id := strings.TrimSpace(string(cached)); id != "" {
			return id, nil
		}
	}

	rootsText, ok := gitOutput(dir, "rev-list", "--max-parents=0", "HEAD")
	if !ok {
		return DefaultProjectID, nil
	}
	roots := strings.Fields(rootsText)
	sort.Strings(roots)
	if len(roots) == 0 {
		return DefaultProjectID, nil
	}
	id := roots[0]
	_ = os.WriteFile(filepath.Join(commonDir, "opencode"), []byte(id+"\n"), 0o644)
	return id, nil
}

// NewSessionID returns an OpenCode-shaped session ID.
func NewSessionID(t time.Time) (string, error) {
	id, err := createID(true, t, 1)
	if err != nil {
		return "", err
	}
	return "ses_" + id, nil
}

// NewRequestID returns an OpenCode-shaped user request/message ID.
func NewRequestID(t time.Time) (string, error) {
	id, err := createID(false, t, 1)
	if err != nil {
		return "", err
	}
	return "msg_" + id, nil
}

func resolveDir(dir string) (string, error) {
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get current directory: %w", err)
		}
		dir = cwd
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve directory: %w", err)
	}
	if stat, err := os.Stat(absDir); err != nil {
		return "", fmt.Errorf("stat directory %s: %w", absDir, err)
	} else if !stat.IsDir() {
		return "", fmt.Errorf("not a directory: %s", absDir)
	}
	return absDir, nil
}

func gitOutput(dir string, args ...string) (string, bool) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

func resolveGitPath(cwd, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return cwd
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(cwd, path))
}

func createID(descending bool, t time.Time, counter uint16) (string, error) {
	if t.IsZero() {
		t = time.Now()
	}
	now := uint64(t.UnixMilli())*0x1000 + uint64(counter)
	if descending {
		now = ^now
	}

	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], now)
	random, err := randomBase62(idRandom)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x%s", buf[2:], random), nil
}

func randomBase62(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random id bytes: %w", err)
	}
	var b strings.Builder
	b.Grow(length)
	for _, value := range bytes {
		b.WriteByte(base62Chars[int(value)%len(base62Chars)])
	}
	return b.String(), nil
}

func clientValue(value string) string {
	if value != "" {
		return value
	}
	if value := os.Getenv("OPENCODE_CLIENT"); value != "" {
		return value
	}
	return DefaultClient
}

func versionValue(value string) string {
	if value != "" {
		return value
	}
	if value := os.Getenv("OPENCODE_VERSION"); value != "" {
		return value
	}
	return DefaultVersion
}
