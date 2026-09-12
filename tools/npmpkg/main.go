// Lays out npm packages from GoReleaser's dist/artifacts.json:
// one @envbuckets/<os>-<arch> package per binary plus the envbuckets
// umbrella that depends on all of them optionally.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type artifact struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Goos   string `json:"goos"`
	Goarch string `json:"goarch"`
}

var (
	npmOS   = map[string]string{"linux": "linux", "darwin": "darwin", "windows": "win32"}
	npmArch = map[string]string{"amd64": "x64", "arm64": "arm64"}
)

func main() {
	version := flag.String("version", "", "package version without v prefix (default: from dist/metadata.json)")
	dist := flag.String("dist", "dist", "GoReleaser dist directory")
	umbrella := flag.String("umbrella", "npm/envbuckets", "umbrella package source directory")
	out := flag.String("out", "npm/dist", "output directory")
	flag.Parse()

	if *version == "" {
		v, err := readVersion(filepath.Join(*dist, "metadata.json"))
		if err != nil {
			fail(err.Error())
		}
		*version = v
	}

	artifacts, err := readArtifacts(filepath.Join(*dist, "artifacts.json"))
	if err != nil {
		fail(err.Error())
	}

	optional := map[string]string{}
	for _, a := range artifacts {
		if a.Type != "Binary" {
			continue
		}
		osName, okOS := npmOS[a.Goos]
		arch, okArch := npmArch[a.Goarch]
		if !okOS || !okArch {
			fail(fmt.Sprintf("no npm mapping for %s/%s", a.Goos, a.Goarch))
		}
		name := fmt.Sprintf("@envbuckets/%s-%s", osName, arch)
		if err := writePlatformPackage(filepath.Join(*out, name), name, *version, osName, arch, a); err != nil {
			fail(err.Error())
		}
		optional[name] = *version
	}
	if len(optional) == 0 {
		fail("no Binary artifacts found in " + *dist)
	}

	if err := writeUmbrella(*umbrella, filepath.Join(*out, "envbuckets"), *version, optional); err != nil {
		fail(err.Error())
	}

	names := make([]string, 0, len(optional))
	for n := range optional {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Println(n)
	}
	fmt.Println("envbuckets")
}

func readVersion(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var meta struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(b, &meta); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	if meta.Version == "" {
		return "", fmt.Errorf("%s: no version field", path)
	}
	return meta.Version, nil
}

func readArtifacts(path string) ([]artifact, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []artifact
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return out, nil
}

func writePlatformPackage(dir, name, version, osName, arch string, a artifact) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := copyFile(a.Path, filepath.Join(dir, a.Name), 0o755); err != nil {
		return err
	}
	pkg := map[string]any{
		"name":        name,
		"version":     version,
		"description": "envbuckets binary for " + osName + "-" + arch,
		"author":      "Sanket Vaghela <sanketvgh@gmail.com>",
		"license":     "MIT",
		"repository": map[string]string{
			"type": "git",
			"url":  "git+https://github.com/sanketvgh/envbuckets.git",
		},
		"os":    []string{osName},
		"cpu":   []string{arch},
		"files": []string{a.Name},
	}
	return writeJSON(filepath.Join(dir, "package.json"), pkg)
}

func writeUmbrella(src, dir, version string, optional map[string]string) error {
	b, err := os.ReadFile(filepath.Join(src, "package.json"))
	if err != nil {
		return err
	}
	var pkg map[string]any
	if err := json.Unmarshal(b, &pkg); err != nil {
		return err
	}
	pkg["version"] = version
	pkg["optionalDependencies"] = optional

	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(src, "bin", "envbuckets.js"), filepath.Join(dir, "bin", "envbuckets.js"), 0o755); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "package.json"), pkg)
}

func writeJSON(path string, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "npmpkg:", msg)
	os.Exit(1)
}
