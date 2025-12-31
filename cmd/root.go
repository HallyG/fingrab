package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/HallyG/fingrab/internal/api"
	"github.com/HallyG/fingrab/internal/export"
	"github.com/HallyG/fingrab/internal/log"
	"github.com/HallyG/fingrab/internal/monzo"
	monzoexporter "github.com/HallyG/fingrab/internal/monzo/exporter"
	"github.com/HallyG/fingrab/internal/starling"
	starlingexporter "github.com/HallyG/fingrab/internal/starling/exporter"
	"github.com/spf13/cobra"

	_ "embed"
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
	//go:embed cmd_example.txt
	cmdExample string
)

func getBankCommand(exportType export.ExportType) *cobra.Command {
	switch exportType {
	case monzoexporter.ExportTypeMonzo:
		return monzoCmd
	case starlingexporter.ExportTypeStarling:
		return starlingCmd
	default:
		return nil
	}
}

func init() {
	export.Register(starlingexporter.ExportTypeStarling, func(opts export.Options) (export.Exporter, error) {
		client := &http.Client{
			Timeout: opts.Timeout,
		}

		api := starling.New(client, api.WithAuthToken(opts.BearerAuthToken()))
		return starlingexporter.New(api)
	})

	export.Register(monzoexporter.ExportTypeMonzo, func(opts export.Options) (export.Exporter, error) {
		client := &http.Client{
			Timeout: opts.Timeout,
		}

		api := monzo.New(client, api.WithAuthToken(opts.BearerAuthToken()))
		return monzoexporter.New(api)
	})

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolP("no-colour", "", false, "disable coloured output")

	for _, exportType := range export.All() {
		bankCmd := getBankCommand(exportType)
		if bankCmd != nil {
			bankCmd.AddCommand(newTransactionsCommand(exportType))
			bankCmd.AddCommand(newAccountsCommand(exportType))
			rootCmd.AddCommand(bankCmd)
		}
	}
}

func Main(ctx context.Context, args []string, output io.Writer, errOutput io.Writer) error {
	rootCmd.SetOut(output)
	rootCmd.SetErr(errOutput)
	rootCmd.SetArgs(args[1:])

	verbose, _ := rootCmd.Flags().GetBool("verbose")
	logger := log.New(
		log.WithWriter(errOutput),
		log.WithVerbose(verbose),
		log.WithAttrs(
			slog.String("build.version", BuildVersion),
			slog.String("build.sha", BuildShortSHA),
		),
	)
	return rootCmd.ExecuteContext(log.WithContext(ctx, logger))
}

func setupLogger(cmd *cobra.Command, _ []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	noColour, _ := cmd.Flags().GetBool("no-colour")

	handlerOpt := log.WithColourTextHandler()
	if noColour {
		handlerOpt = log.WithTextHandler()
	}

	logger := log.New(
		log.WithWriter(cmd.ErrOrStderr()),
		log.WithVerbose(verbose),
		handlerOpt,
		log.WithAttrs(
			slog.String("build.version", BuildVersion),
			slog.String("build.sha", BuildShortSHA),
		),
	)

	ctx := log.WithContext(cmd.Context(), logger)
	cmd.SetContext(ctx)
	return nil
}
