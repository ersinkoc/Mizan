package ir

import "encoding/json"

func CanonicalJSON(m *Model) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}
