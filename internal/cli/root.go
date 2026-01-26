package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/REDFOX1899/ask-sh/internal/config"
	"github.com/REDFOX1899/ask-sh/internal/prompt"
	"github.com/REDFOX1899/ask-sh/internal/provider"
	"github.com/REDFOX1899/ask-sh/internal/ui"
)

var (
	verbose bool
	cfgMgr  *config.Manager
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "x [instruction]",
	Short: "Natural language shell command executor",
	Long: `x converts natural language instructions into shell commands.
It supports OpenAI, Anthropic, Gemini, and Ollama API providers.

Set one of: OPENAI_API_KEY, ANTHROPIC_API_KEY, GEMINI_API_KEY, or OLLAMA_MODEL`,
	Example: `  x get all the git branches
  x list all files modified in the last 7 days
  x show disk usage of current directory
  x count lines in all python files`,
	Args:          cobra.MinimumNArgs(1),
	RunE:          runCommand,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// Persistent flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug output")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the CLI application
func Execute() error {
	// Initialize config manager
	var err error
	cfgMgr, err = config.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	return rootCmd.Execute()
}

func runCommand(cmd *cobra.Command, args []string) error {
	// Combine all arguments into instruction
	instruction := strings.Join(args, " ")

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Instruction: %s\n", instruction)
	}

	// Load configuration
	cfg, err := cfgMgr.Load()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
		return err
	}
	cfg.Verbose = verbose

	// Create provider registry and detect provider
	registry := provider.NewRegistry(cfg, verbose)
	p, err := registry.Detect()
	if err != nil {
		ui.PrintError(err.Error())
		return err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Using API provider: %s\n", p.Name())
	}

	// Build prompt
	promptText := prompt.Build(instruction)

	// Generate command
	ctx := context.Background()
	resp, err := p.GenerateCommand(ctx, promptText)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to generate command: %v", err))
		return err
	}

	// Save working model to config
	if err := cfgMgr.SaveWorkingModel(resp.Provider, resp.Model); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "DEBUG: Failed to save working model: %v\n", err)
		}
	} else if verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Saved working model: %s\n", resp.Model)
	}

	// Run interactive TUI loop
	return runInteractiveLoop(ctx, p, resp.Command, resp.Provider, resp.Model)
}

func runInteractiveLoop(ctx context.Context, p provider.Provider, command, providerName, modelName string) error {
	for {
		// Run TUI
		result, err := ui.RunTUI(command, providerName, modelName)
		if err != nil {
			return err
		}

		switch result.Action {
		case ui.ActionExecute:
			// Execute the command
			return executeShellCommand(result.Command)

		case ui.ActionCancel:
			fmt.Println("Command execution cancelled")
			return nil

		case ui.ActionEdit:
			// Command was edited in TUI, update and continue loop
			command = result.Command
			continue

		case ui.ActionRefine:
			// Refine with AI
			fmt.Println("Refining command...")
			resp, err := p.RefineCommand(ctx, command, result.RefinementQuery)
			if err != nil {
				ui.PrintError(fmt.Sprintf("Failed to refine command: %v", err))
				continue
			}
			command = resp.Command
			continue

		case ui.ActionExplain:
			// Get explanation and show TUI with explanation
			fmt.Println("Getting explanation...")
			explanation, err := p.ExplainCommand(ctx, command)
			if err != nil {
				ui.PrintError(fmt.Sprintf("Failed to explain command: %v", err))
				// Continue without explanation
				continue
			}

			// Run TUI with explanation
			result, err := ui.RunTUIWithExplanation(command, explanation, providerName, modelName)
			if err != nil {
				return err
			}

			// Handle the result from explanation view
			switch result.Action {
			case ui.ActionExecute:
				return executeShellCommand(result.Command)
			case ui.ActionCancel:
				fmt.Println("Command execution cancelled")
				return nil
			case ui.ActionEdit:
				command = result.Command
				continue
			case ui.ActionRefine:
				fmt.Println("Refining command...")
				resp, err := p.RefineCommand(ctx, command, result.RefinementQuery)
				if err != nil {
					ui.PrintError(fmt.Sprintf("Failed to refine command: %v", err))
					continue
				}
				command = resp.Command
				continue
			default:
				continue
			}

		default:
			continue
		}
	}
}

func executeShellCommand(command string) error {
	// Use the user's shell to execute the command
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell, "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
