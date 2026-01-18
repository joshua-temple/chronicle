package chronicle

import "gopkg.in/yaml.v3"

// YAMLRenderer outputs human-readable YAML
type YAMLRenderer struct{}

// Render renders an entry to YAML
func (r *YAMLRenderer) Render(entry *Entry) ([]byte, error) {
	return yaml.Marshal(entry)
}

// RenderSuite renders a suite chronicle to YAML
func (r *YAMLRenderer) RenderSuite(suite *SuiteChronicle) ([]byte, error) {
	return yaml.Marshal(suite)
}
