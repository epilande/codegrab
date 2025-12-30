package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epilande/codegrab/internal/secrets"
)

// BenchmarkFileCollection compares sequential vs concurrent file reading
func BenchmarkFileCollection(b *testing.B) {
	// Create test directory structure with files
	tmpDir, err := os.MkdirTemp("", "generator-benchmark-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Setup test files
	setupBenchmarkFiles(b, tmpDir)

	// Build test tree
	tree := createTestTree(tmpDir)

	// Initialize secret scanner (not used in benchmark for performance testing)
	_, scannerErr := secrets.NewGitleaksScanner()
	if scannerErr != nil {
		b.Fatalf("Failed to create secret scanner: %v", scannerErr)
	}

	b.Run("Concurrent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var files []FileData
			collector := NewConcurrentFileCollector(tmpDir)
			result, err := collector.CollectFilesConcurrent(tree, nil) // Pass nil for now, secret scanning happens later
			if err != nil {
				b.Fatalf("Concurrent collection failed: %v", err)
			}
			files = result
			if len(files) == 0 {
				b.Fatalf("No files collected")
			}
		}
	})

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var files []FileData
			collectFilesSequential(tree, &files, tmpDir, nil) // Pass nil for now
			if len(files) == 0 {
				b.Fatalf("No files collected")
			}
		}
	})
}

// BenchmarkLargeFileCollection tests with many files
func BenchmarkLargeFileCollection(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "generator-large-benchmark-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create larger test structure
	setupLargeBenchmarkFiles(b, tmpDir)
	tree := createTestTree(tmpDir)

	_, scannerErr := secrets.NewGitleaksScanner()
	if scannerErr != nil {
		b.Fatalf("Failed to create secret scanner: %v", scannerErr)
	}

	b.Run("Concurrent-Large", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			collector := NewConcurrentFileCollector(tmpDir)
			files, err := collector.CollectFilesConcurrent(tree, nil) // Pass nil for now
			if err != nil {
				b.Fatalf("Concurrent collection failed: %v", err)
			}
			if len(files) == 0 {
				b.Fatalf("No files collected")
			}
		}
	})

	b.Run("Sequential-Large", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var files []FileData
			collectFilesSequential(tree, &files, tmpDir, nil) // Pass nil for now
			if len(files) == 0 {
				b.Fatalf("No files collected")
			}
		}
	})
}

// setupBenchmarkFiles creates realistic test files
func setupBenchmarkFiles(b *testing.B, baseDir string) {
	files := []struct {
		path    string
		content string
	}{
		{"main.go", generateGoFileContent("main", 100)},
		{"utils/helper.go", generateGoFileContent("utils", 200)},
		{"utils/parser.go", generateGoFileContent("utils", 150)},
		{"internal/config.go", generateGoFileContent("internal", 300)},
		{"internal/service.go", generateGoFileContent("internal", 250)},
		{"cmd/cli.go", generateGoFileContent("main", 400)},
		{"README.md", generateMarkdownContent(50)},
		{"config.yaml", generateYamlContent()},
	}

	for _, file := range files {
		fullPath := filepath.Join(baseDir, file.path)
		dir := filepath.Dir(fullPath)
		
		if err := os.MkdirAll(dir, 0755); err != nil {
			b.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		
		if err := os.WriteFile(fullPath, []byte(file.content), 0644); err != nil {
			b.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}
}

// setupLargeBenchmarkFiles creates many files for stress testing
func setupLargeBenchmarkFiles(b *testing.B, baseDir string) {
	// Create 50 directories with 5 files each = 250 files
	for i := 0; i < 50; i++ {
		for j := 0; j < 5; j++ {
			dir := filepath.Join(baseDir, fmt.Sprintf("pkg%d", i))
			if err := os.MkdirAll(dir, 0755); err != nil {
				b.Fatalf("Failed to create dir %s: %v", dir, err)
			}

			fileName := filepath.Join(dir, fmt.Sprintf("file%d.go", j))
			content := generateGoFileContent(fmt.Sprintf("pkg%d", i), 100+j*50)
			
			if err := os.WriteFile(fileName, []byte(content), 0644); err != nil {
				b.Fatalf("Failed to create file %s: %v", fileName, err)
			}
		}
	}
}

// createTestTree builds a Node tree from the directory structure
func createTestTree(rootPath string) *Node {
	root := &Node{
		Name:     filepath.Base(rootPath),
		IsDir:    true,
		Children: []*Node{},
		Path:     "",
	}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == rootPath {
			return err
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}

		// Build path segments
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		current := root

		for i, part := range parts {
			isLast := i == len(parts)-1
			found := false

			// Look for existing child
			for _, child := range current.Children {
				if child.Name == part {
					current = child
					found = true
					break
				}
			}

			// Create new node if not found
			if !found {
				newNode := &Node{
					Name:     part,
					IsDir:    !isLast || info.IsDir(),
					Children: []*Node{},
					Path:     relPath,
				}
				
				if !newNode.IsDir {
					newNode.Language = determineLanguage(part)
				}

				current.Children = append(current.Children, newNode)
				current = newNode
			}
		}
		return nil
	})

	if err != nil {
		return root
	}

	sortTree(root)
	return root
}

// Helper functions to generate test content
func generateGoFileContent(packageName string, lines int) string {
	content := fmt.Sprintf("package %s\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\n", packageName)
	
	for i := 0; i < lines; i++ {
		content += fmt.Sprintf("// Line %d of generated content\n", i)
		if i%10 == 0 {
			content += fmt.Sprintf("func Function%d() {\n\tfmt.Println(\"Function %d\")\n}\n\n", i/10, i/10)
		}
	}
	
	return content
}

func generateMarkdownContent(lines int) string {
	content := "# Test README\n\nThis is a test README file.\n\n"
	for i := 0; i < lines; i++ {
		content += fmt.Sprintf("- Item %d in the list\n", i)
	}
	return content
}

func generateYamlContent() string {
	return `
name: test-project
version: 1.0.0
description: Test project for benchmarking
dependencies:
  - go
  - yaml
settings:
  debug: true
  port: 8080
`
}