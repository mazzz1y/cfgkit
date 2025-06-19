package renderer

import (
	"cfgkit/internal/config"
	"encoding/json"
	"testing"
)

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
			"d": {TemplateName: "t", Variables: map[string]any{"dv": "D"}},
		},
		Variables: map[string]any{"gv": "G"},
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
			"d": {TemplateName: "j", Variables: map[string]any{}},
		},
		Variables: map[string]any{"v": "V"},
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
			"d": {TemplateName: "b", Variables: map[string]any{}},
		},
		Variables: map[string]any{"v": "X"},
		Templates: map[string]config.TemplateConfig{
			"b": {Type: "json", Template: `{"field": {{.Global.v}}}`},
		},
	}
	r, err := New(cfg, "", "d")
	if err != nil {
		t.Fatalf("create renderer: %v", err)
	}
	_, err = r.Render()
	if err == nil {
		t.Fatal("expected error for invalid json output")
	}
}
