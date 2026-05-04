package cmd

import (
	"context"
	"fmt"
	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/utils"
	"github.com/kubara-io/kubara/internal/workflow"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

type InitOptions struct {
	copyPrepFolder bool
	force          bool
	cwd            string
	configFilePath string
	dotEnvFilePath string
	envVarPrefix   string
	catalogOptions catalog.LoadOptions
}

type InitFlags struct {
	PrepFlag      bool
	ForceFlag     bool
	EnvFileFlag   string
	EnvPrefixFlag string
}

func NewInitFlags() *InitFlags {
	return &InitFlags{
		PrepFlag:      false,
		ForceFlag:     false,
		EnvFileFlag:   ".env",
		EnvPrefixFlag: "KUBARA_",
	}
}

func NewInitCmd() *cli.Command {
	flags := NewInitFlags()

	cmd := &cli.Command{
		Name:  "init",
		Usage: "Initialize a new kubara directory",
		Action: func(c context.Context, cmd *cli.Command) error {
			o, _ := flags.ToOptions(cmd)
			return o.Run()
		},
	}

	flags.AddFlags(cmd)

	return cmd
}

func (flags *InitFlags) ToOptions(cmd *cli.Command) (*InitOptions, error) {
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get config file path: %w", err)
	}
	dotEnvFilePath, err := utils.GetFullPath(cmd.String("env-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get env file path: %w", err)
	}
	catalogOptions, err := catalogLoadOptionsFromCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("get catalog options: %w", err)
	}

	o := &InitOptions{
		copyPrepFolder: flags.PrepFlag,
		force:          flags.ForceFlag,
		cwd:            cwd,
		configFilePath: configFilePath,
		dotEnvFilePath: dotEnvFilePath,
		envVarPrefix:   flags.EnvPrefixFlag,
		catalogOptions: catalogOptions,
	}
	return o, nil
}

func (flags *InitFlags) AddFlags(cmd *cli.Command) {
	initFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:        "prep",
			Value:       flags.PrepFlag,
			Usage:       "Copy embedded prep/ folder into current working directory",
			Destination: &flags.PrepFlag,
		},
		&cli.BoolFlag{
			Name:        "overwrite",
			Value:       flags.ForceFlag,
			Usage:       "Overwrite config if exists",
			Destination: &flags.ForceFlag,
		},
		&cli.StringFlag{
			Name:        "envVarPrefix",
			Value:       flags.EnvPrefixFlag,
			Usage:       "Prefix for envs read from envVars",
			Destination: &flags.EnvPrefixFlag,
		},
	}

	cmd.Flags = initFlags
}

func (o *InitOptions) Run() error {
	es := envconfig.NewEnvStore(o.dotEnvFilePath, ".", o.envVarPrefix)
	cs := config.NewConfigStoreWithCatalog(o.configFilePath, o.catalogLoadOptions())

	envLoadErr := es.Load()
	configLoadErr := cs.Load()
	envValidateErr := es.Validate()

	es.SetDefaults()

	if envLoadErr != nil {
		log.Error().Msgf("Reading Env failed. %s", envLoadErr)
		return envLoadErr
	}

	if o.copyPrepFolder {
		return o.runPrepMode(es)
	}

	if o.force {
		return o.runForceMode(es, cs, envValidateErr, configLoadErr)
	}

	return o.runNormalMode(es, cs, envValidateErr)
}

func (o *InitOptions) catalogLoadOptions() catalog.LoadOptions {
	return o.catalogOptions
}

func (o *InitOptions) runPrepMode(es *envconfig.EnvStore) error {
	if err := utils.AddGitignore(o.cwd); err != nil {
		return err
	}

	_, err := os.Stat(o.dotEnvFilePath)
	if err == nil {
		log.Info().Msgf("Skipping dotenv creation. File exist: %v", es.GetFilepath())
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	exampleEnvMap, err := es.GenerateEnvExample()
	if err != nil {
		return err
	}
	if err := os.WriteFile(o.dotEnvFilePath, exampleEnvMap, 0600); err != nil {
		return err
	}

	log.Info().Msgf("Generated dotenv in path: %v", es.GetFilepath())
	return nil
}

func (o *InitOptions) runForceMode(es *envconfig.EnvStore, cs *config.ConfigStore, envValidateErr, configLoadErr error) error {
	if envValidateErr != nil {
		return fmt.Errorf("validate env: %w", envValidateErr)
	}

	fileExists, _ := utils.FileExist(cs.GetFilepath())
	if !fileExists {
		return fmt.Errorf("load config file: %w", configLoadErr)
	}

	if err := workflow.CreateOrUpdateClusterFromEnvWithCatalog(cs.GetConfig(), es.GetConfig(), o.catalogLoadOptions()); err != nil {
		return fmt.Errorf("create or update cluster from env: %w", err)
	}
	if err := cs.Validate(); err != nil {
		return fmt.Errorf("validate config file: %w", err)
	}
	if err := cs.SaveToFile(); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	log.Info().Msgf("overwritten config file: %s", cs.GetFilepath())
	log.Info().Msg("Initialized successfully")
	return nil
}

func (o *InitOptions) runNormalMode(es *envconfig.EnvStore, cs *config.ConfigStore, envValidateErr error) error {
	fileExists, err := utils.FileExist(cs.GetFilepath())
	if err != nil {
		return err
	}

	if fileExists {
		log.Info().Msgf("Config file already exist. To overwrite existing variables in the config from env: set flag \"--overwrite\"")
		if err := cs.Validate(); err != nil {
			return err
		}
		log.Info().Msg("Initialized successfully")
		return nil
	}

	if envValidateErr != nil {
		log.Info().Msgf("Env validation error. If you want to generate an example dotenv, pass the \"--prep\" flag.")
		return fmt.Errorf("validate env: %w", envValidateErr)
	}

	newCluster, err := config.NewClusterFromEnvWithCatalog(es.GetConfig(), o.catalogLoadOptions())
	if err != nil {
		return fmt.Errorf("create cluster from env: %w", err)
	}

	cs.GetConfig().Clusters = []config.Cluster{newCluster}
	if err := cs.SaveToFile(); err != nil {
		return err
	}

	log.Info().Msgf("Generated config in path: %v", cs.GetFilepath())
	return nil
}
