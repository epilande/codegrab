package formats

import (
	"strings"
	"testing"

	"github.com/epilande/codegrab/internal/generator"
)

func TestGetFormat(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
	}{
		{"markdown", "markdown"},
		{"text", "text"},
		{"xml", "xml"},
		{"unknown", "markdown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			format := GetFormat(tc.name)
			if format.Name() != tc.expected {
				t.Errorf("GetFormat(%q) returned format with name %q, expected %q",
					tc.name, format.Name(), tc.expected)
			}
		})
	}
}

func TestGetFormatNames(t *testing.T) {
	names := GetFormatNames()

	expectedFormats := []string{"markdown", "text", "xml"}
	for _, expected := range expectedFormats {
		found := false
		for _, name := range names {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected format %q to be in the list of format names", expected)
		}
	}
}

func TestMarkdownFormat(t *testing.T) {
	format := &MarkdownFormat{}

	if format.Name() != "markdown" {
		t.Errorf("Expected Name() to return %q, got %q", "markdown", format.Name())
	}
	if format.Extension() != ".md" {
		t.Errorf("Expected Extension() to return %q, got %q", ".md", format.Extension())
	}

	data := createTestTemplateData()
	content, tokens, err := format.Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(content, "# Project Structure") {
		t.Errorf("Expected content to contain '# Project Structure'")
	}
	if !strings.Contains(content, "```go") {
		t.Errorf("Expected content to contain '```go'")
	}
	if !strings.Contains(content, "package main") {
		t.Errorf("Expected content to contain 'package main'")
	}

	if tokens <= 0 {
		t.Errorf("Expected tokens to be positive, got %d", tokens)
	}
}

func TestTxtFormat(t *testing.T) {
	format := &TxtFormat{}

	if format.Name() != "text" {
		t.Errorf("Expected Name() to return %q, got %q", "text", format.Name())
	}
	if format.Extension() != ".txt" {
		t.Errorf("Expected Extension() to return %q, got %q", ".txt", format.Extension())
	}

	data := createTestTemplateData()
	content, tokens, err := format.Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(content, "PROJECT STRUCTURE") {
		t.Errorf("Expected content to contain 'PROJECT STRUCTURE'")
	}
	if !strings.Contains(content, "FILE: main.go") {
		t.Errorf("Expected content to contain 'FILE: main.go'")
	}
	if !strings.Contains(content, "package main") {
		t.Errorf("Expected content to contain 'package main'")
	}

	if tokens <= 0 {
		t.Errorf("Expected tokens to be positive, got %d", tokens)
	}
}

func TestXMLFormat(t *testing.T) {
	format := &XMLFormat{}

	if format.Name() != "xml" {
		t.Errorf("Expected Name() to return %q, got %q", "xml", format.Name())
	}
	if format.Extension() != ".xml" {
		t.Errorf("Expected Extension() to return %q, got %q", ".xml", format.Extension())
	}

	data := createTestTemplateData()
	content, tokens, err := format.Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(content, "<?xml") {
		t.Errorf("Expected content to contain '<?xml'")
	}
	if !strings.Contains(content, "<project>") {
		t.Errorf("Expected content to contain '<project>'")
	}
	if !strings.Contains(content, "<file path=\"main.go\"") {
		t.Errorf("Expected content to contain '<file path=\"main.go\"'")
	}
	if !strings.Contains(content, "package main") {
		t.Errorf("Expected content to contain 'package main'")
	}

	if tokens <= 0 {
		t.Errorf("Expected tokens to be positive, got %d", tokens)
	}
}

func createTestTemplateData() generator.TemplateData {
	return generator.TemplateData{
		Structure: "test-project/\n└── main.go\n",
		Files: []generator.FileData{
			{
				Path:     "main.go",
				Content:  "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n",
				Language: "go",
			},
		},
	}
}

func createTreeOnlyTemplateData() generator.TemplateData {
	return generator.TemplateData{
		Structure: "test-project/\n├── src/\n│   └── main.go\n└── README.md\n",
		Files:     []generator.FileData{},
	}
}

func TestMarkdownFormatTreeOnly(t *testing.T) {
	format := &MarkdownFormat{}
	data := createTreeOnlyTemplateData()

	content, tokens, err := format.Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(content, "# Project Structure") {
		t.Errorf("Expected content to contain '# Project Structure'")
	}
	if !strings.Contains(content, "test-project/") {
		t.Errorf("Expected content to contain 'test-project/'")
	}
	if strings.Contains(content, "# Project Files") {
		t.Errorf("Expected content to NOT contain '# Project Files' in tree-only mode")
	}
	if tokens <= 0 {
		t.Errorf("Expected tokens to be positive, got %d", tokens)
	}
}

func TestTxtFormatTreeOnly(t *testing.T) {
	format := &TxtFormat{}
	data := createTreeOnlyTemplateData()

	content, tokens, err := format.Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(content, "PROJECT STRUCTURE") {
		t.Errorf("Expected content to contain 'PROJECT STRUCTURE'")
	}
	if !strings.Contains(content, "test-project/") {
		t.Errorf("Expected content to contain 'test-project/'")
	}
	if strings.Contains(content, "PROJECT FILES") {
		t.Errorf("Expected content to NOT contain 'PROJECT FILES' in tree-only mode")
	}
	if tokens <= 0 {
		t.Errorf("Expected tokens to be positive, got %d", tokens)
	}
}

func TestXMLFormatTreeOnly(t *testing.T) {
	format := &XMLFormat{}
	data := createTreeOnlyTemplateData()

	content, tokens, err := format.Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(content, "<?xml") {
		t.Errorf("Expected content to contain '<?xml'")
	}
	if !strings.Contains(content, "<project>") {
		t.Errorf("Expected content to contain '<project>'")
	}
	if strings.Contains(content, "<file path=") {
		t.Errorf("Expected content to NOT contain file elements in tree-only mode")
	}
	if tokens <= 0 {
		t.Errorf("Expected tokens to be positive, got %d", tokens)
	}
}

func TestAddFileToTree(t *testing.T) {
	root := &directoryEntry{
		name:    ".",
		subdirs: make(map[string]*directoryEntry),
		files:   []string{},
	}

	addFileToTree(root, "file1.txt")
	addFileToTree(root, "dir1/file2.txt")
	addFileToTree(root, "dir1/subdir/file3.txt")

	if len(root.files) != 1 || root.files[0] != "file1.txt" {
		t.Errorf("Expected root to have 1 file 'file1.txt', got %v", root.files)
	}

	dir1, exists := root.subdirs["dir1"]
	if !exists {
		t.Fatalf("Expected to find dir1 in root subdirs")
	}
	if len(dir1.files) != 1 || dir1.files[0] != "file2.txt" {
		t.Errorf("Expected dir1 to have 1 file 'file2.txt', got %v", dir1.files)
	}

	subdir, exists := dir1.subdirs["subdir"]
	if !exists {
		t.Fatalf("Expected to find subdir in dir1 subdirs")
	}
	if len(subdir.files) != 1 || subdir.files[0] != "file3.txt" {
		t.Errorf("Expected subdir to have 1 file 'file3.txt', got %v", subdir.files)
	}
}

func TestConvertToXMLDirectory(t *testing.T) {
	root := &directoryEntry{
		name:    "root",
		subdirs: make(map[string]*directoryEntry),
		files:   []string{"file1.txt", "file2.txt"},
	}

	subdir := &directoryEntry{
		name:    "subdir",
		subdirs: make(map[string]*directoryEntry),
		files:   []string{"file3.txt"},
	}

	root.subdirs["subdir"] = subdir

	xmlDir := convertToXMLDirectory(root)

	if xmlDir.Name != "root" {
		t.Errorf("Expected Name to be %q, got %q", "root", xmlDir.Name)
	}

	if len(xmlDir.Files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(xmlDir.Files))
	}

	if xmlDir.Files[0].Name != "file1.txt" || xmlDir.Files[1].Name != "file2.txt" {
		t.Errorf("Expected files to be sorted, got %q and %q", xmlDir.Files[0].Name, xmlDir.Files[1].Name)
	}

	if len(xmlDir.Directories) != 1 {
		t.Fatalf("Expected 1 subdirectory, got %d", len(xmlDir.Directories))
	}

	if xmlDir.Directories[0].Name != "subdir" {
		t.Errorf("Expected subdirectory name to be %q, got %q", "subdir", xmlDir.Directories[0].Name)
	}

	if len(xmlDir.Directories[0].Files) != 1 || xmlDir.Directories[0].Files[0].Name != "file3.txt" {
		t.Errorf("Expected subdir to have 1 file 'file3.txt', got %v", xmlDir.Directories[0].Files)
	}
}
