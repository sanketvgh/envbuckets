package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackageDocumentation(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(filepath.Join(src, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	inputs := map[string]string{
		"README.md":             "# Synthetic user documentation\n",
		"LICENSE":               "Synthetic license text\n",
		"src/package.json":      `{"name":"envbuckets","files":["bin"]}`,
		"src/bin/envbuckets.js": "// synthetic launcher\n",
		"envbuckets":            "synthetic binary\n",
	}
	for name, content := range inputs {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	license := filepath.Join(root, "LICENSE")
	umbrella := filepath.Join(root, "umbrella")
	if err := writeUmbrella(src, filepath.Join(root, "README.md"), license, umbrella, "1.2.3", map[string]string{"@envbuckets/linux-x64": "1.2.3"}); err != nil {
		t.Fatal(err)
	}
	platform := filepath.Join(root, "platform")
	if err := writePlatformPackage(platform, "@envbuckets/linux-x64", "1.2.3", "linux", "x64", license, artifact{Name: "envbuckets", Path: filepath.Join(root, "envbuckets")}); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"umbrella/README.md": inputs["README.md"],
		"umbrella/LICENSE":   inputs["LICENSE"],
		"platform/LICENSE":   inputs["LICENSE"],
	} {
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("%s differs from its source", name)
		}
	}
}
