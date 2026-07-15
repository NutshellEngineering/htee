package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"app.getnutshell/htee/internal/config"
	"app.getnutshell/htee/internal/openapi"
	"app.getnutshell/htee/internal/tui"
)

// newInitCommand builds the `ht init` subcommand: an interactive wizard
// that scaffolds a project-local .ht/conf.toml. Given an OpenAPI 3 spec
// path, it pre-populates the wizard's servers, suggested entrypoint, and
// openapispec from that file instead of asking for them.
func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init [SWAGGER_FILE]",
		Short: "Create a .ht/conf.toml configuration file",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	tuiOpts := tui.Options{Exists: config.Exists(dir)}
	if len(args) == 1 {
		specPath := args[0]
		extracted, err := openapi.Load(specPath)
		if err != nil {
			return err
		}
		tuiOpts.PresetServers = extracted.Servers
		tuiOpts.SuggestedEntrypoint = extracted.Entrypoint
		tuiOpts.PresetOpenAPISpec = specPath
	}

	cfg, confirmed, err := tui.Run(tuiOpts)
	if err != nil {
		return err
	}
	if !confirmed {
		_, fe := fmt.Fprintln(cmd.OutOrStdout(), "ht init: cancelled, nothing written")
		if fe != nil {
			panic(fe)
		}
		return nil
	}

	if err := config.Write(dir, cfg); err != nil {
		return err
	}
	_, fe := fmt.Fprintln(cmd.OutOrStdout(), "wrote "+config.Path(dir))
	if fe != nil {
		panic(fe)
	}
	return nil
}
