package chronicle

import "os"

// Renderer transforms a chronicle into output
type Renderer interface {
	Render(entry *Entry) ([]byte, error)
	RenderSuite(suite *SuiteChronicle) ([]byte, error)
}

// RenderTo writes rendered output to a file
func RenderTo(entry *Entry, renderer Renderer, path string) error {
	data, err := renderer.Render(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// RenderSuiteTo writes rendered suite output to a file
func RenderSuiteTo(suite *SuiteChronicle, renderer Renderer, path string) error {
	data, err := renderer.RenderSuite(suite)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
