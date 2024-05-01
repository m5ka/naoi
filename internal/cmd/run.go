package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"go.m5ka.dev/naoi/internal/pipeline"
	"go.m5ka.dev/naoi/internal/runner"
	"os"
)

var runCmd = &cobra.Command{
	Use:   "run FILE",
	Short: "runs the pipeline specified by FILE",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "naoi run requires a single FILE argument")
			os.Exit(1)
		}

		content, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not open file: %s\n", args[0])
			os.Exit(1)
		}

		config, err := pipeline.Parse(content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not parse the config file %s:\n%s\n", args[0], err.Error())
			os.Exit(1)
		}

		if err := config.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "config file %s was not valid:\n%s\n", args[0], err.Error())
			os.Exit(1)
		}

		r, err := runner.New(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not create runner for pipeline %s:\n%s\n", args[0], err.Error())
			os.Exit(1)
		}
		defer func() {
			if err := r.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "fatal exception encountered terminating runner: %s\n", err.Error())
			}
		}()

		if ret, err := r.Run(); ret > 0 || err != nil {
			os.Exit(ret)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
