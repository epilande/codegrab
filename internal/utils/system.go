package utils

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"math/big"
)

// CopyFileObject attempts to place a "file object" onto the clipboard.
// - On macOS, it calls AppleScript to set the clipboard to a POSIX file reference.
// - On non-macOS, it does nothing.
func CopyFileObject(filePath string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`set the clipboard to (POSIX file "%s")`, filePath))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy file to clipboard: %w", err)
	}
	return nil
}

// IsTextFile returns true if the file at filePath appears to be a text file.
// It checks the content type and also looks for null bytes.
func IsTextFile(filePath string) (bool, error) {
	const sampleSize = 512

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsPermission(err) {
			fmt.Fprintf(os.Stderr, "Warning: Permission denied checking file type for %s\n", filePath)
			return false, nil
		}
		return false, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	buf := make([]byte, sampleSize)
	n, err := file.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("failed to read file content for %s: %w", filePath, err)
	}
	if n == 0 {
		return true, nil
	}

	buffer := buf[:n]

	contentType := http.DetectContentType(buffer)
	isLikelyTextType := strings.HasPrefix(contentType, "text/") ||
		contentType == "application/json" ||
		contentType == "application/xml" ||
		contentType == "application/javascript" ||
		contentType == "application/octet-stream"

	hasNullByte := false
	for _, b := range buffer {
		if b == 0 {
			hasNullByte = true
			break
		}
	}

	if hasNullByte {
		return false, nil
	}

	if isLikelyTextType {
		return true, nil
	}

	return false, nil
}

// IsHiddenPath checks if any component of the path starts with a dot.
func IsHiddenPath(path string) bool {
	cleanedPath := filepath.Clean(path)
	if strings.HasPrefix(filepath.Base(cleanedPath), ".") {
		return true
	}
	current := cleanedPath
	for {
		parent := filepath.Dir(current)
		if parent == "." || parent == "/" || parent == current {
			break
		}
		if strings.HasPrefix(filepath.Base(parent), ".") {
			return true
		}
		current = parent
	}
	return false
}

// ParseSizeString converts a human-readable size string (e.g., "100kb", "2MB") into bytes (int64).
func ParseSizeString(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(strings.ToLower(sizeStr))
	if sizeStr == "" {
		return 0, errors.New("size string cannot be empty")
	}

	var multiplier int64 = 1
	var numPart string

	// Separate numeric part from unit part
	lastDigitIndex := -1
	for i, r := range sizeStr {
		if unicode.IsDigit(r) || r == '.' {
			lastDigitIndex = i
		} else {
			break
		}
	}

	if lastDigitIndex == -1 {
		return 0, fmt.Errorf("invalid size format: no numeric part found in %q", sizeStr)
	}

	numPart = sizeStr[:lastDigitIndex+1]
	unitPart := strings.TrimSpace(sizeStr[lastDigitIndex+1:])

	switch unitPart {
	case "", "b":
		multiplier = 1
	case "kb", "k":
		multiplier = 1024
	case "mb", "m":
		multiplier = 1024 * 1024
	case "gb", "g":
		multiplier = 1024 * 1024 * 1024
	case "tb", "t":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("invalid size unit: %q in %q", unitPart, sizeStr)
	}

	// Use math/big for high precision to avoid float64 overflow or loss.
	bf, _, err := big.ParseFloat(numPart, 10, 256, big.ToZero)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value %q: %w", numPart, err)
	}

	if bf.Sign() < 0 {
		return 0, fmt.Errorf("size cannot be negative: %s", numPart)
	}

	bf.Mul(bf, new(big.Float).SetInt64(multiplier))
	resultInt, _ := bf.Int(nil) // truncate towards zero
	if !resultInt.IsInt64() {
		return 0, fmt.Errorf("size %q overflows int64", sizeStr)
	}

	return resultInt.Int64(), nil
}
