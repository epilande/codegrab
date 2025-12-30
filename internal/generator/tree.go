package generator

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/epilande/codegrab/internal/cache"
	"github.com/epilande/codegrab/internal/secrets"
	"github.com/epilande/codegrab/internal/utils"
)

// Node represents a node in the in-memory file tree.
type Node struct {
	Name     string
	Path     string
	Content  string
	Language string
	Children []*Node
	IsDir    bool
	Findings []secrets.Finding
}

func (g *Generator) buildTree() *Node {
	root := &Node{
		Name:     filepath.Base(g.RootPath),
		IsDir:    true,
		Children: []*Node{},
	}
	var paths []string
	for path, selected := range g.SelectedFiles {
		if selected {
			fullPath := filepath.Join(g.RootPath, path)
			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}
			if !info.IsDir() {
				fileCache := cache.GetGlobalFileCache()
				if ok, err := fileCache.GetTextFileStatus(fullPath, utils.IsTextFile); err != nil || !ok {
					continue
				}
			}
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	dirSet := make(map[string]bool)
	for _, p := range paths {
		parts := strings.Split(p, "/")
		current := root
		fullPath := ""
		for i, part := range parts {
			fullPath = path.Join(fullPath, part)
			isLast := i == len(parts)-1
			found := false
			isDir := !isLast || dirSet[fullPath]
			if isLast {
				if !dirSet[fullPath] {
					if info, err := os.Stat(filepath.Join(g.RootPath, fullPath)); err == nil && info.IsDir() {
						isDir = true
						dirSet[fullPath] = true
					}
				}
			} else {
				isDir = true
				dirSet[fullPath] = true
			}
			for _, child := range current.Children {
				if child.Name == part {
					current = child
					found = true
					break
				}
			}
			if !found {
				newNode := &Node{
					Name:     part,
					IsDir:    isDir,
					Children: []*Node{},
					Path:     fullPath,
				}
				current.Children = append(current.Children, newNode)
				current = newNode
				if !isDir {
					fileCache := cache.GetGlobalFileCache()
					absolutePath := filepath.Join(g.RootPath, fullPath)
					if err := fileCache.CacheMetadataOnly(absolutePath); err == nil {
						newNode.Language = determineLanguage(part)
					} else {
						fmt.Fprintf(os.Stderr, "Warning: failed to cache metadata for %s: %v\n", fullPath, err)
					}
				}
			}
		}
	}
	pruneEmptyDirectories(root)
	sortTree(root)
	return root
}

func pruneEmptyDirectories(node *Node) bool {
	var newChildren []*Node
	for _, child := range node.Children {
		if pruneEmptyDirectories(child) {
			newChildren = append(newChildren, child)
		}
	}
	node.Children = newChildren
	return !node.IsDir || len(node.Children) > 0
}

func sortTree(node *Node) {
	sort.Slice(node.Children, func(i, j int) bool {
		if node.Children[i].IsDir != node.Children[j].IsDir {
			return node.Children[i].IsDir
		}
		return node.Children[i].Name < node.Children[j].Name
	})
	for _, child := range node.Children {
		sortTree(child)
	}
}

func renderTree(node *Node, prefix string, isLast bool, builder *strings.Builder, rootName string, deselectedFiles map[string]bool) {
	if deselectedFiles[node.Path] {
		return
	}
	if node.Name != rootName {
		branch := "├── "
		if isLast {
			branch = "└── "
		}
		name := node.Name
		if node.IsDir {
			name += "/"
		}
		fmt.Fprintf(builder, "%s%s%s\n", prefix, branch, name)
	}
	newPrefix := prefix
	if isLast {
		newPrefix += "    "
	} else {
		newPrefix += "│   "
	}
	for i, child := range node.Children {
		renderTree(child, newPrefix, i == len(node.Children)-1, builder, rootName, deselectedFiles)
	}
}

// fileWorkItem represents a file to be processed
type fileWorkItem struct {
	node *Node
	path string
}

// ConcurrentFileCollector handles concurrent file content reading
type ConcurrentFileCollector struct {
	maxWorkers int
	rootPath   string
}

// NewConcurrentFileCollector creates a new concurrent file collector
func NewConcurrentFileCollector(rootPath string) *ConcurrentFileCollector {
	return &ConcurrentFileCollector{
		maxWorkers: runtime.NumCPU(),
		rootPath:   rootPath,
	}
}

// collectFiles intelligently chooses between concurrent and sequential based on file count
func collectFiles(node *Node, files *[]FileData, rootPath string, secretScanner *secrets.Scanner) {
	// Count files to determine best approach
	fileCount := countFiles(node)
	
	// Use concurrent approach only for larger file sets where the overhead is worth it
	const concurrentThreshold = 50
	
	if fileCount >= concurrentThreshold {
		collector := NewConcurrentFileCollector(rootPath)
		result, err := collector.CollectFilesConcurrent(node, secretScanner)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: concurrent file collection failed, falling back to sequential: %v\n", err)
			collectFilesSequential(node, files, rootPath, secretScanner)
			return
		}
		*files = result
	} else {
		// Use sequential for smaller file sets
		collectFilesSequential(node, files, rootPath, secretScanner)
	}
}

// countFiles recursively counts the number of files in the tree
func countFiles(node *Node) int {
	if !node.IsDir {
		return 1
	}
	
	count := 0
	for _, child := range node.Children {
		count += countFiles(child)
	}
	return count
}

// CollectFilesConcurrent performs concurrent file content reading
func (c *ConcurrentFileCollector) CollectFilesConcurrent(node *Node, secretScanner *secrets.Scanner) ([]FileData, error) {
	// First pass: collect all file work items
	var workItems []fileWorkItem
	c.collectWorkItems(node, &workItems)

	if len(workItems) == 0 {
		return []FileData{}, nil
	}

	// Create channels for work distribution
	workQueue := make(chan fileWorkItem, len(workItems))
	resultQueue := make(chan FileData, len(workItems))
	errorChan := make(chan error, c.maxWorkers)

	// Populate work queue
	for _, item := range workItems {
		workQueue <- item
	}
	close(workQueue)

	var wg sync.WaitGroup
	var firstError error

	// Start worker goroutines
	for i := 0; i < c.maxWorkers; i++ {
		wg.Add(1)
		go c.fileWorker(workQueue, resultQueue, errorChan, &wg)
	}

	// Start error collector
	errorDone := make(chan struct{})
	go func() {
		defer close(errorDone)
		for err := range errorChan {
			if firstError == nil {
				firstError = err
			}
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}()

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultQueue)
		close(errorChan)
	}()

	// Collect results
	var files []FileData
	for fileData := range resultQueue {
		files = append(files, fileData)
	}

	// Wait for error collector to finish
	<-errorDone

	// Sort files to maintain consistent order (by path)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, firstError
}

// collectWorkItems recursively gathers all file work items from the tree
func (c *ConcurrentFileCollector) collectWorkItems(node *Node, workItems *[]fileWorkItem) {
	if !node.IsDir {
		*workItems = append(*workItems, fileWorkItem{
			node: node,
			path: filepath.Join(c.rootPath, node.Path),
		})
	}
	for _, child := range node.Children {
		c.collectWorkItems(child, workItems)
	}
}

// fileWorker processes files from the work queue
func (c *ConcurrentFileCollector) fileWorker(workQueue <-chan fileWorkItem, resultQueue chan<- FileData, errorChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	fileCache := cache.GetGlobalFileCache()

	for item := range workQueue {
		content, err := fileCache.GetLazy(item.path)
		if err != nil {
			select {
			case errorChan <- fmt.Errorf("failed to read file %s: %w", item.node.Path, err):
			default:
			}
			continue
		}

		// Update node content for potential tree rendering
		item.node.Content = content

		// Send result to collector
		select {
		case resultQueue <- FileData{
			Path:     item.node.Path,
			Content:  content,
			Language: item.node.Language,
			Findings: nil, // Will be populated by secret scanner later
		}:
		default:
		}
	}
}

// collectFilesSequential is the original sequential implementation as fallback
func collectFilesSequential(node *Node, files *[]FileData, rootPath string, secretScanner *secrets.Scanner) {
	if !node.IsDir {
		fileCache := cache.GetGlobalFileCache()
		absolutePath := filepath.Join(rootPath, node.Path)

		if content, err := fileCache.GetLazy(absolutePath); err == nil {
			node.Content = content

			*files = append(*files, FileData{
				Path:     node.Path,
				Content:  content,
				Language: node.Language,
				Findings: nil,
			})
		} else {
			fmt.Fprintf(os.Stderr, "Warning: failed to read file %s: %v\n", node.Path, err)
		}
	}
	for _, child := range node.Children {
		collectFilesSequential(child, files, rootPath, secretScanner)
	}
}

func determineLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		// Check for special filenames without extensions
		name := strings.ToLower(filepath.Base(filename))
		switch name {
		case "makefile", "dockerfile", "jenkinsfile", "vagrantfile":
			return name
		case "gemfile", "rakefile":
			return "ruby"
		case ".gitignore", ".dockerignore", ".gitattributes":
			return "gitignore"
		case ".env", ".env.local", ".env.development", ".env.production":
			return "dotenv"
		case "readme", "license", "contributing", "changelog":
			return "markdown"
		default:
			return "text"
		}
	}

	ext = ext[1:]

	extensionMap := map[string]string{
		"html":       "html",
		"htm":        "html",
		"xhtml":      "html",
		"css":        "css",
		"scss":       "scss",
		"sass":       "sass",
		"less":       "less",
		"js":         "javascript",
		"jsx":        "jsx",
		"ts":         "typescript",
		"tsx":        "tsx",
		"vue":        "vue",
		"svelte":     "svelte",
		"go":         "go",
		"py":         "python",
		"rb":         "ruby",
		"php":        "php",
		"java":       "java",
		"c":          "c",
		"h":          "c",
		"cpp":        "cpp",
		"cc":         "cpp",
		"cxx":        "cpp",
		"hpp":        "cpp",
		"hxx":        "cpp",
		"cs":         "csharp",
		"fs":         "fsharp",
		"fsx":        "fsharp",
		"rs":         "rust",
		"swift":      "swift",
		"kt":         "kotlin",
		"kts":        "kotlin",
		"scala":      "scala",
		"clj":        "clojure",
		"cljs":       "clojure",
		"cljc":       "clojure",
		"edn":        "clojure",
		"hs":         "haskell",
		"lhs":        "haskell",
		"elm":        "elm",
		"ex":         "elixir",
		"exs":        "elixir",
		"erl":        "erlang",
		"hrl":        "erlang",
		"dart":       "dart",
		"pl":         "perl",
		"pm":         "perl",
		"r":          "r",
		"lua":        "lua",
		"groovy":     "groovy",
		"tcl":        "tcl",
		"m":          "objectivec",
		"mm":         "objectivec",
		"d":          "d",
		"jl":         "julia",
		"cr":         "crystal",
		"nim":        "nim",
		"zig":        "zig",
		"v":          "v",
		"sh":         "bash",
		"bash":       "bash",
		"zsh":        "bash",
		"fish":       "fish",
		"ps1":        "powershell",
		"psm1":       "powershell",
		"bat":        "batch",
		"cmd":        "batch",
		"awk":        "awk",
		"json":       "json",
		"yaml":       "yaml",
		"yml":        "yaml",
		"toml":       "toml",
		"xml":        "xml",
		"csv":        "csv",
		"tsv":        "tsv",
		"ini":        "ini",
		"conf":       "conf",
		"cfg":        "conf",
		"plist":      "xml",
		"md":         "markdown",
		"markdown":   "markdown",
		"rst":        "restructuredtext",
		"tex":        "latex",
		"latex":      "latex",
		"txt":        "text",
		"adoc":       "asciidoc",
		"asciidoc":   "asciidoc",
		"org":        "org",
		"sql":        "sql",
		"graphql":    "graphql",
		"gql":        "graphql",
		"proto":      "protobuf",
		"tf":         "terraform",
		"tfvars":     "terraform",
		"hcl":        "hcl",
		"dockerfile": "dockerfile",
		"lock":       "text",
		"gradle":     "gradle",
		"properties": "properties",
		"diff":       "diff",
		"patch":      "diff",
		"svg":        "svg",
		"log":        "log",
	}

	if language, ok := extensionMap[ext]; ok {
		return language
	}

	// If not found, return the extension itself
	return ext
}
