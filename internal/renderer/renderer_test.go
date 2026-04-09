package renderer

import (
	"cfgkit/internal/config"
	"encoding/json"
	"gopkg.in/yaml.v3"
	"testing"
)

func mustNode(t *testing.T, s string) yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(s), &doc); err != nil {
		t.Fatalf("parse yaml: %v", err)
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) == 1 {
		return *doc.Content[0]
	}
	return doc
}

func TestNewDeviceNotFound(t *testing.T) {
	_, err := New(&config.Config{}, "no", "")
	if err == nil {
		t.Fatal("expected error for missing device")
	}
}

func TestNewTemplateNotFound(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "x"},
		},
		Templates: map[string]config.TemplateConfig{},
	}
	_, err := New(cfg, "d", "")
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestRenderText(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "t", Variables: mustNode(t, "dv: D")},
		},
		Variables: mustNode(t, "gv: G"),
		Templates: map[string]config.TemplateConfig{
			"t": {Type: "text", Template: "{{.Global.gv}}-{{.Device.dv}}"},
		},
	}
	r, err := New(cfg, "", "d")
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	res, err := r.Render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if res.ContentType != "text/plain; charset=utf-8" {
		t.Errorf("unexpected content type: %s", res.ContentType)
	}
	if got := res.Data.String(); got != "G-D" {
		t.Errorf("unexpected data: %s", got)
	}
}

func TestRenderJSON(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "j"},
		},
		Variables: mustNode(t, "v: V"),
		Templates: map[string]config.TemplateConfig{
			"j": {Type: "json", Template: `{"field":"{{.Global.v}}"}`},
		},
	}
	r, err := New(cfg, "", "d")
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	res, err := r.Render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if res.ContentType != "application/json" {
		t.Errorf("unexpected content type: %s", res.ContentType)
	}
	var obj map[string]string
	if err := json.Unmarshal(res.Data.Bytes(), &obj); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if obj["field"] != "V" {
		t.Errorf("unexpected field value: %s", obj["field"])
	}
}

func TestRenderInvalidJSON(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "b"},
		},
		Variables: mustNode(t, "v: X"),
		Templates: map[string]config.TemplateConfig{
			"b": {Type: "json", Template: `{"field": {{.Global.v}}}`},
		},
	}
	r, err := New(cfg, "", "d")
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	if _, err := r.Render(); err == nil {
		t.Fatal("expected error for invalid json output")
	}
}

func TestResolveGlobalSequential(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "t"},
		},
		Variables: mustNode(t, "a: hello\nb: \"{{ .Global.a }} world\"\n"),
		Templates: map[string]config.TemplateConfig{
			"t": {Type: "text", Template: "{{.Global.b}}"},
		},
	}
	r, err := New(cfg, "", "d")
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	res, err := r.Render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := res.Data.String(); got != "hello world" {
		t.Errorf("unexpected data: %q", got)
	}
}

func TestResolveNestedSiblingVisible(t *testing.T) {
	vars := `
functions:
  first: alpha
  second: "{{ .Global.functions.first }}-beta"
`
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "t"},
		},
		Variables: mustNode(t, vars),
		Templates: map[string]config.TemplateConfig{
			"t": {Type: "text", Template: "{{ .Global.functions.second }}"},
		},
	}
	r, err := New(cfg, "", "d")
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	res, err := r.Render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := res.Data.String(); got != "alpha-beta" {
		t.Errorf("unexpected data: %q", got)
	}
}

func TestResolveGlobalReferencesDevice(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"phone": {TemplateName: "t", Variables: mustNode(t, "tag: alpha")},
		},
		Variables: mustNode(t, "label: \"device-{{ .Device.tag }}\""),
		Templates: map[string]config.TemplateConfig{
			"t": {Type: "text", Template: "{{.Global.label}}"},
		},
	}
	r, err := New(cfg, "", "phone")
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	res, err := r.Render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := res.Data.String(); got != "device-alpha" {
		t.Errorf("unexpected data: %q", got)
	}
}
