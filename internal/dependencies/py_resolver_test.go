package dependencies

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestPyResolver_Resolve(t *testing.T) {
	files := map[string]string{
		// Multiple import styles: aliased imports, third-party packages, standard lib
		"main.py": "import sys\nimport pkg.util as util\nimport local\nimport requests\nimport project.empty_dir\n\ndef main():\n\tprint(\"Hello\")\n\tutil.helper()\n\tlocal.do()\n\t_ = requests.get('https://example.com')",

		// Relative imports within a package structure
		"pkg/util.py": "from .nested import nested\n\ndef helper():\n\tnested.nested_func()",

		// Basic importable module for dependency checking
		"pkg/helper.py": "def another_helper():\n\tpass",

		// Handling of test files in dependency resolution
		"pkg/test_util.py": "import unittest\n\nclass TestHelper(unittest.TestCase):\n\tdef test_helper(self):\n\t\tpass",

		// Nested module dependency resolution
		"pkg/nested/nested.py": "def nested_func():\n\tpass",

		// Package-level import resolution
		"package.py": "import pkg\ndef do():\n\tpkg.helper.another_helper()",

		// Multi-level relative imports including parent directory
		"local/local.py": "from .other import other_func\nfrom ..sub.sub import sub_func\n\ndef do():\n\tsub_func()\n\nother_func()",

		// Target for relative import resolution
		"local/other.py": "def func():\n\tpass",

		// Parent directory module import resolution
		"sub/sub.py": "def sub_func():\n\tpass",

		// Invalid Python syntax error handling
		"invalid.py": "def bad_func(:", 

		// Empty file handling
		"empty.py": "# Empty but valid Python file",

		// Non-Python file handling in package directory
		"project/empty_dir/readme.md": "This directory intentionally left empty of Python files.",

		// Relative imports without explicit module name
		"no_module/main.py": "from .util import util_func\n\ndef main():\n\tpass",

		// Empty module dependency resolution
		"no_module/util.py": "# Utility module with no content",
	}

	tempDir, cleanup := setupTestEnv(t, files)
	defer cleanup()

	resolver := PyResolver{}

	tests := []struct {
		name         string
		filePath     string
		moduleName   string
		expectedDeps []string
		expectError  bool
	}{
		{
			name:     "Main file with various imports",
			filePath: "main.py",
			expectedDeps: []string{
				"local/local.py",
				"local/other.py",
				"pkg/util.py",
			},
			expectError: false,
		},
		{
			name:     "Package util file",
			filePath: "pkg/util.py",
			expectedDeps: []string{
				"pkg/nested/nested.py",
			},
			expectError: false,
		},
		{
			name:     "Local relative import",
			filePath: "local/local.py",
			expectedDeps: []string{
				"sub/sub.py",
				"local/other.py",
			},
			expectError: false,
		},
		{
			name:         "File with no relevant imports",
			filePath:     "pkg/nested/nested.py",
			expectedDeps: []string{},
			expectError:  false,
		},
		{
			name:         "Empty py file",
			filePath:     "empty.py",
			expectedDeps: []string{},
			expectError:  false,
		},
		{
			name:         "Non-existent file path",
			filePath:     "nonexistent/file.py",
			expectedDeps: nil,
			expectError:  true,
		},
		{
			name:         "Invalid py syntax",
			filePath:     "invalid.py",
			expectedDeps: nil,
			expectError:  true,
		},
		{
			name:       "Resolve without module name (should only resolve relative)",
			filePath:   "no_module/main.py",
			moduleName: "",
			expectedDeps: []string{
				"no_module/util.py",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePathAbs := filepath.Join(tempDir, filepath.FromSlash(tt.filePath))
			fileContent, readErr := os.ReadFile(filePathAbs)

			if tt.filePath == "nonexistent/file.py" {
				if !errors.Is(readErr, fs.ErrNotExist) {
					t.Fatalf("Expected os.ReadFile to fail with ErrNotExist for %q, but got: %v", tt.filePath, readErr)
				}
				if tt.expectError {
					return
				}
			}

			if readErr != nil && !tt.expectError {
				t.Fatalf("Failed to read test file %s: %v", tt.filePath, readErr)
			}


			deps, err := resolver.Resolve(fileContent, tt.filePath, tempDir, "")

			if tt.expectError {
				if err == nil && !errors.Is(readErr, fs.ErrNotExist) {
					t.Errorf("Resolve(%q) error = nil, want error (readErr was: %v)", tt.filePath, readErr)
				}
				if tt.filePath == "invalid.go" && err != nil && !strings.Contains(err.Error(), "parsing error detected") {
					t.Errorf("Resolve(%q) expected parsing error, but got: %v", tt.filePath, err)
				}
			} else { // Not expecting error
				if err != nil {
					t.Errorf("Resolve(%q) unexpected error: %v", tt.filePath, err)
				}
				if readErr != nil {
					t.Errorf("Resolve(%q) had unexpected file read error: %v", tt.filePath, readErr)
				}

				sort.Strings(deps)
				sort.Strings(tt.expectedDeps)

				if !reflect.DeepEqual(deps, tt.expectedDeps) {
					t.Errorf("Resolve(%q) deps = %v, want %v", tt.filePath, deps, tt.expectedDeps)
				}
			}
		})
	}
}
