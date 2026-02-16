// Package lib provides a library interface to the Crush AI assistant.
// You can use this package to embed Crush in your own applications.
package lib

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/ui/anim"
	"github.com/charmbracelet/crush/internal/ui/common"
	ui "github.com/charmbracelet/crush/internal/ui/model"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/crush/internal/format"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	_ "github.com/mattn/go-sqlite3"
	uv "github.com/charmbracelet/ultraviolet"
)

// Config is the configuration for the Crush application.
type Config = config.Config

// App is the main Crush application instance.
type App = app.App

// NewConfig creates a new configuration with the given working directory.
// The data directory will be created as <cwd>/.crush if not specified.
func NewConfig(cwd, dataDir string, debug bool) (*Config, error) {
	return config.Init(cwd, dataDir, debug)
}

// NewApp creates a new Crush application instance.
// You need to call Connect first to get the database connection.
func NewApp(ctx context.Context, conn *sql.DB, cfg *Config) (*App, error) {
	return app.New(ctx, conn, cfg)
}

// Connect connects to the Crush database.
// The dataDir should be the same as used in NewConfig.
func Connect(ctx context.Context, dataDir string) (*sql.DB, error) {
	return db.Connect(ctx, dataDir)
}

// RunTUI runs the Crush TUI (Bubble Tea interface).
// This blocks until the TUI exits.
func RunTUI(ctx context.Context, appInstance *App) error {
	var env uv.Environ = os.Environ()

	com := common.DefaultCommon(appInstance)
	model := ui.New(com)

	program := tea.NewProgram(
		model,
		tea.WithEnvironment(env),
		tea.WithContext(ctx),
		tea.WithFilter(ui.MouseEventFilter),
	)
	go appInstance.Subscribe(program)

	if _, err := program.Run(); err != nil {
		slog.Error("TUI run error", "error", err)
		return err
	}
	return nil
}

// RunWithProgressBar runs the Crush TUI with a progress bar.
// This is similar to running `crush` from the command line.
func RunWithProgressBar(ctx context.Context, cfg *Config, cwd string) error {
	// Check if progress bar is supported
	supportsProgress := supportsProgressBar()
	progressEnabled := cfg.Options.Progress == nil || *cfg.Options.Progress

	if progressEnabled && supportsProgress {
		_, _ = fmt.Fprint(os.Stderr, ansi.SetIndeterminateProgressBar)
		defer func() { _, _ = fmt.Fprintf(os.Stderr, ansi.ResetProgressBar) }()
	}

	// Connect to database
	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		return err
	}

	// Create app instance
	appInstance, err := app.New(ctx, conn, cfg)
	if err != nil {
		return err
	}

	// Initialize TUI
	var env uv.Environ = os.Environ()
	com := common.DefaultCommon(appInstance)
	model := ui.New(com)

	program := tea.NewProgram(
		model,
		tea.WithEnvironment(env),
		tea.WithContext(ctx),
		tea.WithFilter(ui.MouseEventFilter),
	)
	go appInstance.Subscribe(program)

	if _, err := program.Run(); err != nil {
		return err
	}
	return nil
}

// supportsProgressBar checks if the terminal supports progress bars.
func supportsProgressBar() bool {
	if !term.IsTerminal(os.Stderr.Fd()) {
		return false
	}
	termProg := os.Getenv("TERM_PROGRAM")
	_, isWindowsTerminal := os.LookupEnv("WT_SESSION")
	return isWindowsTerminal || strings.Contains(strings.ToLower(termProg), "ghostty")
}

// Styles returns the default styles for the TUI.
func Styles() styles.Styles {
	return styles.DefaultStyles()
}

// NewSpinner creates a new spinner for the TUI.
func NewSpinner(ctx context.Context, cancel context.CancelFunc, settings anim.Settings) *format.Spinner {
	return format.NewSpinner(ctx, cancel, settings)
}

// IsTerminal checks if the given file descriptor is a terminal.
func IsTerminal(fd uintptr) bool {
	return term.IsTerminal(fd)
}

// Shutdown performs cleanup when shutting down the application.
func Shutdown(appInstance *App) {
	appInstance.Shutdown()
}
