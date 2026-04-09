package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Devices   map[string]DeviceConfig   `yaml:"devices"`
	Variables yaml.Node                 `yaml:"variables"`
	Templates map[string]TemplateConfig `yaml:"templates"`
}

type DeviceConfig struct {
	Password     string    `yaml:"password"`
	TemplateName string    `yaml:"template"`
	Variables    yaml.Node `yaml:"variables"`
}

type TemplateConfig struct {
	Type     string `yaml:"type"`
	File     string `yaml:"file"`
	Template string `yaml:"data"`
}

func Load(dir string) (*Config, error) {
	c := &Config{
		Devices:   map[string]DeviceConfig{},
		Templates: map[string]TemplateConfig{},
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %q: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		if err := loadFile(c, filepath.Join(dir, entry.Name())); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func loadFile(c *Config, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening file %q: %w", path, err)
	}
	defer f.Close()

	var file Config
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&file); err != nil {
		return fmt.Errorf("decoding yaml %q: %w", path, err)
	}

	for k, v := range file.Devices {
		c.Devices[k] = v
	}
	for k, v := range file.Templates {
		c.Templates[k] = v
	}
	c.Variables = mergeMapping(c.Variables, file.Variables)
	return nil
}

func mergeMapping(dst, src yaml.Node) yaml.Node {
	if src.Kind == 0 {
		return dst
	}
	if dst.Kind == 0 {
		return src
	}
	merged := yaml.Node{Kind: yaml.MappingNode, Tag: dst.Tag}
	indexByKey := map[string]int{}
	for i := 0; i+1 < len(dst.Content); i += 2 {
		indexByKey[dst.Content[i].Value] = len(merged.Content) + 1
		merged.Content = append(merged.Content, dst.Content[i], dst.Content[i+1])
	}
	for i := 0; i+1 < len(src.Content); i += 2 {
		key := src.Content[i].Value
		if valueIdx, ok := indexByKey[key]; ok {
			merged.Content[valueIdx] = src.Content[i+1]
			continue
		}
		indexByKey[key] = len(merged.Content) + 1
		merged.Content = append(merged.Content, src.Content[i], src.Content[i+1])
	}
	return merged
}
