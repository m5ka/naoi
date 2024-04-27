package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"go.m5ka.dev/naoi/internal/pipeline"
	"os"
)

var prettyPrint bool

var checkConfigCommand = &cobra.Command{
	Use:   "check FILE",
	Short: "parses a pipeline configuration file to ensure it's valid",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Fprint(os.Stderr, "naoi check requires a single FILE parameter\n")
			os.Exit(1)
		}
		content, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not open file %s\n", args[0])
			os.Exit(1)
		}
		config, err := pipeline.Parse(content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not parse the config file %s\n%s\n", args[0], err.Error())
			os.Exit(1)
		}
		if err := config.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "config file %s was not valid:\n%s\n", args[0], err.Error())
			os.Exit(1)
		}
		if prettyPrint {
			config.PrettyPrint()
		}
	},
}

func init() {
	checkConfigCommand.Flags().BoolVarP(&prettyPrint, "print", "p", false, "pretty print parsed configuration")
	rootCmd.AddCommand(checkConfigCommand)
}
