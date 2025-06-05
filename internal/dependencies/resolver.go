package dependencies

import (
	"os"
	"path/filepath"
	"strings"
)

// Resolver defines the interface for finding direct dependencies of a file.
type Resolver interface {
	// Resolve finds direct, project-local dependencies for the given file path.
	// fileContent is the content of the file to parse.
	// filePath is the path relative to the project root.
	// projectRoot is the absolute path to the project root.
	// projectModuleName is the go module name (if applicable, empty otherwise).
	// Returns a slice of dependency paths, also relative to the project root.
	Resolve(fileContent []byte, filePath string, projectRoot string, projectModuleName string) ([]string, error)
}

// GetResolver returns the appropriate resolver based on the file extension.
func GetResolver(filePath string) Resolver {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return &GoResolver{}
	case ".js", ".jsx", ".ts", ".tsx":
		return &JSResolver{}
	case ".py":
		return &PyResolver{}
	default:
		return nil
	}
}

// isProjectLocal checks if a resolved path is likely within the project boundaries.
func isProjectLocal(absPath, projectRoot string) bool {
	rel, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") && rel != "."
}

// fileExists checks if a file exists at the given absolute path.
func fileExists(absPath string) bool {
	info, err := os.Stat(absPath)
	return err == nil && !info.IsDir()
}

// normalizePath cleans and converts a path to slash format, relative to root.
func normalizePath(path, containingDir, projectRoot string) (string, bool) {
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(containingDir, path)
	}
	absPath = filepath.Clean(absPath)

	if !isProjectLocal(absPath, projectRoot) {
		return "", false
	}

	relPath, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		return "", false
	}

	return filepath.ToSlash(relPath), true
}
