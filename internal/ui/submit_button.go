package ui

import tea "charm.land/bubbletea/v2"

type SubmitButton struct {
	focused bool
}

func NewSubmitButton() *SubmitButton {
	return &SubmitButton{focused: false}
}

func (s *SubmitButton) Focus() tea.Cmd {
	s.focused = true
	return nil
}

func (s *SubmitButton) Blur() {
	s.focused = false
}

func (s *SubmitButton) IsFocused() bool {
	return s.focused
}
