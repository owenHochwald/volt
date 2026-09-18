package design

import "time"

const (
	StartupFrameInterval = 100 * time.Millisecond
	StartupMaxDuration   = 750 * time.Millisecond
	RequestFrameInterval = 100 * time.Millisecond
	LoadTestRefresh      = 200 * time.Millisecond
)

var ActivityFrames = [...]string{
	"ϟ····",
	"·ϟ···",
	"··ϟ··",
	"···ϟ·",
	"····ϟ",
}
