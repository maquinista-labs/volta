package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "volta",
	Short: "Unified agent orchestration platform",
	Long:  "Volta combines Telegram bot management, pull-based task coordination, and pluggable agent runners into a single CLI.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("volta", version)
	},
}

func init() {
	rootCmd.PersistentFlags().String("db", "", "database URL (overrides DATABASE_URL)")
	rootCmd.PersistentFlags().String("session", "", "tmux session name (overrides VOLTA_SESSION)")
	rootCmd.PersistentFlags().String("config", "", "config file path")

	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
