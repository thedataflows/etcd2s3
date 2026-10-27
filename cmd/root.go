package cmd

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/thedataflows/etcd2s3/pkg/appconfig"
)

const (
	PKG_CMD = "cmd"
	KEY_PKG = "pkg"
)

// CLI represents the main CLI structure
type CLI struct {
	LogLevel  string              `kong:"help='Log level (trace,debug,info,warn,error)',default='info'"`
	LogFormat string              `kong:"help='Log format (console,json)',default='console'"`
	Version   VersionCmd          `kong:"cmd,help='Show version information'"`
	Snapshot  SnapshotCmd         `kong:"cmd,help='Take a snapshot of etcd and upload to S3'"`
	Restore   RestoreCmd          `kong:"cmd,help='Restore etcd from a snapshot stored in S3'"`
	List      ListCmd             `kong:"cmd,help='List snapshots stored locally and in S3'"`
	Cleanup   CleanupCmd          `kong:"cmd,help='Delete snapshots based on retention policies'"`
	Config    appconfig.AppConfig `kong:"embed"`
}

// AfterApply is called after Kong parses the CLI but before the command runs
func (cli *CLI) AfterApply(ctx *kong.Context) error {
	// Skip initialization for version command
	if ctx.Command() == "version" || slices.Contains(ctx.Args, "--help") || slices.Contains(ctx.Args, "-h") {
		return nil
	}

	// Set log level and format
	level, err := zerolog.ParseLevel(cli.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to set log level: %w", err)
	}
	zerolog.SetGlobalLevel(level)

	switch cli.LogFormat {
	case "", "console":
		log.Logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Logger()
	case "json":
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	default:
		return fmt.Errorf("failed to set log format: invalid log format '%s'. Supported formats: [console json]", cli.LogFormat)
	}

	return nil
}

// Run executes the CLI with the given version
func Run(version string, args []string) error {
	// Optionally load .env file if it exists
	_ = godotenv.Load(
		".env",             // Current directory
		".env.local",       // Local overrides (common in web development)
		".env.development", // Development environment
	)

	var cli CLI

	parser, err := kong.New(&cli,
		kong.Name("etcd2s3"),
		kong.Description("A CLI tool for managing etcd snapshots to and from S3"),
		kong.Configuration(kongyaml.Loader),
		kong.UsageOnError(),
		kong.DefaultEnvars(""),
	)
	if err != nil {
		return fmt.Errorf("failed to create CLI parser: %w", err)
	}

	ctx, err := parser.Parse(args)
	if slices.Contains(args, "--help") || slices.Contains(args, "-h") {
		return nil
	}
	if err != nil {
		parser.FatalIfErrorf(err)
		return err
	}

	// Check if this is the version command - handle it specially without logging/config
	if ctx.Command() == "version" {
		cliCtx := &CLIContext{
			Version: version,
		}
		return ctx.Run(cliCtx)
	}

	// Create CLI context with shared config and S3 factory
	cliCtx := NewCLIContext(version, &cli.Config)

	log.Info().Str(KEY_PKG, PKG_CMD).Str("app", ctx.Model.Name).Str("version", version).Msg("Starting application")

	return ctx.Run(cliCtx)
}
