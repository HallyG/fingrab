package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/HallyG/fingrab/cmd/fingrab"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	if err := run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "err: %s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	fingrab.RootCmd.SetArgs(args[1:])

	if err := fingrab.RootCmd.ExecuteContext(ctx); err != nil {
		return err
	}

	return nil
}
