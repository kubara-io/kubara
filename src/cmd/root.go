package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kubara-io/kubara/internal/k8s"
	"github.com/kubara-io/kubara/internal/updatecheck"
	"github.com/rs/zerolog/log"
	docs "github.com/urfave/cli-docs/v3"
	"github.com/urfave/cli/v3"
)

const AppName = "kubara"

var Authors = []any{
	"Contributors: https://github.com/kubara-io/kubara/graphs/contributors"}

var version string

type rootActionDeps struct {
	notifyUpdate   func(string)
	checkUpdate    func(string) error
	testConnection func(string)
	writeDocs      func(*cli.Command, string) error
}

var defaultRootActionDeps = rootActionDeps{
	notifyUpdate: func(ver string) {
		updatecheck.NotifyIfNewReleaseAvailable(ver, os.Stderr)
	},
	checkUpdate: func(ver string) error {
		if err := updatecheck.PrintLiveCheck(ver, os.Stdout); err != nil {
			return cli.Exit(fmt.Sprintf("Error: update check failed: %v", err), 1)
		}
		return nil
	},
	testConnection: testConnection,
	writeDocs:      writeCommandDocs,
}

// NewRootCmd builds and returns the root CLI command. ver is injected from
// main via ldflags.
func NewRootCmd(ver string) *cli.Command {
	version = ver
	globalFlags := NewGlobalFlags()

	return &cli.Command{
		Name:                  AppName,
		Version:               ver,
		Authors:               Authors,
		Copyright:             "",
		Usage:                 "Opinionated CLI for Kubernetes platform engineering",
		Description:           "kubara is an opinionated CLI to bootstrap and operate Kubernetes platforms with GitOps-first workflows.",
		Flags:                 globalFlags.CLIFlags(),
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			NewInitCmd(),
			NewGenerateCmd(),
			NewBootstrapCmd(),
			NewSchemaCmd(),
		},
		Before: func(ctx context.Context, _ *cli.Command) (context.Context, error) {
			if shouldNotifyStartupUpdate(globalFlags.ToRootOptions()) {
				defaultRootActionDeps.notifyUpdate(ver)
			}
			return ctx, nil
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return newAppAction(cmd, globalFlags.ToRootOptions(), defaultRootActionDeps)
		},
	}
}

func newAppAction(cmd *cli.Command, options RootOptions, deps rootActionDeps) error {
	if strings.TrimSpace(options.DocsOutputPath) != "" {
		return deps.writeDocs(cmd, options.DocsOutputPath)
	}

	if options.Base64Mode {
		return runBase64Mode(options.Base64)
	}

	if cmd.NumFlags() == 0 {
		cli.ShowAppHelpAndExit(cmd, 0)
	}

	if err := executeRootAction(options, deps); err != nil {
		return err
	}

	if !options.TestK8sConnection && !options.CheckUpdateFlag {
		cli.ShowAppHelpAndExit(cmd, 0)
	}

	return nil
}

func shouldNotifyStartupUpdate(options RootOptions) bool {
	return !options.CheckUpdateFlag && strings.TrimSpace(options.DocsOutputPath) == ""
}

func writeCommandDocs(app *cli.Command, outputPath string) error {
	md, err := docs.ToMarkdown(app)
	if err != nil {
		return fmt.Errorf("create markdown: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create docs directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte("# kubara commands\n\n"+md), 0o644); err != nil {
		return fmt.Errorf("write docs file: %w", err)
	}

	log.Info().Str("path", outputPath).Msg("successfully created docs")
	return nil
}

func executeRootAction(options RootOptions, deps rootActionDeps) error {
	switch {
	case options.TestK8sConnection:
		deps.testConnection(options.KubeconfigFilePath)
	case options.CheckUpdateFlag:
		return deps.checkUpdate(version)
	}

	return nil
}

func runBase64Mode(options Base64Options) error {
	if (options.Encode && options.Decode) || (!options.Encode && !options.Decode) {
		return cli.Exit("Error: specify either --encode or --decode", 1)
	}

	if (options.InputString != "" && options.InputFile != "") || (options.InputString == "" && options.InputFile == "") {
		return cli.Exit("Error: specify exactly one of --string or --file", 1)
	}

	var data []byte
	var err error
	if options.InputFile != "" {
		data, err = os.ReadFile(options.InputFile)
		if err != nil {
			log.Fatal().Err(err).Msgf("Cannot read file: %s", options.InputFile)
			return cli.Exit("Error: reading file", 1)
		}
	} else {
		data = []byte(options.InputString)
	}

	if options.Encode {
		fmt.Print(base64.StdEncoding.EncodeToString(data))
		return nil
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid base64 input")
		return cli.Exit("Error: invalid base64 input", 1)
	}

	_, err = os.Stdout.Write(decoded)
	if err != nil {
		return cli.Exit("Error: writing decoded base64 input", 1)
	}

	return nil
}

func testConnection(kubeconfig string) {
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Err(err).Msg("home dir")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	log.Info().Msgf("Testing connection to your cluster using: %s", kubeconfig)
	client, err := k8s.NewClient(k8s.Config{
		KubeconfigPath: kubeconfig,
		Timeout:        30 * time.Second,
		UserAgent:      "kubara/1.0",
	})
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to setup k8s client using: %s", kubeconfig)
	}

	if _, err := client.ListNamespaces(context.Background()); err != nil {
		log.Fatal().Err(err).Msgf("Failed to test k8s connection using")
	}

	log.Info().Msgf("Successful connection to: %s", client.RESTConfig.Host)
}
