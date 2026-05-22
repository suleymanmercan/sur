// Package stack defines the core data types for the sur stack catalog.
package stack

// StackMeta is a lightweight entry from index.yaml.
type StackMeta struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Index is the top-level structure of catalog/stacks/index.yaml.
type Index struct {
	Stacks []StackMeta `yaml:"stacks"`
}

// ConfigFieldType enumerates the supported TUI input types.
type ConfigFieldType string

const (
	FieldTypeText   ConfigFieldType = "text"
	FieldTypeNumber ConfigFieldType = "number"
	FieldTypeSelect ConfigFieldType = "select"
	FieldTypeBool   ConfigFieldType = "bool"
	FieldTypeSecret ConfigFieldType = "secret"
)

// ConfigField describes one configurable parameter in a stack.
type ConfigField struct {
	ID       string          `yaml:"id"`
	Label    string          `yaml:"label"`
	Type     ConfigFieldType `yaml:"type"`
	Default  string          `yaml:"default"`
	Options  []string        `yaml:"options,omitempty"`
	Generate bool            `yaml:"generate,omitempty"` // only for secret type
}

// StackDef is the full parsed stack.yaml.
type StackDef struct {
	ID          string        `yaml:"id"`
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	RiskLevel   string        `yaml:"risk_level"`
	Config      []ConfigField `yaml:"config"`
	// Source is set at runtime: "official" or "custom"
	Source string `yaml:"-"`
}

// InstalledStack represents a stack that has been deployed on this machine.
type InstalledStack struct {
	Def     StackDef
	Dir     string // /opt/sur/stacks/<id>
	Running bool   // true when docker compose ps shows at least one running container
}

// ContainerStatus holds a parsed row from `docker compose ps`.
type ContainerStatus struct {
	Name   string
	State  string // "running", "exited", etc.
	Status string // human-readable
}
