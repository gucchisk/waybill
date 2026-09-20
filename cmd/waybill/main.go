// waybill はコンテナイメージの manifest を閲覧・ダウンロードする CLI。
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/gucchisk/waybill/internal/adapter/filesystem"
	"github.com/gucchisk/waybill/internal/adapter/registry"
	"github.com/gucchisk/waybill/internal/infrastructure/tui"
	"github.com/gucchisk/waybill/internal/usecase"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "waybill:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) != 1 || arguments[0] == "-h" || arguments[0] == "--help" {
		return fmt.Errorf("usage: waybill <image-reference>  (e.g. waybill ghcr.io/regclient/regctl:latest)")
	}
	imageReference := arguments[0]

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
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
	})
}
