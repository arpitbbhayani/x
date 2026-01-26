package safety

import (
	"regexp"
	"strings"
)

// RiskLevel indicates how dangerous a command is
type RiskLevel int

const (
	RiskNone     RiskLevel = iota // Safe command
	RiskLow                       // Slightly risky, normal confirmation
	RiskMedium                    // Moderately risky, highlighted warning
	RiskHigh                      // Very dangerous, requires typed confirmation
	RiskCritical                  // Extremely dangerous, requires typing full command
)

// RiskAssessment contains the result of analyzing a command
type RiskAssessment struct {
	Level       RiskLevel
	Warnings    []string
	Suggestions []string
}

// DangerousPattern defines a pattern to detect dangerous commands
type DangerousPattern struct {
	Pattern     *regexp.Regexp
	Description string
	Level       RiskLevel
	Suggestion  string
}

var dangerousPatterns = []DangerousPattern{
	// CRITICAL - System destruction
	{
		Pattern:     regexp.MustCompile(`rm\s+(-[a-zA-Z]*[rf][a-zA-Z]*\s+)*(/|/\*|\s+/\s|"\s*/\s*")`),
		Description: "Removes root filesystem - THIS WILL DESTROY YOUR SYSTEM",
		Level:       RiskCritical,
		Suggestion:  "Never run rm -rf on root. Specify the exact path you want to delete.",
	},
	{
		Pattern:     regexp.MustCompile(`rm\s+(-[a-zA-Z]*[rf][a-zA-Z]*\s+)*(~|~/\*|/home/\*|/Users/\*)`),
		Description: "Removes entire home directory",
		Level:       RiskCritical,
		Suggestion:  "Specify the exact subdirectory you want to delete.",
	},
	{
		Pattern:     regexp.MustCompile(`mkfs\s`),
		Description: "Formats a filesystem - ALL DATA WILL BE LOST",
		Level:       RiskCritical,
		Suggestion:  "Double-check the device path. This is irreversible.",
	},
	{
		Pattern:     regexp.MustCompile(`dd\s+.*of\s*=\s*/dev/(sd[a-z]|nvme|hd[a-z]|disk)\b`),
		Description: "Writes directly to disk - CAN DESTROY DATA",
		Level:       RiskCritical,
		Suggestion:  "Verify the output device is correct. Consider backing up first.",
	},
	{
		Pattern:     regexp.MustCompile(`>\s*/dev/(sd[a-z]|nvme|hd[a-z])`),
		Description: "Redirects output to raw disk device",
		Level:       RiskCritical,
		Suggestion:  "This will overwrite the disk. Use a file path instead.",
	},
	{
		Pattern:     regexp.MustCompile(`:(){ :|:& };:`),
		Description: "Fork bomb - WILL CRASH YOUR SYSTEM",
		Level:       RiskCritical,
		Suggestion:  "This is a malicious command. Do not run it.",
	},

	// HIGH - Data loss potential
	{
		Pattern:     regexp.MustCompile(`rm\s+(-[a-zA-Z]*[rf][a-zA-Z]*\s+)+`),
		Description: "Recursive/forced deletion",
		Level:       RiskHigh,
		Suggestion:  "Consider using 'rm -i' for interactive confirmation, or list files first with 'ls'.",
	},
	{
		Pattern:     regexp.MustCompile(`chmod\s+(-R\s+)?(000|777)\s`),
		Description: "Dangerous permission change",
		Level:       RiskHigh,
		Suggestion:  "777 makes files world-writable. 000 removes all access. Use more specific permissions.",
	},
	{
		Pattern:     regexp.MustCompile(`chmod\s+-R\s`),
		Description: "Recursive permission change",
		Level:       RiskMedium,
		Suggestion:  "Verify the target directory before applying recursive permission changes.",
	},
	{
		Pattern:     regexp.MustCompile(`chown\s+-R\s`),
		Description: "Recursive ownership change",
		Level:       RiskMedium,
		Suggestion:  "Verify the target directory and new owner before applying.",
	},
	{
		Pattern:     regexp.MustCompile(`>\s*/etc/`),
		Description: "Overwrites system configuration file",
		Level:       RiskHigh,
		Suggestion:  "Back up the original file first. Consider using '>>' to append instead.",
	},
	{
		Pattern:     regexp.MustCompile(`dd\s+`),
		Description: "Low-level disk operation",
		Level:       RiskHigh,
		Suggestion:  "Double-check if= and of= parameters. Data can be lost if reversed.",
	},
	{
		Pattern:     regexp.MustCompile(`mv\s+.*\s+/dev/null`),
		Description: "Moving files to /dev/null deletes them permanently",
		Level:       RiskHigh,
		Suggestion:  "Use 'rm' if you want to delete. This is irreversible.",
	},
	{
		Pattern:     regexp.MustCompile(`curl\s+.*\|\s*(sudo\s+)?(ba)?sh`),
		Description: "Piping remote script directly to shell",
		Level:       RiskHigh,
		Suggestion:  "Download the script first, review it, then execute.",
	},
	{
		Pattern:     regexp.MustCompile(`wget\s+.*\|\s*(sudo\s+)?(ba)?sh`),
		Description: "Piping remote script directly to shell",
		Level:       RiskHigh,
		Suggestion:  "Download the script first, review it, then execute.",
	},
	{
		Pattern:     regexp.MustCompile(`eval\s+.*\$`),
		Description: "Executing dynamically constructed command",
		Level:       RiskHigh,
		Suggestion:  "Avoid eval when possible. It can execute unintended code.",
	},

	// MEDIUM - Potential issues
	{
		Pattern:     regexp.MustCompile(`sudo\s+rm\s`),
		Description: "Deleting files with elevated privileges",
		Level:       RiskMedium,
		Suggestion:  "Verify the files to be deleted before running with sudo.",
	},
	{
		Pattern:     regexp.MustCompile(`sudo\s+`),
		Description: "Running with elevated privileges",
		Level:       RiskLow,
		Suggestion:  "Command runs as root. Verify this is necessary.",
	},
	{
		Pattern:     regexp.MustCompile(`rm\s`),
		Description: "Deleting files",
		Level:       RiskLow,
		Suggestion:  "Consider using trash/recycle instead of permanent deletion.",
	},
	{
		Pattern:     regexp.MustCompile(`kill\s+-9`),
		Description: "Force killing process",
		Level:       RiskMedium,
		Suggestion:  "SIGKILL doesn't allow graceful shutdown. Try 'kill' without -9 first.",
	},
	{
		Pattern:     regexp.MustCompile(`killall\s`),
		Description: "Killing all processes by name",
		Level:       RiskMedium,
		Suggestion:  "This affects ALL processes with that name. Be specific.",
	},
	{
		Pattern:     regexp.MustCompile(`pkill\s`),
		Description: "Killing processes by pattern",
		Level:       RiskMedium,
		Suggestion:  "Verify which processes will be affected with 'pgrep' first.",
	},
	{
		Pattern:     regexp.MustCompile(`shutdown|reboot|poweroff|halt`),
		Description: "System shutdown/reboot",
		Level:       RiskMedium,
		Suggestion:  "This will terminate all running programs.",
	},
	{
		Pattern:     regexp.MustCompile(`systemctl\s+(stop|disable|mask)\s`),
		Description: "Stopping/disabling system service",
		Level:       RiskMedium,
		Suggestion:  "Verify this won't affect critical system functionality.",
	},
	{
		Pattern:     regexp.MustCompile(`iptables\s+-F`),
		Description: "Flushing firewall rules",
		Level:       RiskHigh,
		Suggestion:  "This removes all firewall rules. Your system may become exposed.",
	},
	{
		Pattern:     regexp.MustCompile(`ufw\s+disable`),
		Description: "Disabling firewall",
		Level:       RiskHigh,
		Suggestion:  "This disables the firewall entirely. Your system may become exposed.",
	},
	{
		Pattern:     regexp.MustCompile(`history\s+-c`),
		Description: "Clearing shell history",
		Level:       RiskLow,
		Suggestion:  "This is often used to hide malicious activity.",
	},
	{
		Pattern:     regexp.MustCompile(`shred\s`),
		Description: "Securely erasing files (unrecoverable)",
		Level:       RiskHigh,
		Suggestion:  "Shredded files cannot be recovered. Verify targets carefully.",
	},
	{
		Pattern:     regexp.MustCompile(`truncate\s`),
		Description: "Truncating files",
		Level:       RiskMedium,
		Suggestion:  "This can cause data loss. Verify the target file.",
	},
	{
		Pattern:     regexp.MustCompile(`>\s*[^|&]`),
		Description: "Overwriting file with redirect",
		Level:       RiskLow,
		Suggestion:  "This overwrites the file. Use '>>' to append instead if needed.",
	},
}

// AnalyzeCommand checks a command for dangerous patterns
func AnalyzeCommand(command string) RiskAssessment {
	assessment := RiskAssessment{
		Level:       RiskNone,
		Warnings:    []string{},
		Suggestions: []string{},
	}

	// Normalize command
	cmd := strings.TrimSpace(command)
	cmd = strings.ToLower(cmd)

	for _, pattern := range dangerousPatterns {
		if pattern.Pattern.MatchString(cmd) {
			// Update to highest risk level found
			if pattern.Level > assessment.Level {
				assessment.Level = pattern.Level
			}
			assessment.Warnings = append(assessment.Warnings, pattern.Description)
			if pattern.Suggestion != "" {
				assessment.Suggestions = append(assessment.Suggestions, pattern.Suggestion)
			}
		}
	}

	return assessment
}

// GetRiskLevelName returns a human-readable risk level name
func GetRiskLevelName(level RiskLevel) string {
	switch level {
	case RiskNone:
		return "Safe"
	case RiskLow:
		return "Low Risk"
	case RiskMedium:
		return "Medium Risk"
	case RiskHigh:
		return "High Risk"
	case RiskCritical:
		return "CRITICAL DANGER"
	default:
		return "Unknown"
	}
}

// GetConfirmationWord returns the word user must type for high-risk commands
func GetConfirmationWord(level RiskLevel) string {
	switch level {
	case RiskHigh:
		return "CONFIRM"
	case RiskCritical:
		return "I UNDERSTAND THE RISK"
	default:
		return ""
	}
}
