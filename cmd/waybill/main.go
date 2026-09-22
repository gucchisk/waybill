// waybill is a CLI for browsing and downloading container image manifests.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/gucchisk/waybill/internal/adapter/filesystem"
	"github.com/gucchisk/waybill/internal/adapter/registry"
	"github.com/gucchisk/waybill/internal/infrastructure/tui"
	"github.com/gucchisk/waybill/internal/usecase"
)

func main() {
	rootCommand, runError := newRootCommand()
	if err := rootCommand.Execute(); err != nil {
		os.Exit(1)
	}
	if *runError != nil {
		fmt.Fprintln(os.Stderr, "waybill:", *runError)
		os.Exit(1)
	}
}

// newRootCommand returns the root command and the destination for runtime errors.
// Returning a runtime error from RunE makes cobra print the usage, so it is handed to main through this destination.
func newRootCommand() (*cobra.Command, *error) {
	var runError error
	theme := tui.ThemeAuto
	rootCommand := &cobra.Command{
		Use:     "waybill <image-ref>",
		Short:   "Browse and download container image manifests",
		Long:    "Show a container image manifest as JSON, follow the objects it references, and download their contents.",
		Example: "  waybill ghcr.io/regclient/regctl:latest",
		Args:    cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			runError = run(command.Context(), arguments[0], theme)
			return nil
		},
	}
	rootCommand.Flags().Var(&theme, "theme", "terminal background: auto, dark, or light (auto detects it, but some terminals, e.g. tmux, may not answer)")
	return rootCommand, &runError
}

func run(parentCtx context.Context, imageReference string, theme tui.Theme) error {
	ctx, stop := signal.NotifyContext(parentCtx, os.Interrupt)
	defer stop()

	workingDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	registryRepository := registry.NewRegclientRepository()
	viewContent := usecase.NewViewContent(registryRepository)
	downloadContent := usecase.NewDownloadContent(registryRepository, filesystem.NewFileWriter(workingDirectory))

	repository, rootDescriptor, err := usecase.NewOpenImage(registryRepository).Execute(ctx, imageReference)
	if err != nil {
		return err
	}
	rootContent, err := viewContent.Execute(ctx, repository, rootDescriptor)
	if err != nil {
		return err
	}

	return tui.Run(ctx, repository, rootDescriptor, rootContent, tui.Dependencies{
		ViewContent:     viewContent,
		DownloadContent: downloadContent,
	}, theme)
}
