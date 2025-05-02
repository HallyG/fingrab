package fingrab

import (
	"os"
	"time"

	"github.com/HallyG/fingrab/internal/export"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	BuildShortSHA = `(missing)`

	RootCmd = &cobra.Command{
		Use:               "fingrab",
		Short:             "Financial data exporter",
		Long:              `A powerful tool to export transactions from various banks.`,
		PersistentPreRunE: setupLogger,
	}
)

func init() {
	RootCmd.SilenceUsage = true
	RootCmd.SilenceErrors = true
	RootCmd.CompletionOptions.DisableDefaultCmd = true
	RootCmd.SetOut(os.Stderr)

	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(exportCmd)

	for _, exportType := range export.All() {
		cmd := newExportCommand(exportType)
		exportCmd.AddCommand(cmd)
	}
}

func setupLogger(cmd *cobra.Command, _ []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	writer := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stderr
		w.TimeFormat = time.RFC3339
		w.PartsExclude = []string{"time", "level"}
	})

	logger := zerolog.New(writer).
		With().
		Timestamp().
		Str("build.sha", BuildShortSHA).
		Logger()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	ctx := logger.WithContext(cmd.Context())
	cmd.SetContext(ctx)

	return nil
}
