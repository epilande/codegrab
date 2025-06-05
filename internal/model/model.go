package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	"github.com/epilande/codegrab/internal/dependencies"
	"github.com/epilande/codegrab/internal/filesystem"
	"github.com/epilande/codegrab/internal/generator"
	"github.com/epilande/codegrab/internal/generator/formats"
	"github.com/epilande/codegrab/internal/ui"
	"github.com/epilande/codegrab/internal/utils"
)

// FileNode represents a file/folder in the display list.
type FileNode struct {
	Path         string
	Name         string
	Icon         string
	IconColor    string
	Level        int
	IsDir        bool
	IsLast       bool
	Selected     bool
	IsDeselected bool
	IsDependency bool
}

type Model struct {
	err                   error
	selected              map[string]bool
	deselected            map[string]bool
	collapsed             map[string]bool
	isDependency          map[string]bool
	gitIgnoreMgr          *filesystem.GitIgnoreManager
	filterMgr             *filesystem.FilterManager
	generator             *generator.Generator
	rootPath              string
	projectModuleName     string
	successMsg            string
	warningMsg            string
	viewport              viewport.Model
	previewViewport       viewport.Model
	files                 []filesystem.FileItem
	displayNodes          []FileNode
	searchResults         []FileNode
	searchInput           textinput.Model
	cursor                int
	width                 int
	height                int
	maxDepth              int
	maxFileSize           int64
	showHelp              bool
	useGitIgnore          bool
	showHidden            bool
	showIcons             bool
	isSearching           bool
	isGrabbing            bool
	redactSecrets         bool
	resolveDeps           bool
	showTokenCount        bool
	showPreview           bool
	previewFocused        bool
	currentPreviewPath    string
	currentPreviewContent string
	currentPreviewIsDir   bool
	lastKeyTime           int64  // Last key press time
	lastKey               string // Last key pressed
}

type Config struct {
	FilterMgr      *filesystem.FilterManager
	RootPath       string
	OutputPath     string
	Format         string
	MaxDepth       int
	MaxFileSize    int64
	UseTempFile    bool
	SkipRedaction  bool
	ResolveDeps    bool
	ShowIcons      bool
	ShowTokenCount bool
}

// updatePreview reads the content of the file at the cursor and updates the preview viewport
func (m *Model) updatePreview() {
	var nodes []FileNode
	if m.isSearching && len(m.searchResults) > 0 {
		nodes = m.searchResults
	} else {
		nodes = m.displayNodes
	}

	if m.cursor >= 0 && m.cursor < len(nodes) {
		node := nodes[m.cursor]

		// Don't update if it's the same file
		if m.currentPreviewPath == node.Path {
			return
		}

		m.currentPreviewPath = node.Path
		m.currentPreviewIsDir = node.IsDir

		if node.IsDir {
			// For directories, just show the directory name
			m.currentPreviewContent = fmt.Sprintf("Directory: %s", node.Path)
		} else {
			// For files, read the content
			fullPath := filepath.Join(m.rootPath, node.Path)

			// Check if it's a text file
			isText, err := utils.IsTextFile(fullPath)
			if err != nil {
				m.currentPreviewContent = fmt.Sprintf("Error reading file: %v", err)
				return
			}

			if !isText {
				m.currentPreviewContent = "[Binary file]"
				return
			}

			// Read file content
			content, err := os.ReadFile(fullPath)
			if err != nil {
				m.currentPreviewContent = fmt.Sprintf("Error reading file: %v", err)
				return
			}

			// Set the content
			// Calculate the available width for text in the preview pane
			// Account for border (1 char on each side) and padding (1 char on each side)
			availableWidth := m.previewViewport.Width - 4

			// Process the content to handle line wrapping consistently
			rawContent := string(content)

			// If the preview viewport width is set, ensure content fits properly
			if m.previewViewport.Width > 0 && availableWidth > 0 {
				// Split content into lines
				lines := strings.Split(rawContent, "\n")
				processedLines := make([]string, 0, len(lines))

				// Process each line to ensure consistent wrapping
				for _, line := range lines {
					// For very long lines, add some padding to ensure consistent wrapping
					if len(line) > availableWidth {
						// Add the line with proper wrapping
						processedLines = append(processedLines, line)
					} else {
						// Keep shorter lines as is
						processedLines = append(processedLines, line)
					}
				}

				// Join the processed lines back together
				rawContent = strings.Join(processedLines, "\n")
			}

			// Store the processed content
			m.currentPreviewContent = rawContent
		}

		// Store the current viewport dimensions before updating content
		oldWidth := m.previewViewport.Width
		oldHeight := m.previewViewport.Height

		// Set the content and reset position
		m.previewViewport.SetContent(m.currentPreviewContent)
		m.previewViewport.GotoTop()

		// Ensure viewport dimensions remain consistent
		m.previewViewport.Width = oldWidth
		m.previewViewport.Height = oldHeight
	}
}

func NewModel(config Config) Model {
	gitIgnoreMgr, err := filesystem.NewGitIgnoreManager(config.RootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading .gitignore: %v\n", err)
		os.Exit(1)
	}

	gen := generator.NewGenerator(config.RootPath, gitIgnoreMgr, config.FilterMgr, config.OutputPath, config.UseTempFile)
	format := formats.GetFormat(config.Format)
	gen.SetFormat(format)
	gen.SetRedactionMode(!config.SkipRedaction)

	moduleName := dependencies.ReadGoModFile(config.RootPath)

	return Model{
		rootPath:          config.RootPath,
		selected:          make(map[string]bool),
		deselected:        make(map[string]bool),
		collapsed:         make(map[string]bool),
		isDependency:      make(map[string]bool),
		useGitIgnore:      true,
		gitIgnoreMgr:      gitIgnoreMgr,
		filterMgr:         config.FilterMgr,
		generator:         gen,
		redactSecrets:     !config.SkipRedaction,
		resolveDeps:       config.ResolveDeps,
		showIcons:         config.ShowIcons,
		maxDepth:          config.MaxDepth,
		maxFileSize:       config.MaxFileSize,
		projectModuleName: moduleName,
		showHidden:        false,
		searchInput:       ui.NewSearchInput(),
		viewport: viewport.Model{
			Width:  80,
			Height: 10,
		},
		previewViewport: viewport.Model{
			Width:  80,
			Height: 10,
		},
		cursor:         0,
		showTokenCount: config.ShowTokenCount,
		showPreview:    false,
	}
}
