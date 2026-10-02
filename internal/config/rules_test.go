package config

import (
	"errors"
	"reflect"
	"regexp"
	"testing"
)

var unsafe = regexp.MustCompile(`[^a-zA-Z0-9]`)

func bucketFor(pattern string) string {
	return "b" + unsafe.ReplaceAllString(pattern, "x")
}

func cfgWith(patterns ...string) *Config {
	c := New()
	for _, p := range patterns {
		c.Rules = append(c.Rules, Rule{Pattern: p, Bucket: bucketFor(p)})
	}
	return c
}

func order(c *Config) []string {
	out := make([]string, len(c.Rules))
	for i, r := range c.Rules {
		out[i] = r.Pattern
	}
	return out
}

func TestAddRuleInsertsBeforeCatchAll(t *testing.T) {
	c := cfgWith("main", "*")
	at, err := c.AddRule(Rule{Pattern: "feature/*", Bucket: "dev"})
	if err != nil || at != 1 {
		t.Fatalf("at=%d err=%v", at, err)
	}
	if got := order(c); !reflect.DeepEqual(got, []string{"main", "feature/*", "*"}) {
		t.Fatalf("order %v", got)
	}
}

func TestAddRuleCatchAllGoesLast(t *testing.T) {
	c := cfgWith("main", "release/*")
	at, err := c.AddRule(Rule{Pattern: "*", Bucket: "dev"})
	if err != nil || at != 2 {
		t.Fatalf("at=%d err=%v", at, err)
	}
	if _, err := c.AddRule(Rule{Pattern: "*", Bucket: "prod"}); !errors.Is(err, ErrDuplicateRule) {
		t.Fatalf("second catch-all: %v", err)
	}
}

func TestAddRuleRejectsInvalidWithoutChange(t *testing.T) {
	c := cfgWith("main")
	for _, r := range []Rule{{Pattern: "main", Bucket: "x"}, {Pattern: "a b", Bucket: "x"}, {Pattern: "ok", Bucket: "../x"}} {
		if _, err := c.AddRule(r); err == nil {
			t.Fatalf("%+v accepted", r)
		}
	}
	if got := order(c); !reflect.DeepEqual(got, []string{"main"}) {
		t.Fatalf("order changed: %v", got)
	}
}

func TestUpdateRuleKeepsPriority(t *testing.T) {
	c := cfgWith("main", "release/*", "*")
	prev, at, err := c.UpdateRule("release/*", "prod2")
	if err != nil || prev != bucketFor("release/*") || at != 1 {
		t.Fatalf("prev=%s at=%d err=%v", prev, at, err)
	}
	if c.Rules[1] != (Rule{Pattern: "release/*", Bucket: "prod2"}) || len(c.Rules) != 3 {
		t.Fatalf("rules %+v", c.Rules)
	}
	if _, _, err := c.UpdateRule("nope", "x"); !errors.Is(err, ErrNoRule) {
		t.Fatalf("unknown: %v", err)
	}
	if _, _, err := c.UpdateRule("main", "bad name"); err == nil || c.Rules[0].Bucket != bucketFor("main") {
		t.Fatalf("invalid bucket: %v %+v", err, c.Rules[0])
	}
}

func TestMoveRule(t *testing.T) {
	cases := []struct {
		name    string
		from    []string
		pattern string
		anchor  string
		after   bool
		want    []string
		changed bool
		wantErr error
	}{
		{name: "before", from: []string{"a", "b", "c"}, pattern: "c", anchor: "a", want: []string{"c", "a", "b"}, changed: true},
		{name: "after", from: []string{"a", "b", "c"}, pattern: "a", anchor: "c", after: true, want: []string{"b", "c", "a"}, changed: true},
		{name: "already in place", from: []string{"a", "b", "c"}, pattern: "a", anchor: "b", want: []string{"a", "b", "c"}},
		{name: "after catch-all", from: []string{"a", "b", "*"}, pattern: "a", anchor: "*", after: true, wantErr: ErrCatchAllOrder},
		{name: "catch-all before rule", from: []string{"a", "b", "*"}, pattern: "*", anchor: "b", wantErr: ErrCatchAllOrder},
		{name: "catch-all after last rule", from: []string{"a", "b", "*"}, pattern: "*", anchor: "b", after: true, want: []string{"a", "b", "*"}},
		{name: "before catch-all", from: []string{"a", "b", "*"}, pattern: "a", anchor: "*", want: []string{"b", "a", "*"}, changed: true},
		{name: "self", from: []string{"a", "b"}, pattern: "a", anchor: "a", wantErr: ErrSelfReference},
		{name: "unknown source", from: []string{"a", "b"}, pattern: "x", anchor: "a", wantErr: ErrNoRule},
		{name: "unknown anchor", from: []string{"a", "b"}, pattern: "a", anchor: "x", wantErr: ErrNoRule},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := cfgWith(tc.from...)
			changed, err := c.MoveRule(tc.pattern, tc.anchor, tc.after)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err %v, want %v", err, tc.wantErr)
				}
				if got := order(c); !reflect.DeepEqual(got, tc.from) {
					t.Fatalf("order changed on error: %v", got)
				}
				return
			}
			if err != nil || changed != tc.changed {
				t.Fatalf("changed=%v err=%v", changed, err)
			}
			if got := order(c); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("order %v, want %v", got, tc.want)
			}
			for _, r := range c.Rules {
				if r.Bucket != bucketFor(r.Pattern) {
					t.Fatalf("mapping changed: %+v", r)
				}
			}
		})
	}
}
