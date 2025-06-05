package model

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/epilande/codegrab/internal/utils"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/epilande/codegrab/internal/filesystem"
	"github.com/epilande/codegrab/internal/generator/formats"
	"github.com/epilande/codegrab/internal/ui"
)

// doubleKeyTimeoutMs is the maximum time in milliseconds between two 'g' keypresses to be considered a 'gg' command
const doubleKeyTimeoutMs = 500

// defaultFileTreePreviewRatio is the ratio of the screen width allocated to the file tree panel
const defaultFileTreePreviewRatio = 0.5

type filesLoadedMsg struct {
	err   error
	files []filesystem.FileItem
}

type outputGeneratedMsg struct {
	err         error
	path        string
	format      string
	tokenCount  int
	secretCount int
}

type clipboardCopiedMsg struct {
	err         error
	format      string
	tokenCount  int
	secretCount int
}

type refreshMsg struct{}

func (m Model) Init() tea.Cmd {
	return m.reloadFiles()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.successMsg = ""
	m.warningMsg = ""
	m.isGrabbing = false

	switch msg := msg.(type) {
	case filesLoadedMsg:
		m.err = msg.err
		m.files = msg.files
		for _, f := range m.files {
			if f.IsDir {
				m.collapsed[f.Path] = true
			}
		}
		m.buildDisplayNodes()
		m.refreshViewportContent()

		// Initialize preview if enabled
		if m.showPreview {
			m.updatePreview()
		}

		return m, nil

	case outputGeneratedMsg:
		m.isGrabbing = true
		if msg.err != nil {
			m.err = msg.err
			m.successMsg = ""
			m.warningMsg = ""
		} else {
			m.err = nil
			m.successMsg = fmt.Sprintf("✅ Generated %s (%d tokens)", msg.path, msg.tokenCount)
			if msg.secretCount > 0 && !m.redactSecrets {
				m.warningMsg = fmt.Sprintf("⚠️ %d secrets NOT redacted", msg.secretCount)
			} else if msg.secretCount > 0 && m.redactSecrets {
				m.warningMsg = fmt.Sprintf("ℹ️ %d secrets redacted", msg.secretCount)
			} else {
				m.warningMsg = ""
			}
		}
		m.refreshViewportContent()
		return m, nil

	case clipboardCopiedMsg:
		m.isGrabbing = true
		if msg.err != nil {
			m.err = msg.err
			m.successMsg = ""
			m.warningMsg = ""
		} else {
			m.err = nil
			m.successMsg = fmt.Sprintf("✅ %s copied to clipboard! (%d tokens)", msg.format, msg.tokenCount)
			if msg.secretCount > 0 && !m.redactSecrets {
				m.warningMsg = fmt.Sprintf("⚠️ %d secrets NOT redacted", msg.secretCount)
			} else if msg.secretCount > 0 && m.redactSecrets {
				m.warningMsg = fmt.Sprintf("ℹ️ %d secrets redacted", msg.secretCount)
			} else {
				m.warningMsg = ""
			}
		}
		m.refreshViewportContent()
		return m, nil

	case refreshMsg:
		m.successMsg = "🔄 Refreshed files and reset selection"
		m.refreshViewportContent()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update layout based on new window size
		m.calculateLayout()

		m.refreshViewportContent()

		// Update preview content if needed
		if m.showPreview {
			m.updatePreview()
		}

		return m, nil

	case tea.KeyMsg:

		// Get the current key
		currentKey := msg.String()
		currentTime := utils.GetCurrentTimeMillis()

		// Handle 'gg' key sequence for Vim-like navigation (consolidated logic)
		if currentKey == "g" {
			if m.lastKey == "g" && (currentTime-m.lastKeyTime) < doubleKeyTimeoutMs {
				// This is the second 'g' within time window - execute 'gg' command
				m.lastKey = "" // Reset after handling the sequence

				// Go to top (Vim style) - action depends on current state
				if m.showHelp {
					// In help mode, just scroll help to top
					m.viewport.GotoTop()
				} else if m.previewFocused && m.showPreview {
					// When preview is focused, scroll preview to top
					m.previewViewport.GotoTop()
				} else {
					// When file tree is focused, move cursor to top and scroll
					m.viewport.GotoTop()
					m.cursor = 0
					m.refreshViewportContent()
				}
				return m, nil
			} else {
				// First 'g' press - record it and wait for second 'g'
				m.lastKey = "g"
				m.lastKeyTime = currentTime
				return m, nil
			}
		} else {
			// For non-'g' keys, update tracking info
			m.lastKey = currentKey
			m.lastKeyTime = currentTime
		}

		// Handle half-page scrolling keys
		if currentKey == "ctrl+u" {
			if m.previewFocused && m.showPreview {
				// Scroll preview half page up when preview is focused
				m.previewViewport.HalfViewUp()
			} else {
				// Move cursor half page up in file tree
				m.halfPageUp()
				m.refreshViewportContent()
				// Update preview if enabled
				if m.showPreview {
					m.updatePreview()
				}
			}
			return m, nil
		} else if currentKey == "ctrl+d" {
			if m.previewFocused && m.showPreview {
				// Scroll preview half page down when preview is focused
				m.previewViewport.HalfViewDown()
			} else {
				// Move cursor half page down in file tree
				m.halfPageDown()
				m.refreshViewportContent()
				// Update preview if enabled
				if m.showPreview {
					m.updatePreview()
				}
			}
			return m, nil
		}

		// Special handling for help mode
		if m.showHelp {
			// Handle help mode keys
			switch currentKey {
			case "q", "esc", "?":
				m.showHelp = false
				m.refreshViewportContent()
				return m, nil
			case "j", "down":
				m.viewport.LineDown(1)
				return m, nil
			case "k", "up":
				m.viewport.LineUp(1)
				return m, nil
			case "G":
				// Vim style: go to bottom
				m.viewport.GotoBottom()
				return m, nil
			default:
				return m, nil
			}
		}

		if m.isSearching {
			var cmd tea.Cmd

			switch msg.String() {
			case "esc":
				m.isSearching = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				m.searchResults = nil
				m.cursor = 0
				m.viewport.GotoTop()
				m.collapseAllDirectories()
				m.refreshViewportContent()
				// Update preview if enabled
				if m.showPreview {
					m.updatePreview()
				}
				return m, nil
			case "tab", "enter":
				if len(m.searchResults) > 0 && m.cursor < len(m.searchResults) {
					node := m.searchResults[m.cursor]
					m.toggleSelection(node.Path, node.IsDir)
					m.buildDisplayNodes()
					m.updateSearchResults()
					m.ensureCursorVisible()
					m.refreshViewportContent()
					// Update preview if enabled
					if m.showPreview {
						m.updatePreview()
					}
				}
				return m, nil
			case "ctrl+n", "down":
				if len(m.searchResults) > 0 {
					m.cursor = (m.cursor + 1) % len(m.searchResults)
					m.ensureCursorVisible()
					m.refreshViewportContent()
					// Update preview if enabled
					if m.showPreview {
						m.updatePreview()
					}
				}
				return m, nil
			case "ctrl+p", "up":
				if len(m.searchResults) > 0 {
					m.cursor--
					if m.cursor < 0 {
						m.cursor = len(m.searchResults) - 1
					}
					m.ensureCursorVisible()
					m.refreshViewportContent()
					// Update preview if enabled
					if m.showPreview {
						m.updatePreview()
					}
				}
				return m, nil
			}

			m.searchInput, cmd = m.searchInput.Update(msg)

			m.updateSearchResults()
			m.refreshViewportContent()

			// Update preview if enabled and we have search results
			if m.showPreview && len(m.searchResults) > 0 {
				m.updatePreview()
			}

			return m, cmd
		}

		// Handle single keys
		switch currentKey {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down", "ctrl+n":
			if m.previewFocused && m.showPreview {
				// Scroll preview down when preview is focused
				m.previewViewport.LineDown(1)
			} else if m.cursor < len(m.displayNodes)-1 {
				m.cursor++
				m.ensureCursorVisible()
				m.refreshViewportContent()
				if m.showPreview {
					m.updatePreview()
				}
			}
		case "k", "up", "ctrl+p":
			if m.previewFocused && m.showPreview {
				// Scroll preview up when preview is focused
				m.previewViewport.LineUp(1)
			} else if m.cursor > 0 {
				m.cursor--
				m.ensureCursorVisible()
				m.refreshViewportContent()
				if m.showPreview {
					m.updatePreview()
				}
			}
		case "r":
			m.selected = make(map[string]bool)
			m.deselected = make(map[string]bool)
			m.isDependency = make(map[string]bool)
			m.cursor = 0
			m.viewport.GotoTop()
			return m, tea.Sequence(
				m.reloadFiles(),
				func() tea.Msg { return refreshMsg{} },
			)

		case "h", "left":
			if m.previewFocused && m.showPreview {
				// Return focus to file tree
				m.previewFocused = false
				m.refreshViewportContent()
			} else if m.cursor < len(m.displayNodes) {
				node := m.displayNodes[m.cursor]
				if node.IsDir && !m.collapsed[node.Path] {
					m.toggleCollapse(node.Path)
					m.buildDisplayNodes()
					if m.cursor >= len(m.displayNodes) {
						m.cursor = len(m.displayNodes) - 1
					}
					m.ensureCursorVisible()
					m.refreshViewportContent()
				}
			}
		case "l", "right":
			if !m.previewFocused && m.showPreview && m.cursor < len(m.displayNodes) {
				node := m.displayNodes[m.cursor]
				if !node.IsDir {
					// Move focus to preview panel for files
					m.previewFocused = true
					m.refreshViewportContent()
					return m, nil
				}
			}

			if m.cursor < len(m.displayNodes) {
				node := m.displayNodes[m.cursor]
				if node.IsDir && m.collapsed[node.Path] {
					m.toggleCollapse(node.Path)
					m.buildDisplayNodes()
					m.ensureCursorVisible()
					m.refreshViewportContent()
				}
			}
		case "e":
			if len(m.collapsed) > 0 {
				m.expandAllDirectories()
			} else {
				m.cursor = 0
				m.viewport.GotoTop()
				m.collapseAllDirectories()
			}
			m.ensureCursorVisible()
			m.refreshViewportContent()
		case " ", "tab":
			if m.cursor < len(m.displayNodes) {
				node := m.displayNodes[m.cursor]
				cmds := []tea.Cmd{m.toggleSelection(node.Path, node.IsDir)}
				m.buildDisplayNodes()
				m.ensureCursorVisible()
				m.refreshViewportContent()
				return m, tea.Batch(cmds...)
			}
		case "/":
			m.isSearching = true
			m.searchInput.Focus()
			m.searchInput.SetValue("")
			m.searchResults = nil
			m.cursor = 0
			m.viewport.GotoTop()
			m.expandAllDirectories()
			m.refreshViewportContent()
			return m, nil
		case "?":
			m.showHelp = !m.showHelp
			if m.showHelp {
				m.showHelpScreen()
			} else {
				m.refreshViewportContent()
			}
		case "esc":
			if m.showHelp {
				m.showHelp = false
				m.refreshViewportContent()
			}
		case "y":
			return m, m.copyOutputToClipboard()
		case "i":
			m.useGitIgnore = !m.useGitIgnore
			m.generator.UseGitIgnore = m.useGitIgnore
			m.filterSelections()
			m.cursor = 0
			m.viewport.GotoTop()
			return m, m.reloadFiles()
		case ".":
			m.showHidden = !m.showHidden
			m.generator.ShowHidden = m.showHidden
			m.filterSelections()
			m.cursor = 0
			m.viewport.GotoTop()
			return m, m.reloadFiles()
		case "D":
			m.resolveDeps = !m.resolveDeps
			if m.resolveDeps {
				m.successMsg = "Dependency resolution enabled"
			} else {
				m.successMsg = "Dependency resolution disabled"
			}
			m.buildDisplayNodes()
			m.refreshViewportContent()
		case "F":
			formatNames := formats.GetFormatNames()
			if len(formatNames) == 0 {
				break
			}

			currentFormatName := m.generator.GetFormatName()

			nextIndex := 0
			for i, name := range formatNames {
				if name == currentFormatName {
					nextIndex = (i + 1) % len(formatNames)
					break
				}
			}

			nextFormat := formats.GetFormat(formatNames[nextIndex])
			m.generator.SetFormat(nextFormat)

			m.successMsg = fmt.Sprintf("Format changed: %s", m.generator.GetFormatName())
			m.refreshViewportContent()
		case "S":
			m.redactSecrets = !m.redactSecrets
			m.generator.SetRedactionMode(m.redactSecrets)
			if m.redactSecrets {
				m.successMsg = "Secret redaction enabled"
			} else {
				m.successMsg = "Secret redaction disabled"
			}
			m.warningMsg = ""
			m.refreshViewportContent()
		case "P":
			// Toggle preview pane
			m.showPreview = !m.showPreview

			// Update layout based on new preview state
			m.calculateLayout()

			// Update the preview content if needed
			if m.showPreview {
				m.updatePreview()
				m.successMsg = "Preview pane enabled"
			} else {
				m.successMsg = "Preview pane disabled"
			}

			m.refreshViewportContent()
		// Preview navigation keys
		case "J":
			if m.showPreview {
				m.previewViewport.LineDown(1)
			}
		case "K":
			if m.showPreview {
				m.previewViewport.LineUp(1)
			}
		case "G":
			// Go to bottom (Vim style)
			if m.previewFocused && m.showPreview {
				// When preview is focused, just scroll to bottom
				m.previewViewport.GotoBottom()
			} else {
				// When file tree is focused, move cursor to last item and scroll to bottom
				m.viewport.GotoBottom()
				if len(m.displayNodes) > 0 {
					m.cursor = len(m.displayNodes) - 1
					m.ensureCursorVisible()
					m.refreshViewportContent()
					// Update preview if needed
					if m.showPreview {
						m.updatePreview()
					}
				}
			}

		case "ctrl+g":
			// Generate output
			return m, m.generateOutput()
		}
	}
	return m, nil
}

func (m *Model) reloadFiles() tea.Cmd {
	return func() tea.Msg {
		files, err := filesystem.WalkDirectory(m.rootPath, m.gitIgnoreMgr, m.filterMgr, m.useGitIgnore, m.showHidden, m.maxFileSize)
		if err != nil {
			return filesLoadedMsg{files: nil, err: fmt.Errorf("failed to reload files: %w", err)}
		}

		m.isDependency = make(map[string]bool)

		return filesLoadedMsg{files: files, err: nil}
	}
}

func (m *Model) toggleCollapse(path string) {
	m.collapsed[path] = !m.collapsed[path]
}

// halfPageUp moves the cursor up by half the viewport height
func (m *Model) halfPageUp() {
	// Get the nodes to work with
	var nodes []FileNode
	if m.isSearching && len(m.searchResults) > 0 {
		nodes = m.searchResults
	} else {
		nodes = m.displayNodes
	}

	// Check for empty list
	if len(nodes) == 0 {
		m.cursor = 0
		return
	}

	// Calculate half the viewport height
	halfHeight := m.viewport.Height / 2

	// Calculate how many lines we can actually move up
	// This is the minimum of half the viewport height and the current cursor position
	moveLines := halfHeight
	if moveLines > m.cursor {
		moveLines = m.cursor
	}

	// Move cursor up by calculated amount
	m.cursor -= moveLines

	// Ensure cursor is visible
	m.ensureCursorVisible()
}

// halfPageDown moves the cursor down by half the viewport height
func (m *Model) halfPageDown() {
	// Calculate half the viewport height
	halfHeight := m.viewport.Height / 2

	// Get the nodes to work with
	var nodes []FileNode
	if m.isSearching && len(m.searchResults) > 0 {
		nodes = m.searchResults
	} else {
		nodes = m.displayNodes
	}

	// Safety check for empty list
	if len(nodes) == 0 {
		m.cursor = 0
		return
	}

	// Calculate how many lines we can actually move down
	// This is the minimum of half the viewport height and the remaining lines
	moveLines := halfHeight
	remaining := len(nodes) - m.cursor - 1
	if moveLines > remaining {
		moveLines = remaining
	}

	// Move cursor down by calculated amount
	m.cursor += moveLines

	// Ensure cursor is visible
	m.ensureCursorVisible()
}

// calculateLayout updates the viewport dimensions based on the current window size and preview state
func (m *Model) calculateLayout() {
	// Calculate actual header and footer heights dynamically
	renderedHeader := m.renderHeader()
	renderedFooter := m.renderFooter()
	headerHeight := lipgloss.Height(renderedHeader)
	footerHeight := lipgloss.Height(renderedFooter)

	// Calculate available height precisely
	availablePanelHeight := m.height - headerHeight - footerHeight
	if availablePanelHeight < 0 {
		availablePanelHeight = 0
	}

	// Available width after terminal edge padding
	availableWidth := m.width - ui.FileTreePaddingL - ui.FileTreePaddingR

	if m.showPreview {
		// Split the screen for file tree and preview
		adjustedWidth := availableWidth - ui.PanelGap - (4 * ui.BorderSize)
		if adjustedWidth < 0 {
			adjustedWidth = 0
		}

		// Allocate space based on the defined ratio
		fileTreeInnerWidth := int(float64(adjustedWidth) * defaultFileTreePreviewRatio)
		previewInnerWidth := adjustedWidth - fileTreeInnerWidth

		if fileTreeInnerWidth < 0 {
			fileTreeInnerWidth = 0
		}
		if previewInnerWidth < 0 {
			previewInnerWidth = 0
		}

		// Set viewport dimensions with precise width calculation
		m.viewport.Width = fileTreeInnerWidth

		// Create file tree panel header to calculate its actual height
		fileTreePanelHeader := ui.GetStyleFileTreePanelHeader().
			Width(fileTreeInnerWidth).
			Render("📚 Files")
		fileTreePanelHeaderHeight := lipgloss.Height(fileTreePanelHeader)

		// Calculate viewport height precisely accounting for all UI elements
		m.viewport.Height = availablePanelHeight - (2 * ui.BorderSize) - fileTreePanelHeaderHeight
		if m.viewport.Height < 0 {
			m.viewport.Height = 0
		}

		m.previewViewport.Width = previewInnerWidth

		// Prepare preview header text
		previewHeaderText := "No file selected"
		if m.currentPreviewPath != "" {
			previewHeaderText = "📄 " + m.currentPreviewPath
			if m.currentPreviewIsDir {
				previewHeaderText = "📁 " + m.currentPreviewPath
			}
		}

		// Calculate actual preview header height
		previewOuterWidth := previewInnerWidth + (2 * ui.BorderSize)
		styledPreviewHeader := ui.GetStylePreviewHeader().
			Width(previewOuterWidth).
			Render(previewHeaderText)
		previewHeaderActualHeight := lipgloss.Height(styledPreviewHeader)

		// Calculate preview viewport height with precise measurement
		m.previewViewport.Height = availablePanelHeight - previewHeaderActualHeight - (2 * ui.BorderSize)
		if m.previewViewport.Height < 0 {
			m.previewViewport.Height = 0
		}
	} else {
		// Full width for file tree (single panel mode)
		innerWidth := availableWidth - (2 * ui.BorderSize)
		if innerWidth < 0 {
			innerWidth = 0
		}

		// Set viewport dimensions with precise width calculation
		m.viewport.Width = innerWidth

		// Create file tree panel header to calculate its actual height
		fileTreePanelHeader := ui.GetStyleFileTreePanelHeader().
			Width(availableWidth - (2 * ui.BorderSize)).
			Render("📚 Files")
		fileTreePanelHeaderHeight := lipgloss.Height(fileTreePanelHeader)

		// Calculate viewport height precisely accounting for all UI elements
		m.viewport.Height = availablePanelHeight - (2 * ui.BorderSize) - fileTreePanelHeaderHeight
		if m.viewport.Height < 0 {
			m.viewport.Height = 0
		}
	}
}

// generateOutput creates the output in the current format
func (m Model) generateOutput() tea.Cmd {
	return func() tea.Msg {
		m.generator.SelectedFiles = m.selected
		m.generator.DeselectedFiles = m.deselected
		outPath, tokenCount, secretCount, err := m.generator.Generate()
		if err != nil {
			return outputGeneratedMsg{
				err:         fmt.Errorf("failed to generate output: %w", err),
				path:        "",
				tokenCount:  0,
				format:      m.generator.GetFormatName(),
				secretCount: secretCount,
			}
		}
		return outputGeneratedMsg{
			err:         nil,
			path:        outPath,
			tokenCount:  tokenCount,
			format:      m.generator.GetFormatName(),
			secretCount: secretCount,
		}
	}
}

// copyOutputToClipboard copies the generated content to the clipboard
func (m Model) copyOutputToClipboard() tea.Cmd {
	return func() tea.Msg {
		m.generator.SelectedFiles = m.selected
		m.generator.DeselectedFiles = m.deselected

		content, tokenCount, secretCount, err := m.generator.GenerateString()
		if err != nil {
			return clipboardCopiedMsg{
				err:         err,
				tokenCount:  0,
				format:      m.generator.GetFormatName(),
				secretCount: secretCount,
			}
		}

		if err = clipboard.WriteAll(content); err != nil {
			return clipboardCopiedMsg{
				err:         fmt.Errorf("failed to copy to clipboard: %w", err),
				tokenCount:  tokenCount,
				format:      m.generator.GetFormatName(),
				secretCount: secretCount,
			}
		}

		return clipboardCopiedMsg{
			err:         nil,
			tokenCount:  tokenCount,
			format:      m.generator.GetFormatName(),
			secretCount: secretCount,
		}
	}
}
