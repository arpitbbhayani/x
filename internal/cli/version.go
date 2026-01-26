package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Version can be set at build time
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version number of x`,
	Run: func(cmd *cobra.Command, args []string) {
		// Try to read from VERSION file first
		version := getVersion()
		fmt.Println(version)
	},
}

func getVersion() string {
	// If Version was set at build time, use it
	if Version != "dev" {
		return Version
	}

	// Try to read VERSION file from executable directory
	execPath, err := os.Executable()
	if err == nil {
		versionFile := filepath.Join(filepath.Dir(execPath), "VERSION")
		if data, err := os.ReadFile(versionFile); err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	// Try current directory
	if data, err := os.ReadFile("VERSION"); err == nil {
		return strings.TrimSpace(string(data))
	}

	return Version
}
