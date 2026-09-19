package utils

type Panel int

const (
	SidebarPanel Panel = iota
	RequestPanel
	ResponsePanel
	// WSPanel is only a valid focus target while the WebSocket panel is
	// docked (see app.Model.wsDocked) - undocked, WebSocket only ever
	// shows as a floating modal, never as a focusable panel.
	WSPanel
)
