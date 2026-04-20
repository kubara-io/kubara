package cmd

import (
	"context"
	"errors"
	"fmt"
	"kubara/assets/app"
	"kubara/assets/config"
	"kubara/assets/envmap"
	"kubara/internal/tui"
	"kubara/utils"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

type InitOptions struct {
	copyPrepFolder bool
	force          bool
	nonInteractive bool
	cwd            string
	configFilePath string
	dotEnvFilePath string
	envVarPrefix   string
}

type InitFlags struct {
	PrepFlag       bool
	ForceFlag      bool
	NonInteractive bool
	EnvFileFlag    string
	EnvPrefixFlag  string
}

func NewInitFlags() *InitFlags {
	return &InitFlags{
		PrepFlag:       false,
		ForceFlag:      false,
		NonInteractive: false,
		EnvFileFlag:    ".env",
		EnvPrefixFlag:  "KUBARA_",
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

	o := &InitOptions{
		copyPrepFolder: flags.PrepFlag,
		force:          flags.ForceFlag,
		nonInteractive: flags.NonInteractive,
		cwd:            cwd,
		configFilePath: configFilePath,
		dotEnvFilePath: dotEnvFilePath,
		envVarPrefix:   flags.EnvPrefixFlag,
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
		&cli.BoolFlag{
			Name:        "non-interactive",
			Value:       flags.NonInteractive,
			Usage:       "Disable interactive TUI mode and use env-only init",
			Destination: &flags.NonInteractive,
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
	em := envmap.NewEnvMapManager(o.dotEnvFilePath, ".", o.envVarPrefix)
	cm := config.NewConfigManager(o.configFilePath)

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
			app.CreateOrUpdateClusterFromEnv(cm.GetConfig(), em.GetConfig())
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
		if o.shouldUseInteractiveMode() {
			answers, errUI := tui.RunInitialConfigWizard(tui.AnswersFromEnvMap(em.GetConfig()))
			if errUI != nil {
				if errors.Is(errUI, tui.ErrUserCancelled) {
					log.Info().Msg("Initialization cancelled by user")
					return nil
				}
				return errUI
			}

			answers.ApplyToEnvMap(em.GetConfig())
			if errValidate := em.Validate(); errValidate != nil {
				return fmt.Errorf("error validating env: %w", errValidate)
			}

			newCluster := config.NewClusterFromEnv(em.GetConfig())
			applyServiceSelection(&newCluster.Services, answers.Services)
			cm.GetConfig().Clusters = []config.Cluster{newCluster}
			errSave := cm.SaveToFile()
			if errSave != nil {
				return errSave
			}
			log.Info().Msgf("Generated config in path: %v", cm.GetFilepath())
			return nil
		}

		if EnvValidateErr != nil {
			log.Info().Msgf("Env validation error. If you want to generate an example dotenv, pass the \"--prep\" flag.")
			return fmt.Errorf("error validating env: %w", EnvValidateErr)
		}
		newCluster := config.NewClusterFromEnv(em.GetConfig())
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

func (o *InitOptions) shouldUseInteractiveMode() bool {
	if o.nonInteractive {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func setGenericService(enabled bool, svc *config.GenericService) {
	if enabled {
		svc.Status = config.StatusEnabled
		return
	}
	svc.Status = config.StatusDisabled
}

func applyServiceSelection(services *config.Services, selected map[string]bool) {
	for key, enabled := range selected {
		switch key {
		case "argocd":
			setGenericService(enabled, &services.Argocd)
		case "cert-manager":
			if enabled {
				services.CertManager.Status = config.StatusEnabled
			} else {
				services.CertManager.Status = config.StatusDisabled
			}
		case "external-dns":
			setGenericService(enabled, &services.ExternalDns)
		case "external-secrets":
			setGenericService(enabled, &services.ExternalSecrets)
		case "kube-prometheus-stack":
			if enabled {
				services.KubePrometheusStack.Status = config.StatusEnabled
			} else {
				services.KubePrometheusStack.Status = config.StatusDisabled
			}
		case "traefik":
			setGenericService(enabled, &services.Traefik)
		case "kyverno":
			setGenericService(enabled, &services.Kyverno)
		case "kyverno-policies":
			setGenericService(enabled, &services.KyvernoPolicies)
		case "kyverno-policy-reporter":
			setGenericService(enabled, &services.KyvernoPolicyReport)
		case "loki":
			if enabled {
				services.Loki.Status = config.StatusEnabled
			} else {
				services.Loki.Status = config.StatusDisabled
			}
		case "homer-dashboard":
			setGenericService(enabled, &services.HomerDashboard)
		case "oauth2-proxy":
			setGenericService(enabled, &services.Oauth2Proxy)
		case "metrics-server":
			setGenericService(enabled, &services.MetricsServer)
		case "metallb":
			setGenericService(enabled, &services.MetalLb)
		case "longhorn":
			setGenericService(enabled, &services.Longhorn)
		}
	}
}
