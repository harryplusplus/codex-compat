package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Goblin adapter server",
	Long: `Start the adapter server that translates between Codex App / Codex CLI
and any OpenAI-compatible provider.

The server implements the OpenAI Responses API and handles app-level
features such as title generation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServe()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe() error {
	host := getServerHost()
	port := getServerPort()
	logLevel := getLogLevel()

	var level slog.Level
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		return fmt.Errorf("invalid log_level %q: %w", logLevel, err)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := fmt.Fprint(w, `{"status":"ok"}`); err != nil {
			slog.Error("health check write error", "err", err)
		}
	})

	addr := fmt.Sprintf("%s:%d", host, port)
	slog.Info("server starting", "addr", addr, "log_level", logLevel)
	return http.ListenAndServe(addr, mux)
}
