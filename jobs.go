package runner

import "time"

type Jobs []Job

type Job struct {
	Detail     string        `json:"detail"`
	Cmd        []string      `json:"cmd,omitempty"`
	Background *Background   `json:"background,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`
	RunTwice   bool          `json:"runTwice,omitempty"`
	SendOut    bool          `json:"sendOut,omitempty"`
	Base64     bool          `json:"base64,omitempty"`
	LogKey     string        `json:"logKey,omitempty"`
	URL        string        `json:"url,omitempty"`
	Save       string        `json:"save,omitempty"`
}

// Run this process in background, delaying by Delay before calling the main command
type Background struct {
	Delay time.Duration `json:"delay"`
	Cmd   []string      `json:"cmd"`
}

type Log struct {
	Label   string    `json:"label"`
	Batch   time.Time `json:"batch"`
	Path    string    `json:"path,omitempty"`
	Machine string    `json:"machine,omitempty"`
	Reports []Report  `json:"reports"`
}

type Report struct {
	Detail   string        `json:"detail"`
	Start    time.Time     `json:"start"`
	Duration time.Duration `json:"duration"`
	Output   string        `json:"output,omitempty"`
	Err      string        `json:"error,omitempty"`
}
