package ui

import (
	"log/slog"
	"sync"
	"time"
)

// Manager manages the candidate window UI
type Manager struct {
	window   *CandidateWindow
	renderer *Renderer
	logger   *slog.Logger

	mu         sync.Mutex
	candidates []Candidate
	input      string
	page       int
	totalPages int
	caretX     int
	caretY     int

	ready    bool
	readyCh  chan struct{}
}

// NewManager creates a new UI manager
func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		window:   NewCandidateWindow(logger),
		renderer: NewRenderer(DefaultRenderConfig()),
		logger:   logger,
		readyCh:  make(chan struct{}),
	}
}

// Start starts the UI manager (creates window and runs message loop)
// This should be called from a dedicated goroutine
func (m *Manager) Start() error {
	m.logger.Info("Starting UI Manager...")

	// Create window
	if err := m.window.Create(); err != nil {
		return err
	}

	m.mu.Lock()
	m.ready = true
	m.mu.Unlock()
	close(m.readyCh)

	m.logger.Info("UI Manager ready")

	// Run message loop (blocking)
	m.window.Run()

	return nil
}

// WaitReady waits until the UI manager is ready
func (m *Manager) WaitReady() {
	<-m.readyCh
}

// IsReady returns whether the UI manager is ready
func (m *Manager) IsReady() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ready
}

// ShowCandidates shows candidates at the given position
func (m *Manager) ShowCandidates(candidates []Candidate, input string, x, y, page, totalPages int) error {
	m.mu.Lock()
	if !m.ready {
		m.mu.Unlock()
		return nil
	}
	m.candidates = candidates
	m.input = input
	m.page = page
	m.totalPages = totalPages
	m.caretX = x
	m.caretY = y
	m.mu.Unlock()

	m.logger.Info("ShowCandidates", "input", input, "count", len(candidates), "x", x, "y", y)

	// Render
	img := m.renderer.RenderCandidates(candidates, input, page, totalPages)

	// Update window
	if err := m.window.UpdateContent(img, x, y); err != nil {
		m.logger.Error("UpdateContent failed", "error", err)
		return err
	}

	// Show window
	m.window.Show()

	return nil
}

// Hide hides the candidate window
func (m *Manager) Hide() {
	m.logger.Info("Hide called")
	m.window.Hide()
}

// UpdatePosition updates the window position
func (m *Manager) UpdatePosition(x, y int) {
	m.mu.Lock()
	m.caretX = x
	m.caretY = y
	m.mu.Unlock()

	m.window.SetPosition(x, y)
}

// Destroy destroys the UI manager
func (m *Manager) Destroy() {
	m.window.Destroy()
}

// IsVisible returns whether the window is visible
func (m *Manager) IsVisible() bool {
	return m.window.IsVisible()
}

// ShowModeIndicator shows a brief mode indicator (中/En)
func (m *Manager) ShowModeIndicator(mode string, x, y int) {
	m.mu.Lock()
	if !m.ready {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	m.logger.Info("ShowModeIndicator", "mode", mode)

	// Render mode indicator
	img := m.renderer.RenderModeIndicator(mode)

	// Update window
	if err := m.window.UpdateContent(img, x, y); err != nil {
		m.logger.Error("UpdateContent failed", "error", err)
		return
	}

	// Show window briefly
	m.window.Show()

	// Hide after a short delay (in a goroutine)
	go func() {
		time.Sleep(800 * time.Millisecond)
		m.window.Hide()
	}()
}
