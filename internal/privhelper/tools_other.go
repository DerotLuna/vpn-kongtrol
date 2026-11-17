//go:build !(linux || darwin)

package privhelper

// toolsManifestPath is empty: Windows tools are found in Program Files.
func toolsManifestPath() string { return "" }
