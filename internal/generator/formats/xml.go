package formats

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/epilande/codegrab/internal/generator"
	"github.com/epilande/codegrab/internal/utils"
)

// XMLFormat implements the generator.Format interface for XML output
type XMLFormat struct{}

// XMLProject represents the root XML element
type XMLProject struct {
	XMLName    xml.Name      `xml:"project"`
	Filesystem XMLFilesystem `xml:"filesystem"`
	Files      []XMLFile     `xml:"files>file"`
}

// XMLFilesystem represents the directory structure
type XMLFilesystem struct {
	Root XMLDirectory `xml:"directory"`
}

// XMLDirectory represents a directory in the structure
type XMLDirectory struct {
	Name        string         `xml:"name,attr"`
	Directories []XMLDirectory `xml:"directory,omitempty"`
	Files       []XMLFileRef   `xml:"file,omitempty"`
}

// XMLFileRef represents a file reference in the directory structure
type XMLFileRef struct {
	Name string `xml:"name,attr"`
}

// XMLFile represents a file with its content
type XMLFile struct {
	Path     string `xml:"path,attr"`
	Language string `xml:"language,attr"`
	Content  string `xml:",cdata"`
}

// directoryEntry represents a directory in our internal tree structure
type directoryEntry struct {
	name    string
	subdirs map[string]*directoryEntry
	files   []string
}

// Render converts the template data into XML format
func (f *XMLFormat) Render(data generator.TemplateData) (string, int, error) {
	// Create the root directory
	root := &directoryEntry{
		name:    ".",
		subdirs: make(map[string]*directoryEntry),
		files:   []string{},
	}

	for _, path := range data.FilePaths {
		addFileToTree(root, path)
	}

	// Convert our internal tree to the XML structure
	xmlRoot := convertToXMLDirectory(root)

	// Create the XML project
	xmlProject := XMLProject{
		Filesystem: XMLFilesystem{
			Root: xmlRoot,
		},
		Files: make([]XMLFile, len(data.Files)),
	}

	for i, file := range data.Files {
		xmlProject.Files[i] = XMLFile{
			Path:     file.Path,
			Language: file.Language,
			Content:  file.Content,
		}
	}

	output, err := xml.MarshalIndent(xmlProject, "", "  ")
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal XML: %w", err)
	}

	xmlContent := xml.Header + string(output)
	tokenCount := utils.EstimateTokens(xmlContent)
	return xmlContent, tokenCount, nil
}

// addFileToTree adds a file path to our directory tree
func addFileToTree(root *directoryEntry, path string) {
	// Split the path into directory components and filename
	dir, file := filepath.Split(path)

	// Remove trailing slash from directory
	dir = strings.TrimSuffix(dir, "/")

	// If the directory is empty, the file is in the root
	if dir == "" {
		root.files = append(root.files, file)
		return
	}

	// Split the directory path into components
	components := strings.Split(dir, "/")

	// Start at the root directory
	current := root

	// Process each directory component
	for _, component := range components {
		// Skip empty components
		if component == "" {
			continue
		}

		// Check if we've already created this directory
		if subdir, exists := current.subdirs[component]; exists {
			current = subdir
		} else {
			// Create a new directory
			newDir := &directoryEntry{
				name:    component,
				subdirs: make(map[string]*directoryEntry),
				files:   []string{},
			}

			// Add it to the current directory's children
			current.subdirs[component] = newDir

			// Update current directory
			current = newDir
		}
	}

	// Add the file to the final directory
	current.files = append(current.files, file)
}

// convertToXMLDirectory converts our internal directory structure to the XML structure
func convertToXMLDirectory(dir *directoryEntry) XMLDirectory {
	xmlDir := XMLDirectory{
		Name:        dir.name,
		Directories: []XMLDirectory{},
		Files:       []XMLFileRef{},
	}

	// Sort subdirectory names for consistent output
	var subdirNames []string
	for name := range dir.subdirs {
		subdirNames = append(subdirNames, name)
	}
	sort.Strings(subdirNames)

	// Add subdirectories
	for _, name := range subdirNames {
		subdir := dir.subdirs[name]
		xmlDir.Directories = append(xmlDir.Directories, convertToXMLDirectory(subdir))
	}

	// Sort file names for consistent output
	sort.Strings(dir.files)

	// Add files
	for _, file := range dir.files {
		xmlDir.Files = append(xmlDir.Files, XMLFileRef{Name: file})
	}

	return xmlDir
}

// Extension returns the file extension for XML
func (f *XMLFormat) Extension() string {
	return ".xml"
}

// Name returns the name of the format
func (f *XMLFormat) Name() string {
	return "xml"
}
