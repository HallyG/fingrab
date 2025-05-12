package root

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/HallyG/fingrab/cmd/fingrab/exporter"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	BuildVersion  = `(missing)`
	BuildShortSHA = `(missing)`

	RootCmd = &cobra.Command{
		Use:               "fingrab",
		Short:             "Financial data exporter",
		Long:              `A CLI for exporting financial data from various banks.`,
		PersistentPreRunE: setupLogger,
		Version:           fmt.Sprintf("%s (%s)", BuildVersion, BuildShortSHA),
	}
)

func init() {
	RootCmd.SilenceUsage = true
	RootCmd.SilenceErrors = true
	RootCmd.CompletionOptions.DisableDefaultCmd = true
	RootCmd.SetOut(os.Stderr)

	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	RootCmd.AddCommand(exporter.ExportCmd)
}

func Main(ctx context.Context, args []string, output io.Writer) error {
	RootCmd.SetOut(output)
	RootCmd.SetArgs(args[1:])

	return RootCmd.ExecuteContext(ctx)
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
		Str("build.version", BuildShortSHA).
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
