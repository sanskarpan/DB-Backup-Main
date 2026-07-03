package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

// Supported shells for completion generation.
const (
	shellBash       = "bash"
	shellZsh        = "zsh"
	shellFish       = "fish"
	shellPowerShell = "powershell"

	osDarwin = "darwin"
)

// completionCmd represents the completion command.
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate completion scripts for your shell.

To load completions:

Bash:
  $ source <(db-backup completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ db-backup completion bash > /etc/bash_completion.d/db-backup
  # macOS:
  $ db-backup completion bash > /usr/local/etc/bash_completion.d/db-backup

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ db-backup completion zsh > "${fpath[1]}/_db-backup"

  # You will need to start a new shell for this setup to take effect.

  # For oh-my-zsh users:
  $ db-backup completion zsh > ~/.oh-my-zsh/completions/_db-backup

Fish:
  $ db-backup completion fish | source

  # To load completions for each session, execute once:
  $ db-backup completion fish > ~/.config/fish/completions/db-backup.fish

PowerShell:
  PS> db-backup completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> db-backup completion powershell > db-backup.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{shellBash, shellZsh, shellFish, shellPowerShell},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE:                  runCompletion,
}

func init() {
	rootCmd.AddCommand(completionCmd)

	completionCmd.Flags().Bool("install", false, "Install completion script to system location")
	completionCmd.Flags().String("output", "", "Output file (default: stdout)")
}

func runCompletion(cmd *cobra.Command, args []string) error {
	shell := args[0]

	install := flagBool(cmd, "install")
	output := flagString(cmd, "output")

	if install {
		return installCompletion(shell)
	}

	if output != "" {
		return generateCompletionToFile(cmd, shell, output)
	}

	return generateCompletion(cmd, shell)
}

func generateCompletion(cmd *cobra.Command, shell string) error {
	switch shell {
	case shellBash:
		return cmd.Root().GenBashCompletion(os.Stdout)
	case shellZsh:
		return cmd.Root().GenZshCompletion(os.Stdout)
	case shellFish:
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	case shellPowerShell:
		return cmd.Root().GenPowerShellCompletion(os.Stdout)
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

func generateCompletionToFile(cmd *cobra.Command, shell, output string) error {
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	switch shell {
	case shellBash:
		return cmd.Root().GenBashCompletion(file)
	case shellZsh:
		return cmd.Root().GenZshCompletion(file)
	case shellFish:
		return cmd.Root().GenFishCompletion(file, true)
	case shellPowerShell:
		return cmd.Root().GenPowerShellCompletion(file)
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

func installCompletion(shell string) error {
	var installPath string

	switch shell {
	case shellBash:
		installPath = getBashCompletionPath()
	case shellZsh:
		installPath = getZshCompletionPath()
	case shellFish:
		installPath = getFishCompletionPath()
	case shellPowerShell:
		installPath = getPowerShellCompletionPath()
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}

	// Read the completion script from embedded files or generate it
	if installPath == "" {
		return fmt.Errorf("could not determine installation path for %s", shell)
	}

	// Ensure directory exists
	dir := filepath.Dir(installPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Generate completion script
	file, err := os.Create(installPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", installPath, err)
	}
	defer file.Close()

	var genErr error
	switch shell {
	case shellBash:
		genErr = rootCmd.GenBashCompletion(file)
	case shellZsh:
		genErr = rootCmd.GenZshCompletion(file)
	case shellFish:
		genErr = rootCmd.GenFishCompletion(file, true)
	case shellPowerShell:
		genErr = rootCmd.GenPowerShellCompletion(file)
	}

	if genErr != nil {
		return fmt.Errorf("failed to generate completion: %w", genErr)
	}

	fmt.Printf("Completion script installed to: %s\n", installPath)
	fmt.Println("\nPlease restart your shell or source the completion file.")

	switch shell {
	case shellBash:
		fmt.Printf("\nRun: source %s\n", installPath)
	case shellZsh:
		fmt.Println("\nRun: exec zsh")
	case shellFish:
		fmt.Println("\nRun: exec fish")
	case shellPowerShell:
		fmt.Printf("\nRun: . %s\n", installPath)
	}

	return nil
}

func getBashCompletionPath() string {
	if runtime.GOOS == osDarwin {
		// macOS with Homebrew
		if _, err := os.Stat("/usr/local/etc/bash_completion.d"); err == nil {
			return "/usr/local/etc/bash_completion.d/db-backup"
		}
		if _, err := os.Stat("/opt/homebrew/etc/bash_completion.d"); err == nil {
			return "/opt/homebrew/etc/bash_completion.d/db-backup"
		}
	}

	// Linux
	if _, err := os.Stat("/etc/bash_completion.d"); err == nil {
		return "/etc/bash_completion.d/db-backup"
	}

	// User-local fallback
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".bash_completion.d", "db-backup")
}

func getZshCompletionPath() string {
	// oh-my-zsh
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	ohmyzsh := filepath.Join(home, ".oh-my-zsh", "completions", "_db-backup")
	if _, statErr := os.Stat(filepath.Dir(ohmyzsh)); statErr == nil {
		return ohmyzsh
	}

	// Standard zsh
	if runtime.GOOS == osDarwin {
		// macOS with Homebrew
		if _, statErr := os.Stat("/usr/local/share/zsh/site-functions"); statErr == nil {
			return "/usr/local/share/zsh/site-functions/_db-backup"
		}
		if _, statErr := os.Stat("/opt/homebrew/share/zsh/site-functions"); statErr == nil {
			return "/opt/homebrew/share/zsh/site-functions/_db-backup"
		}
	}

	// User-local fallback
	return filepath.Join(home, ".zfunc", "_db-backup")
}

func getFishCompletionPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// User config
	userConfig := filepath.Join(home, ".config", "fish", "completions", "db-backup.fish")
	if _, statErr := os.Stat(filepath.Dir(userConfig)); statErr == nil {
		return userConfig
	}

	// System-wide (macOS with Homebrew)
	if runtime.GOOS == osDarwin {
		if _, statErr := os.Stat("/usr/local/share/fish/vendor_completions.d"); statErr == nil {
			return "/usr/local/share/fish/vendor_completions.d/db-backup.fish"
		}
		if _, statErr := os.Stat("/opt/homebrew/share/fish/vendor_completions.d"); statErr == nil {
			return "/opt/homebrew/share/fish/vendor_completions.d/db-backup.fish"
		}
	}

	// Create user config dir and return path
	if mkErr := os.MkdirAll(filepath.Dir(userConfig), 0o755); mkErr != nil {
		return ""
	}
	return userConfig
}

func getPowerShellCompletionPath() string {
	if runtime.GOOS != "windows" {
		return ""
	}

	// PowerShell profile directory
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return ""
	}

	// Documents\PowerShell\Modules (PowerShell 7+)
	psModules := filepath.Join(userProfile, "Documents", "PowerShell", "Modules", "DbBackupCompletion")
	if _, err := os.Stat(filepath.Dir(psModules)); err == nil {
		return filepath.Join(psModules, "db-backup.ps1")
	}

	// Documents\WindowsPowerShell\Modules (Windows PowerShell 5.1)
	wsModules := filepath.Join(userProfile, "Documents", "WindowsPowerShell", "Modules", "DbBackupCompletion")
	return filepath.Join(wsModules, "db-backup.ps1")
}
