package config

import (
	"os"
	"path/filepath"
	"testing"
)

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
	err := os.WriteFile(f1, []byte(content1), 0644)
	if err != nil {
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
	err = os.WriteFile(f2, []byte(content2), 0644)
	if err != nil {
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
	if v, ok := dev.Variables["key1"]; !ok || v != "val1" {
		t.Errorf("device replacement mismatch")
	}
	if cfg.Variables["global1"] != 100 {
		t.Errorf("variables global1 mismatch")
	}
	if cfg.Variables["global2"] != "yes" {
		t.Errorf("variables global2 mismatch")
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

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "invalid.yaml")
	err := os.WriteFile(f, []byte(":::bad yaml"), 0644)
	if err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err = Load(dir)
	if err == nil {
		t.Fatalf("expected error for invalid yaml")
	}
}

func TestLoadNonexistent(t *testing.T) {
	_, err := Load("no-such-dir")
	if err == nil {
		t.Fatalf("expected error for missing dir")
	}
}
