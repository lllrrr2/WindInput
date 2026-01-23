// Package config handles application configuration
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	AppName        = "WindInput"
	ConfigFileName = "config.yaml"
	UserDictFile   = "user_dict.txt"
)

// Config represents the application configuration
type Config struct {
	General    GeneralConfig    `yaml:"general"`
	Dictionary DictionaryConfig `yaml:"dictionary"`
	Engine     EngineConfig     `yaml:"engine"`
	Hotkeys    HotkeyConfig     `yaml:"hotkeys"`
	UI         UIConfig         `yaml:"ui"`
}

// GeneralConfig contains general settings
type GeneralConfig struct {
	StartInChineseMode bool   `yaml:"start_in_chinese_mode"`
	LogLevel           string `yaml:"log_level"`
}

// DictionaryConfig contains dictionary settings
type DictionaryConfig struct {
	SystemDict string `yaml:"system_dict"`
	UserDict   string `yaml:"user_dict"`
	PinyinDict string `yaml:"pinyin_dict"` // 拼音词库（用于反查）
}

// EngineConfig 引擎配置
type EngineConfig struct {
	Type   string       `yaml:"type"` // pinyin / wubi
	Pinyin PinyinConfig `yaml:"pinyin"`
	Wubi   WubiConfig   `yaml:"wubi"`
}

// PinyinConfig 拼音引擎配置
type PinyinConfig struct {
	ShowWubiHint bool `yaml:"show_wubi_hint"` // 显示五笔编码提示（反查）
}

// WubiConfig 五笔引擎配置
type WubiConfig struct {
	AutoCommit    string `yaml:"auto_commit"`     // none / unique / unique_at_4 / unique_full_match
	EmptyCode     string `yaml:"empty_code"`      // none / clear / clear_at_4 / to_english
	TopCodeCommit bool   `yaml:"top_code_commit"` // 五码顶字上屏
	PunctCommit   bool   `yaml:"punct_commit"`    // 标点顶字上屏
}

// HotkeyConfig contains hotkey settings
type HotkeyConfig struct {
	ToggleMode   string `yaml:"toggle_mode"`   // "shift", "ctrl+space", etc.
	SwitchEngine string `yaml:"switch_engine"` // "ctrl+`", "ctrl+shift+e", etc.
}

// UIConfig contains UI settings
type UIConfig struct {
	FontSize          float64 `yaml:"font_size"`
	CandidatesPerPage int     `yaml:"candidates_per_page"`
	FontPath          string  `yaml:"font_path"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			StartInChineseMode: true,
			LogLevel:           "info",
		},
		Dictionary: DictionaryConfig{
			SystemDict: "dict/pinyin/pinyin.txt",
			UserDict:   UserDictFile,
			PinyinDict: "dict/pinyin/pinyin.txt",
		},
		Engine: EngineConfig{
			Type: "pinyin",
			Pinyin: PinyinConfig{
				ShowWubiHint: true, // 默认显示五笔编码提示
			},
			Wubi: WubiConfig{
				AutoCommit:    "unique_at_4",
				EmptyCode:     "clear_at_4",
				TopCodeCommit: true,
				PunctCommit:   true,
			},
		},
		Hotkeys: HotkeyConfig{
			ToggleMode:   "shift",
			SwitchEngine: "ctrl+`",
		},
		UI: UIConfig{
			FontSize:          18,
			CandidatesPerPage: 9,
			FontPath:          "",
		},
	}
}

// GetConfigDir returns the configuration directory path
// On Windows: %APPDATA%\WindInput
func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}
	return filepath.Join(configDir, AppName), nil
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, ConfigFileName), nil
}

// GetUserDictPath returns the full path to the user dictionary
func GetUserDictPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, UserDictFile), nil
}

// EnsureConfigDir ensures the config directory exists
func EnsureConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(configDir, 0755)
}

// Load loads the configuration from file
// If the file doesn't exist, returns default configuration
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return DefaultConfig(), nil
		}
		return DefaultConfig(), fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return DefaultConfig(), fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// Save saves the configuration to file
func Save(config *Config) error {
	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SaveDefault saves the default configuration to file
func SaveDefault() error {
	return Save(DefaultConfig())
}

// UpdateEngineType updates the engine type in config and saves
func UpdateEngineType(engineType string) error {
	cfg, err := Load()
	if err != nil {
		cfg = DefaultConfig()
	}

	cfg.Engine.Type = engineType

	// Update system dict based on engine type
	switch engineType {
	case "wubi":
		cfg.Dictionary.SystemDict = "dict/wubi/wubi86.txt"
	case "pinyin":
		cfg.Dictionary.SystemDict = "dict/pinyin/pinyin.txt"
	}

	return Save(cfg)
}

// GetWubiDictPath returns the path to the wubi dictionary
func GetWubiDictPath() string {
	return "dict/wubi/wubi86.txt"
}

// GetPinyinDictPath returns the path to the pinyin dictionary
func GetPinyinDictPath() string {
	return "dict/pinyin/pinyin.txt"
}
