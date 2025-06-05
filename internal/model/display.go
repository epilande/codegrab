package model

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/epilande/codegrab/internal/filesystem"
	"github.com/epilande/go-devicons"
)

// buildDisplayNodes constructs a hierarchical view of files and directories for display.
func (m *Model) buildDisplayNodes() {
	m.displayNodes = nil
	nodesToAdd := make(map[string]filesystem.FileItem)
	for _, f := range m.files {
		nodesToAdd[f.Path] = f
	}

	var rootItems []filesystem.FileItem
	processed := make(map[string]bool)

	var addNode func(item filesystem.FileItem, level int)
	addNode = func(item filesystem.FileItem, level int) {
		if processed[item.Path] {
			return
		}
		processed[item.Path] = true

		icon := ""
		iconColor := ""
		if m.showIcons {
			fullPath := filepath.Join(m.rootPath, item.Path)
			style := devicons.IconForPath(fullPath)
			icon = style.Icon
			iconColor = style.Color
		}

		node := FileNode{
			Path:         item.Path,
			Name:         filepath.Base(item.Path),
			IsDir:        item.IsDir,
			Level:        level,
			Selected:     m.selected[item.Path],
			IsDeselected: m.deselected[item.Path],
			IsDependency: m.isDependency[item.Path],
			Icon:         icon,
			IconColor:    iconColor,
		}
		m.displayNodes = append(m.displayNodes, node)

		if item.IsDir && !m.collapsed[item.Path] {
			prefix := item.Path + string(os.PathSeparator)
			var children []filesystem.FileItem
			directChildren := make(map[string]filesystem.FileItem)

			for _, f := range m.files {
				if strings.HasPrefix(f.Path, prefix) {
					sub := strings.TrimPrefix(f.Path, prefix)
					if !strings.Contains(sub, string(os.PathSeparator)) {
						directChildren[f.Path] = f
					}
				}
			}

			for _, childItem := range directChildren {
				children = append(children, childItem)
			}

			sort.Slice(children, func(i, j int) bool {
				if children[i].IsDir != children[j].IsDir {
					return children[i].IsDir
				}
				return children[i].Path < children[j].Path
			})

			for _, child := range children {
				addNode(child, level+1)
			}
		}
	}

	for _, item := range m.files {
		if !strings.Contains(item.Path, string(os.PathSeparator)) {
			rootItems = append(rootItems, item)
		}
	}

	sort.Slice(rootItems, func(i, j int) bool {
		if rootItems[i].IsDir != rootItems[j].IsDir {
			return rootItems[i].IsDir
		}
		return rootItems[i].Path < rootItems[j].Path
	})

	for _, rootItem := range rootItems {
		addNode(rootItem, 0)
	}

	if len(m.displayNodes) > 0 {
		levelLast := make(map[int]string)
		for _, node := range m.displayNodes {
			levelLast[node.Level] = node.Path
		}

		for i := range m.displayNodes {
			node := &m.displayNodes[i]
			node.IsLast = (node.Path == levelLast[node.Level])
		}
	}
}
