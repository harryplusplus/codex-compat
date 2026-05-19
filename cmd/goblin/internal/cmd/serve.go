package cmd

import (
	"fmt"
	"log"
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
	port := getServerPort()
	logLevel := getLogLevel()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := fmt.Fprint(w, `{"status":"ok"}`); err != nil {
			log.Printf("health check write error: %v", err)
		}
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Fprintf(os.Stderr, "goblin server listening on %s (log level: %s)\n", addr, logLevel)
	return http.ListenAndServe(addr, mux)
}
