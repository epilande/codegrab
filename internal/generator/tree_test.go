package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildTree tests tree building from selected files.
// Note: All internal paths use forward slashes ("/") for cross-platform compatibility.
// The walker normalizes paths with filepath.ToSlash(), so tree building must use
// path.Join (not filepath.Join) to maintain consistency.
func TestBuildTree(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tree-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFiles := []struct {
		path    string
		content string
	}{
		{"file1.txt", "Content of file1"},
		{"file2.go", "package main\n\nfunc main() {}"},
		{"subdir/file3.txt", "Content of file3"},
		{"subdir/nested/file4.js", "console.log('Hello');"},
	}

	for _, tf := range testFiles {
		path := filepath.Join(tempDir, filepath.FromSlash(tf.path))
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(path, []byte(tf.content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	gen := NewGenerator(tempDir, nil, nil, "", false)

	gen.SelectedFiles = map[string]bool{
		"file1.txt":              true,
		"file2.go":               true,
		"subdir/file3.txt":       true,
		"subdir/nested/file4.js": true,
	}

	root := gen.buildTree()

	if root.Name != filepath.Base(tempDir) {
		t.Errorf("Expected root name to be %q, got %q", filepath.Base(tempDir), root.Name)
	}
	if !root.IsDir {
		t.Errorf("Expected root to be a directory")
	}

	if len(root.Children) != 3 {
		t.Errorf("Expected root to have 3 children, got %d", len(root.Children))
	}

	foundFiles := make(map[string]bool)
	var collectFilesTest func(*Node)
	collectFilesTest = func(node *Node) {
		if !node.IsDir {
			foundFiles[node.Path] = true
		}
		for _, child := range node.Children {
			collectFilesTest(child)
		}
	}
	collectFilesTest(root)

	// Test lazy loading by using the actual collectFiles function
	var files []FileData
	collectFiles(root, &files, tempDir, nil)

	if len(files) != len(testFiles) {
		t.Errorf("Expected %d files from collectFiles, got %d", len(testFiles), len(files))
	}

	contentMap := make(map[string]string)
	for _, file := range files {
		contentMap[file.Path] = file.Content
	}

	for _, tf := range testFiles {
		if content, ok := contentMap[tf.path]; ok {
			if content != tf.content {
				t.Errorf("Expected content of %q to be %q, got %q", tf.path, tf.content, content)
			}
		}
	}

	for _, tf := range testFiles {
		if !foundFiles[tf.path] {
			t.Errorf("Expected file %q to be in the tree", tf.path)
		}
	}
}

func TestRenderTree(t *testing.T) {
	root := &Node{
		Children: []*Node{
			{
				Name:  "file1.txt",
				Path:  "file1.txt",
				IsDir: false,
			},
			{
				Name:  "subdir",
				Path:  "subdir",
				IsDir: true,
				Children: []*Node{
					{
						Name:  "file2.txt",
						Path:  "subdir/file2.txt",
						IsDir: false,
					},
				},
			},
		},
	}

	var builder strings.Builder
	renderTree(root.Children[0], "", true, &builder, "root", nil)
	renderTree(root.Children[1], "", false, &builder, "root", nil)

	expected := "└── file1.txt\n├── subdir/\n│   └── file2.txt\n"
	if builder.String() != expected {
		t.Errorf("Expected tree rendering to be:\n%s\nGot:\n%s", expected, builder.String())
	}
}

func TestRenderTreeWithDeselectedFiles(t *testing.T) {
	root := &Node{
		Children: []*Node{
			{
				Name:  "subdir",
				Path:  "subdir",
				IsDir: true,
				Children: []*Node{
					{
						Name:  "file2.txt",
						Path:  "subdir/file2.txt",
						IsDir: false,
					},
					{
						Name:  "file3.txt",
						Path:  "subdir/file3.txt",
						IsDir: false,
					},
				},
			},
			{
				Name:  "file1.txt",
				Path:  "file1.txt",
				IsDir: false,
			},
		},
	}

	deselected := map[string]bool{
		"subdir/file2.txt": true,
	}

	var builder strings.Builder
	renderTree(root.Children[0], "", false, &builder, "root", deselected)
	renderTree(root.Children[1], "", true, &builder, "root", nil)

	expected := "├── subdir/\n│   └── file3.txt\n└── file1.txt\n"
	if builder.String() != expected {
		t.Errorf("Expected tree rendering to be:\n%s\nGot:\n%s", expected, builder.String())
	}
}

func TestDetermineLanguage(t *testing.T) {
	testCases := []struct {
		filename string
		expected string
	}{
		{"file.go", "go"},
		{"script.js", "javascript"},
		{"code.py", "python"},
		{"module.rs", "rust"},
		{"app.ts", "typescript"},
		{"index.html", "html"},
		{"styles.css", "css"},
		{"README.md", "markdown"},
		{"config.json", "json"},
		{"settings.yaml", "yaml"},
		{"data.yml", "yaml"},
		{"unknown.xyz", "xyz"},
		{"noextension", "text"},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := determineLanguage(tc.filename)
			if result != tc.expected {
				t.Errorf("determineLanguage(%q) = %q, want %q", tc.filename, result, tc.expected)
			}
		})
	}
}

func TestSortTree(t *testing.T) {
	root := &Node{
		Name:  "root",
		IsDir: true,
		Children: []*Node{
			{
				Name:  "file.txt",
				IsDir: false,
			},
			{
				Name:  "adir",
				IsDir: true,
			},
			{
				Name:  "bfile.txt",
				IsDir: false,
			},
			{
				Name:  "cdir",
				IsDir: true,
			},
		},
	}

	sortTree(root)

	expectedOrder := []struct {
		name  string
		isDir bool
	}{
		{"adir", true},
		{"cdir", true},
		{"bfile.txt", false},
		{"file.txt", false},
	}

	if len(root.Children) != len(expectedOrder) {
		t.Fatalf("Expected %d children, got %d", len(expectedOrder), len(root.Children))
	}

	for i, expected := range expectedOrder {
		if root.Children[i].Name != expected.name || root.Children[i].IsDir != expected.isDir {
			t.Errorf("Expected child %d to be %q (isDir=%v), got %q (isDir=%v)",
				i, expected.name, expected.isDir, root.Children[i].Name, root.Children[i].IsDir)
		}
	}
}

func TestCollectFiles(t *testing.T) {
	// Create temporary files for testing
	tempDir, err := os.MkdirTemp("", "collect-files-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFiles := []struct {
		path    string
		content string
	}{
		{"file1.txt", "Content 1"},
		{"subdir/file2.go", "Content 2"},
	}

	// Create the test files
	for _, tf := range testFiles {
		path := filepath.Join(tempDir, filepath.FromSlash(tf.path))
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(path, []byte(tf.content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	root := &Node{
		Name:  "root",
		IsDir: true,
		Children: []*Node{
			{
				Name:     "file1.txt",
				Path:     "file1.txt",
				Language: "text",
				IsDir:    false,
			},
			{
				Name:  "subdir",
				Path:  "subdir",
				IsDir: true,
				Children: []*Node{
					{
						Name:     "file2.go",
						Path:     "subdir/file2.go",
						Language: "go",
						IsDir:    false,
					},
				},
			},
		},
	}

	var files []FileData
	collectFiles(root, &files, tempDir, nil)

	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(files))
	}

	fileMap := make(map[string]FileData)
	for _, file := range files {
		fileMap[file.Path] = file
	}

	file1, ok := fileMap["file1.txt"]
	if !ok {
		t.Fatalf("Expected to find file1.txt")
	}
	if file1.Content != "Content 1" || file1.Language != "text" {
		t.Errorf("Expected file1.txt to have content %q and language %q, got %q and %q",
			"Content 1", "text", file1.Content, file1.Language)
	}

	file2, ok := fileMap["subdir/file2.go"]
	if !ok {
		t.Fatalf("Expected to find subdir/file2.go")
	}
	if file2.Content != "Content 2" || file2.Language != "go" {
		t.Errorf("Expected subdir/file2.go to have content %q and language %q, got %q and %q",
			"Content 2", "go", file2.Content, file2.Language)
	}
}
