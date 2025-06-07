package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type Config struct {
	Devices   DevicesConfig             `yaml:"devices"`
	Variables VariablesConfig           `yaml:"variables"`
	Templates map[string]TemplateConfig `yaml:"templates"`
}

type DevicesConfig map[string]DeviceConfig

type DeviceConfig struct {
	Password     string          `yaml:"password"`
	TemplateName string          `yaml:"template"`
	Variables    VariablesConfig `yaml:"variables"`
}

type VariablesConfig map[string]any

type TemplateConfig struct {
	Type     string `yaml:"type"`
	Template string `yaml:"data"`
}

func Load(dir string) (*Config, error) {
	c := &Config{}
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
		path := filepath.Join(dir, entry.Name())
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening file %q: %w", path, err)
		}
		dec := yaml.NewDecoder(f)
		dec.KnownFields(true)
		if err := dec.Decode(c); err != nil {
			f.Close()
			return nil, fmt.Errorf("decoding yaml %q: %w", path, err)
		}
		f.Close()
	}
	return c, nil
}
