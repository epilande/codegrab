package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/epilande/codegrab/internal/cache"
	"github.com/epilande/codegrab/internal/filesystem"
	"github.com/epilande/codegrab/internal/secrets"
	"github.com/epilande/codegrab/internal/utils"
)

// Generator organizes how we generate the output in different formats
type Generator struct {
	format          Format
	SecretScanner   secrets.Scanner
	SelectedFiles   map[string]bool
	DeselectedFiles map[string]bool
	GitIgnoreMgr    *filesystem.GitIgnoreManager
	FilterMgr       *filesystem.FilterManager
	OutputPath      string
	RootPath        string
	UseTempFile     bool
	UseGitIgnore    bool
	ShowHidden      bool
	RedactSecrets   bool
	TreeOnly        bool
	lastSecretCount int
}

// NewGenerator constructs a generator with default settings
func NewGenerator(rootPath string, gitIgnoreMgr *filesystem.GitIgnoreManager, filterMgr *filesystem.FilterManager, outputPath string, useTempFile bool) *Generator {
	secretScanner, err := secrets.NewGitleaksScanner()
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to initialize secret scanner: %v\n", err)
		os.Exit(1)
	}

	return &Generator{
		RootPath:        rootPath,
		OutputPath:      outputPath,
		UseTempFile:     useTempFile,
		SelectedFiles:   make(map[string]bool),
		DeselectedFiles: make(map[string]bool),
		GitIgnoreMgr:    gitIgnoreMgr,
		FilterMgr:       filterMgr,
		UseGitIgnore:    true,
		ShowHidden:      false,
		SecretScanner:   secretScanner,
		RedactSecrets:   true,
	}
}

// SetFormat changes the output format
func (g *Generator) SetFormat(format Format) {
	g.format = format
}

// GetFormat returns the current format
func (g *Generator) GetFormat() Format {
	return g.format
}

// GetFormatName returns the name of the current format
func (g *Generator) GetFormatName() string {
	if g.format == nil {
		return "unknown"
	}
	return g.format.Name()
}

// SetRedactionMode enables or disables secret redaction.
func (g *Generator) SetRedactionMode(redact bool) {
	g.RedactSecrets = redact
}

// SetTreeOnlyMode enables or disables tree-only output (structure without file contents).
func (g *Generator) SetTreeOnlyMode(treeOnly bool) {
	g.TreeOnly = treeOnly
}

// Generate creates an output file in the specified format
func (g *Generator) Generate() (string, int, int, error) {
	if len(g.SelectedFiles) == 0 {
		return "", 0, 0, fmt.Errorf("no files selected, skipping generation")
	}

	if g.format == nil {
		return "", 0, 0, fmt.Errorf("no format set, cannot generate output")
	}

	data, err := g.PrepareTemplateData()
	if err != nil {
		return "", 0, g.lastSecretCount, fmt.Errorf("failed to prepare template data: %w", err)
	}

	content, tokenCount, err := g.format.Render(data)
	if err != nil {
		return "", 0, g.lastSecretCount, fmt.Errorf("failed to render %s: %w", g.format.Name(), err)
	}

	var outputPath string
	var displayPath string

	if g.UseTempFile {
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("codegrab-*%s", g.format.Extension()))
		if err != nil {
			return "", 0, g.lastSecretCount, fmt.Errorf("failed to create temporary file: %w", err)
		}
		defer tmpFile.Close()

		if _, err := tmpFile.Write([]byte(content)); err != nil {
			return tmpFile.Name(), tokenCount, g.lastSecretCount, fmt.Errorf("failed to write to temporary file: %w", err)
		}
		outputPath = tmpFile.Name()
		displayPath = outputPath
	} else {
		if g.OutputPath != "" {
			if !strings.HasSuffix(g.OutputPath, g.format.Extension()) {
				g.OutputPath += g.format.Extension()
			}
			outputPath = g.OutputPath
		} else {
			outputPath = fmt.Sprintf("./codegrab-output%s", g.format.Extension())
		}

		displayPath = outputPath
		absPath, err := filepath.Abs(outputPath)
		if err != nil {
			return displayPath, tokenCount, g.lastSecretCount, fmt.Errorf("failed to get absolute path: %w", err)
		}

		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			return displayPath, tokenCount, g.lastSecretCount, fmt.Errorf("failed to write to output file %s: %w", absPath, err)
		}

		outputPath = absPath
	}

	if err := utils.CopyFileObject(outputPath); err != nil {
		return displayPath, tokenCount, g.lastSecretCount, fmt.Errorf("clipboard copy failed: %w", err)
	}

	return displayPath, tokenCount, g.lastSecretCount, nil
}

// GenerateString returns the rendered content as a string along with counts
func (g *Generator) GenerateString() (string, int, int, error) {
	if len(g.SelectedFiles) == 0 {
		return "", 0, 0, fmt.Errorf("no files selected, skipping generation")
	}

	if g.format == nil {
		return "", 0, 0, fmt.Errorf("no format set, cannot generate output")
	}

	data, err := g.PrepareTemplateData()
	if err != nil {
		return "", 0, g.lastSecretCount, fmt.Errorf("failed to prepare template data: %w", err)
	}

	content, tokenCount, err := g.format.Render(data)
	return content, tokenCount, g.lastSecretCount, err
}

// PrepareTemplateData finalizes the selection, scans/redacts secrets, and builds TemplateData
func (g *Generator) PrepareTemplateData() (TemplateData, error) {
	g.lastSecretCount = 0

	expandedSelection := make(map[string]bool)

	for path := range g.SelectedFiles {
		fullPath := filepath.Join(g.RootPath, path)
		info, err := os.Stat(fullPath)
		if err != nil {
			// Skip files that no longer exist
			continue
		}
		if !g.ShowHidden && strings.HasPrefix(info.Name(), ".") {
			continue
		}
		if g.UseGitIgnore && g.GitIgnoreMgr.IsIgnored(fullPath) {
			continue
		}
		if info.IsDir() {
			continue
		} else {
			fileCache := cache.GetGlobalFileCache()
			isText, textErr := fileCache.GetTextFileStatus(fullPath, utils.IsTextFile)
			if textErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: Skipping %s during preparation (error checking type): %v\n", path, textErr)
				continue
			}
			if !isText {
				continue
			}
			expandedSelection[path] = true
		}
	}

	g.SelectedFiles = expandedSelection

	filePaths := make([]string, 0, len(expandedSelection))
	for path := range expandedSelection {
		filePaths = append(filePaths, path)
	}

	rootNode := g.buildTree()
	var structureBuilder strings.Builder
	baseRootName := filepath.Base(g.RootPath)
	if baseRootName == "." || baseRootName == "/" {
		cwd, _ := os.Getwd()
		baseRootName = filepath.Base(cwd)
	}
	fmt.Fprintf(&structureBuilder, "%s/\n", baseRootName)

	for i, child := range rootNode.Children {
		isLast := i == len(rootNode.Children)-1
		renderTree(child, "", isLast, &structureBuilder, baseRootName, make(map[string]bool))
	}

	var filesData []FileData
	if !g.TreeOnly {
		collectFiles(rootNode, &filesData, g.RootPath, &g.SecretScanner)
	}

	secretCount := 0

	if g.SecretScanner != nil {
		for i := range filesData {
			if len(filesData[i].Findings) > 0 {
				secretCount += len(filesData[i].Findings)
				if g.RedactSecrets {
					filesData[i].Content = g.SecretScanner.Redact(filesData[i].Content, filesData[i].Findings)
				}
			}
		}
	}

	g.lastSecretCount = secretCount

	return TemplateData{
		Structure: structureBuilder.String(),
		Files:     filesData,
		FilePaths: filePaths,
	}, nil
}
