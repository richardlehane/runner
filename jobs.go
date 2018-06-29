package runner

import "time"

type Jobs []Job

type Job struct {
	Detail   string   `json:"detail"`
	Cmd      []string `json:"cmd,omitempty"`
	RunTwice bool     `json:"runTwice"`
	SendOut  bool     `json:"sendOut"`
	LogKey   string   `json:"logKey,omitempty"`
	URL      string   `json:"url,omitempty"`
	Save     string   `json:"save,omitempty"`
}

type Log struct {
	Detail  string   `json:"detail"`
	Reports []Report `json:"reports"`
}

type Report struct {
	Detail   string        `json:"detail"`
	Start    time.Time     `json:"start"`
	Duration time.Duration `json:"duration"`
	Output   string        `json:"output,omitempty"`
	Err      string        `json:"error,omitempty"`
}
