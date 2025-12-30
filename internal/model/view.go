package model

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/epilande/codegrab/internal/filesystem"
	"github.com/epilande/codegrab/internal/ui"
	"github.com/epilande/codegrab/internal/ui/themes"
)

// View renders the entire UI model.
func (m Model) View() string {
	// If help is shown, render a simple help view.
	if m.showHelp {
		header := ui.GetStyleHeader().Render("❔ Help Menu")
		headerHeight := lipgloss.Height(header)
		footerText := "Exit: esc" // Example footer text for height calculation
		footer := ui.GetStyleHelp().Render(footerText)
		footerHeight := lipgloss.Height(footer)
		availableHeight := m.height - headerHeight - footerHeight - (2 * ui.BorderSize)
		if availableHeight < 0 {
			availableHeight = 0
		}
		m.viewport.Height = availableHeight // Set viewport height for help content

		// Calculate inner width accounting for borders and padding
		innerWidth := m.width - (2 * ui.BorderSize) - ui.FileTreePaddingL - ui.FileTreePaddingR
		if innerWidth < 0 {
			innerWidth = 0
		}
		content := ui.GetStyleBorderedViewport().Width(innerWidth).Height(availableHeight).Render(m.viewport.View())

		result := header + "\n" + content + "\n" + footer

		// Apply terminal edge padding with consistent spacing
		return lipgloss.NewStyle().
			PaddingLeft(ui.FileTreePaddingL).
			PaddingRight(ui.FileTreePaddingR).
			Render(result)
	}

	// Render header and footer strings first to determine their heights.
	renderedHeader := m.renderHeader()
	renderedFooter := m.renderFooter()
	headerHeight := lipgloss.Height(renderedHeader)
	footerHeight := lipgloss.Height(renderedFooter)

	// Calculate available height precisely - don't add the +1 adjustment
	availablePanelHeight := m.height - headerHeight - footerHeight
	if availablePanelHeight < 0 {
		availablePanelHeight = 0
	}

	// Available width after terminal edge padding
	availableWidth := m.width - ui.FileTreePaddingL - ui.FileTreePaddingR

	// If preview mode is active, render a split-screen view.
	if m.showPreview {
		// Calculate total horizontal space consumed by borders and gap
		totalHorizontalBorderSpace := 2 * (2 * ui.BorderSize)

		// Calculate the total inner width available for the content of both panels
		availableInnerContentWidth := availableWidth - ui.PanelGap - totalHorizontalBorderSpace
		if availableInnerContentWidth < 0 {
			availableInnerContentWidth = 0
		}

		// Distribute the available inner width between the file tree and preview content
		fileTreeInnerWidth := int(float64(availableInnerContentWidth) * defaultFileTreePreviewRatio)
		previewInnerWidth := availableInnerContentWidth - fileTreeInnerWidth

		// Apply minimum width constraints
		if fileTreeInnerWidth < ui.MinInnerContentWidth {
			fileTreeInnerWidth = ui.MinInnerContentWidth
			previewInnerWidth = availableInnerContentWidth - fileTreeInnerWidth
			if previewInnerWidth < ui.MinInnerContentWidth {
				previewInnerWidth = ui.MinInnerContentWidth
			}
		} else if previewInnerWidth < ui.MinInnerContentWidth {
			previewInnerWidth = ui.MinInnerContentWidth
			fileTreeInnerWidth = availableInnerContentWidth - previewInnerWidth
			if fileTreeInnerWidth < ui.MinInnerContentWidth {
				fileTreeInnerWidth = ui.MinInnerContentWidth
			}
		}

		// Ensure non-negative widths
		if fileTreeInnerWidth < 0 {
			fileTreeInnerWidth = 0
		}
		if previewInnerWidth < 0 {
			previewInnerWidth = 0
		}

		// Create file tree panel header
		fileTreePanelHeader := ui.GetStyleFileTreePanelHeader().
			Width(fileTreeInnerWidth).
			Render("📚 Files")
		fileTreePanelHeaderHeight := lipgloss.Height(fileTreePanelHeader)

		// Precise viewport height calculation - account for everything
		m.viewport.Height = availablePanelHeight - (2 * ui.BorderSize) - fileTreePanelHeaderHeight
		if m.viewport.Height < 0 {
			m.viewport.Height = 0
		}

		// Prepare preview header text
		previewHeaderText := "No file selected"
		if m.currentPreviewPath != "" {
			previewHeaderText = "📄 " + m.currentPreviewPath
			if m.currentPreviewIsDir {
				previewHeaderText = "📁 " + m.currentPreviewPath
			}
		}

		// Set preview header width
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

		// Define panel styles
		commonBorder := lipgloss.RoundedBorder()
		defaultBorderColor := themes.CurrentTheme.Colors().Border
		highlightedBorderColor := ui.GetStyleHighlightedBorder().GetForeground()

		// File tree style - account for content only (header is separate)
		// Use exact height to prevent extra space at the bottom
		fileTreeContentHeight := m.viewport.Height
		fileTreeStyle := lipgloss.NewStyle().
			Border(commonBorder).
			BorderTop(true).
			BorderRight(true).
			BorderBottom(true).
			BorderLeft(true).
			BorderForeground(defaultBorderColor).
			Width(fileTreeInnerWidth).
			Height(fileTreeContentHeight)

		// Preview content style
		// Use exact height to prevent extra space at the bottom
		previewContentBoxHeight := m.previewViewport.Height
		if previewContentBoxHeight < 0 {
			previewContentBoxHeight = 0
		}

		previewContentBoxStyle := lipgloss.NewStyle().
			Border(commonBorder).
			BorderTop(true).
			BorderRight(true).
			BorderBottom(true).
			BorderLeft(true).
			BorderForeground(defaultBorderColor).
			Width(previewInnerWidth).
			Height(previewContentBoxHeight)

		// Apply highlighted border to focused panel
		if m.previewFocused {
			previewContentBoxStyle = previewContentBoxStyle.BorderForeground(highlightedBorderColor)
		} else {
			fileTreeStyle = fileTreeStyle.BorderForeground(highlightedBorderColor)
		}

		// Render file tree with its header
		renderedFileTreeContent := fileTreeStyle.Render(m.viewport.View())
		renderedFileTree := lipgloss.JoinVertical(lipgloss.Left,
			fileTreePanelHeader,
			renderedFileTreeContent)

		// Render preview content
		renderedPreviewContent := previewContentBoxStyle.Render(m.previewViewport.View())

		// Combine preview header and content
		renderedPreview := lipgloss.JoinVertical(lipgloss.Left,
			styledPreviewHeader,
			renderedPreviewContent,
		)

		// Join panels with gap
		gapFiller := strings.Repeat(" ", ui.PanelGap)
		mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
			renderedFileTree,
			gapFiller,
			renderedPreview,
		)

		// Build the final layout content
		finalContent := lipgloss.JoinVertical(
			lipgloss.Left,
			renderedHeader,
			mainContent,
			renderedFooter,
		)

		// Apply terminal edge padding with consistent spacing
		return lipgloss.NewStyle().
			PaddingLeft(ui.FileTreePaddingL).
			PaddingRight(ui.FileTreePaddingR).
			Render(finalContent)

	} else {
		// Single-pane view (non-preview mode)
		// Create file tree panel header
		fileTreePanelHeader := ui.GetStyleFileTreePanelHeader().
			Width(availableWidth - (2 * ui.BorderSize)).
			Render("📚 Files")
		fileTreePanelHeaderHeight := lipgloss.Height(fileTreePanelHeader)

		// Calculate viewport height accounting for the file tree panel header
		m.viewport.Height = availablePanelHeight - (2 * ui.BorderSize) - fileTreePanelHeaderHeight
		if m.viewport.Height < 0 {
			m.viewport.Height = 0
		}

		// Calculate bordered viewport height for content only (not header)
		// Use exact height to prevent extra space at the bottom
		borderedViewportHeight := m.viewport.Height

		// Calculate inner width
		innerWidth := availableWidth - (2 * ui.BorderSize)
		if innerWidth < 0 {
			innerWidth = 0
		}

		// Create bordered viewport for file content
		borderedViewportStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderTop(true).
			BorderRight(true).
			BorderBottom(true).
			BorderLeft(true).
			BorderForeground(themes.CurrentTheme.Colors().Border).
			Width(innerWidth).
			Height(borderedViewportHeight)

		// Render content and join with header
		borderedContent := borderedViewportStyle.Render(m.viewport.View())
		content := lipgloss.JoinVertical(lipgloss.Left,
			fileTreePanelHeader,
			borderedContent)

		// Build the final layout content explicitly
		finalContent := lipgloss.JoinVertical(
			lipgloss.Left,
			renderedHeader,
			content,
			renderedFooter,
		)

		// Apply terminal edge padding with consistent spacing
		return lipgloss.NewStyle().
			PaddingLeft(ui.FileTreePaddingL).
			PaddingRight(ui.FileTreePaddingR).
			Render(finalContent)
	}
}

// renderHeader renders the header part of the UI.
func (m Model) renderHeader() string {
	headerIcon := "✋"
	if m.isGrabbing {
		headerIcon = "✊"
	}
	leftContent := ui.GetStyleHeader().Render(headerIcon + " Code Grab")
	rightContent := ""
	totalFiles := m.getTotalFileCount()
	selectedCount := m.getSelectedFileCount()

	formatExt := strings.TrimPrefix(m.generator.GetFormat().Extension(), ".")
	formatIndicator := ui.GetStyleFormatIndicator().Render(formatExt)

	if m.isSearching {
		// When searching, the search input takes the left side
		leftContent = m.searchInput.View()
		matchCount := 0
		for _, node := range m.searchResults {
			if !node.IsDir {
				matchCount++
			}
		}
		rightContent = ui.GetStyleSearchCount().Render(fmt.Sprintf("%d/%d [%d] %s",
			matchCount,
			totalFiles,
			selectedCount,
			formatIndicator,
		))
	} else {
		rightContent = ui.GetStyleSearchCount().Render(fmt.Sprintf("%d [%d] %s",
			totalFiles,
			selectedCount,
			formatIndicator,
		))
	}

	// Calculate spacing to push rightContent to the right edge
	spacing := m.width - lipgloss.Width(leftContent) - lipgloss.Width(rightContent) - 1
	if spacing < 0 {
		spacing = 0
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftContent,
		strings.Repeat(" ", spacing),
		rightContent,
	)
}

// renderFooter renders the footer part of the UI.
func (m Model) renderFooter() string {
	// If help is shown, the main View() handles a minimal footer
	if m.showHelp {
		return ""
	}

	var leftParts []string
	var rightParts []string

	// Left side: Status/Error/Help prompts
	if m.isSearching {
		searchHelp := "Next: ctrl+n | Prev: ctrl+p | Select: tab | Exit: esc"
		leftParts = append(leftParts, ui.GetStyleHelp().Render(searchHelp))
	} else if m.err != nil {
		leftParts = append(leftParts, ui.GetStyleError().Render(m.err.Error()))
	} else if m.successMsg != "" {
		leftParts = append(leftParts, ui.GetStyleSuccess().Render(m.successMsg))
	} else {
		helpText := "Press '?' for help | Select: space | Generate: ctrl+g | Copy: y"
		leftParts = append(leftParts, ui.GetStyleHelp().Render(helpText))
	}

	// Right side: Warning/Redaction status/Dependency status
	if m.warningMsg != "" {
		rightParts = append(rightParts, ui.GetStyleWarning().Render(m.warningMsg))
	} else if m.redactSecrets {
		rightParts = append(rightParts, ui.GetStyleInfo().Render("🛡️ Redacting"))
	} else {
		rightParts = append(rightParts, ui.GetStyleWarning().Render("⚠️ NOT Redacting"))
	}

	// Dependency status
	if m.resolveDeps {
		rightParts = append(rightParts, ui.GetStyleInfo().Render(" | 🔗 Deps"))
	}

	leftContent := lipgloss.JoinHorizontal(lipgloss.Top, leftParts...)
	rightContent := lipgloss.JoinHorizontal(lipgloss.Top, rightParts...)

	// Calculate spacing ensuring proper alignment
	availableFooterWidth := m.width - 2 // Simple 1-char padding on each side
	if availableFooterWidth < 0 {
		availableFooterWidth = 0
	}

	spacing := availableFooterWidth - lipgloss.Width(leftContent) - lipgloss.Width(rightContent)
	if spacing < 0 {
		spacing = 0
	}

	footerContent := lipgloss.JoinHorizontal(
		lipgloss.Bottom,
		leftContent,
		lipgloss.NewStyle().Width(spacing).Render(""),
		rightContent,
	)

	// Add proper padding to ensure consistent spacing
	return lipgloss.NewStyle().
		PaddingLeft(1).
		PaddingRight(1).
		Width(m.width).
		Render(footerContent)
}

// refreshViewportContent regenerates the lines for our displayNodes, highlights
// the cursor, and sets that as the viewport content.
func (m *Model) refreshViewportContent() {
	var nodes []FileNode
	if m.isSearching && len(m.searchResults) > 0 {
		nodes = m.searchResults
	} else {
		nodes = m.displayNodes
	}

	dirsWithSelectedChildren := make(map[string]bool)
	dirSelectedCounts := make(map[string]int)
	
	// When in search mode, only consider files that are in search results
	var relevantFiles []filesystem.FileItem
	if m.isSearching && len(m.searchResults) > 0 {
		// Create a map of search result paths for quick lookup
		searchPaths := make(map[string]bool)
		for _, node := range m.searchResults {
			searchPaths[node.Path] = true
		}
		
		// Only include files that are in search results
		for _, file := range m.files {
			if searchPaths[file.Path] {
				relevantFiles = append(relevantFiles, file)
			}
		}
	} else {
		relevantFiles = m.files
	}
	
	// Calculate directory statistics based on relevant files only
	for path := range m.selected {
		// Skip if this path is not in our relevant set when searching
		if m.isSearching && len(m.searchResults) > 0 {
			found := false
			for _, file := range relevantFiles {
				if file.Path == path {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		dir := filepath.Dir(path)
		// Mark parent directories that contain selected items
		for dir != "." && dir != "/" && dir != "" {
			dirsWithSelectedChildren[dir] = true
			prevDir := dir
			dir = filepath.Dir(dir)
			if dir == prevDir {
				break
			}
		}
	}

	for _, file := range relevantFiles {
		if !file.IsDir && m.selected[file.Path] && !m.deselected[file.Path] {
			dir := filepath.Dir(file.Path)
			for dir != "." && dir != "/" && dir != "" {
				dirSelectedCounts[dir]++
				prevDir := dir
				dir = filepath.Dir(dir)
				if dir == prevDir {
					break
				}
			}
		}
	}

	var lines []string
	parentIsLast := make(map[int]bool) // Tracks if parent at a certain level is the last child

	for i, node := range nodes {
		var prefixBuilder strings.Builder
		for l := 0; l < node.Level; l++ {
			if parentIsLast[l] {
				prefixBuilder.WriteString("    ") // Parent was last, no vertical line
			} else {
				prefixBuilder.WriteString("│   ") // Vertical line for ongoing parent branch
			}
		}

		treeBranch := "├── "
		if node.IsLast {
			treeBranch = "└── "
			parentIsLast[node.Level] = true // This node is the last at its level
		} else {
			parentIsLast[node.Level] = false // Not the last, siblings will follow
		}
		// Reset parentIsLast for deeper levels
		for l := node.Level + 1; l < len(parentIsLast); l++ {
			parentIsLast[l] = false
		}

		treePrefix := prefixBuilder.String() + treeBranch

		icon := node.Icon
		iconColor := node.IconColor
		// Handle directory icon state (open/closed)
		if node.IsDir && icon == "" {
			if m.collapsed[node.Path] {
				icon = "" // Collapsed icon
			} else {
				icon = "" // Expanded icon
			}
			iconColor = "" // Use theme directory color by default
		}

		isPartialDir := !m.selected[node.Path] && dirsWithSelectedChildren[node.Path] && node.IsDir
		rawCheckbox := "[ ]" // Default empty checkbox
		if node.IsDir {
			if m.selected[node.Path] {
				rawCheckbox = "[x]" // Directory fully selected
			} else if isPartialDir {
				rawCheckbox = "[~]" // Directory partially selected
			}
		} else {
			if node.Selected {
				rawCheckbox = "[x]"
			}
		}

		rawName := node.Name
		if node.IsDir {
			rawName += "/" // Append slash to directory names
		}

		rawSuffix := ""
		if node.IsDir {
			if count := dirSelectedCounts[node.Path]; count > 0 {
				rawSuffix = fmt.Sprintf(" [%d]", count) // Show count of selected files
			}
		} else { // It's a file
			if node.IsDependency {
				rawSuffix += " [dep]"
			}
			if m.showTokenCount {
				// Use cached tokens for non-blocking UI rendering
				tokensFormatted := m.tokenCache.GetTokensFormatted(node.Path)
				rawSuffix += tokensFormatted
			}
		}
		isCursorLine := i == m.cursor

		// Render the full line with proper left padding applied
		rendered := ui.StyleFileLine(
			rawCheckbox,
			treePrefix,
			icon,
			iconColor,
			rawName,
			rawSuffix,
			node.IsDir,
			m.selected[node.Path] || isPartialDir,
			isCursorLine,
			isPartialDir,
			m.viewport.Width,
		)
		lines = append(lines, rendered)
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
}

// ensureCursorVisible adjusts viewport.YOffset so the selected line is visible.
func (m *Model) ensureCursorVisible() {
	if m.cursor < m.viewport.YOffset {
		m.viewport.YOffset = m.cursor
	} else if m.cursor >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = m.cursor - m.viewport.Height + 1
		if m.viewport.YOffset < 0 {
			m.viewport.YOffset = 0
		}
	}
}

// showHelpScreen prepares and displays the help content in the main viewport.
func (m *Model) showHelpScreen() {
	helpContent := ui.GetStyleHelp().Render(ui.HelpText + "\n\nPress '?' or 'esc' to close this help menu.")
	m.viewport.SetContent(helpContent)
	m.viewport.GotoTop() // Reset scroll to the top when help is shown
}

// getTotalFileCount counts all non-directory files.
func (m *Model) getTotalFileCount() int {
	count := 0
	for _, file := range m.files {
		if !file.IsDir {
			count++
		}
	}
	return count
}

// getSelectedFileCount calculates the effective number of selected files.
func (m *Model) getSelectedFileCount() int {
	effectiveSelection := make(map[string]bool)

	// Create a set of paths that are part of the current search results, if any
	searchResultPaths := make(map[string]bool)
	isFilteringBySearch := m.isSearching && len(m.searchResults) > 0
	if isFilteringBySearch {
		for _, node := range m.searchResults {
			searchResultPaths[node.Path] = true
		}
	}

	for _, item := range m.files {
		// Skip if already counted or explicitly deselected
		if effectiveSelection[item.Path] || m.deselected[item.Path] {
			continue
		}

		// If filtering by search results, check if item is in scope
		if isFilteringBySearch {
			inSearchResultScope := false
			currentPath := item.Path
			for {
				if searchResultPaths[currentPath] {
					inSearchResultScope = true
					break
				}
				parent := filepath.Dir(currentPath)
				if parent == currentPath || parent == "." || parent == "/" {
					break
				}
				currentPath = parent
			}
			if !inSearchResultScope {
				continue
			}
		}

		// Handle explicitly selected items
		if m.selected[item.Path] {
			if !item.IsDir {
				effectiveSelection[item.Path] = true
			} else {
				// For selected directories, count all non-deselected files
				prefix := item.Path + "/"
				for _, child := range m.files {
					if !child.IsDir && strings.HasPrefix(child.Path, prefix) && !m.deselected[child.Path] {
						if isFilteringBySearch && !searchResultPaths[child.Path] {
							isChildInSearchResultScope := false
							childPathEval := child.Path
							for {
								if searchResultPaths[childPathEval] {
									isChildInSearchResultScope = true
									break
								}
								parent := filepath.Dir(childPathEval)
								if parent == childPathEval || parent == "." || parent == "/" {
									break
								}
								childPathEval = parent
							}
							if !isChildInSearchResultScope {
								continue
							}
						}
						effectiveSelection[child.Path] = true
					}
				}
			}
			continue
		}

		// Handle files within selected directories (implicit selection)
		if !item.IsDir {
			parent := filepath.Dir(item.Path)
			isInSelectedDir := false
			for parent != "." && parent != "/" && parent != "" {
				if m.selected[parent] {
					isInSelectedDir = true
					break
				}
				prevParent := parent
				parent = filepath.Dir(parent)
				if parent == prevParent {
					break
				}
			}

			if isInSelectedDir {
				effectiveSelection[item.Path] = true
			}
		}
	}
	return len(effectiveSelection)
}
