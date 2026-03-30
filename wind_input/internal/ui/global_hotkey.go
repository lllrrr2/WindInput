package ui

import (
	"log/slog"
	"strings"
)

var (
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
)

// Windows RegisterHotKey modifier constants
const (
	hotkeyModAlt      = 0x0001
	hotkeyModControl  = 0x0002
	hotkeyModShift    = 0x0004
	hotkeyModNoRepeat = 0x4000

	wmHotkey = 0x0312
)

// GlobalHotkeyEntry defines a global hotkey to register
type GlobalHotkeyEntry struct {
	ID        int    // Unique ID (1-based)
	Modifiers uint32 // hotkeyModControl, hotkeyModShift, etc.
	VK        uint32 // Virtual key code
	Command   string // Command name for callback dispatch
}

// ParseHotkeyString parses a hotkey config string (e.g., "ctrl+`") into a GlobalHotkeyEntry.
// Returns ok=false if the string is empty, "none", or unrecognized.
func ParseHotkeyString(s string, id int, command string) (GlobalHotkeyEntry, bool) {
	if s == "" || s == "none" {
		return GlobalHotkeyEntry{}, false
	}

	var mods uint32
	var vk uint32

	switch s {
	case "ctrl+`":
		mods = hotkeyModControl
		vk = 0xC0 // VK_OEM_3
	case "shift+space":
		mods = hotkeyModShift
		vk = 0x20 // VK_SPACE
	case "ctrl+.":
		mods = hotkeyModControl
		vk = 0xBE // VK_OEM_PERIOD
	case "ctrl+,":
		mods = hotkeyModControl
		vk = 0xBC // VK_OEM_COMMA
	case "ctrl+shift+e":
		mods = hotkeyModControl | hotkeyModShift
		vk = 0x45 // 'E'
	case "ctrl+shift+space":
		mods = hotkeyModControl | hotkeyModShift
		vk = 0x20 // VK_SPACE
	default:
		// Generic parser: split by "+" and resolve modifiers + key
		parts := strings.Split(strings.ToLower(s), "+")
		for i, part := range parts {
			switch part {
			case "ctrl":
				mods |= hotkeyModControl
			case "shift":
				mods |= hotkeyModShift
			case "alt":
				mods |= hotkeyModAlt
			default:
				if i == len(parts)-1 {
					vk = resolveVK(part)
				}
			}
		}
		if vk == 0 {
			return GlobalHotkeyEntry{}, false
		}
	}

	return GlobalHotkeyEntry{ID: id, Modifiers: mods, VK: vk, Command: command}, true
}

// resolveVK converts a lowercase key name string to a Windows virtual key code (uint32).
// Returns 0 if the name is not recognized.
func resolveVK(name string) uint32 {
	// Single letter a-z → 0x41-0x5A
	if len(name) == 1 {
		ch := name[0]
		if ch >= 'a' && ch <= 'z' {
			return uint32(ch-'a') + 0x41
		}
		// Digit 0-9 → 0x30-0x39
		if ch >= '0' && ch <= '9' {
			return uint32(ch-'0') + 0x30
		}
	}

	// F1-F12 → 0x70-0x7B
	if len(name) >= 2 && name[0] == 'f' {
		rest := name[1:]
		num := uint32(0)
		valid := true
		for _, c := range rest {
			if c < '0' || c > '9' {
				valid = false
				break
			}
			num = num*10 + uint32(c-'0')
		}
		if valid && num >= 1 && num <= 12 {
			return 0x70 + num - 1
		}
	}

	// Special keys
	switch name {
	case "`":
		return 0xC0 // VK_OEM_3
	case "space":
		return 0x20 // VK_SPACE
	case ".":
		return 0xBE // VK_OEM_PERIOD
	case ",":
		return 0xBC // VK_OEM_COMMA
	case ";":
		return 0xBA // VK_OEM_1
	case "'":
		return 0xDE // VK_OEM_7
	case "/":
		return 0xBF // VK_OEM_2
	case "\\":
		return 0xDC // VK_OEM_5
	case "[":
		return 0xDB // VK_OEM_4
	case "]":
		return 0xDD // VK_OEM_6
	case "-":
		return 0xBD // VK_OEM_MINUS
	case "=":
		return 0xBB // VK_OEM_PLUS
	case "tab":
		return 0x09
	case "escape", "esc":
		return 0x1B
	}
	return 0
}

// globalHotkeyState tracks registered hotkeys on the UI thread
type globalHotkeyState struct {
	entries  []GlobalHotkeyEntry
	callback func(command string)
	logger   *slog.Logger
}

func (s *globalHotkeyState) register(entries []GlobalHotkeyEntry) {
	// Unregister any previously registered hotkeys first
	s.unregister()

	for _, e := range entries {
		ret, _, err := procRegisterHotKey.Call(
			0, // NULL hwnd = thread-level hotkey
			uintptr(e.ID),
			uintptr(e.Modifiers|hotkeyModNoRepeat),
			uintptr(e.VK),
		)
		if ret == 0 {
			if s.logger != nil {
				s.logger.Warn("Failed to register global hotkey",
					"command", e.Command, "id", e.ID, "error", err)
			}
		} else {
			if s.logger != nil {
				s.logger.Debug("Registered global hotkey",
					"command", e.Command, "id", e.ID)
			}
		}
	}
	s.entries = entries
}

func (s *globalHotkeyState) unregister() {
	for _, e := range s.entries {
		procUnregisterHotKey.Call(0, uintptr(e.ID))
	}
	if len(s.entries) > 0 && s.logger != nil {
		s.logger.Debug("Unregistered all global hotkeys", "count", len(s.entries))
	}
	s.entries = nil
}

func (s *globalHotkeyState) handleWMHotkey(id int) {
	if s.callback == nil {
		return
	}
	for _, e := range s.entries {
		if e.ID == id {
			go s.callback(e.Command)
			return
		}
	}
}
