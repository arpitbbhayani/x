package prompt

import (
	"fmt"
	"os"
	"runtime"
)

// Build creates the system prompt for the AI model
func Build(instruction string) string {
	return fmt.Sprintf(`You are a shell command generator. Convert the user's natural language instruction into a shell command.

Rules:
- Return ONLY the shell command, nothing else
- No explanations, no markdown formatting, no code block markers
- No backticks, no `+"`"+`bash`+"`"+`, no comments
- Just the raw executable command(s)
- Use pipes (|) and operators (&&, ||) as needed
- If multiple commands are needed, combine them with && or ;

Context:
- Current directory: %s
- Shell: %s
- OS: %s

Instruction: %s

Command:`, getCurrentDir(), getShell(), getOS(), instruction)
}

// getCurrentDir returns the current working directory
func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return dir
}

// getShell returns the current shell
func getShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "unknown"
	}
	return shell
}

// getOS returns the operating system
func getOS() string {
	return runtime.GOOS
}
