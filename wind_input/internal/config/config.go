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
	General    GeneralConfig    `yaml:"general" json:"general"`
	Dictionary DictionaryConfig `yaml:"dictionary" json:"dictionary"`
	Engine     EngineConfig     `yaml:"engine" json:"engine"`
	Hotkeys    HotkeyConfig     `yaml:"hotkeys" json:"hotkeys"`
	UI         UIConfig         `yaml:"ui" json:"ui"`
	Toolbar    ToolbarConfig    `yaml:"toolbar" json:"toolbar"`
	Input      InputConfig      `yaml:"input" json:"input"`
}

// GeneralConfig contains general settings
type GeneralConfig struct {
	StartInChineseMode bool   `yaml:"start_in_chinese_mode" json:"start_in_chinese_mode"`
	LogLevel           string `yaml:"log_level" json:"log_level"`
}

// DictionaryConfig contains dictionary settings
type DictionaryConfig struct {
	SystemDict string `yaml:"system_dict" json:"system_dict"`
	UserDict   string `yaml:"user_dict" json:"user_dict"`
	PinyinDict string `yaml:"pinyin_dict" json:"pinyin_dict"` // 拼音词库（用于反查）
}

// EngineConfig 引擎配置
type EngineConfig struct {
	Type       string       `yaml:"type" json:"type"`               // pinyin / wubi
	FilterMode string       `yaml:"filter_mode" json:"filter_mode"` // general / gb18030 / smart
	Pinyin     PinyinConfig `yaml:"pinyin" json:"pinyin"`
	Wubi       WubiConfig   `yaml:"wubi" json:"wubi"`
}

// PinyinConfig 拼音引擎配置
type PinyinConfig struct {
	ShowWubiHint bool `yaml:"show_wubi_hint" json:"show_wubi_hint"` // 显示五笔编码提示（反查）
}

// WubiConfig 五笔引擎配置
type WubiConfig struct {
	AutoCommit    string `yaml:"auto_commit" json:"auto_commit"`         // none / unique / unique_at_4 / unique_full_match
	EmptyCode     string `yaml:"empty_code" json:"empty_code"`           // none / clear / clear_at_4 / to_english
	TopCodeCommit bool   `yaml:"top_code_commit" json:"top_code_commit"` // 五码顶字上屏
	PunctCommit   bool   `yaml:"punct_commit" json:"punct_commit"`       // 标点顶字上屏
}

// HotkeyConfig contains hotkey settings
type HotkeyConfig struct {
	ToggleMode      string `yaml:"toggle_mode" json:"toggle_mode"`             // "shift", "ctrl+space", etc.
	SwitchEngine    string `yaml:"switch_engine" json:"switch_engine"`         // "ctrl+`", "ctrl+shift+e", etc.
	ToggleFullWidth string `yaml:"toggle_full_width" json:"toggle_full_width"` // "shift+space", "ctrl+shift+space", "none"
	TogglePunct     string `yaml:"toggle_punct" json:"toggle_punct"`           // "ctrl+.", "ctrl+,", "none"
}

// UIConfig contains UI settings
type UIConfig struct {
	FontSize          float64 `yaml:"font_size" json:"font_size"`
	CandidatesPerPage int     `yaml:"candidates_per_page" json:"candidates_per_page"`
	FontPath          string  `yaml:"font_path" json:"font_path"`
	InlinePreedit     bool    `yaml:"inline_preedit" json:"inline_preedit"` // 启用嵌入式编码行
}

// ToolbarConfig contains toolbar settings
type ToolbarConfig struct {
	Visible   bool `yaml:"visible" json:"visible"`
	PositionX int  `yaml:"position_x" json:"position_x"`
	PositionY int  `yaml:"position_y" json:"position_y"`
}

// InputConfig contains input behavior settings
type InputConfig struct {
	FullWidth          bool   `yaml:"full_width" json:"full_width"`                       // 全角模式
	ChinesePunctuation bool   `yaml:"chinese_punctuation" json:"chinese_punctuation"`     // 中文标点
	PunctFollowMode    bool   `yaml:"punct_follow_mode" json:"punct_follow_mode"`         // 标点随中英文切换
	SelectKey2         string `yaml:"select_key_2" json:"select_key_2"`                   // 第2候选选择键: "semicolon"(;), "comma"(,), "lshift", "lctrl", "none"
	SelectKey3         string `yaml:"select_key_3" json:"select_key_3"`                   // 第3候选选择键: "quote"('), "period"(.), "rshift", "rctrl", "none"
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
			FilterMode: "smart",
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
			ToggleMode:      "shift",
			SwitchEngine:    "ctrl+`",
			ToggleFullWidth: "shift+space",   // 默认 Shift+空格 切换全半角
			TogglePunct:     "ctrl+.",        // 默认 Ctrl+. 切换中英文标点
		},
		UI: UIConfig{
			FontSize:          18,
			CandidatesPerPage: 9,
			FontPath:          "",
			InlinePreedit:     true, // 默认开启嵌入式编码行
		},
		Toolbar: ToolbarConfig{
			Visible:   false, // 默认不显示工具栏
			PositionX: 0,     // 0 表示使用自动计算位置（屏幕右下角）
			PositionY: 0,
		},
		Input: InputConfig{
			FullWidth:          false,      // 默认半角
			ChinesePunctuation: true,       // 默认中文标点
			PunctFollowMode:    false,      // 默认不跟随模式切换（可在设置中启用）
			SelectKey2:         "semicolon", // 默认分号选第2候选
			SelectKey3:         "quote",     // 默认引号选第3候选
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
