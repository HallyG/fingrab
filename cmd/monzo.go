package cmd

import (
	"github.com/spf13/cobra"
)

var (
	monzoCmd = &cobra.Command{
		Use:   "monzo",
		Short: "Monzo bank commands",
		Long:  "Commands for interacting with the Monzo API",
	}
)
