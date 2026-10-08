package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStrictAndRuleOrder(t *testing.T) {
	c, err := Parse([]byte(`{"default":"dev","rules":[{"branch":"release/**","bucket":"prod"},{"branch":"release/1.*","bucket":"stage"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	got, rule := c.BucketFor("release/1.2")
	if got != "prod" || rule != "release/**" {
		t.Fatalf("got %q via %q", got, rule)
	}
}

func TestParseErrorsNameFileAndProblem(t *testing.T) {
	for _, tc := range []struct{ input, part string }{
		{`{"default":"dev","bad":true}`, "bad"},
		{`{"rules":[]}`, "default"},
		{`{"default":""}`, "default"},
		{`{"default":"bad/name"}`, "bucket"},
		{`{"default":"-x"}`, "bucket"},
		{`{"default":"_x"}`, "bucket"},
		{`{"default":"dev","rules":[{"branch":"main","bucket":"-x"}]}`, "bucket"},
		{`{"default":"dev","rules":null}`, "array"},
		{`{"default":"dev","rules":[{"branch":"","bucket":"prod"}]}`, "branch"},
		{`{"default":"dev"} {}`, "invalid json"},
		{`{"default":"dev","rules":[{"branch":"[","bucket":"prod"}]}`, "pattern"},
	} {
		_, err := Parse([]byte(tc.input))
		if err == nil || !strings.Contains(err.Error(), ".envbuckets.json:") || !strings.Contains(strings.ToLower(err.Error()), tc.part) {
			t.Errorf("Parse(%s) error = %v, want %q", tc.input, err, tc.part)
		}
	}
}

func TestParseStrictV2Cases(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"duplicate key", []byte(`{"default":"dev","default":"prod"}`), "duplicate key"},
		{"nested unknown key", []byte(`{"default":"dev","rules":[{"branch":"main","bucket":"prod","typo":true}]}`), `unknown key "typo"`},
		{"wrong case", []byte(`{"Default":"dev"}`), `unknown key "Default"`},
		{"invalid UTF-8", []byte{'{', '"', 'd', 'e', 'f', 'a', 'u', 'l', 't', '"', ':', '"', 0xff, '"', '}'}, "invalid json"},
		{"trailing data", []byte(`{"default":"dev"} {}`), "invalid json"},
		{"trailing comma", []byte(`{"default":"dev",}`), "invalid json"},
		{"wrong type", []byte(`{"default":42}`), "invalid value at /default"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.data)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Errorf("Parse error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidBucketBoundaries(t *testing.T) {
	for _, name := range []string{"dev", "x-", "x_y", "9-prod"} {
		if !ValidBucket(name) {
			t.Errorf("ValidBucket(%q) = false", name)
		}
	}
	for _, name := range []string{"", "-x", "_x"} {
		if ValidBucket(name) {
			t.Errorf("ValidBucket(%q) = true", name)
		}
	}
}

func TestLoadIgnoresLegacyTOML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".envbuckets.toml"), []byte("invalid = ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".envbuckets.json"), []byte(`{"default":"dev"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	c, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if c.Default != "dev" {
		t.Fatalf("default = %q, want dev", c.Default)
	}
}
