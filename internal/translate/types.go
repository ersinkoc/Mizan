package translate

import "github.com/mizanproxy/mizan/internal/ir"

type SourceMapEntry struct {
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	EntityID  string `json:"entity_id"`
}

type Warning struct {
	EntityID string `json:"entity_id,omitempty"`
	Message  string `json:"message"`
}

type Result struct {
	Target    ir.Engine        `json:"target"`
	Config    string           `json:"config"`
	SourceMap []SourceMapEntry `json:"source_map"`
	Warnings  []Warning        `json:"warnings"`
}
