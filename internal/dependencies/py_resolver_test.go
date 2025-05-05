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
		"go.mod":                 "module example.com/project\n\ngo 1.20\n",
		"main.go":                "package main\n\nimport (\n\t\"fmt\"\n\t\"example.com/project/pkg/util\"\n\t\"./local\"\n\t\"github.com/gin-gonic/gin\"\n\t\"example.com/project/empty_dir\"\n)\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n\tutil.Helper()\n\tlocal.Do()\n\t_ = gin.Default()\n}\n",
		"pkg/util/util.go":       "package util\n\nimport \"example.com/project/pkg/nested\"\n\nfunc Helper() {\n\tnested.NestedFunc()\n}\n",
		"pkg/util/helper.go":     "package util\n\nfunc AnotherHelper(){}\n",                               // Sibling file in same package
		"pkg/util/util_test.go":  "package util\n\nimport \"testing\"\n\nfunc TestHelper(t *testing.T) {}", // Should be ignored
		"pkg/nested/nested.go":   "package nested\n\nfunc NestedFunc() {}\n",
		"local/local.go":         "package local\n\nimport (\n\t\"../other\"\n\t\"example.com/project/local/sub\"\n)\n\nfunc Do() {\n other.OtherFunc()\n sub.SubFunc()\n}\n",
		"local/sub/sub.go":       "package sub\n\nfunc SubFunc() {}",
		"other/other.go":         "package other\n\nfunc OtherFunc() {}\n",
		"invalid.go":             "package invalid\n func Bad() {\n",                            // Invalid syntax
		"empty.go":               "package empty",                                               // Empty but valid Go file
		"empty_dir/readme.md":    "This directory intentionally left empty of Go files.",        // Import should resolve dir, but find no deps
		"no_module/main.go":      "package main\n\nimport (\n\t\"./util\"\n)\n\nfunc main() {}", // For testing with empty module name
		"no_module/util/util.go": "package util",                                                // Sibling for no_module test
	}

	tempDir, cleanup := setupTestEnv(t, files)
	defer cleanup()

	resolver := GoResolver{}

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
				"pkg/util/helper.py",
				"pkg/util/util.py",
			},
			expectError: false,
		},
		{
			name:     "Package util file",
			filePath: "pkg/util/util.py",
			expectedDeps: []string{
				"pkg/nested/nested.py",
			},
			expectError: false,
		},
		{
			name:     "Local relative import",
			filePath: "local/local.py",
			expectedDeps: []string{
				"local/sub/sub.py",
				"other/other.py",
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
			name:     "Import resolves to dir with no py files",
			filePath: "main.py",
			expectedDeps: []string{
				"local/local.py",
				"pkg/util/helper.py",
				"pkg/util/util.py",
			},
			expectError: false,
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
				"no_module/util/util.py",
			},
			expectError: false,
		},
		{
			name:       "Resolve module path when module name unknown",
			filePath:   "main.py",
			moduleName: "",
			expectedDeps: []string{
				"local/local.py",
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

