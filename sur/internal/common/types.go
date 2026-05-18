// Package common holds shared types used across sur subsystems.
package common

// Severity classifies how serious a finding is.
type Severity string

const (
	SeverityInfo Severity = "info"
	SeverityLow  Severity = "low"
	SeverityMed  Severity = "medium"
	SeverityHigh Severity = "high"
)

// Status of a check result.
type Status string

const (
	StatusPass Status = "PASS"
	StatusWarn Status = "WARN"
	StatusFail Status = "FAIL"
	StatusSkip Status = "SKIP"
)

// Finding is a single result from a check.
type Finding struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	Status      Status   `json:"status"`
	Severity    Severity `json:"severity"`
	Detail      string   `json:"detail,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
	Source      string   `json:"source"` // "builtin" | "lynis"
}

// Report bundles findings with a derived score.
type Report struct {
	Findings []Finding `json:"findings"`
	Score    int       `json:"score"`
	MaxScore int       `json:"max_score"`
	Issues   int       `json:"issues"`
}
