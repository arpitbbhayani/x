package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	// errorStyle for error messages
	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")) // Red
)

// PrintError displays an error message
func PrintError(msg string) {
	fmt.Fprintln(os.Stderr, errorStyle.Render("Error: "+msg))
}
