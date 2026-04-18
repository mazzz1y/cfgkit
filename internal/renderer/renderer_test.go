package renderer

import (
	"cfgkit/internal/config"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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

func TestValidateNotConfigured(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "t"},
		},
		Variables: mustNode(t, "v: hello"),
		Templates: map[string]config.TemplateConfig{
			"t": {Type: "text", Template: "{{.Global.v}}"},
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
	if got := res.Data.String(); got != "hello" {
		t.Errorf("unexpected data: %q", got)
	}
}

func TestValidateSuccess(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "t"},
		},
		Variables: mustNode(t, "v: hello"),
		Templates: map[string]config.TemplateConfig{
			"t": {Type: "text", Template: "{{.Global.v}}", Check: []any{"cat", "{{ .TemplateFilePath }}"}},
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
	if got := res.Data.String(); got != "hello" {
		t.Errorf("unexpected data: %q", got)
	}
}

func TestValidateFailure(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "t"},
		},
		Variables: mustNode(t, "v: hello"),
		Templates: map[string]config.TemplateConfig{
			"t": {Type: "text", Template: "{{.Global.v}}", Check: []any{"false", "{{ .TemplateFilePath }}"}},
		},
	}
	r, err := New(cfg, "", "d")
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	_, err = r.Render()
	if err == nil {
		t.Fatal("expected error from check command")
	}
	if !strings.Contains(err.Error(), "check") {
		t.Errorf("expected 'check' in error, got: %v", err)
	}
}

func TestValidateShellForm(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "t"},
		},
		Variables: mustNode(t, "v: hello"),
		Templates: map[string]config.TemplateConfig{
			"t": {Type: "text", Template: "{{.Global.v}}", Check: "true {{ .TemplateFilePath }}"},
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
	if got := res.Data.String(); got != "hello" {
		t.Errorf("unexpected data: %q", got)
	}
}

func TestValidateShellFormFailure(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "t"},
		},
		Variables: mustNode(t, "v: hello"),
		Templates: map[string]config.TemplateConfig{
			"t": {Type: "text", Template: "{{.Global.v}}", Check: "exit 1"},
		},
	}
	r, err := New(cfg, "", "d")
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	_, err = r.Render()
	if err == nil {
		t.Fatal("expected error from shell check command")
	}
}

func TestValidateReceivesTempFilePath(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "t"},
		},
		Variables: mustNode(t, "v: hello"),
		Templates: map[string]config.TemplateConfig{
			"t": {Type: "text", Template: "{{.Global.v}}",
				Check: []any{"grep", "-q", "hello", "{{ .TemplateFilePath }}"}},
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
	if got := res.Data.String(); got != "hello" {
		t.Errorf("unexpected data: %q", got)
	}
}

func TestParseCheckNil(t *testing.T) {
	result, err := parseCheck(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestParseCheckExecForm(t *testing.T) {
	result, err := parseCheck([]any{"mycheck", "--validate", "-c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 || result[0] != "mycheck" || result[1] != "--validate" || result[2] != "-c" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestParseCheckShellForm(t *testing.T) {
	result, err := parseCheck("mycheck --validate -c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 || result[0] != "sh" || result[1] != "-c" || result[2] != "mycheck --validate -c" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestParseCheckEmptyString(t *testing.T) {
	result, err := parseCheck("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestParseCheckInvalidType(t *testing.T) {
	_, err := parseCheck(123)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestParseCheckInvalidListElement(t *testing.T) {
	_, err := parseCheck([]any{"ok", 123})
	if err == nil {
		t.Fatal("expected error for non-string list element")
	}
}

func TestValidateJSONFileExtension(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "t"},
		},
		Templates: map[string]config.TemplateConfig{
			"t": {Type: "json", Template: `{"ok":true}`,
				Check: "case {{ .TemplateFilePath }} in *.json) exit 0;; *) exit 1;; esac"},
		},
	}
	r, err := New(cfg, "", "d")
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	if _, err := r.Render(); err != nil {
		t.Fatalf("expected .json extension: %v", err)
	}
}

func TestValidateTextNoExtension(t *testing.T) {
	cfg := &config.Config{
		Devices: map[string]config.DeviceConfig{
			"d": {TemplateName: "t"},
		},
		Templates: map[string]config.TemplateConfig{
			"t": {Type: "text", Template: "hello",
				Check: "case {{ .TemplateFilePath }} in *.*) exit 1;; *) exit 0;; esac"},
		},
	}
	r, err := New(cfg, "", "d")
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	if _, err := r.Render(); err != nil {
		t.Fatalf("expected no extension: %v", err)
	}
}
