package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/epilande/codegrab/internal/dependencies"
	"github.com/epilande/codegrab/internal/filesystem"
	"github.com/epilande/codegrab/internal/generator"
	"github.com/epilande/codegrab/internal/generator/formats"
	"github.com/epilande/codegrab/internal/model"
	"github.com/epilande/codegrab/internal/ui"
	"github.com/epilande/codegrab/internal/ui/themes"
	"github.com/epilande/codegrab/internal/utils"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ", ")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	themes.Initialize()

	var err error
	var globPatterns stringSliceFlag
	var showHelp bool
	var showVersion bool
	var nonInteractive bool
	var outputPath string
	var useTempFile bool
	var themeName string
	var formatName string
	var skipRedaction bool
	var resolveDeps bool
	var maxDepth int
	var maxFileSizeStr string
	var showIcons bool
	var showTokenCount bool

	flag.BoolVar(&showHelp, "help", false, "Display help information")
	flag.BoolVar(&showHelp, "h", false, "Display help information (shorthand)")

	flag.BoolVar(&showVersion, "version", false, "Display version information")
	flag.BoolVar(&showVersion, "v", false, "Display version information (shorthand)")

	flag.BoolVar(&nonInteractive, "non-interactive", false, "Run in non-interactive mode")
	flag.BoolVar(&nonInteractive, "n", false, "Run in non-interactive mode (shorthand)")

	flag.Var(&globPatterns, "glob", "Include/exclude files and directories (e.g., --glob=\"*.{ts,tsx}\" --glob=\"\\!*.spec.ts\")")
	flag.Var(&globPatterns, "g", "Include/exclude files and directories (shorthand)")

	flag.StringVar(&outputPath, "output", "", "Output file path (default: current directory)")
	flag.StringVar(&outputPath, "o", "", "Output file path (shorthand)")

	flag.BoolVar(&useTempFile, "temp", false, "Use system temporary directory for output file")
	flag.BoolVar(&useTempFile, "t", false, "Use system temporary directory for output file (shorthand)")

	availableThemes := strings.Join(themes.GetThemeNames(), ", ")
	themeUsage := fmt.Sprintf("UI theme (available: %s)", availableThemes)
	flag.StringVar(&themeName, "theme", "catppuccin-mocha", themeUsage)

	availableFormats := strings.Join(formats.GetFormatNames(), ", ")
	formatUsage := fmt.Sprintf("Output format (available: %s)", availableFormats)
	flag.StringVar(&formatName, "format", "markdown", formatUsage)
	flag.StringVar(&formatName, "f", "markdown", formatUsage+" (shorthand)")

	flag.BoolVar(&resolveDeps, "deps", false, "Automatically include direct dependencies (Go, TS/JS, Python)")

	flag.IntVar(&maxDepth, "max-depth", 1, "Maximum depth for dependency resolution (-1 for unlimited)")

	flag.BoolVar(&skipRedaction, "skip-redaction", false, "Skip automatic secret redaction (WARNING: this may expose secrets)")
	flag.BoolVar(&skipRedaction, "S", false, "Skip automatic secret redaction (shorthand)")

	maxFileSizeUsage := "Maximum file size to include (e.g., 50kb, 2MB). No limit by default."
	flag.StringVar(&maxFileSizeStr, "max-file-size", "", maxFileSizeUsage)

	flag.BoolVar(&showIcons, "icons", false, "Display Nerd Font icons")

	flag.BoolVar(&showTokenCount, "show-tokens", false, "Show the number of tokens for each file")

	flag.Parse()

	if showHelp {
		fmt.Println(ui.UsageText)
		fmt.Println()
		fmt.Println(ui.HelpText)
		os.Exit(0)
	}

	if showVersion {
		fmt.Println(utils.VersionInfo())
		os.Exit(0)
	}

	if themeName != "" {
		if err := themes.SetTheme(themeName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v, using default theme\n", err)
		}
	}

	if maxDepth < 0 {
		maxDepth = math.MaxInt
	}

	// Default to no limit if the flag is not set
	var maxFileSize int64 = math.MaxInt64
	if maxFileSizeStr != "" {
		maxFileSize, err = utils.ParseSizeString(maxFileSizeStr)
		if err != nil {
			log.Fatalf("Error parsing max file size %q: %v", maxFileSizeStr, err)
		}
	}

	// Use current directory if no argument is provided
	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		log.Fatalf("Error getting absolute path for %q: %v", root, err)
	}
	root = absRoot

	// Validate the provided path
	if stat, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("Error: Directory %q does not exist\n", root)
		} else {
			log.Fatalf("Error accessing %q: %v\n", root, err)
		}
	} else if !stat.IsDir() {
		log.Fatalf("Error: %q is not a directory\n", root)
	}

	filterMgr := filesystem.NewFilterManager()

	for _, pattern := range globPatterns {
		normalizedPattern, isValid := utils.NormalizeGlobPattern(pattern, root)
		if isValid {
			filterMgr.AddGlobPattern(normalizedPattern)
		}
	}

	if nonInteractive {
		runNonInteractive(root, filterMgr, outputPath, useTempFile, formatName, skipRedaction, resolveDeps, maxDepth, maxFileSize)
	} else {
		config := model.Config{
			RootPath:       root,
			FilterMgr:      filterMgr,
			OutputPath:     outputPath,
			UseTempFile:    useTempFile,
			Format:         formatName,
			SkipRedaction:  skipRedaction,
			ResolveDeps:    resolveDeps,
			ShowIcons:      showIcons,
			ShowTokenCount: showTokenCount,
			MaxDepth:       maxDepth,
			MaxFileSize:    maxFileSize,
		}

		m := model.NewModel(config)
		p := tea.NewProgram(m, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			log.Fatalf("Error running program: %v\n", err)
		}
	}
}

// runNonInteractive processes files and generates output without user interaction
func runNonInteractive(rootPath string, filterMgr *filesystem.FilterManager, outputPath string, useTempFile bool, formatName string, skipRedaction bool, resolveDeps bool, maxDepth int, maxFileSize int64) {
	gitIgnoreMgr, err := filesystem.NewGitIgnoreManager(rootPath)
	if err != nil {
		log.Fatalf("Error reading .gitignore: %v\n", err)
	}

	files, err := filesystem.WalkDirectory(rootPath, gitIgnoreMgr, filterMgr, true, false, maxFileSize)
	if err != nil {
		log.Fatalf("Error walking directory: %v\n", err)
	}

	// Automatically select all non-directory files
	selectedFiles := make(map[string]bool)
	for _, file := range files {
		if !file.IsDir {
			selectedFiles[file.Path] = true
		}
	}

	if resolveDeps {
		fmt.Println("ℹ️ Resolving dependencies...")
		projectModuleName := dependencies.ReadGoModFile(rootPath)

		queue := make([]model.QueuedDep, 0, len(selectedFiles))
		processed := make(map[string]bool)

		for path := range selectedFiles {
			queue = append(queue, model.QueuedDep{Path: path, Depth: 0})
			processed[path] = true
		}

		i := 0
		for i < len(queue) {
			currentItem := queue[i]
			i++

			filePath := currentItem.Path
			currentDepth := currentItem.Depth

			if currentDepth >= maxDepth {
				continue
			}

			resolver := dependencies.GetResolver(filePath)
			if resolver == nil {
				continue
			}

			content, err := os.ReadFile(filepath.Join(rootPath, filePath))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cannot read %s for dep resolution: %v\n", filePath, err)
				continue
			}

			deps, err := resolver.Resolve(content, filePath, rootPath, projectModuleName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: cannot resolve deps for %s: %v\n", filePath, err)
				continue
			}

			for _, depPath := range deps {
				depPath = filepath.ToSlash(filepath.Clean(depPath))

				depFullPath := filepath.Join(rootPath, depPath)
				depInfo, statErr := os.Stat(depFullPath)
				if statErr != nil || depInfo.IsDir() ||
					(gitIgnoreMgr.IsIgnored(depFullPath)) ||
					(utils.IsHiddenPath(depPath)) ||
					(depInfo.Size() > maxFileSize) {
					continue
				}

				if !processed[depPath] {
					fmt.Printf("Adding dependency: %s (depth %d, required by %s)\n", depPath, currentDepth+1, filePath)
					selectedFiles[depPath] = true
					processed[depPath] = true

					queue = append(queue, model.QueuedDep{Path: depPath, Depth: currentDepth + 1})
				}
			}
		}
		fmt.Printf("ℹ️ Dependency resolution complete. Total files selected: %d\n", len(selectedFiles))
	}

	gen := generator.NewGenerator(rootPath, gitIgnoreMgr, filterMgr, outputPath, useTempFile)
	format := formats.GetFormat(formatName)
	gen.SetFormat(format)
	gen.SetRedactionMode(!skipRedaction)

	gen.SelectedFiles = selectedFiles

	outputFilePath, tokenCount, secretCount, err := gen.Generate()
	if err != nil {
		log.Fatalf("Error generating output: %v\n", err)
	}

	fmt.Printf("✅ Generated %s (%d tokens)\n", outputFilePath, tokenCount)

	if secretCount > 0 && skipRedaction {
		fmt.Fprintf(os.Stderr, "⚠️ WARNING: %d secrets detected in the output and redaction was skipped!\n", secretCount)
	} else if secretCount > 0 && !skipRedaction {
		fmt.Fprintf(os.Stderr, "🛡️ INFO: %d secrets detected and redacted in the output.\n", secretCount)
	} else {
		fmt.Println("🛡️ No secrets detected in the output.")
	}
}
