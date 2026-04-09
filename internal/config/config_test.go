package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"testing"
)

func decodeNode(t *testing.T, n yaml.Node) map[string]any {
	t.Helper()
	if n.Kind == 0 {
		return nil
	}
	var m map[string]any
	if err := n.Decode(&m); err != nil {
		t.Fatalf("decode node: %v", err)
	}
	return m
}

func TestLoadSuccess(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.yaml")
	content1 := `
devices:
  dev1:
    password: "pass"
    template: "tpl1"
    variables:
      key1: "val1"
variables:
  global1: 100
templates:
  tpl1:
    type: "inline"
    data: "hello"
`
	if err := os.WriteFile(f1, []byte(content1), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	f2 := filepath.Join(dir, "b.yaml")
	content2 := `
templates:
  tpl2:
    type: "file"
    data: "world"
variables:
  global2: "yes"
`
	if err := os.WriteFile(f2, []byte(content2), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(cfg.Devices))
	}

	dev, ok := cfg.Devices["dev1"]
	if !ok {
		t.Fatalf("device dev1 missing")
	}
	if dev.Password != "pass" {
		t.Errorf("password mismatch")
	}
	if dev.TemplateName != "tpl1" {
		t.Errorf("template name mismatch")
	}

	deviceVars := decodeNode(t, dev.Variables)
	if v, ok := deviceVars["key1"]; !ok || v != "val1" {
		t.Errorf("device variables mismatch: %v", deviceVars)
	}

	globalVars := decodeNode(t, cfg.Variables)
	if globalVars["global1"] != 100 {
		t.Errorf("variables global1 mismatch: %v", globalVars["global1"])
	}
	if globalVars["global2"] != "yes" {
		t.Errorf("variables global2 mismatch: %v", globalVars["global2"])
	}

	if len(cfg.Templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(cfg.Templates))
	}
	tpl1 := cfg.Templates["tpl1"]
	if tpl1.Type != "inline" || tpl1.Template != "hello" {
		t.Errorf("template tpl1 mismatch")
	}
	tpl2 := cfg.Templates["tpl2"]
	if tpl2.Type != "file" || tpl2.Template != "world" {
		t.Errorf("template tpl2 mismatch")
	}
}

func TestLoadVariablesMergeOverride(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.yaml"), []byte("variables:\n  shared: first\n  onlyA: a\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.yaml"), []byte("variables:\n  shared: second\n  onlyB: b\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	merged := decodeNode(t, cfg.Variables)
	if merged["shared"] != "second" {
		t.Errorf("expected override, got %v", merged["shared"])
	}
	if merged["onlyA"] != "a" {
		t.Errorf("expected onlyA preserved, got %v", merged["onlyA"])
	}
	if merged["onlyB"] != "b" {
		t.Errorf("expected onlyB merged, got %v", merged["onlyB"])
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(f, []byte(":::bad yaml"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatalf("expected error for invalid yaml")
	}
}

func TestLoadNonexistent(t *testing.T) {
	if _, err := Load("no-such-dir"); err == nil {
		t.Fatalf("expected error for missing dir")
	}
}
