package cmd

import (
	"github.com/spf13/cobra"
)

var (
	starlingCmd = &cobra.Command{
		Use:   "starling",
		Short: "Starling bank commands",
		Long:  "Commands for interacting with the Starling API",
	}
)
