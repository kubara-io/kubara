package cmd

import (
	"context"
	"fmt"
	"kubara/assets/app"
	"kubara/assets/catalog"
	"kubara/assets/config"
	"kubara/assets/envmap"
	"kubara/utils"
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
	catalogPath    string
	catalogForce   bool
}

type InitFlags struct {
	PrepFlag      bool
	ForceFlag     bool
	EnvFileFlag   string
	EnvPrefixFlag string
	CatalogPath   string
	CatalogForce  bool
}

func NewInitFlags() *InitFlags {
	return &InitFlags{
		PrepFlag:      false,
		ForceFlag:     false,
		EnvFileFlag:   ".env",
		EnvPrefixFlag: "KUBARA_",
		CatalogPath:   "",
		CatalogForce:  false,
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
		return nil, err
	}
	configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
	if err != nil {
		return nil, err
	}
	dotEnvFilePath, err := utils.GetFullPath(cmd.String("env-file"), cwd)
	if err != nil {
		return nil, err
	}
	catalogPath := ""
	if flags.CatalogPath != "" {
		catalogPath, err = utils.GetFullPath(flags.CatalogPath, cwd)
		if err != nil {
			return nil, fmt.Errorf("failed to get catalog path: %w", err)
		}
	}

	o := &InitOptions{
		copyPrepFolder: flags.PrepFlag,
		force:          flags.ForceFlag,
		cwd:            cwd,
		configFilePath: configFilePath,
		dotEnvFilePath: dotEnvFilePath,
		envVarPrefix:   flags.EnvPrefixFlag,
		catalogPath:    catalogPath,
		catalogForce:   flags.CatalogForce,
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
		&cli.StringFlag{
			Name:        "catalog",
			Value:       flags.CatalogPath,
			Usage:       "Path to external ServiceDefinition catalog/distribution directory.",
			Destination: &flags.CatalogPath,
		},
		&cli.BoolFlag{
			Name:        "force",
			Aliases:     []string{"catalog-overwrite"},
			Value:       flags.CatalogForce,
			Usage:       "Allow external service definitions from --catalog to overwrite built-in definitions on name collisions.",
			Destination: &flags.CatalogForce,
		},
	}

	cmd.Flags = initFlags
}

func (o *InitOptions) Run() error {
	em := envmap.NewEnvMapManager(o.dotEnvFilePath, ".", o.envVarPrefix)
	cm := config.NewConfigManagerWithCatalog(o.configFilePath, o.catalogLoadOptions())

	EnvLoadErr := em.Load()
	CnfLoadErr := cm.Load()
	EnvValidateErr := em.Validate()

	em.SetDefaults()

	if EnvLoadErr != nil {
		log.Error().Msgf("Reading Env failed. %s", EnvLoadErr)
		return EnvLoadErr
	}

	// prep mode
	if o.copyPrepFolder {
		// add or merge .gitignore
		errPrep := utils.AddGitignore(o.cwd)
		if errPrep != nil {
			return errPrep
		}

		_, dotenvStatError := os.Stat(o.dotEnvFilePath)
		if dotenvStatError == nil {
			log.Info().Msgf("Skipping dotenv creation. File exist: %v", em.GetFilepath())
		} else if os.IsNotExist(dotenvStatError) {
			exampleEnvMap, err := em.GenerateEnvExample()
			if err != nil {
				return err
			}
			if errWrite := os.WriteFile(o.dotEnvFilePath, exampleEnvMap, 0600); errWrite != nil {
				return errWrite
			}
			log.Info().Msgf("Generated dotenv in path: %v", em.GetFilepath())
		} else {
			return dotenvStatError
		}
		return nil
	}

	// force mode
	if o.force {
		if EnvValidateErr != nil {
			return fmt.Errorf("error validating env: %w", EnvValidateErr)
		}

		if fileExist, _ := utils.FileExist(cm.GetFilepath()); fileExist {
			app.CreateOrUpdateClusterFromEnvWithCatalog(cm.GetConfig(), em.GetConfig(), o.catalogLoadOptions())
		} else {
			return fmt.Errorf("error loading config file. %s", CnfLoadErr)
		}

		errValidate := cm.Validate()
		if errValidate != nil {
			return fmt.Errorf("error validating config file. %s", errValidate)
		}
		errSave := cm.SaveToFile()
		if errSave != nil {
			return fmt.Errorf("error writing config file. %s", errSave)
		}
		log.Info().Msgf("overwritten config file: %s", cm.GetFilepath())
		log.Info().Msg("Initialized successfully")
		return nil
	}

	// normal mode
	if fileExist, err := utils.FileExist(cm.GetFilepath()); fileExist {
		log.Info().Msgf("Config file already exist. To overwrite existing variables in the config from env: set flag \"--overwrite\"")
		errV := cm.Validate()
		if errV != nil {
			return errV
		}
	} else if err != nil {
		return err
	} else {
		if EnvValidateErr != nil {
			log.Info().Msgf("Env validation error. If you want to generate an example dotenv, pass the \"--prep\" flag.")
			return fmt.Errorf("error validating env: %w", EnvValidateErr)
		}
		newCluster := config.NewClusterFromEnvWithCatalog(em.GetConfig(), o.catalogLoadOptions())
		cm.GetConfig().Clusters = []config.Cluster{newCluster}
		errSave := cm.SaveToFile()
		if errSave != nil {
			return errSave
		}
		log.Info().Msgf("Generated config in path: %v", cm.GetFilepath())
		// return here to not log as successful as no validation was run on config
		return nil
	}

	log.Info().Msg("Initialized successfully")

	return nil

}

func (o *InitOptions) catalogLoadOptions() catalog.LoadOptions {
	return catalog.LoadOptions{
		DistributionPath: o.catalogPath,
		Overwrite:        o.catalogForce,
	}
}
