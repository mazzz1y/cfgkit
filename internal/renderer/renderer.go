package renderer

import (
	"bytes"
	"cfgkit/internal/config"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

const defaultTemplateName = "default"

type Renderer struct {
	tmpl       *template.Template
	ctx        map[string]any
	resultType string
	check      []string
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

	check, err := parseCheck(tmplCfg.Check)
	if err != nil {
		return nil, fmt.Errorf("parse check command: %w", err)
	}

	return &Renderer{
		tmpl:       tmpl,
		ctx:        resolver.Context(),
		resultType: tmplCfg.Type,
		check:      check,
	}, nil
}

func (r *Renderer) Render() (*Result, error) {
	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, r.ctx); err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}
	res, err := r.format(&buf)
	if err != nil {
		return nil, err
	}
	if err := r.validate(res.Data.Bytes()); err != nil {
		return nil, err
	}
	return res, nil
}

func parseCheck(raw any) ([]string, error) {
	if raw == nil {
		return nil, nil
	}
	switch v := raw.(type) {
	case string:
		if v == "" {
			return nil, nil
		}
		return []string{"sh", "-c", v}, nil
	case []any:
		cmd := make([]string, 0, len(v))
		for i, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("check[%d]: expected string, got %T", i, item)
			}
			cmd = append(cmd, s)
		}
		if len(cmd) == 0 {
			return nil, nil
		}
		return cmd, nil
	default:
		return nil, fmt.Errorf("check: expected string or list, got %T", raw)
	}
}

func (r *Renderer) validate(data []byte) error {
	if len(r.check) == 0 {
		return nil
	}

	tmpFile, err := os.CreateTemp("", "cfgkit-check-*")
	if err != nil {
		return fmt.Errorf("check: create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("check: write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("check: close temp file: %w", err)
	}

	tmplCtx := map[string]string{"TemplatePath": tmpFile.Name()}
	args := make([]string, len(r.check))
	for i, raw := range r.check {
		arg, err := renderCheckArg(raw, tmplCtx)
		if err != nil {
			return fmt.Errorf("check: render arg %d: %w", i, err)
		}
		args[i] = arg
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("check: timed out after 5s")
		}
		msg := stderr.String()
		if msg != "" {
			return fmt.Errorf("check: %s", msg)
		}
		return fmt.Errorf("check: %w", err)
	}
	return nil
}

func renderCheckArg(raw string, ctx map[string]string) (string, error) {
	if !strings.Contains(raw, "{{") {
		return raw, nil
	}
	tmpl, err := template.New("").Parse(raw)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", err
	}
	return buf.String(), nil
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
