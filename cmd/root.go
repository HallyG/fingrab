package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/HallyG/fingrab/internal/export"
	"github.com/HallyG/fingrab/internal/monzo"
	monzoexporter "github.com/HallyG/fingrab/internal/monzo/exporter"
	"github.com/HallyG/fingrab/internal/starling"
	starlingexporter "github.com/HallyG/fingrab/internal/starling/exporter"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	BuildVersion  = `(missing)`
	BuildShortSHA = `(missing)`

	rootCmd = &cobra.Command{
		Use:               "fingrab",
		Short:             "Financial data exporter",
		Long:              `A CLI for exporting financial data from various banks.`,
		PersistentPreRunE: setupLogger,
		Version:           fmt.Sprintf("%s (%s)", BuildVersion, BuildShortSHA),
	}
)

func init() {
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	for _, exportType := range export.All() {
		cmd := newExportCommand(exportType)
		exportCmd.AddCommand(cmd)
	}

	rootCmd.AddCommand(exportCmd)
}

func init() {
	export.Register(starlingexporter.ExportTypeStarling, func(opts export.Options) (export.Exporter, error) {
		client := &http.Client{
			Timeout: opts.Timeout,
		}

		api := starling.New(client, starling.WithAuthToken(opts.BearerAuthToken()))
		return starlingexporter.New(api)
	})

	export.Register(monzoexporter.ExportTypeMonzo, func(opts export.Options) (export.Exporter, error) {
		client := &http.Client{
			Timeout: opts.Timeout,
		}

		api := monzo.New(client, monzo.WithAuthToken(opts.BearerAuthToken()))
		return monzoexporter.New(api)
	})
}

func Main(ctx context.Context, args []string, output io.Writer, errOutput io.Writer) error {
	rootCmd.SetOut(output)
	rootCmd.SetErr(errOutput)
	rootCmd.SetArgs(args[1:])

	return rootCmd.ExecuteContext(ctx)
}

func setupLogger(cmd *cobra.Command, _ []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	writer := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = cmd.ErrOrStderr()
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
