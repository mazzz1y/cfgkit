package renderer

import (
	"bytes"
	"cfgkit/internal/config"
	"encoding/json"
	"fmt"
	"text/template"
)

type Renderer struct {
	tmpl       *template.Template
	resultType string
	globalVars map[string]any
	deviceVars map[string]any
}

type Result struct {
	Data        bytes.Buffer
	ContentType string
}

func New(cfg *config.Config, name string) (*Renderer, error) {
	d, ok := cfg.Devices[name]
	if !ok {
		return nil, fmt.Errorf("device %s not found", name)
	}

	t, ok := cfg.Templates[d.TemplateName]
	if !ok {
		return nil, fmt.Errorf("template %s not found", d.TemplateName)
	}

	funcMap := template.FuncMap{
		"toJSON": toJSON,
	}

	tmpl, err := template.New("").Funcs(funcMap).Parse(t.Template)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	return &Renderer{
		tmpl:       tmpl,
		resultType: t.Type,
		globalVars: cfg.Variables,
		deviceVars: d.Variables,
	}, nil
}

func (r *Renderer) Render() (*Result, error) {
	buf := &bytes.Buffer{}
	data := struct {
		Global map[string]any
		Device map[string]any
	}{
		Global: r.globalVars,
		Device: r.deviceVars,
	}
	if err := r.tmpl.Execute(buf, data); err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}

	switch r.resultType {
	case "json":
		var out bytes.Buffer
		if err := json.Indent(&out, buf.Bytes(), "", "  "); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		return &Result{
			Data:        out,
			ContentType: "application/json",
		}, nil
	default:
		return &Result{
			Data:        *buf,
			ContentType: "text/plain; charset=utf-8",
		}, nil
	}
}

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
