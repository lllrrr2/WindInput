package ui

// DPI constants
const (
	DefaultDPI = 96
	LOGPIXELSX = 88
	LOGPIXELSY = 90
)

// GetSystemDPI returns the system DPI
func GetSystemDPI() int {
	// Try Windows 10 1607+ API first
	if procGetDpiForSystem.Find() == nil {
		ret, _, _ := procGetDpiForSystem.Call()
		if ret != 0 {
			return int(ret)
		}
	}

	// Fallback: Use GetDeviceCaps with screen DC
	hdcScreen, _, _ := procGetDC.Call(0)
	if hdcScreen != 0 {
		defer procReleaseDC.Call(0, hdcScreen)
		dpi, _, _ := procGetDeviceCaps.Call(hdcScreen, LOGPIXELSX)
		if dpi != 0 {
			return int(dpi)
		}
	}

	return DefaultDPI
}

// GetDPIScale returns the DPI scale factor (1.0 = 100%, 1.5 = 150%, etc.)
func GetDPIScale() float64 {
	dpi := GetSystemDPI()
	return float64(dpi) / float64(DefaultDPI)
}

// ScaleForDPI scales a value according to the current DPI
func ScaleForDPI(value float64) float64 {
	return value * GetDPIScale()
}

// ScaleIntForDPI scales an integer value according to the current DPI
func ScaleIntForDPI(value int) int {
	return int(float64(value) * GetDPIScale())
}
