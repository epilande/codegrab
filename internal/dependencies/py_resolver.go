package dependencies

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

type PyResolver struct{}

func (r *PyResolver) Resolve(fileContent []byte, filePath string, projectRoot string, projectModuleName string) ([]string, error) {
	if len(fileContent) == 0 {
		return []string{}, nil
	}
	dependencies := make(map[string]struct{})

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, fileContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Python file %s: %w", filePath, err)
	}
	defer tree.Close()

	if tree.RootNode().HasError() {
		return nil, fmt.Errorf("parsing error detected in Python file %s", filePath)
	}

	queryString := `
		[
  			(import_statement [
 	     		(dotted_name) @import_path 
 	     		(aliased_import name: (dotted_name) @import_path) 
 	   		])
			(import_from_statement 
				module_name: [
					(dotted_name) @import_path
					(relative_import) @import_path
				])
		]
	`

	query, err := sitter.NewQuery([]byte(queryString), python.GetLanguage())
	if err != nil {
		return nil, fmt.Errorf("failed to create Python query: %w", err)
	}
	defer query.Close()

	qc := sitter.NewQueryCursor()
	qc.Exec(query, tree.RootNode())
	defer qc.Close()

	fileDir := filepath.Dir(filePath)
	for {
		match, ok := qc.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			node := capture.Node
			importPath := node.Content(fileContent)
			if importPath == "" {
				continue
			}
			// Handle relative imports
			if strings.HasPrefix(importPath, ".") {
				parts := strings.Split(importPath, ".")
				currentDir := fileDir
				dotCount := 0

				// Count leading dots for relative imports
				for _, part := range parts {
					if part == "" {
						dotCount++
					} else {
						break
					}
				}

				// Move up directories based on dot count
				for i := 1; i < dotCount; i++ {
					currentDir = filepath.Dir(currentDir)
				}

				// Get the remaining path after the dots
				remainingPath := strings.Join(parts[dotCount:], "/")
				if remainingPath != "" {
					currentDir = filepath.Join(currentDir, remainingPath)
				}

				// Try as a .py file
				pyFile := currentDir + ".py"
				if fileExists(filepath.Join(projectRoot, pyFile)) {
					dependencies[filepath.ToSlash(pyFile)] = struct{}{}
					continue
				}

				// Try as a directory
				dirPath := filepath.Join(projectRoot, currentDir)
				if dirEntries, err := os.ReadDir(dirPath); err == nil {
					for _, entry := range dirEntries {
						if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
							pyFilePath := filepath.Join(currentDir, entry.Name())
							dependencies[filepath.ToSlash(pyFilePath)] = struct{}{}
						}
					}
				}
			} else {
				// Handle absolute imports
				importParts := strings.Split(importPath, ".")

				// Try direct file first
				filePath := filepath.Join(importParts...) + ".py"
				if fileExists(filepath.Join(projectRoot, filePath)) {
					dependencies[filepath.ToSlash(filePath)] = struct{}{}
					continue
				}

				// Try as directory
				dirPath := filepath.Join(projectRoot, filepath.Join(importParts...))
				if dirEntries, err := os.ReadDir(dirPath); err == nil {
					for _, entry := range dirEntries {
						if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
							pyFilePath := filepath.Join(filepath.Join(importParts...), entry.Name())
							dependencies[filepath.ToSlash(pyFilePath)] = struct{}{}
						}
					}
				}
			}
		}
	}

	depList := make([]string, 0, len(dependencies))
	for dep := range dependencies {
		depList = append(depList, dep)
	}
	return depList, nil
}
