package model

import (
	"reflect"
	"testing"

	"github.com/epilande/codegrab/internal/filesystem"
)

// TestBuildDisplayNodes tests the display node building.
// Note: All paths use forward slashes ("/") for cross-platform compatibility.
// The walker normalizes all paths to use "/" regardless of OS.
func TestBuildDisplayNodes(t *testing.T) {
	m := Model{
		selected:     make(map[string]bool),
		deselected:   make(map[string]bool),
		collapsed:    make(map[string]bool),
		isDependency: make(map[string]bool),
		files: []filesystem.FileItem{
			{Path: "dir1", IsDir: true, Level: 0},
			{Path: "dir1/file1.txt", IsDir: false, Level: 1},
			{Path: "dir2", IsDir: true, Level: 0},
			{Path: "file2.txt", IsDir: false, Level: 0},
			{Path: "file3.txt", IsDir: false, Level: 0},
		},
	}

	// Mark some files as selected
	m.selected["dir1"] = true
	m.selected["file2.txt"] = true

	// Mark dir1 as collapsed
	m.collapsed["dir1"] = true

	// Build display nodes
	m.buildDisplayNodes()

	// Expected nodes (directories first, then files, all sorted alphabetically)
	expected := []FileNode{
		{Path: "dir1", Name: "dir1", IsDir: true, Level: 0, IsLast: false, Selected: true},
		{Path: "dir2", Name: "dir2", IsDir: true, Level: 0, IsLast: false, Selected: false},
		{Path: "file2.txt", Name: "file2.txt", IsDir: false, Level: 0, IsLast: false, Selected: true},
		{Path: "file3.txt", Name: "file3.txt", IsDir: false, Level: 0, IsLast: true, Selected: false},
	}

	// Check length
	if len(m.displayNodes) != len(expected) {
		t.Fatalf("Expected %d display nodes, got %d", len(expected), len(m.displayNodes))
	}

	// Check each node
	for i, expectedNode := range expected {
		actualNode := m.displayNodes[i]
		if !reflect.DeepEqual(actualNode, expectedNode) {
			t.Errorf("Node %d mismatch:\nExpected: %+v\nActual: %+v", i, expectedNode, actualNode)
		}
	}

	// Test with expanded directory
	m.collapsed["dir1"] = false
	m.buildDisplayNodes()

	// Expected nodes with dir1 expanded
	expectedExpanded := []FileNode{
		{Path: "dir1", Name: "dir1", IsDir: true, Level: 0, IsLast: false, Selected: true},
		{Path: "dir1/file1.txt", Name: "file1.txt", IsDir: false, Level: 1, IsLast: true, Selected: false},
		{Path: "dir2", Name: "dir2", IsDir: true, Level: 0, IsLast: false, Selected: false},
		{Path: "file2.txt", Name: "file2.txt", IsDir: false, Level: 0, IsLast: false, Selected: true},
		{Path: "file3.txt", Name: "file3.txt", IsDir: false, Level: 0, IsLast: true, Selected: false},
	}

	// Check length with dir1 expanded
	if len(m.displayNodes) != len(expectedExpanded) {
		t.Fatalf("Expected %d display nodes with dir1 expanded, got %d", len(expectedExpanded), len(m.displayNodes))
	}

	// Check each node with dir1 expanded
	for i, expectedNode := range expectedExpanded {
		actualNode := m.displayNodes[i]
		if !reflect.DeepEqual(actualNode, expectedNode) {
			t.Errorf("Node %d mismatch (expanded):\nExpected: %+v\nActual: %+v", i, expectedNode, actualNode)
		}
	}

	// Check that dir1/file1.txt is now included
	found := false
	for _, node := range m.displayNodes {
		if node.Path == "dir1/file1.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected dir1/file1.txt to be included in display nodes when dir1 is expanded")
	}
}

// TestBuildDisplayNodesDeeplyNested tests that deeply nested paths work correctly.
// This is important for cross-platform compatibility since paths must use "/"
// consistently (not os.PathSeparator which would be "\" on Windows).
func TestBuildDisplayNodesDeeplyNested(t *testing.T) {
	m := Model{
		selected:     make(map[string]bool),
		deselected:   make(map[string]bool),
		collapsed:    make(map[string]bool),
		isDependency: make(map[string]bool),
		files: []filesystem.FileItem{
			{Path: "a", IsDir: true, Level: 0},
			{Path: "a/b", IsDir: true, Level: 1},
			{Path: "a/b/c", IsDir: true, Level: 2},
			{Path: "a/b/c/d", IsDir: true, Level: 3},
			{Path: "a/b/c/d/file.txt", IsDir: false, Level: 4},
			{Path: "a/b/other.txt", IsDir: false, Level: 2},
		},
	}

	m.buildDisplayNodes()

	// With all directories expanded (default), we should see all nodes
	expectedPaths := []string{
		"a",
		"a/b",
		"a/b/c",
		"a/b/c/d",
		"a/b/c/d/file.txt",
		"a/b/other.txt",
	}

	if len(m.displayNodes) != len(expectedPaths) {
		t.Fatalf("Expected %d display nodes, got %d", len(expectedPaths), len(m.displayNodes))
	}

	for i, expectedPath := range expectedPaths {
		if m.displayNodes[i].Path != expectedPath {
			t.Errorf("Node %d: expected path %q, got %q", i, expectedPath, m.displayNodes[i].Path)
		}
	}

	// Collapse a/b and verify children are hidden
	m.collapsed["a/b"] = true
	m.buildDisplayNodes()

	// Should only show a and a/b (children of a/b are hidden)
	if len(m.displayNodes) != 2 {
		t.Fatalf("Expected 2 display nodes when a/b is collapsed, got %d", len(m.displayNodes))
	}
	if m.displayNodes[0].Path != "a" {
		t.Errorf("Expected first node to be 'a', got %q", m.displayNodes[0].Path)
	}
	if m.displayNodes[1].Path != "a/b" {
		t.Errorf("Expected second node to be 'a/b', got %q", m.displayNodes[1].Path)
	}
}
