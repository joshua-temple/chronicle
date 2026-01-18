package chronicle

import "encoding/json"

// JSONRenderer outputs machine-parseable JSON
type JSONRenderer struct {
	Indent bool
}

// Render renders an entry to JSON
func (r *JSONRenderer) Render(entry *Entry) ([]byte, error) {
	if r.Indent {
		return json.MarshalIndent(entry, "", "  ")
	}
	return json.Marshal(entry)
}

// RenderSuite renders a suite chronicle to JSON
func (r *JSONRenderer) RenderSuite(suite *SuiteChronicle) ([]byte, error) {
	if r.Indent {
		return json.MarshalIndent(suite, "", "  ")
	}
	return json.Marshal(suite)
}
