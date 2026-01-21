// Package ui provides native Windows UI for candidate window
package ui

// Candidate represents a candidate word
type Candidate struct {
	Text    string
	Index   int
	Comment string
	Weight  int
}
