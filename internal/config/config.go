// Package config reads and writes .envbuckets.toml, which holds structure
// only and is refused wholesale when it cannot be parsed.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/sanketvgh/envbuckets/internal/fsx"
)

// FileName is the committed config file at the repo root.
const FileName = ".envbuckets.toml"

const (
	header        = "# .envbuckets.toml — managed by `envbuckets`. Do not edit by hand.\n"
	schemaVersion = 1
	maxPattern    = 128
)

// Sentinel errors for the two config failure classes (both exit 2).
var (
	ErrMissing = errors.New("config missing")
	ErrCorrupt = errors.New("config unreadable")
)

var nameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// Rule maps a branch pattern to a bucket. Order is priority.
type Rule struct {
	Pattern string `toml:"pattern"`
	Bucket  string `toml:"bucket"`
}

// Scope is a registered directory with its own symlink farm.
type Scope struct {
	Name string `toml:"name"`
	Path string `toml:"path"`
}

// Config is the on-disk document.
type Config struct {
	Schema int     `toml:"schema"`
	Rules  []Rule  `toml:"rules,omitempty"`
	Scopes []Scope `toml:"scopes,omitempty"`
}

// New returns an empty schema-1 config.
func New() *Config {
	return &Config{Schema: schemaVersion}
}

// Path returns the config path for a repo root.
func Path(root string) string {
	return filepath.Join(root, FileName)
}

// Load reads and validates the config, returning ErrMissing when absent
// and ErrCorrupt on any parse, schema, or validation failure.
func Load(root string) (*Config, error) {
	data, err := os.ReadFile(Path(root))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrMissing
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCorrupt, err)
	}
	var cfg Config
	meta, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCorrupt, err)
	}
	if !meta.IsDefined("schema") {
		return nil, fmt.Errorf("%w: schema missing", ErrCorrupt)
	}
	if cfg.Schema != schemaVersion {
		return nil, fmt.Errorf("%w: schema %d unsupported (this binary supports %d)", ErrCorrupt, cfg.Schema, schemaVersion)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCorrupt, err)
	}
	return &cfg, nil
}

// Save writes the config atomically in a fixed order: header, rules, scopes.
func (c *Config) Save(root string) error {
	var buf bytes.Buffer
	buf.WriteString(header)
	if err := toml.NewEncoder(&buf).Encode(c); err != nil {
		return err
	}
	return fsx.WriteFileAtomic(Path(root), buf.Bytes(), 0o644)
}

func (c *Config) validate() error {
	seenPattern := map[string]bool{}
	for i, r := range c.Rules {
		if err := ValidatePattern(r.Pattern); err != nil {
			return fmt.Errorf("rules[%d]: %w", i, err)
		}
		if seenPattern[r.Pattern] {
			return fmt.Errorf("rules[%d]: duplicate pattern %q", i, r.Pattern)
		}
		seenPattern[r.Pattern] = true
		if r.Pattern == "*" && i != len(c.Rules)-1 {
			return fmt.Errorf("rules[%d]: catch-all must be last", i)
		}
		if err := ValidateName(r.Bucket); err != nil {
			return fmt.Errorf("rules[%d]: bucket: %w", i, err)
		}
	}
	seenName, seenPath := map[string]bool{}, map[string]bool{}
	for i, s := range c.Scopes {
		if err := ValidateName(s.Name); err != nil {
			return fmt.Errorf("scopes[%d]: name: %w", i, err)
		}
		if seenName[s.Name] {
			return fmt.Errorf("scopes[%d]: duplicate name %q", i, s.Name)
		}
		seenName[s.Name] = true
		p := CleanScopePath(s.Path)
		if p == "" || filepath.IsAbs(s.Path) || p == ".." || strings.HasPrefix(p, "../") {
			return fmt.Errorf("scopes[%d]: invalid path %q", i, s.Path)
		}
		if seenPath[p] {
			return fmt.Errorf("scopes[%d]: duplicate path %q", i, s.Path)
		}
		seenPath[p] = true
	}
	return nil
}

// ValidateName checks a bucket or scope name.
func ValidateName(name string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf("invalid name %q (want ^[a-zA-Z0-9][a-zA-Z0-9_-]*$)", name)
	}
	return nil
}

// ValidatePattern checks a branch pattern.
func ValidatePattern(p string) error {
	switch {
	case p == "":
		return errors.New("pattern is empty")
	case strings.ContainsAny(p, " \t\r\n"):
		return fmt.Errorf("pattern %q contains whitespace", p)
	case len(p) > maxPattern:
		return fmt.Errorf("pattern longer than %d chars", maxPattern)
	}
	return nil
}

// CleanScopePath normalizes a scope path to slash form with "." for root.
func CleanScopePath(p string) string {
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(p)))
}

// Match returns the first rule matching branch, or nil.
func (c *Config) Match(branch string, match func(pattern, name string) bool) *Rule {
	for i := range c.Rules {
		if match(c.Rules[i].Pattern, branch) {
			return &c.Rules[i]
		}
	}
	return nil
}

// HasCatchAll reports whether a `*` rule exists.
func (c *Config) HasCatchAll() bool {
	for _, r := range c.Rules {
		if r.Pattern == "*" {
			return true
		}
	}
	return false
}

// RulesFor lists the patterns that reference bucket.
func (c *Config) RulesFor(bucket string) []string {
	var out []string
	for _, r := range c.Rules {
		if r.Bucket == bucket {
			out = append(out, r.Pattern)
		}
	}
	return out
}

// EffectiveScopes returns the declared scopes or the implicit root scope.
func (c *Config) EffectiveScopes() []Scope {
	if len(c.Scopes) == 0 {
		return []Scope{{Name: "root", Path: "."}}
	}
	out := make([]Scope, len(c.Scopes))
	for i, s := range c.Scopes {
		out[i] = Scope{Name: s.Name, Path: CleanScopePath(s.Path)}
	}
	return out
}
