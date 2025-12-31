package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/epilande/codegrab/internal/cache"
	"github.com/epilande/codegrab/internal/filesystem"
)

func TestNewGenerator(t *testing.T) {
	rootPath := "."
	gitIgnoreMgr, _ := filesystem.NewGitIgnoreManager(rootPath)
	filterMgr := filesystem.NewFilterManager()
	outputPath := "test-output.md"
	useTempFile := false

	gen := NewGenerator(rootPath, gitIgnoreMgr, filterMgr, outputPath, useTempFile)

	if gen.RootPath != rootPath {
		t.Errorf("Expected RootPath to be %q, got %q", rootPath, gen.RootPath)
	}
	if gen.OutputPath != outputPath {
		t.Errorf("Expected OutputPath to be %q, got %q", outputPath, gen.OutputPath)
	}
	if gen.UseTempFile != useTempFile {
		t.Errorf("Expected UseTempFile to be %v, got %v", useTempFile, gen.UseTempFile)
	}
	if gen.GitIgnoreMgr != gitIgnoreMgr {
		t.Errorf("Expected GitIgnoreMgr to be the provided instance")
	}
	if gen.FilterMgr != filterMgr {
		t.Errorf("Expected FilterMgr to be the provided instance")
	}
	if len(gen.SelectedFiles) != 0 {
		t.Errorf("Expected SelectedFiles to be empty, got %d items", len(gen.SelectedFiles))
	}
	if len(gen.DeselectedFiles) != 0 {
		t.Errorf("Expected DeselectedFiles to be empty, got %d items", len(gen.DeselectedFiles))
	}
	if !gen.UseGitIgnore {
		t.Errorf("Expected UseGitIgnore to be true by default")
	}
	if gen.ShowHidden {
		t.Errorf("Expected ShowHidden to be false by default")
	}
}

func TestSetFormat(t *testing.T) {
	mockFormat := &mockFormat{
		name:      "mock",
		extension: ".mock",
	}

	gen := NewGenerator(".", nil, nil, "", false)

	gen.SetFormat(mockFormat)

	if gen.format != mockFormat {
		t.Errorf("Expected format to be set to the mock format")
	}

	if gen.GetFormat() != mockFormat {
		t.Errorf("GetFormat() should return the set format")
	}

	if gen.GetFormatName() != "mock" {
		t.Errorf("Expected GetFormatName() to return %q, got %q", "mock", gen.GetFormatName())
	}
}

func TestGetFormatNameWithNoFormat(t *testing.T) {
	gen := NewGenerator(".", nil, nil, "", false)

	if gen.GetFormatName() != "unknown" {
		t.Errorf("Expected GetFormatName() to return %q when no format is set, got %q", "unknown", gen.GetFormatName())
	}
}

func TestPrepareTemplateData(t *testing.T) {
	// Reset cache to ensure clean state
	cache.ResetGlobalCache()

	tempDir, err := os.MkdirTemp("", "generator-test")
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

	gitIgnoreMgr, _ := filesystem.NewGitIgnoreManager(tempDir)
	filterMgr := filesystem.NewFilterManager()
	gen := NewGenerator(tempDir, gitIgnoreMgr, filterMgr, "", false)

	gen.SelectedFiles = map[string]bool{
		"file1.txt":        true,
		"file2.go":         true,
		"subdir/file3.txt": true,
	}

	data, err := gen.PrepareTemplateData()
	if err != nil {
		t.Fatalf("PrepareTemplateData failed: %v", err)
	}

	if data.Structure == "" {
		t.Errorf("Expected Structure to be non-empty")
	}

	if len(data.Files) != 3 {
		t.Errorf("Expected 3 files in template data, got %d", len(data.Files))
	}

	fileMap := make(map[string]FileData)
	for _, file := range data.Files {
		fileMap[file.Path] = file
	}

	for _, tf := range testFiles {
		file, ok := fileMap[tf.path]
		if !ok {
			t.Errorf("Expected file %q to be included in template data", tf.path)
			continue
		}

		if file.Content != tf.content {
			t.Errorf("Expected content of %q to be %q, got %q", tf.path, tf.content, file.Content)
		}

		if tf.path == "file2.go" && file.Language != "go" {
			t.Errorf("Expected language for %q to be %q, got %q", tf.path, "go", file.Language)
		}
	}
}

func TestGenerateString(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "generator-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	gitIgnoreMgr, _ := filesystem.NewGitIgnoreManager(tempDir)
	filterMgr := filesystem.NewFilterManager()
	gen := NewGenerator(tempDir, gitIgnoreMgr, filterMgr, "", false)

	mockFormat := &mockFormat{
		name:      "mock",
		extension: ".mock",
		content:   "Mock content with 1 file",
		tokens:    10,
	}
	gen.SetFormat(mockFormat)

	gen.SelectedFiles = map[string]bool{
		"test.txt": true,
	}

	content, tokens, _, err := gen.GenerateString()
	if err != nil {
		t.Fatalf("GenerateString failed: %v", err)
	}

	if content != "Mock content with 1 file" {
		t.Errorf("Expected content to be %q, got %q", "Mock content with 1 file", content)
	}
	if tokens != 10 {
		t.Errorf("Expected tokens to be %d, got %d", 10, tokens)
	}
}

func TestGenerateStringWithNoFiles(t *testing.T) {
	gen := NewGenerator(".", nil, nil, "", false)
	gen.SetFormat(&mockFormat{})

	_, _, _, err := gen.GenerateString()
	if err == nil {
		t.Errorf("Expected error when no files are selected")
	}
}

func TestGenerateStringWithNoFormat(t *testing.T) {
	gen := NewGenerator(".", nil, nil, "", false)
	gen.SelectedFiles = map[string]bool{"test.txt": true}

	_, _, _, err := gen.GenerateString()
	if err == nil {
		t.Errorf("Expected error when no format is set")
	}
}

func TestSetTreeOnlyMode(t *testing.T) {
	gen := NewGenerator(".", nil, nil, "", false)

	if gen.TreeOnly {
		t.Errorf("Expected TreeOnly to be false by default")
	}

	gen.SetTreeOnlyMode(true)
	if !gen.TreeOnly {
		t.Errorf("Expected TreeOnly to be true after SetTreeOnlyMode(true)")
	}

	gen.SetTreeOnlyMode(false)
	if gen.TreeOnly {
		t.Errorf("Expected TreeOnly to be false after SetTreeOnlyMode(false)")
	}
}

func TestPrepareTemplateDataTreeOnly(t *testing.T) {
	cache.ResetGlobalCache()

	tempDir, err := os.MkdirTemp("", "generator-test-treeonly")
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

	gitIgnoreMgr, _ := filesystem.NewGitIgnoreManager(tempDir)
	filterMgr := filesystem.NewFilterManager()
	gen := NewGenerator(tempDir, gitIgnoreMgr, filterMgr, "", false)
	gen.SelectedFiles = map[string]bool{
		"file1.txt":        true,
		"file2.go":         true,
		"subdir/file3.txt": true,
	}
	gen.SetTreeOnlyMode(true)

	data, err := gen.PrepareTemplateData()
	if err != nil {
		t.Fatalf("PrepareTemplateData failed: %v", err)
	}

	if data.Structure == "" {
		t.Errorf("Expected Structure to be non-empty in tree-only mode")
	}
	if len(data.Files) != 0 {
		t.Errorf("Expected Files to be empty in tree-only mode, got %d files", len(data.Files))
	}
}

type mockFormat struct {
	err       error
	name      string
	extension string
	content   string
	tokens    int
}

func (m *mockFormat) Render(data TemplateData) (string, int, error) {
	if m.err != nil {
		return "", 0, m.err
	}
	return m.content, m.tokens, nil
}

func (m *mockFormat) Extension() string {
	return m.extension
}

func (m *mockFormat) Name() string {
	return m.name
}
