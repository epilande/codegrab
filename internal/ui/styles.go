package ui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/epilande/codegrab/internal/ui/themes"
	"strings"
)

// GetStyleHeader returns the header style using the current theme
func GetStyleHeader() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(themes.CurrentTheme.Colors().Primary).
		PaddingLeft(FileTreePaddingL).
		PaddingRight(FileTreePaddingR)
}

// GetStyleFormatIndicator returns the format indicator style using the current theme
func GetStyleFormatIndicator() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(themes.CurrentTheme.Colors().Highlight)
}

// GetStyleSuccess returns the success style using the current theme
func GetStyleSuccess() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(themes.CurrentTheme.Colors().Success).
		Bold(true)
}

// GetStyleError returns the error style using the current theme
func GetStyleError() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(themes.CurrentTheme.Colors().Error).
		Bold(true)
}

// GetStyleHelp returns the help style using the current theme
func GetStyleHelp() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(themes.CurrentTheme.Colors().Muted)
}

// GetStyleBorderedViewport returns the bordered viewport style using the current theme
func GetStyleBorderedViewport() lipgloss.Style {
	return lipgloss.NewStyle().
		// Use rounded border on all sides
		Border(lipgloss.RoundedBorder()).
		// Explicitly set all borders to ensure they're visible
		BorderTop(true).
		BorderRight(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderForeground(themes.CurrentTheme.Colors().Border).
		PaddingLeft(FileTreePaddingL).
		PaddingRight(FileTreePaddingR) // Use separate padding constants for left and right
}

// GetStyleHighlightedBorder returns a style for highlighted borders
func GetStyleHighlightedBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(themes.CurrentTheme.Colors().Primary)
}

// GetStylePreviewHeader returns the preview header style using the current theme
func GetStylePreviewHeader() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(themes.CurrentTheme.Colors().Secondary).
		PaddingLeft(FileTreePaddingL).
		PaddingRight(FileTreePaddingR)
}

// GetStyleSearchCount returns the search count style using the current theme
func GetStyleSearchCount() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(themes.CurrentTheme.Colors().Muted)
}

// GetStyleWarning returns the warning style using the current theme
func GetStyleWarning() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(themes.CurrentTheme.Colors().Warning).
		Bold(false).
		PaddingLeft(FileTreePaddingL).
		PaddingRight(FileTreePaddingR)
}

// GetStyleInfo returns the info style using the current theme
func GetStyleInfo() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(themes.CurrentTheme.Colors().Info).
		Bold(false).
		PaddingLeft(FileTreePaddingL).
		PaddingRight(FileTreePaddingR)
}

// GetStyleFileTreePanelHeader returns the style for the file tree panel header
func GetStyleFileTreePanelHeader() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(themes.CurrentTheme.Colors().Secondary).
		PaddingLeft(FileTreePaddingL).
		PaddingRight(FileTreePaddingR)
}

// StyleFileLine styles a file line based on its properties and the current theme
func StyleFileLine(
	rawCheckbox string,
	treePrefix string,
	icon string,
	iconColor string,
	name string,
	rawSuffix string,
	isDir bool,
	isSelected bool,
	isCursor bool,
	isPartialDir bool,
	viewportWidth int,
) string {
	colors := themes.CurrentTheme.Colors()

	checkboxStyle := lipgloss.NewStyle()
	prefixStyle := lipgloss.NewStyle().Foreground(colors.Text)
	nameStyle := lipgloss.NewStyle()
	suffixStyle := lipgloss.NewStyle().Foreground(colors.Muted)
	iconStyle := lipgloss.NewStyle()

	// Style checkbox based on selection state
	switch rawCheckbox {
	case "[x]":
		checkboxStyle = checkboxStyle.Foreground(colors.Success)
	case "[~]":
		checkboxStyle = checkboxStyle.Foreground(colors.Info)
	default:
		checkboxStyle = checkboxStyle.Foreground(colors.Muted)
	}

	// Apply icon styling
	if iconColor != "" {
		iconStyle = iconStyle.Foreground(lipgloss.Color(iconColor))
	} else {
		if isDir {
			iconStyle = iconStyle.Foreground(colors.Directory)
		} else {
			iconStyle = iconStyle.Foreground(colors.File)
		}
	}

	// Set text styling based on item type and selection state
	shouldBold := isSelected || isPartialDir
	if isDir {
		nameStyle = nameStyle.Foreground(colors.Directory)
	} else {
		if isSelected {
			nameStyle = nameStyle.Foreground(colors.Selected)
		} else {
			nameStyle = nameStyle.Foreground(colors.File)
		}
	}
	if shouldBold {
		nameStyle = nameStyle.Bold(true)
		checkboxStyle = checkboxStyle.Bold(true)
	}

	// Calculate available width for the name and suffix
	// Fixed components: checkbox (4 chars), cursor indicator (3 chars), icon (varies)
	// We need to account for the rendered width of all components
	const cursorIndicatorWidth = 3 // " ❯ "
	const leftPaddingWidth = 3     // "   "

	checkboxWidth := len(rawCheckbox) + 1 // +1 for padding
	prefixWidth := lipgloss.Width(treePrefix)
	iconWidth := 0
	if icon != "" {
		iconWidth = lipgloss.Width(icon) + 1 // +1 for space after icon
	}

	// Calculate available width for name and suffix
	baseWidth := checkboxWidth + prefixWidth + iconWidth
	availableWidth := 0

	if isCursor {
		availableWidth = viewportWidth - baseWidth - cursorIndicatorWidth
	} else {
		availableWidth = viewportWidth - baseWidth - leftPaddingWidth
	}

	// Ensure we have at least some space
	if availableWidth < 5 {
		availableWidth = 5 // Minimum width to show at least a few characters
	}

	// Truncate name if needed
	suffixWidth := lipgloss.Width(rawSuffix)
	nameWidth := lipgloss.Width(name)
	maxNameWidth := availableWidth - suffixWidth

	// If name is too long, truncate it
	truncatedName := name
	if maxNameWidth < 3 {
		maxNameWidth = 3 // Minimum to show at least "..."
	}

	if nameWidth > maxNameWidth {
		// Truncate the name and add ellipsis
		if maxNameWidth <= 3 {
			truncatedName = "..."
		} else {
			truncatedName = name[:maxNameWidth-3] + "..."
		}
	}

	// Render individual parts
	renderedCheckbox := checkboxStyle.PaddingRight(1).Render(rawCheckbox)
	renderedPrefix := prefixStyle.Render(treePrefix)
	renderedIcon := ""
	if icon != "" {
		renderedIcon = iconStyle.Render(icon + " ")
	}
	renderedName := nameStyle.Render(truncatedName)
	renderedSuffix := suffixStyle.Render(rawSuffix)

	// Handle cursor highlighting
	if isCursor {
		cursorBaseStyle := lipgloss.NewStyle().Background(colors.HighlightBackground).Bold(true)
		cursorIndicator := cursorBaseStyle.Foreground(colors.Text).Render(" ❯ ")

		cursorCheckboxStyle := checkboxStyle
		if rawCheckbox == "[ ]" {
			cursorCheckboxStyle = cursorCheckboxStyle.Foreground(colors.Text)
		}

		cursorIconStyle := iconStyle

		renderedCheckbox = cursorBaseStyle.Inherit(cursorCheckboxStyle).PaddingRight(1).Render(rawCheckbox)
		renderedPrefix = cursorBaseStyle.Inherit(prefixStyle).Render(treePrefix)
		if icon != "" {
			renderedIcon = cursorBaseStyle.Inherit(cursorIconStyle).Bold(false).Render(icon + " ")
		} else {
			renderedIcon = ""
		}
		renderedName = cursorBaseStyle.Inherit(nameStyle).Render(truncatedName)
		renderedSuffix = cursorBaseStyle.Inherit(suffixStyle).Render(rawSuffix)

		// Build the full line with cursor highlight
		cursorLineContent := fmt.Sprintf("%s%s%s%s%s",
			renderedCheckbox,
			renderedPrefix,
			renderedIcon,
			renderedName,
			renderedSuffix,
		)

		// Calculate remaining width to fill the entire line
		lineWidth := lipgloss.Width(cursorLineContent) + lipgloss.Width(cursorIndicator)
		remainingWidth := viewportWidth - lineWidth
		if remainingWidth < 0 {
			remainingWidth = 0
		}

		// Create padding to extend highlight to full width
		padding := strings.Repeat(" ", remainingWidth)
		paddingWithHighlight := cursorBaseStyle.Render(padding)

		// Add cursor indicator with proper spacing and padding for full-width highlight
		return cursorIndicator + cursorLineContent + paddingWithHighlight
	} else {
		// Build the non-cursor line with consistent spacing
		lineContent := fmt.Sprintf("%s%s%s%s%s",
			renderedCheckbox,
			renderedPrefix,
			renderedIcon,
			renderedName,
			renderedSuffix,
		)

		// Ensure consistent left padding (matches cursor indicator width)
		return "   " + lineContent
	}
}

// StylePreviewContent styles the preview content when the preview panel is focused
func StylePreviewContent(content string, isFocused bool, viewportWidth int) string {
	if !isFocused {
		return content
	}

	colors := themes.CurrentTheme.Colors()
	lines := strings.Split(content, "\n")
	styledLines := make([]string, 0, len(lines))

	highlightStyle := lipgloss.NewStyle().Background(colors.HighlightBackground)

	for _, line := range lines {
		// Ensure the highlight extends across the full width by padding the line
		// We need to account for the viewport width minus border and padding
		effectiveWidth := viewportWidth - (2 * FileTreePaddingL) - (2 * FileTreePaddingR)
		if effectiveWidth < 0 {
			effectiveWidth = 0
		}

		// Calculate how much padding we need to add to make the highlight span the full width
		lineWidth := lipgloss.Width(line)
		padding := 0
		if lineWidth < effectiveWidth {
			padding = effectiveWidth - lineWidth
		}

		// Apply the highlight style to the line with padding
		styledLine := highlightStyle.Render(line + strings.Repeat(" ", padding))
		styledLines = append(styledLines, styledLine)
	}

	// Join the styled lines back together
	return strings.Join(styledLines, "\n")
}

// NewSearchInput creates a new search input with styling from the current theme
func NewSearchInput() textinput.Model {
	colors := themes.CurrentTheme.Colors()

	ti := textinput.New()
	ti.Placeholder = "Search files..."
	ti.PromptStyle = ti.PromptStyle.
		Foreground(colors.Tertiary).
		PaddingLeft(FileTreePaddingL).
		PaddingRight(FileTreePaddingR)
	ti.TextStyle = ti.TextStyle.Foreground(colors.Tertiary)
	ti.Prompt = "🔍"
	ti.Width = 50
	return ti
}
