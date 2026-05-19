package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string

	// version is the build version injected at build time via ldflags.
	version = "0.0.0"
)

var rootCmd = &cobra.Command{
	Use:   "goblin",
	Short: "Codex app adapter server for any OpenAI-compatible provider",
	Long: `Goblin is an adapter server that lets you use the Codex app with any
OpenAI-compatible provider.

It implements the OpenAI Responses API that Codex CLI uses, as well
as reverse-engineers Codex app-level requests such as title generation.`,
	Version: version,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.goblin.toml)")
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func initConfig() {
	setDefaults()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("toml")
		viper.SetConfigName(".goblin")
	}

	viper.SetEnvPrefix("goblin")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
