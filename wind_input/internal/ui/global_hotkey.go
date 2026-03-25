package ui

import "log/slog"

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
		return GlobalHotkeyEntry{}, false
	}

	return GlobalHotkeyEntry{ID: id, Modifiers: mods, VK: vk, Command: command}, true
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
