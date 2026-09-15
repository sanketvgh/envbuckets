package config

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestSaveLoadRoundTripAndOrder(t *testing.T) {
	root := t.TempDir()
	cfg := New()
	cfg.Rules = []Rule{{Pattern: "main", Bucket: "staging"}, {Pattern: "*", Bucket: "dev"}}
	cfg.Scopes = []Scope{{Name: "root", Path: "."}, {Name: "api", Path: "apps/api"}}
	if err := cfg.Save(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(Path(root))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	iHeader := strings.Index(text, "# .envbuckets.toml")
	iSchema := strings.Index(text, "schema = 1")
	iRules := strings.Index(text, "[[rules]]")
	iScopes := strings.Index(text, "[[scopes]]")
	if iHeader != 0 || iSchema < iHeader || iRules < iSchema || iScopes < iRules {
		t.Fatalf("unexpected order:\n%s", text)
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Rules) != 2 || got.Rules[1].Pattern != "*" || len(got.Scopes) != 2 || got.Scopes[1].Path != "apps/api" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load(t.TempDir())
	if !errors.Is(err, ErrMissing) {
		t.Fatalf("want ErrMissing, got %v", err)
	}
}

func TestLoadCorruptVariants(t *testing.T) {
	cases := map[string]string{
		"truncated":       "schema = 1\n[[rules]]\npattern = \"ma",
		"schema too new":  "schema = 2\n",
		"schema missing":  "[[rules]]\npattern = \"main\"\nbucket = \"dev\"\n",
		"catch-all first": "schema = 1\n[[rules]]\npattern = \"*\"\nbucket = \"dev\"\n[[rules]]\npattern = \"main\"\nbucket = \"dev\"\n",
		"duplicate":       "schema = 1\n[[rules]]\npattern = \"main\"\nbucket = \"dev\"\n[[rules]]\npattern = \"main\"\nbucket = \"x\"\n",
		"bad scope name":  "schema = 1\n[[scopes]]\nname = \"-bad\"\npath = \"apps\"\n",
		"escaping scope":  "schema = 1\n[[scopes]]\nname = \"x\"\npath = \"../x\"\n",
	}
	for name, body := range cases {
		root := t.TempDir()
		if err := os.WriteFile(Path(root), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(root)
		if !errors.Is(err, ErrCorrupt) {
			t.Errorf("%s: want ErrCorrupt, got %v", name, err)
		}
	}
}

func TestEffectiveScopesImplicitRoot(t *testing.T) {
	scopes := New().EffectiveScopes()
	if len(scopes) != 1 || scopes[0].Name != "root" || scopes[0].Path != "." {
		t.Fatalf("got %+v", scopes)
	}
}

func TestValidatePattern(t *testing.T) {
	if ValidatePattern("") == nil || ValidatePattern("a b") == nil || ValidatePattern(strings.Repeat("x", 129)) == nil {
		t.Fatal("invalid patterns accepted")
	}
	if ValidatePattern("release/*") != nil {
		t.Fatal("valid pattern rejected")
	}
}
