package context

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	maxFileLines      = 50   // Maximum lines to read from a file
	maxHistoryLines   = 5    // Last N commands from history
	maxFileSize       = 8192 // Max file size to read (8KB)
	maxFilesPerPrompt = 3    // Max files to include in context
)

// Context holds all gathered context information
type Context struct {
	CurrentDir    string
	OS            string
	Shell         string
	ShellHistory  []string
	ReferencedFiles map[string]string // filename -> content
	DirectoryListing []string
}

// GetContext gathers all relevant context for the AI prompt
func GetContext(instruction string) *Context {
	ctx := &Context{
		ReferencedFiles: make(map[string]string),
	}

	// 1. System Info
	ctx.CurrentDir = getCurrentDir()
	ctx.OS = runtime.GOOS
	ctx.Shell = getShell()

	// 2. Directory listing (for awareness of available files)
	ctx.DirectoryListing = getDirectoryListing()

	// 3. File Awareness - detect and read referenced files
	ctx.ReferencedFiles = detectAndReadFiles(instruction, ctx.DirectoryListing)

	// 4. Shell History
	ctx.ShellHistory = getShellHistory(ctx.Shell)

	return ctx
}

// Format returns the context as a formatted string for the AI prompt
func (c *Context) Format() string {
	var b strings.Builder

	// System info
	b.WriteString("=== SYSTEM CONTEXT ===\n")
	b.WriteString(fmt.Sprintf("Current Directory: %s\n", c.CurrentDir))
	b.WriteString(fmt.Sprintf("Operating System: %s\n", c.OS))
	b.WriteString(fmt.Sprintf("Shell: %s\n", c.Shell))

	// Directory listing (abbreviated)
	if len(c.DirectoryListing) > 0 {
		b.WriteString("\n=== FILES IN CURRENT DIRECTORY ===\n")
		// Show up to 20 files
		count := len(c.DirectoryListing)
		if count > 20 {
			count = 20
		}
		for i := 0; i < count; i++ {
			b.WriteString(fmt.Sprintf("  %s\n", c.DirectoryListing[i]))
		}
		if len(c.DirectoryListing) > 20 {
			b.WriteString(fmt.Sprintf("  ... and %d more files\n", len(c.DirectoryListing)-20))
		}
	}

	// Recent shell history
	if len(c.ShellHistory) > 0 {
		b.WriteString("\n=== RECENT COMMANDS (for context) ===\n")
		for i, cmd := range c.ShellHistory {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, cmd))
		}
	}

	// Referenced file contents
	if len(c.ReferencedFiles) > 0 {
		for filename, content := range c.ReferencedFiles {
			b.WriteString(fmt.Sprintf("\n=== CONTENT OF '%s' ===\n", filename))
			b.WriteString(content)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// HasFileContext returns true if any files were detected and read
func (c *Context) HasFileContext() bool {
	return len(c.ReferencedFiles) > 0
}

// HasHistoryContext returns true if shell history was gathered
func (c *Context) HasHistoryContext() bool {
	return len(c.ShellHistory) > 0
}

// getCurrentDir returns the current working directory
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return dir
}

// getShell returns the current shell name
func getShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "unknown"
	}
	// Extract just the shell name (e.g., /bin/zsh -> zsh)
	return filepath.Base(shell)
}

// getDirectoryListing returns a list of files in the current directory
func getDirectoryListing() []string {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files and common non-relevant files
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			name += "/"
		}
		files = append(files, name)
	}
	return files
}

// detectAndReadFiles scans the instruction for file references and reads them
func detectAndReadFiles(instruction string, dirListing []string) map[string]string {
	files := make(map[string]string)

	// Create a set of existing files for quick lookup
	existingFiles := make(map[string]bool)
	for _, f := range dirListing {
		// Remove trailing slash for directories
		existingFiles[strings.TrimSuffix(f, "/")] = true
	}

	// Patterns to detect file references
	patterns := []string{
		// Exact word matches for common file extensions
		`\b([\w.-]+\.(?:go|py|js|ts|jsx|tsx|rs|c|cpp|h|hpp|java|rb|php|sh|bash|zsh|yaml|yml|json|xml|html|css|scss|md|txt|sql|env|toml|ini|cfg|conf))\b`,
		// Paths with slashes
		`\b([\w./]+/[\w.-]+)\b`,
	}

	foundFiles := make(map[string]bool)

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(instruction, -1)
		for _, match := range matches {
			if len(match) > 1 {
				filename := match[1]
				// Check if this file exists
				if _, exists := existingFiles[filename]; exists {
					foundFiles[filename] = true
				}
				// Also check without directory listing (for full paths)
				if _, err := os.Stat(filename); err == nil {
					foundFiles[filename] = true
				}
			}
		}
	}

	// Also do simple word matching against directory listing
	words := strings.Fields(instruction)
	for _, word := range words {
		// Clean up the word (remove punctuation)
		word = strings.Trim(word, ",.;:!?\"'()[]{}")
		if _, exists := existingFiles[word]; exists {
			foundFiles[word] = true
		}
	}

	// Read the found files (up to maxFilesPerPrompt)
	count := 0
	for filename := range foundFiles {
		if count >= maxFilesPerPrompt {
			break
		}
		content, err := readFileContent(filename)
		if err == nil && content != "" {
			files[filename] = content
			count++
		}
	}

	return files
}

// readFileContent reads a file and returns its content (truncated if too large)
func readFileContent(filename string) (string, error) {
	// Check file size first
	info, err := os.Stat(filename)
	if err != nil {
		return "", err
	}

	// Skip directories
	if info.IsDir() {
		return "", fmt.Errorf("is a directory")
	}

	// Skip files that are too large
	if info.Size() > maxFileSize {
		return fmt.Sprintf("[File too large: %d bytes, showing first %d bytes]\n", info.Size(), maxFileSize), nil
	}

	// Open and read the file
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() && lineCount < maxFileLines {
		lines = append(lines, scanner.Text())
		lineCount++
	}

	content := strings.Join(lines, "\n")

	// Add truncation notice if we didn't read the whole file
	if lineCount >= maxFileLines {
		content += fmt.Sprintf("\n... [truncated at %d lines]", maxFileLines)
	}

	return content, nil
}

// getShellHistory reads recent commands from shell history
func getShellHistory(shell string) []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	// Determine history file based on shell
	var historyFile string
	switch shell {
	case "zsh":
		historyFile = filepath.Join(homeDir, ".zsh_history")
	case "bash":
		historyFile = filepath.Join(homeDir, ".bash_history")
	case "fish":
		historyFile = filepath.Join(homeDir, ".local/share/fish/fish_history")
	default:
		// Try both common history files
		historyFile = filepath.Join(homeDir, ".bash_history")
		if _, err := os.Stat(historyFile); os.IsNotExist(err) {
			historyFile = filepath.Join(homeDir, ".zsh_history")
		}
	}

	return readLastLines(historyFile, maxHistoryLines, shell)
}

// readLastLines reads the last N lines from a file
func readLastLines(filename string, n int, shell string) []string {
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()

	// Read all lines (not efficient for huge files, but history files are usually small)
	var allLines []string
	scanner := bufio.NewScanner(file)
	// Increase buffer size for zsh history which can have long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		// Parse the command from zsh history format
		// Zsh history format: : timestamp:0;command
		if shell == "zsh" && strings.HasPrefix(line, ": ") {
			parts := strings.SplitN(line, ";", 2)
			if len(parts) == 2 {
				line = parts[1]
			}
		}
		// Skip empty lines and the 'x' command itself
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "x ") {
			allLines = append(allLines, line)
		}
	}

	// Get last N lines
	if len(allLines) <= n {
		return allLines
	}
	return allLines[len(allLines)-n:]
}
