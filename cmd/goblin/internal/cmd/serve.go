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

func initSlog() error {
	logLevel := getLogLevel()
	var level slog.Level
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		return fmt.Errorf("invalid log_level %q: %w", logLevel, err)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
	slog.Info("log level set", "level", logLevel)
	return nil
}

func validateTitleModel() error {
	titleModel := getTitleModel()
	if titleModel == "" {
		return fmt.Errorf("title_model is required")
	}
	models := getAllModelConfigs()
	if _, ok := models[titleModel]; !ok {
		return fmt.Errorf("title_model %q not found in [models]", titleModel)
	}
	slog.Info("using title_model for title generation", "model", titleModel)
	return nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := fmt.Fprint(w, `{"status":"ok"}`); err != nil {
		slog.Error("health check write error", "err", err)
	}
}

func runServe() error {
	if err := initSlog(); err != nil {
		return err
	}

	if err := validateTitleModel(); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)

	addr := fmt.Sprintf("%s:%d", getServerHost(), getServerPort())
	slog.Info("server starting", "addr", addr)
	return http.ListenAndServe(addr, mux)
}
