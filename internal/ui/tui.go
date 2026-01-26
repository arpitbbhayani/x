package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/REDFOX1899/ask-sh/internal/safety"
)

// Action represents the user's chosen action
type Action int

const (
	ActionNone Action = iota
	ActionExecute
	ActionCancel
	ActionEdit
	ActionRefine
	ActionExplain
)

// Result contains the TUI result
type Result struct {
	Action          Action
	Command         string // Final command (possibly edited)
	RefinementQuery string // User's refinement request
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	commandBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			MarginBottom(1)

	commandBoxDangerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("196")).
				Padding(1, 2).
				MarginBottom(1)

	commandTextStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("226"))

	commandTextDangerStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("196"))

	explanationBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(1, 2).
				MarginBottom(1).
				Foreground(lipgloss.Color("252"))

	warningBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(1, 2).
			MarginBottom(1)

	warningTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("196")).
				Background(lipgloss.Color("52"))

	warningTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("208"))

	suggestionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	inputPromptStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	dangerInputPromptStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46"))

	providerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	riskBadgeStyles = map[safety.RiskLevel]lipgloss.Style{
		safety.RiskNone: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true),
		safety.RiskLow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true),
		safety.RiskMedium: lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true),
		safety.RiskHigh: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Blink(true),
		safety.RiskCritical: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Background(lipgloss.Color("52")).
			Bold(true).
			Blink(true),
	}
)

// Model represents the Bubble Tea model
type Model struct {
	command     string
	explanation string
	provider    string
	modelName   string

	// Safety assessment
	riskAssessment safety.RiskAssessment

	showExplanation bool
	editMode        bool
	refineMode      bool
	confirmMode     bool // For dangerous commands requiring typed confirmation
	textInput       textinput.Model

	result Result
	done   bool
	width  int
}

// NewModel creates a new TUI model
func NewModel(command, provider, modelName string) Model {
	ti := textinput.New()
	ti.Placeholder = "Type your refinement request..."
	ti.CharLimit = 500
	ti.Width = 60

	// Analyze command for safety
	assessment := safety.AnalyzeCommand(command)

	return Model{
		command:        command,
		provider:       provider,
		modelName:      modelName,
		textInput:      ti,
		width:          80,
		riskAssessment: assessment,
	}
}

// SetExplanation sets the command explanation
func (m *Model) SetExplanation(explanation string) {
	m.explanation = explanation
	m.showExplanation = true
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		// Handle edit mode
		if m.editMode {
			switch msg.String() {
			case "enter":
				m.command = m.textInput.Value()
				m.editMode = false
				m.textInput.Blur()
				// Re-analyze command after edit
				m.riskAssessment = safety.AnalyzeCommand(m.command)
				return m, nil
			case "esc":
				m.editMode = false
				m.textInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		// Handle confirm mode (for dangerous commands)
		if m.confirmMode {
			switch msg.String() {
			case "enter":
				requiredWord := safety.GetConfirmationWord(m.riskAssessment.Level)
				if strings.TrimSpace(m.textInput.Value()) == requiredWord {
					m.result = Result{Action: ActionExecute, Command: m.command}
					m.done = true
					return m, tea.Quit
				}
				// Wrong confirmation, stay in confirm mode
				m.textInput.SetValue("")
				return m, nil
			case "esc":
				m.confirmMode = false
				m.textInput.Blur()
				m.textInput.SetValue("")
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		// Handle refine mode
		if m.refineMode {
			switch msg.String() {
			case "enter":
				m.result = Result{
					Action:          ActionRefine,
					Command:         m.command,
					RefinementQuery: m.textInput.Value(),
				}
				m.done = true
				return m, tea.Quit
			case "esc":
				m.refineMode = false
				m.textInput.Blur()
				m.textInput.SetValue("")
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		// Normal mode key handling
		switch msg.String() {
		case "y", "Y", "enter":
			// Check if dangerous command requires typed confirmation
			if m.riskAssessment.Level >= safety.RiskHigh {
				m.confirmMode = true
				m.textInput.SetValue("")
				m.textInput.Placeholder = ""
				m.textInput.Focus()
				return m, textinput.Blink
			}
			m.result = Result{Action: ActionExecute, Command: m.command}
			m.done = true
			return m, tea.Quit

		case "n", "N", "q", "esc":
			m.result = Result{Action: ActionCancel, Command: m.command}
			m.done = true
			return m, tea.Quit

		case "e", "E":
			m.editMode = true
			m.textInput.SetValue(m.command)
			m.textInput.Placeholder = ""
			m.textInput.Focus()
			return m, textinput.Blink

		case "r", "R":
			m.refineMode = true
			m.textInput.SetValue("")
			m.textInput.Placeholder = "e.g., add timestamps, make recursive..."
			m.textInput.Focus()
			return m, textinput.Blink

		case "x", "X":
			m.result = Result{Action: ActionExplain, Command: m.command}
			m.done = true
			return m, tea.Quit

		case "ctrl+c":
			m.result = Result{Action: ActionCancel, Command: m.command}
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("🚀 x - Natural Language Shell"))
	b.WriteString("\n\n")

	// Provider info
	providerInfo := fmt.Sprintf("Using %s (%s)", m.provider, m.modelName)
	b.WriteString(providerStyle.Render(providerInfo))
	b.WriteString("\n\n")

	// Risk badge
	if m.riskAssessment.Level > safety.RiskNone {
		riskStyle := riskBadgeStyles[m.riskAssessment.Level]
		riskName := safety.GetRiskLevelName(m.riskAssessment.Level)
		if m.riskAssessment.Level >= safety.RiskHigh {
			b.WriteString(riskStyle.Render("⚠️  " + riskName + " ⚠️"))
		} else {
			b.WriteString(riskStyle.Render("⚡ " + riskName))
		}
		b.WriteString("\n\n")
	}

	// Command box (red border for dangerous commands)
	var commandContent string
	if m.riskAssessment.Level >= safety.RiskHigh {
		commandContent = commandTextDangerStyle.Render(m.command)
		b.WriteString(commandBoxDangerStyle.Render(commandContent))
	} else {
		commandContent = commandTextStyle.Render(m.command)
		b.WriteString(commandBoxStyle.Render(commandContent))
	}
	b.WriteString("\n")

	// Warning box for dangerous commands
	if len(m.riskAssessment.Warnings) > 0 && m.riskAssessment.Level >= safety.RiskMedium {
		var warningContent strings.Builder

		warningContent.WriteString(warningTitleStyle.Render(" ⚠️  SAFETY WARNINGS ⚠️ "))
		warningContent.WriteString("\n\n")

		for _, warning := range m.riskAssessment.Warnings {
			warningContent.WriteString(warningTextStyle.Render("• " + warning))
			warningContent.WriteString("\n")
		}

		if len(m.riskAssessment.Suggestions) > 0 {
			warningContent.WriteString("\n")
			warningContent.WriteString(suggestionStyle.Render("💡 Suggestions:"))
			warningContent.WriteString("\n")
			for _, suggestion := range m.riskAssessment.Suggestions {
				warningContent.WriteString(suggestionStyle.Render("  → " + suggestion))
				warningContent.WriteString("\n")
			}
		}

		b.WriteString(warningBoxStyle.Render(warningContent.String()))
		b.WriteString("\n")
	}

	// Explanation box (if showing)
	if m.showExplanation && m.explanation != "" {
		b.WriteString(explanationBoxStyle.Render(m.explanation))
		b.WriteString("\n")
	}

	// Confirm mode for dangerous commands
	if m.confirmMode {
		requiredWord := safety.GetConfirmationWord(m.riskAssessment.Level)
		b.WriteString(dangerInputPromptStyle.Render(fmt.Sprintf("⚠️  Type '%s' to execute this dangerous command:", requiredWord)))
		b.WriteString("\n")
		b.WriteString(m.textInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press Esc to cancel"))
		return b.String()
	}

	// Edit mode
	if m.editMode {
		b.WriteString(inputPromptStyle.Render("Edit command:"))
		b.WriteString("\n")
		b.WriteString(m.textInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press Enter to confirm, Esc to cancel"))
		return b.String()
	}

	// Refine mode
	if m.refineMode {
		b.WriteString(inputPromptStyle.Render("How would you like to refine this command?"))
		b.WriteString("\n")
		b.WriteString(m.textInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press Enter to refine, Esc to cancel"))
		return b.String()
	}

	// Help text (different for dangerous commands)
	if m.riskAssessment.Level >= safety.RiskHigh {
		b.WriteString(renderDangerHelp())
	} else {
		b.WriteString(renderHelp())
	}

	return b.String()
}

func renderHelp() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"y/Enter", "Execute"},
		{"n/Esc", "Cancel"},
		{"e", "Edit command"},
		{"r", "Refine with AI"},
		{"x", "Explain command"},
	}

	var parts []string
	for _, k := range keys {
		part := keyStyle.Render(k.key) + " " + descStyle.Render(k.desc)
		parts = append(parts, part)
	}

	return helpStyle.Render(strings.Join(parts, "  •  "))
}

func renderDangerHelp() string {
	keys := []struct {
		key   string
		desc  string
		style lipgloss.Style
	}{
		{"y", "Requires typed confirmation", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)},
		{"n/Esc", "Cancel (recommended)", keyStyle},
		{"e", "Edit command", keyStyle},
		{"r", "Refine with AI", keyStyle},
		{"x", "Explain command", keyStyle},
	}

	var parts []string
	for _, k := range keys {
		part := k.style.Render(k.key) + " " + descStyle.Render(k.desc)
		parts = append(parts, part)
	}

	return helpStyle.Render(strings.Join(parts, "  •  "))
}

// GetResult returns the result after the TUI exits
func (m Model) GetResult() Result {
	return m.result
}

// RunTUI runs the interactive TUI and returns the result
func RunTUI(command, provider, modelName string) (Result, error) {
	model := NewModel(command, provider, modelName)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return Result{Action: ActionCancel}, err
	}

	return finalModel.(Model).GetResult(), nil
}

// RunTUIWithExplanation runs the TUI with an explanation shown
func RunTUIWithExplanation(command, explanation, provider, modelName string) (Result, error) {
	model := NewModel(command, provider, modelName)
	model.SetExplanation(explanation)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return Result{Action: ActionCancel}, err
	}

	return finalModel.(Model).GetResult(), nil
}
