package parser

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/mizanproxy/mizan/internal/ir"
)

func ParseFile(name string, data []byte) (*ir.Model, error) {
	ext := strings.ToLower(filepath.Ext(name))
	text := string(data)
	switch {
	case ext == ".cfg" && strings.Contains(text, "frontend "):
		return ParseHAProxy(text)
	case ext == ".conf", strings.Contains(text, "http {"), strings.Contains(text, "upstream "):
		return ParseNginx(text)
	case strings.Contains(text, "frontend ") || strings.Contains(text, "backend "):
		return ParseHAProxy(text)
	default:
		return nil, errors.New("could not detect config format")
	}
}

func normalizeID(prefix, name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer(" ", "_", "-", "_", ".", "_", ":", "_", "/", "_")
	name = replacer.Replace(name)
	name = strings.Trim(name, "_")
	if name == "" {
		name = "default"
	}
	return prefix + "_" + name
}
