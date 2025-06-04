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
	const moduleName = "example.com/project"
	files := map[string]string{
		"main.py":                     "import sys\nimport pkg.util as util\nimport local\nimport requests\nimport project.empty_dir\n\ndef main():\n\tprint(\"Hello\")\n\tutil.helper()\n\tlocal.do()\n\t_ = requests.get('https://example.com')",
		"pkg/util.py":                 "from .nested import nested\n\ndef helper():\n\tnested.nested_func()",
		"pkg/helper.py":               "def another_helper():\n\tpass",
		"pkg/test_util.py":            "import unittest\n\nclass TestHelper(unittest.TestCase):\n\tdef test_helper(self):\n\t\tpass",
		"pkg/nested/nested.py":        "def nested_func():\n\tpass",
		"package.py":                  "import pkg\ndef do():\n\tpkg.helper.another_helper()",
		"local/local.py":              "from .other import other_func\nfrom ..sub.sub import sub_func\n\ndef do():\n\tsub_func()\n\nother_func()",
		"local/other.py":              "def func():\n\tpass",
		"sub/sub.py":                  "def sub_func():\n\tpass",
		"invalid.py":                  "def bad_func(:", // Intentionally invalid
		"empty.py":                    "# Empty but valid Python file",
		"project/empty_dir/readme.md": "This directory intentionally left empty of Python files.",
		"no_module/main.py":           "from .util import util_func\n\ndef main():\n\tpass",
		"no_module/util.py":           "# Utility module with no content",
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

			currentModuleName := tt.moduleName
			useDefaultModule := tt.name != "Resolve without module name (should only resolve relative)" &&
				tt.name != "Resolve module path when module name unknown"

			if currentModuleName == "" && useDefaultModule {
				currentModuleName = moduleName
			}

			deps, err := resolver.Resolve(fileContent, tt.filePath, tempDir, currentModuleName)

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
