package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh]",
	Short: "Generate shell completion scripts",
	Long: `To load completions:

Bash:
  source <(qix completion bash)
  # To load completions for each session, execute once:
  # Linux:
  qix completion bash > /etc/bash_completion.d/qix
  # macOS:
  qix completion bash > /usr/local/etc/bash_completion.d/qix

Zsh:
  qix completion zsh > "${fpath[1]}/_qix"
  autoload -U compinit && compinit
`,
	ValidArgs: []string{"bash", "zsh"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
