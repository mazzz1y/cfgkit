package renderer

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Resolver struct {
	funcMap template.FuncMap
	device  map[string]any
	global  map[string]any
	ctx     map[string]any
}

func NewResolver(funcMap template.FuncMap) *Resolver {
	device := map[string]any{}
	global := map[string]any{}
	return &Resolver{
		funcMap: funcMap,
		device:  device,
		global:  global,
		ctx:     map[string]any{"Device": device, "Global": global},
	}
}

func (r *Resolver) Context() map[string]any {
	return r.ctx
}

func (r *Resolver) ResolveDevice(name string, node yaml.Node) error {
	if err := r.fillMap(r.device, &node, "Device"); err != nil {
		return err
	}
	r.device["Name"] = name
	return nil
}

func (r *Resolver) ResolveGlobal(node yaml.Node) error {
	return r.fillMap(r.global, &node, "Global")
}

func (r *Resolver) fillMap(dst map[string]any, node *yaml.Node, path string) error {
	mapping := unwrapDocument(node)
	if mapping == nil || mapping.Kind == 0 {
		return nil
	}
	if mapping.Kind != yaml.MappingNode {
		return fmt.Errorf("resolve %q: expected mapping node", path)
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i].Value
		childPath := path + "." + key
		child := unwrapDocument(mapping.Content[i+1])
		if child.Kind == yaml.MappingNode {
			nested := map[string]any{}
			dst[key] = nested
			if err := r.fillMap(nested, child, childPath); err != nil {
				return err
			}
			continue
		}
		value, err := r.resolveNode(child, childPath)
		if err != nil {
			return err
		}
		dst[key] = value
	}
	return nil
}

func (r *Resolver) resolveNode(node *yaml.Node, path string) (any, error) {
	node = unwrapDocument(node)
	if node == nil {
		return nil, nil
	}
	switch node.Kind {
	case yaml.MappingNode:
		nested := map[string]any{}
		if err := r.fillMap(nested, node, path); err != nil {
			return nil, err
		}
		return nested, nil
	case yaml.SequenceNode:
		return r.resolveSequence(node, path)
	case yaml.ScalarNode:
		return r.resolveScalar(node, path)
	case yaml.AliasNode:
		return r.resolveNode(node.Alias, path)
	}
	return nil, fmt.Errorf("resolve %q: unsupported yaml node kind", path)
}

func (r *Resolver) resolveSequence(node *yaml.Node, path string) ([]any, error) {
	result := make([]any, 0, len(node.Content))
	for i, child := range node.Content {
		value, err := r.resolveNode(child, fmt.Sprintf("%s[%d]", path, i))
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (r *Resolver) resolveScalar(node *yaml.Node, path string) (any, error) {
	var value any
	if err := node.Decode(&value); err != nil {
		return nil, fmt.Errorf("resolve %q: %w", path, err)
	}
	s, ok := value.(string)
	if !ok {
		return value, nil
	}
	return r.renderString(s, path)
}

func (r *Resolver) renderString(s, path string) (string, error) {
	if !strings.Contains(s, "{{") {
		return s, nil
	}
	tmpl, err := template.New(path).Funcs(r.funcMap).Parse(s)
	if err != nil {
		return "", fmt.Errorf("parse %q: %w", path, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r.ctx); err != nil {
		return "", fmt.Errorf("render %q: %w", path, err)
	}
	return buf.String(), nil
}

func unwrapDocument(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		node = node.Content[0]
	}
	return node
}
