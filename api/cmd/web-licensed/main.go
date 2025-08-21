package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"os"

	"isxcli/internal/app"
)

// Embedded Next.js frontend files
//go:embed all:frontend/*
var frontendFiles embed.FS

func main() {
	// Create frontend filesystem from embedded files
	var frontendFS fs.FS
	if frontendSubFS, err := fs.Sub(frontendFiles, "frontend"); err == nil {
		frontendFS = frontendSubFS
		slog.Info("Frontend embedded successfully")
	} else {
		slog.Info("Warning: Frontend embedding failed", slog.String("error", err.Error()))
		frontendFS = nil
	}

	// Create application instance
	application, err := app.NewApplication(frontendFS)
	if err != nil {
		slog.Error("Failed to initialize application", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Start application
	if err := application.Run(); err != nil {
		slog.Error("Application error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}