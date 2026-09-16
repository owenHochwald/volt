package ui

import tea "github.com/charmbracelet/bubbletea"

type Focusable interface {
	Focus() tea.Cmd
	Blur()
}

type FocusManager struct {
	currentIndex int
	components   []Focusable
}

func (f *FocusManager) Next() {
	f.components[f.currentIndex].Blur()
	f.currentIndex = (f.currentIndex + 1) % len(f.components)
	f.components[f.currentIndex].Focus()
}

func (f *FocusManager) Prev() {
	f.components[f.currentIndex].Blur()
	f.currentIndex--
	if f.currentIndex < 0 {
		f.currentIndex = len(f.components) - 1
	}
	f.components[f.currentIndex].Focus()
}

func (f *FocusManager) CurrentIndex() int {
	return f.currentIndex
}

// FocusIndex moves focus directly to the component at idx, blurring
// whatever's currently focused first - for jumping straight to a specific
// field (e.g. the first one that failed validation on a blocked send)
// rather than stepping through it via Next/Prev. A no-op if idx is out of
// range or already current.
func (f *FocusManager) FocusIndex(idx int) {
	if idx < 0 || idx >= len(f.components) || idx == f.currentIndex {
		return
	}
	f.components[f.currentIndex].Blur()
	f.currentIndex = idx
	f.components[f.currentIndex].Focus()
}

func (f *FocusManager) Current() Focusable {
	return f.components[f.currentIndex]
}

func NewFocusManager(components []Focusable) *FocusManager {
	fm := &FocusManager{
		currentIndex: 0,
		components:   components,
	}

	if len(fm.components) > 0 {
		components[0].Focus()
	}

	return fm
}

// NewFocusManagerWithIndex creates a FocusManager starting at a specific index
func NewFocusManagerWithIndex(components []Focusable, index int) *FocusManager {
	if index < 0 || index >= len(components) {
		index = 0
	}

	fm := &FocusManager{
		currentIndex: index,
		components:   components,
	}

	if len(fm.components) > 0 {
		components[index].Focus()
	}

	return fm
}
