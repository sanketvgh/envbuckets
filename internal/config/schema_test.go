package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestSchemaMatchesConfigTags(t *testing.T) {
	data, err := os.ReadFile("../../schema/envbuckets.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	checkObjectTags(t, schema, reflect.TypeOf(Config{}), []string{"default"})
	props := schema["properties"].(map[string]any)
	rules := props["rules"].(map[string]any)
	item := rules["items"].(map[string]any)
	checkObjectTags(t, item, reflect.TypeOf(Rule{}), []string{"branch", "bucket"})
	for _, field := range []string{"default", "bucket"} {
		obj := schema
		if field == "bucket" {
			obj = item
		}
		p := obj["properties"].(map[string]any)[field].(map[string]any)["pattern"]
		if p != BucketNamePattern {
			t.Errorf("schema %s pattern = %v, want %s", field, p, BucketNamePattern)
		}
	}
	if !regexp.MustCompile(BucketNamePattern).MatchString("bucket_1-prod") {
		t.Fatal("bucket regex rejected a valid sample")
	}
}

func checkObjectTags(t *testing.T, schema map[string]any, typ reflect.Type, required []string) {
	t.Helper()
	if schema["type"] != "object" || schema["additionalProperties"] != false {
		t.Errorf("%s must be a closed object", typ)
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("%s has no properties", typ)
	}
	want := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if name != "-" && name != "" {
			want[name] = true
		}
	}
	if len(props) != len(want) {
		t.Errorf("%s property count = %d, want %d", typ, len(props), len(want))
	}
	for name := range want {
		property, ok := props[name].(map[string]any)
		if !ok {
			t.Errorf("schema missing %s.%s", typ, name)
			continue
		}
		if description, ok := property["description"].(string); !ok || description == "" {
			t.Errorf("schema property %s.%s has no description", typ, name)
		}
	}
	gotRequired, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("%s has no required array", typ)
	}
	got := map[string]bool{}
	for _, name := range gotRequired {
		got[name.(string)] = true
	}
	if len(got) != len(required) {
		t.Errorf("%s required = %v, want %v", typ, got, required)
	}
	for _, name := range required {
		if !got[name] {
			t.Errorf("%s missing required %s", typ, name)
		}
	}
}

func TestProductJSONConfigsParse(t *testing.T) {
	configs := productConfigs(t)
	for i, example := range configs {
		if _, err := Parse(example); err != nil {
			t.Errorf("PRODUCT.md JSON example %d: %v", i+1, err)
		}
	}
}

func TestProductJSONConfigsValidateSchema(t *testing.T) {
	schema := loadSchema(t)
	for i, example := range productConfigs(t) {
		value, err := jsonschema.UnmarshalJSON(strings.NewReader(string(example)))
		if err != nil {
			t.Fatalf("decode PRODUCT.md JSON example %d: %v", i+1, err)
		}
		if err := schema.Validate(value); err != nil {
			t.Errorf("PRODUCT.md JSON example %d fails schema: %v", i+1, err)
		}
	}
}

func TestInvalidFixturesRejectedBySchema(t *testing.T) {
	schema := loadSchema(t)
	fixtures, err := filepath.Glob("../../schema/testdata/invalid/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no invalid schema fixtures")
	}
	for _, fixture := range fixtures {
		data, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse(data); err == nil {
			t.Errorf("parser accepted invalid fixture %s", fixture)
		}
		value, err := jsonschema.UnmarshalJSON(strings.NewReader(string(data)))
		if err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(value); err == nil {
			t.Errorf("schema accepted invalid fixture %s", fixture)
		}
	}
}

func loadSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	data, err := os.ReadFile("../../schema/envbuckets.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft7)
	if err := c.AddResource("envbuckets.schema.json", doc); err != nil {
		t.Fatal(err)
	}
	schema, err := c.Compile("envbuckets.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func productConfigs(t *testing.T) [][]byte {
	t.Helper()
	data, err := os.ReadFile("../../docs-eb/PRODUCT.md")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile("(?s)```json\\s*\\n(.*?)\\n```")
	matches := re.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		t.Fatal("PRODUCT.md contains no JSON examples")
	}
	configs := make([][]byte, len(matches))
	for i, match := range matches {
		configs[i] = match[1]
	}
	return configs
}
