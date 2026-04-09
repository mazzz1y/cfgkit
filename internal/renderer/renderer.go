package renderer

import (
	"bytes"
	"cfgkit/internal/config"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

const defaultTemplateName = "default"

type Renderer struct {
	tmpl       *template.Template
	ctx        map[string]any
	resultType string
}

type Result struct {
	Data        bytes.Buffer
	ContentType string
}

func New(cfg *config.Config, workDir, deviceName string) (*Renderer, error) {
	device, ok := cfg.Devices[deviceName]
	if !ok {
		return nil, fmt.Errorf("device %s not found", deviceName)
	}

	templateName := device.TemplateName
	if templateName == "" {
		templateName = defaultTemplateName
	}

	tmplCfg, ok := cfg.Templates[templateName]
	if !ok {
		return nil, fmt.Errorf("template %s not found", templateName)
	}

	templateSource, err := loadTemplateSource(workDir, tmplCfg)
	if err != nil {
		return nil, fmt.Errorf("load template %s: %w", templateName, err)
	}

	funcMap := buildFuncMap(workDir)
	resolver := NewResolver(funcMap)

	if err := resolver.ResolveDevice(deviceName, device.Variables); err != nil {
		return nil, fmt.Errorf("resolve device variables: %w", err)
	}
	if err := resolver.ResolveGlobal(cfg.Variables); err != nil {
		return nil, fmt.Errorf("resolve global variables: %w", err)
	}

	tmpl, err := template.New(templateName).Funcs(funcMap).Parse(templateSource)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	return &Renderer{
		tmpl:       tmpl,
		ctx:        resolver.Context(),
		resultType: tmplCfg.Type,
	}, nil
}

func (r *Renderer) Render() (*Result, error) {
	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, r.ctx); err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}
	return r.format(&buf)
}

func (r *Renderer) format(buf *bytes.Buffer) (*Result, error) {
	switch r.resultType {
	case "json":
		var out bytes.Buffer
		if err := json.Indent(&out, buf.Bytes(), "", "  "); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		return &Result{Data: out, ContentType: "application/json"}, nil
	default:
		return &Result{Data: *buf, ContentType: "text/plain; charset=utf-8"}, nil
	}
}

func loadTemplateSource(workDir string, tmplCfg config.TemplateConfig) (string, error) {
	if tmplCfg.File != "" {
		content, err := os.ReadFile(filepath.Join(workDir, tmplCfg.File))
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
	return tmplCfg.Template, nil
}

func buildFuncMap(workDir string) template.FuncMap {
	funcMap := sprig.FuncMap()
	funcMap["fromFile"] = fromFile(workDir)
	funcMap["fromYaml"] = fromYaml
	return funcMap
}

func fromFile(workDir string) func(string) (string, error) {
	return func(path string) (string, error) {
		content, err := os.ReadFile(filepath.Join(workDir, path))
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
}

func fromYaml(data string) (any, error) {
	var result any
	if err := yaml.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}
	return result, nil
}
