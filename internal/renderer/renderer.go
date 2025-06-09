package renderer

import (
	"bytes"
	"cfgkit/internal/config"
	"encoding/json"
	"fmt"
	"os"
	"text/template"
)

type Renderer struct {
	tmplStr    string
	resultType string
	vars       Variables
	funcMap    template.FuncMap
}

type Result struct {
	Data        bytes.Buffer
	ContentType string
}

type Variables struct {
	Global map[string]any
	Device map[string]any
}

func New(cfg *config.Config, name string) (*Renderer, error) {
	d, ok := cfg.Devices[name]
	if !ok {
		return nil, fmt.Errorf("device %s not found", name)
	}

	if d.TemplateName == "" {
		d.TemplateName = "default"
	}

	t, ok := cfg.Templates[d.TemplateName]
	if !ok {
		return nil, fmt.Errorf("template %s not found", d.TemplateName)
	}

	return &Renderer{
		tmplStr:    t.Template,
		resultType: t.Type,
		vars: Variables{
			Global: cfg.Variables,
			Device: d.Variables,
		},
		funcMap: template.FuncMap{
			"toJSON":   toJSON,
			"readFile": readFile,
			"readJSON": readJSON,
		},
	}, nil
}

func (r *Renderer) Render() (*Result, error) {
	buf, err := r.renderLoop()
	if err != nil {
		return nil, err
	}

	return r.format(buf)
}

func (r *Renderer) renderLoop() (*bytes.Buffer, error) {
	var prevResult string
	currentTmplStr := r.tmplStr
	buf := &bytes.Buffer{}

	for {
		buf.Reset()

		tmpl, err := template.New("").Funcs(r.funcMap).Parse(currentTmplStr)
		if err != nil {
			return nil, fmt.Errorf("parse template: %w", err)
		}

		if err := tmpl.Execute(buf, r.vars); err != nil {
			return nil, fmt.Errorf("render: %w", err)
		}

		newResult := buf.String()
		if prevResult == newResult {
			break
		}

		prevResult = newResult
		currentTmplStr = newResult
	}

	return buf, nil
}

func (r *Renderer) format(buf *bytes.Buffer) (*Result, error) {
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

func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func readJSON(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var result any
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
