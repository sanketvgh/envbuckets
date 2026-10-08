// Package config loads the shared, data-only envbuckets configuration.
package config

import (
	"bytes"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/sanketvgh/envbuckets/internal/pattern"
)

// BucketNamePattern is the shared RE2 and ECMA-compatible bucket-name regex.
const BucketNamePattern = `^[A-Za-z0-9][A-Za-z0-9_-]*$`

var bucketName = regexp.MustCompile(BucketNamePattern)

// Rule maps one Git branch pattern to a bucket.
type Rule struct {
	Branch string `json:"branch"`
	Bucket string `json:"bucket"`
}

// Config is the shared branch-to-bucket configuration.
type Config struct {
	Schema  string `json:"$schema,omitempty"`
	Default string `json:"default"`
	Rules   []Rule `json:"rules"`
}

// ValidBucket reports whether name follows the bucket naming rules.
func ValidBucket(name string) bool { return bucketName.MatchString(name) }

// Parse strictly decodes one envbuckets configuration document.
func Parse(data []byte) (Config, error) {
	var c Config
	data = bytes.TrimSpace(data)
	var members map[string]jsontext.Value
	if err := jsonv2.Unmarshal(data, &members); err != nil {
		return Config{}, fmt.Errorf(".envbuckets.json: %s", describeJSONError(err))
	}
	if rules, ok := members["rules"]; ok && rules.Kind() != jsontext.KindBeginArray {
		return Config{}, errors.New(".envbuckets.json: invalid value at /rules (expected array)")
	}
	if err := jsonv2.Unmarshal(data, &c, jsonv2.RejectUnknownMembers(true)); err != nil {
		return Config{}, fmt.Errorf(".envbuckets.json: %s", describeJSONError(err))
	}
	if c.Default == "" {
		return Config{}, errors.New(".envbuckets.json: missing default")
	}
	if !ValidBucket(c.Default) {
		return Config{}, fmt.Errorf(".envbuckets.json: invalid default bucket %q", c.Default)
	}
	for i, r := range c.Rules {
		if r.Branch == "" {
			return Config{}, fmt.Errorf(".envbuckets.json: rules[%d] missing branch", i)
		}
		if !ValidBucket(r.Bucket) {
			return Config{}, fmt.Errorf(".envbuckets.json: invalid bucket %q in rules[%d]", r.Bucket, i)
		}
		if err := pattern.Validate(r.Branch); err != nil {
			return Config{}, fmt.Errorf(".envbuckets.json: invalid branch pattern %q: %w", r.Branch, err)
		}
	}
	return c, nil
}

func describeJSONError(err error) string {
	if errors.Is(err, jsontext.ErrDuplicateName) {
		if syntax, ok := errors.AsType[*jsontext.SyntacticError](err); ok && syntax.JSONPointer.LastToken() != "" {
			return fmt.Sprintf("duplicate key %q", syntax.JSONPointer.LastToken())
		}
		return "duplicate key"
	}
	if errors.Is(err, jsonv2.ErrUnknownName) {
		if semantic, ok := errors.AsType[*jsonv2.SemanticError](err); ok {
			return fmt.Sprintf("unknown key %q", semantic.JSONPointer.LastToken())
		}
		return "unknown key"
	}
	if semantic, ok := errors.AsType[*jsonv2.SemanticError](err); ok {
		return "invalid value at " + pointerName(semantic.JSONPointer)
	}
	if syntax, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
		return "invalid JSON at " + pointerName(syntax.JSONPointer)
	}
	return "invalid JSON"
}

func pointerName(pointer jsontext.Pointer) string {
	if pointer == "" {
		return "the top level"
	}
	return string(pointer)
}

// Load reads and strictly parses .envbuckets.json through the repository root.
func Load(root *os.Root) (Config, error) {
	b, err := root.ReadFile(".envbuckets.json")
	if err != nil {
		return Config{}, fmt.Errorf(".envbuckets.json: %w", err)
	}
	return Parse(b)
}

// BucketFor returns the first matching bucket and rule pattern, or the default
// bucket and an empty rule name when no rule matches.
func (c Config) BucketFor(branch string) (string, string) {
	for _, r := range c.Rules {
		if pattern.Match(r.Branch, branch) {
			return r.Bucket, r.Branch
		}
	}
	return c.Default, ""
}
