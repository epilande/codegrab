<div align="center">
  <h1>CodeGrab ✋</h1>
</div>

<p align="center">
  <strong>An interactive CLI tool for selecting and bundling code into a single, LLM-ready output file.</strong>
</p>

![codegrab-demo](https://github.com/user-attachments/assets/0f7d3e41-d023-4763-aff6-17c6be7c4189)

## ❓ Why?

When working with LLMs, sharing code context is essential for getting accurate responses. However,
manually copying files or creating code snippets is tedious. CodeGrab streamlines this process by
providing a clean TUI (Terminal UI), alongside a versatile CLI (Command-Line Interface). This allows
you to easily select files from your project, generate well-formatted output, and copy it directly
to your clipboard, ready for LLM processing.

## ✨ Features

- 🎮 **Interactive Mode**: Navigate your project structure with vim-like keybindings in a TUI environment
- 💻 **CLI Mode**: Run non-interactively (`-n` flag) to grab all valid files based on filters, ideal for scripting
- 🧹 **Filtering Options**: Respect `.gitignore` rules, handle hidden files, apply customizable glob patterns, and skip large files
- 🔍 **Fuzzy Search**: Quickly find files across your project
- ✅ **File Selection**: Toggle files or entire directories (with child items) for inclusion or exclusion
- 📄 **Multiple Output Formats**: Generate Markdown, Plain Text, or XML output
- ⏳ **Temp File**: Generate the output file in your system's temporary directory
- 📋 **Clipboard Integration**: Copy content or output file directly to your clipboard
- 🌲 **Directory Tree View**: Display a tree-style view of your project structure
- 🧮 **Token Estimation**: Get estimated token count for LLM context windows
- 🛡️ **Secret Detection & Redaction**: Uses [gitleaks](https://github.com/gitleaks/gitleaks) to identify potential secrets and prevent sharing sensitive information
- 🔗 **Dependency Resolution**: Automatically include dependencies for Go, JS/TS when using the `--deps` flag

## 📦 Installation

### Homebrew

```sh
brew tap epilande/tap
brew install codegrab
```

### Go Install

```sh
go install github.com/epilande/codegrab/cmd/grab@latest
```

### Binary Download

Download the [latest release](https://github.com/epilande/codegrab/releases/latest) for your platform.

### Build from Source

```sh
git clone https://github.com/epilande/codegrab
cd codegrab
go build ./cmd/grab
```

Then move the binary to your `PATH`.

## 🚀 Quick Start

1. Go to your project directory and run:

   ```sh
   grab
   ```

2. Navigate with arrow keys or <kbd>h</kbd>/<kbd>j</kbd>/<kbd>k</kbd>/<kbd>l</kbd>.
3. Select files using the <kbd>Space</kbd> or <kbd>Tab</kbd> key.
4. Press <kbd>g</kbd> to generate output file or <kbd>y</kbd> to copy contents to clipboard.

CodeGrab will generate `codegrab-output.md` in your current working directory (on macOS this file is automatically copied to your clipboard), which you can immediately send to an AI assistant for better context-aware coding assistance.

https://github.com/user-attachments/assets/48f245f4-695d-4cea-8fc0-4b0158bb46a5

## 🎮 Usage

```sh
grab [options] [directory]
```

### Arguments

| Argument    | Description                                           |
| :---------- | :---------------------------------------------------- |
| `directory` | Optional path to the project directory (default: ".") |

### Options

| Option                   | Description                                                                                                                                                                                          |
| :----------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `-h, --help`             | Display help information.                                                                                                                                                                            |
| `-v, --version`          | Display version information.                                                                                                                                                                         |
| `-n, --non-interactive`  | Run in non-interactive mode (selects all valid files respecting filters).                                                                                                                            |
| `-o, --output <file>`    | Output file path (default: `./codegrab-output.<format>`).                                                                                                                                            |
| `-t, --temp`             | Use system temporary directory for output file.                                                                                                                                                      |
| `-g, --glob <pattern>`   | Include/exclude files and directories using glob patterns. Can be used multiple times. Prefix with '!' to exclude (e.g., `--glob="*.{ts,tsx}" --glob="\!*.spec.ts"`).                                |
| `-f, --format <format>`  | Output format. Available: `markdown`, `text`, `xml` (default: `"markdown"`).                                                                                                                         |
| `-S, --skip-redaction`   | Skip automatic secret redaction via gitleaks (Default: false). WARNING: Disabling this may expose sensitive information!                                                                             |
| `--deps`                 | Automatically include direct dependencies for selected files (Go, JS/TS).                                                                                                                            |
| `--max-depth <depth>`    | Maximum depth for dependency resolution (`-1` for unlimited, default: `1`). Only effective with `--deps`.                                                                                            |
| `--max-file-size <size>` | Maximum file size to include (e.g., `"100kb"`, `"2MB"`). No limit by default. Files exceeding the specified size will be skipped.                                                                    |
| `--theme <name>`         | Set the UI theme. Available: catppuccin-latte, catppuccin-frappe, catppuccin-macchiato, catppuccin-mocha, rose-pine, rose-pine-dawn, rose-pine-moon, dracula, nord. (default: `"catppuccin-mocha"`). |
| `--show-tokens`          | Show the number of tokens for each file in file tree.                                                                                                                                                |
| `--icons`                | Display Nerd Font icons.                                                                                                                                                                             |

### 📖 Examples

1. Run in interactive mode (default):

   ```bash
   grab
   ```

2. Grab all files in current directory (non-interactive), skipping files > 50kb:

   ```bash
   grab -n --max-file-size 50kb
   ```

3. Grab a specific directory interactively including dependencies:

   ```bash
   grab --deps /path/to/project
   ```

4. Grab a specific directory non-interactively including all dependencies (unlimited depth):

   ```bash
   grab -n --deps --max-depth -1 /path/to/project
   ```

5. Specify custom output file:

   ```bash
   grab -o output.md /path/to/project
   ```

6. Generate XML output:

   ```bash
   grab -f xml -o output.xml /path/to/project
   ```

7. Filter files using glob pattern:

   ```bash
   grab -g="*.go" /path/to/project
   ```

8. Use multiple glob patterns for include/exclude:

   ```bash
   grab -g="*.{ts,tsx}" -g="\!*.spec.{ts,tsx}"
   ```

## ⌨️ Keyboard Controls

### Navigation

| Action                     | Key                             | Description                                             |
| :------------------------- | :------------------------------ | :------------------------------------------------------ |
| Move cursor down           | <kbd>j</kbd> or <kbd>↓</kbd>    | Move the cursor to the next item in the list            |
| Move cursor up             | <kbd>k</kbd> or <kbd>↑</kbd>    | Move the cursor to the previous item in the list        |
| Collapse directory         | <kbd>h</kbd> or <kbd>←</kbd>    | Collapse the currently selected directory               |
| Expand directory           | <kbd>l</kbd> or <kbd>→</kbd>    | Expand the currently selected directory                 |
| Go to top                  | <kbd>H</kbd> or <kbd>home</kbd> | Jump to the first item in the list                      |
| Go to bottom               | <kbd>L</kbd> or <kbd>end</kbd>  | Jump to the last item in the list                       |
| Toggle expand/collapse all | <kbd>e</kbd>                    | Toggle between expanding and collapsing all directories |

### Search

| Action                 | Key                               | Description                                                 |
| :--------------------- | :-------------------------------- | :---------------------------------------------------------- |
| Start search           | <kbd>/</kbd>                      | Begin fuzzy searching for files                             |
| Next search result     | <kbd>ctrl+n</kbd> or <kbd>↓</kbd> | Navigate to the next search result                          |
| Previous search result | <kbd>ctrl+p</kbd> or <kbd>↑</kbd> | Navigate to the previous search result                      |
| Select/deselect item   | <kbd>tab</kbd> / <kbd>enter</kbd> | Toggle selection of the item under cursor in search results |
| Exit search            | <kbd>esc</kbd>                    | Exit search mode and return to normal navigation            |

### Selection & Output

| Action                       | Key                                | Description                                                                  |
| :--------------------------- | :--------------------------------- | :--------------------------------------------------------------------------- |
| Select/deselect item         | <kbd>tab</kbd> or <kbd>space</kbd> | Toggle selection of the current file or directory                            |
| Copy to clipboard            | <kbd>y</kbd>                       | Copy the generated output to clipboard                                       |
| Generate output file         | <kbd>g</kbd>                       | Generate the output file with selected content                               |
| Toggle Dependency Resolution | <kbd>D</kbd>                       | Enable/disable automatic dependency resolution for Go & JS/TS (Default: Off) |
| Cycle output formats         | <kbd>F</kbd>                       | Cycle through available output formats (markdown, text, xml)                 |
| Toggle Secret Redaction      | <kbd>S</kbd>                       | Enable/disable automatic secret redaction (Default: On)                      |

### View Options

| Action                     | Key                              | Description                                  |
| :------------------------- | :------------------------------- | :------------------------------------------- |
| Toggle `.gitignore` filter | <kbd>i</kbd>                     | Toggle whether to respect `.gitignore` rules |
| Toggle hidden files        | <kbd>.</kbd>                     | Toggle visibility of hidden files            |
| Refresh files & folders    | <kbd>r</kbd>                     | Reload directory tree and reset selections   |
| Toggle help screen         | <kbd>?</kbd>                     | Show or hide the help screen                 |
| Quit                       | <kbd>q</kbd> / <kbd>ctrl+c</kbd> | Exit the application                         |

## 🔗 Automatic Dependency Resolution

CodeGrab can automatically include dependencies for selected files, making it easier to share complete code snippets with LLMs.

- **How it works**: When enabled, CodeGrab utilizes [tree-sitter](https://tree-sitter.github.io/tree-sitter/) to parse selected source files, identifying language-specific dependency declarations (like `import` or `require`). It then attempts to resolve these dependencies within your project and automatically includes the necessary files (respecting `.gitignore`, hidden, size, and glob filters).
- **Supported Languages**:
  - **Go**: Resolves relative imports and project-local module imports (if `go.mod` is present).
  - **JavaScript/TypeScript**: Resolves relative imports/requires for `.js`, `.jsx`, `.ts`, and `.tsx` files, including directory `index` files.
- **Enabling**:
  - **Interactive Mode**: Press <kbd>D</kbd> to toggle dependency resolution on/off. A `🔗 Deps` indicator will appear in the footer when active. Files added as dependencies will be marked with `[dep]`.
  - **Non-Interactive Mode**: Use the `--deps` flag.
- **Controlling Depth**: The `--max-depth` flag controls how many levels of dependencies are included:
  - `1` (Default): Only includes files directly imported by your initially selected files.
  - `N`: Includes dependencies up to `N` levels deep.
  - `-1`: Includes all dependencies recursively (unlimited depth).

![codegrab-deps](https://github.com/user-attachments/assets/db04805c-a0b0-4249-ab48-da3d7b92000c)

## 🛡️ Secret Detection & Redaction

CodeGrab automatically scans the content of selected files for potential secrets using [gitleaks](https://github.com/gitleaks/gitleaks) with its default rules. This helps prevent accidental exposure of sensitive credentials like API keys, private tokens, and passwords.

- **Enabled by Default**: Secret scanning and redaction are active unless explicitly disabled.
- **Redaction Format**: Detected secrets are replaced with `[REDACTED_RuleID]`, where `RuleID` indicates the type of secret found (e.g., `[REDACTED_generic-api-key]`).
- **Skipping Redaction**: You can disable this feature using the `-S` / `--skip-redaction` flag when running the command, or by pressing <kbd>S</kbd> in the interactive TUI. Use this option with caution, as it may expose sensitive information in the output.

## 🎨 Themes

CodeGrab comes with several built-in themes:

- Catppuccin (Latte, Frappe, Macchiato, Mocha)
- Dracula
- Nord
- Rosé Pine (Main, Moon, Dawn)

Select a theme using the `--theme` flag:

```sh
grab --theme dracula
```

## 📄 Output Formats

### Markdown (Default)

```sh
grab --format markdown
```

#### Example Output

````md
# Project Structure

```
./
└── internal/
    ├── filesystem/
    │   ├── filter.go
    │   ├── gitignore.go
    │   └── walker.go
    └── generator/
        └── formats/
            ├── markdown.go
            └── registry.go
```

# Project Files

## File: `internal/filesystem/filter.go`

```go
package filesystem

import (
    "path/filepath"
    "strings"
)

// ... rest of the file content
```
````

### Plain Text

```sh
grab --format text
```

#### Example Output

```
=============================================================
PROJECT STRUCTURE
=============================================================

./
└── internal/
    ├── filesystem/
    │   ├── filter.go
    │   ├── gitignore.go
    │   └── walker.go
    └── generator/
        └── formats/
            ├── markdown.go
            └── registry.go


=============================================================
PROJECT FILES
=============================================================
=============================================================
FILE: internal/filesystem/filter.go
=============================================================

package filesystem

import (
    "path/filepath"
    "strings"
)

// ... rest of the file content
```

### XML

```sh
grab --format xml
```

#### Example Output

```xml
<?xml version="1.0" encoding="UTF-8"?>
<project>
  <filesystem>
    <directory name=".">
      <directory name="internal">
        <directory name="filesystem">
          <file name="filter.go"/>
          <file name="gitignore.go"/>
          <file name="walker.go"/>
        </directory>
        <directory name="generator">
          <directory name="formats">
            <file name="markdown.go"/>
            <file name="registry.go"/>
          </directory>
        </directory>
      </directory>
    </directory>
  </filesystem>
  <files>
    <file path="internal/filesystem/filter.go" language="go"><![CDATA[
package filesystem

import (
    "path/filepath"
    "strings"
)

// ... rest of the file content
]]></file>
    <!-- Additional files -->
  </files>
</project>
```
