package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Init []string `yaml:"init,omitempty"`
	Bind []string `yaml:"bind,omitempty"`
	Copy []string `yaml:"copy,omitempty"`
}

func Load() (*Config, error) {
	globalConfig, err := loadGlobalConfig()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	localConfig, err := loadLocalConfig()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load local config: %w", err)
	}

	return mergeConfigs(globalConfig, localConfig), nil
}

func loadGlobalConfig() (*Config, error) {
	configPath := filepath.Join(GetConfigDir(), "claudeway", "claudeway.yaml")
	return loadConfigFromFile(configPath)
}

func loadLocalConfig() (*Config, error) {
	return loadConfigFromFile("claudeway.yaml")
}

func loadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	return &config, nil
}

func mergeConfigs(global, local *Config) *Config {
	if global == nil && local == nil {
		return &Config{}
	}
	if global == nil {
		return local
	}
	if local == nil {
		return global
	}

	merged := &Config{}
	
	// Global config takes precedence for init commands
	if len(global.Init) > 0 {
		merged.Init = append(merged.Init, global.Init...)
	}
	if len(local.Init) > 0 {
		merged.Init = append(merged.Init, local.Init...)
	}

	// Merge bind directories
	bindMap := make(map[string]bool)
	for _, bind := range global.Bind {
		bindMap[bind] = true
		merged.Bind = append(merged.Bind, bind)
	}
	for _, bind := range local.Bind {
		if !bindMap[bind] {
			merged.Bind = append(merged.Bind, bind)
		}
	}

	// Merge copy files
	copyMap := make(map[string]bool)
	for _, copy := range global.Copy {
		copyMap[copy] = true
		merged.Copy = append(merged.Copy, copy)
	}
	for _, copy := range local.Copy {
		if !copyMap[copy] {
			merged.Copy = append(merged.Copy, copy)
		}
	}

	return merged
}

func CreateDefaultConfig(path string) error {
	defaultConfig := &Config{
		Init: []string{
			"# Example initialization commands",
			"# npm ci",
			"# go mod download",
		},
		Bind: []string{
			"# Example additional bind mounts",
			"# /opt/bin",
		},
		Copy: []string{
			"# Example files to copy",
			"# ~/.zshrc",
		},
	}

	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}