package theme

// DefaultTheme returns the built-in light theme
func DefaultTheme() *Theme {
	return &Theme{
		Meta: ThemeMeta{
			Name:    "默认主题",
			Version: "1.0",
			Author:  "清风输入法",
		},
		CandidateWindow: CandidateWindowColors{
			BackgroundColor: "#FFFFFF",
			BorderColor:     "#C8C8C8",
			TextColor:       "#1E1E1E",
			IndexColor:      "#FFFFFF",
			IndexBgColor:    "#4285F4", // Blue
			HoverBgColor:    "#E6F0FF", // Light blue
			InputBgColor:    "#F0F0F0",
			InputTextColor:  "#646464",
			CommentColor:    "#969696",
			ShadowColor:     "#0000000F", // 15/255 alpha
		},
		Toolbar: ToolbarColors{
			BackgroundColor:     "#FFFFFF",
			BorderColor:         "#C7D1E0", // Light blue-gray
			GripColor:           "#99ADC7B3",
			ModeChineseBgColor:  "#339AF5", // Bright blue
			ModeEnglishBgColor:  "#737F94", // Slate gray
			ModeTextColor:       "#FFFFFF",
			FullWidthOnBgColor:  "#2EB899", // Teal green
			FullWidthOffBgColor: "#E6EAEF", // Light cool gray
			FullWidthOnColor:    "#FFFFFF",
			FullWidthOffColor:   "#59667A",
			PunctChineseBgColor: "#F58543", // Coral orange
			PunctEnglishBgColor: "#E6EAEF",
			PunctChineseColor:   "#FFFFFF",
			PunctEnglishColor:   "#59667A",
			SettingsBgColor:     "#E6EAEF",
			SettingsIconColor:   "#7A66B8", // Muted purple
			SettingsHoleColor:   "#E6EAEF",
		},
		PopupMenu: PopupMenuColors{
			BackgroundColor: "#FFFFFF",
			BorderColor:     "#C7C7C7",
			TextColor:       "#000000",
			DisabledColor:   "#A1A1A1",
			HoverBgColor:    "#0078D4", // Windows blue
			HoverTextColor:  "#FFFFFF",
			SeparatorColor:  "#DBDBDB",
		},
		Tooltip: TooltipColors{
			BackgroundColor: "#3C3C3CF0",
			TextColor:       "#FFFFFF",
		},
		ModeIndicator: ModeIndicatorColors{
			BackgroundColor: "#323232E6",
			TextColor:       "#FFFFFF",
		},
	}
}

// DarkTheme returns the built-in dark theme
func DarkTheme() *Theme {
	return &Theme{
		Meta: ThemeMeta{
			Name:    "暗色主题",
			Version: "1.0",
			Author:  "清风输入法",
		},
		CandidateWindow: CandidateWindowColors{
			BackgroundColor: "#2D2D2D",
			BorderColor:     "#404040",
			TextColor:       "#E0E0E0",
			IndexColor:      "#FFFFFF",
			IndexBgColor:    "#4285F4",
			HoverBgColor:    "#3D4A5C",
			InputBgColor:    "#3A3A3A",
			InputTextColor:  "#B0B0B0",
			CommentColor:    "#808080",
			ShadowColor:     "#0000001A",
		},
		Toolbar: ToolbarColors{
			BackgroundColor:     "#2D2D2D",
			BorderColor:         "#404040",
			GripColor:           "#5A5A5AB3",
			ModeChineseBgColor:  "#339AF5",
			ModeEnglishBgColor:  "#5A5A5A",
			ModeTextColor:       "#FFFFFF",
			FullWidthOnBgColor:  "#2EB899",
			FullWidthOffBgColor: "#404040",
			FullWidthOnColor:    "#FFFFFF",
			FullWidthOffColor:   "#B0B0B0",
			PunctChineseBgColor: "#F58543",
			PunctEnglishBgColor: "#404040",
			PunctChineseColor:   "#FFFFFF",
			PunctEnglishColor:   "#B0B0B0",
			SettingsBgColor:     "#404040",
			SettingsIconColor:   "#9B8CCE",
			SettingsHoleColor:   "#404040",
		},
		PopupMenu: PopupMenuColors{
			BackgroundColor: "#2D2D2D",
			BorderColor:     "#404040",
			TextColor:       "#E0E0E0",
			DisabledColor:   "#707070",
			HoverBgColor:    "#0078D4",
			HoverTextColor:  "#FFFFFF",
			SeparatorColor:  "#404040",
		},
		Tooltip: TooltipColors{
			BackgroundColor: "#1E1E1EF0",
			TextColor:       "#E0E0E0",
		},
		ModeIndicator: ModeIndicatorColors{
			BackgroundColor: "#1E1E1EE6",
			TextColor:       "#E0E0E0",
		},
	}
}

// MSIMETheme returns a Microsoft IME style theme (Windows 11 Fluent Design)
func MSIMETheme() *Theme {
	return &Theme{
		Meta: ThemeMeta{
			Name:    "微软风格",
			Version: "1.0",
			Author:  "清风输入法",
		},
		CandidateWindow: CandidateWindowColors{
			BackgroundColor: "#FFFFFF",
			BorderColor:     "#E5E5E5",
			TextColor:       "#1A1A1A",
			IndexColor:      "#888888", // Gray index text in "text" mode
			IndexBgColor:    "#0078D4", // Microsoft blue (used for arrows etc.)
			HoverBgColor:    "#EBF0FE",
			InputBgColor:    "#F5F5F5",
			InputTextColor:  "#666666",
			CommentColor:    "#999999",
			ShadowColor:     "#00000008",
		},
		Style: CandidateWindowStyle{
			IndexStyle:     "text",
			AccentBarColor: "#0078D4", // Microsoft blue accent bar
		},
		Toolbar: ToolbarColors{
			BackgroundColor:     "#FFFFFF",
			BorderColor:         "#E0E0E0",
			GripColor:           "#C0C0C0B3",
			ModeChineseBgColor:  "#0078D4",
			ModeEnglishBgColor:  "#8A8A8A",
			ModeTextColor:       "#FFFFFF",
			FullWidthOnBgColor:  "#0078D4",
			FullWidthOffBgColor: "#F0F0F0",
			FullWidthOnColor:    "#FFFFFF",
			FullWidthOffColor:   "#666666",
			PunctChineseBgColor: "#0078D4",
			PunctEnglishBgColor: "#F0F0F0",
			PunctChineseColor:   "#FFFFFF",
			PunctEnglishColor:   "#666666",
			SettingsBgColor:     "#F0F0F0",
			SettingsIconColor:   "#666666",
			SettingsHoleColor:   "#F0F0F0",
		},
		PopupMenu: PopupMenuColors{
			BackgroundColor: "#FFFFFF",
			BorderColor:     "#E0E0E0",
			TextColor:       "#1A1A1A",
			DisabledColor:   "#AAAAAA",
			HoverBgColor:    "#EBF0FE",
			HoverTextColor:  "#1A1A1A",
			SeparatorColor:  "#E5E5E5",
		},
		Tooltip: TooltipColors{
			BackgroundColor: "#2D2D2DF0",
			TextColor:       "#FFFFFF",
		},
		ModeIndicator: ModeIndicatorColors{
			BackgroundColor: "#2D2D2DE6",
			TextColor:       "#FFFFFF",
		},
	}
}

// GetBuiltinTheme returns a built-in theme by name
func GetBuiltinTheme(name string) *Theme {
	switch name {
	case "dark", "暗色主题":
		return DarkTheme()
	case "msime", "微软风格":
		return MSIMETheme()
	default:
		return DefaultTheme()
	}
}
