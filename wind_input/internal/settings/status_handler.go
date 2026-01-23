package settings

import (
	"net/http"
	"runtime"
	"time"
)

// StatusHandler 状态处理器
type StatusHandler struct {
	services  *Services
	startTime time.Time
}

// NewStatusHandler 创建状态处理器
func NewStatusHandler(services *Services) *StatusHandler {
	return &StatusHandler{
		services:  services,
		startTime: time.Now(),
	}
}

// StatusResponse 状态响应
type StatusResponse struct {
	Service ServiceStatus `json:"service"`
	Engine  EngineStatus  `json:"engine"`
	Memory  MemoryStatus  `json:"memory"`
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	UptimeSec int64  `json:"uptimeSec"`
}

// EngineStatus 引擎状态
type EngineStatus struct {
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
	Info        string `json:"info"`
	EntryCount  int    `json:"entryCount,omitempty"`
}

// MemoryStatus 内存状态
type MemoryStatus struct {
	Alloc      uint64 `json:"alloc"`      // 当前分配的内存（字节）
	TotalAlloc uint64 `json:"totalAlloc"` // 累计分配的内存（字节）
	Sys        uint64 `json:"sys"`        // 从系统获取的内存（字节）
	NumGC      uint32 `json:"numGC"`      // GC 次数
	AllocMB    string `json:"allocMB"`    // 当前分配（MB，易读）
	SysMB      string `json:"sysMB"`      // 系统内存（MB，易读）
}

// GetStatus 获取服务状态
func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime)

	// 获取内存统计
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 构建响应
	response := StatusResponse{
		Service: ServiceStatus{
			Name:      "WindInput IME Service",
			Version:   "1.0.0",
			Uptime:    formatDuration(uptime),
			UptimeSec: int64(uptime.Seconds()),
		},
		Memory: MemoryStatus{
			Alloc:      memStats.Alloc,
			TotalAlloc: memStats.TotalAlloc,
			Sys:        memStats.Sys,
			NumGC:      memStats.NumGC,
			AllocMB:    formatBytes(memStats.Alloc),
			SysMB:      formatBytes(memStats.Sys),
		},
	}

	// 获取引擎状态
	if h.services.EngineMgr != nil {
		engineType := h.services.EngineMgr.GetCurrentType()
		response.Engine = EngineStatus{
			Type:        string(engineType),
			DisplayName: h.services.EngineMgr.GetEngineDisplayName(),
			Info:        h.services.EngineMgr.GetEngineInfo(),
		}
	}

	WriteSuccess(w, response)
}

// formatDuration 格式化时长
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return formatInt(days) + "天" + formatInt(hours) + "小时" + formatInt(minutes) + "分钟"
	}
	if hours > 0 {
		return formatInt(hours) + "小时" + formatInt(minutes) + "分钟" + formatInt(seconds) + "秒"
	}
	if minutes > 0 {
		return formatInt(minutes) + "分钟" + formatInt(seconds) + "秒"
	}
	return formatInt(seconds) + "秒"
}

// formatInt 格式化整数
func formatInt(n int) string {
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

// formatBytes 格式化字节数为易读格式
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return formatUint64(bytes) + " B"
	}

	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	val := float64(bytes) / float64(div)
	units := []string{"KB", "MB", "GB", "TB"}

	// 简单格式化浮点数
	intPart := int(val)
	fracPart := int((val - float64(intPart)) * 10)

	return formatUint64(uint64(intPart)) + "." + string(rune('0'+fracPart)) + " " + units[exp]
}

// formatUint64 简单的 uint64 转字符串
func formatUint64(n uint64) string {
	if n == 0 {
		return "0"
	}

	var buf [20]byte
	i := len(buf)

	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}

	return string(buf[i:])
}
