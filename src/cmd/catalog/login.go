package catalog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	internal "github.com/kubara-io/kubara/internal/catalog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

type catalogLoginFlags struct {
	insecure           bool
	username           string
	password           bool
	passwordStdin      bool
	identityToken      bool
	identityTokenStdin bool
}

func NewCatalogLogin() *cli.Command {
	flags := &catalogLoginFlags{}
	cmd := &cli.Command{
		Name:        "login",
		Usage:       "Log into a registry and store credentials",
		UsageText:   "kubara catalog login [flags] <registry>",
		Description: "Asks for your login credentials and stores them under $HOME/.kubara/credentials.json for future registry interactions.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "username",
				Aliases:     []string{"u"},
				Usage:       "Log in using username and password",
				Destination: &flags.username,
			},
			&cli.BoolFlag{
				Name:        "password",
				Aliases:     []string{"p"},
				Usage:       "Log in with password interactively",
				Destination: &flags.password,
			},
			&cli.BoolFlag{
				Name:        "password-stdin",
				Usage:       "Log in with password from stdin",
				Destination: &flags.passwordStdin,
			},
			&cli.BoolFlag{
				Name:        "identity-token",
				Usage:       "Log in with identity token interactively",
				Destination: &flags.identityToken,
			},
			&cli.BoolFlag{
				Name:        "identity-token-stdin",
				Usage:       "Log in with identity token from stdin",
				Destination: &flags.identityTokenStdin,
			},
			&cli.BoolFlag{
				Name:        "insecure",
				Usage:       "Ignore TLS certificate verification issues for registry connections.",
				Destination: &flags.insecure,
			},
		},
		Action: func(c context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				cli.ShowSubcommandHelpAndExit(cmd, 1)
			}

			loginOptions, err := resolveCatalogLoginOptions(flags)
			if err != nil {
				return err
			}
			loginOptions.Registry = cmd.Args().First()

			result, err := internal.LoginRegistry(c, loginOptions)
			if err != nil {
				return err
			}

			log.Info().Msgf(
				"Credentials for registry %q have been stored in %s",
				result.Registry,
				result.CredentialsPath,
			)
			return nil
		},
	}

	return cmd
}

func resolveCatalogLoginOptions(flags *catalogLoginFlags) (internal.LoginOptions, error) {
	var err error

	if flags.password && flags.passwordStdin {
		return internal.LoginOptions{}, fmt.Errorf("password and password-stdin cannot be used together")
	}
	if flags.identityToken && flags.identityTokenStdin {
		return internal.LoginOptions{}, fmt.Errorf("identity-token and identity-token-stdin cannot be used together")
	}

	usesPasswordInputs := flags.username != "" || flags.password || flags.passwordStdin
	usesIdentityTokenAuth := flags.identityToken || flags.identityTokenStdin

	if usesPasswordInputs && usesIdentityTokenAuth {
		return internal.LoginOptions{}, fmt.Errorf("username/password and identity token authentication cannot be combined")
	}

	if usesIdentityTokenAuth {
		var identityToken string

		switch {
		case flags.identityTokenStdin:
			identityToken, err = readLine(io.Discard, "", false)
			if err != nil {
				return internal.LoginOptions{}, fmt.Errorf("read identity token from stdin: %w", err)
			}

		default:
			identityToken, err = readLine(os.Stdout, "Identity Token: ", true)
			if err != nil {
				return internal.LoginOptions{}, fmt.Errorf("read identity token: %w", err)
			}
		}

		return internal.LoginOptions{
			IdentityToken: identityToken,
			Insecure:      flags.insecure,
		}, nil
	}

	username := flags.username
	if username == "" {
		username, err = readLine(os.Stdout, "Username: ", false)
		if err != nil {
			return internal.LoginOptions{}, fmt.Errorf("read username: %w", err)
		}
	}

	var password string
	switch {
	case flags.passwordStdin:
		password, err = readLine(io.Discard, "", false)
		if err != nil {
			return internal.LoginOptions{}, fmt.Errorf("read password from stdin: %w", err)
		}
	default:
		password, err = readLine(os.Stdout, "Password: ", true)
		if err != nil {
			return internal.LoginOptions{}, fmt.Errorf("read password: %w", err)
		}
	}

	return internal.LoginOptions{
		Username: username,
		Password: password,
		Insecure: flags.insecure,
	}, nil
}

func readLine(outWriter io.Writer, prompt string, silent bool) (string, error) {
	var err error
	var raw []byte

	_, _ = fmt.Fprint(outWriter, prompt)

	// silent / password safe terminal input
	fd := int(os.Stdin.Fd())
	if silent && term.IsTerminal(fd) {
		if raw, err = term.ReadPassword(fd); err == nil {
			_, err = fmt.Fprintln(outWriter)
		}
	} else {
		// If not silent, or when stdin is not a terminal, read directly while
		// ensuring to remove trailing CR/LF bytes.
		reader := os.Stdin
		var line []byte
		var buffer [1]byte

		for {
			var n int
			n, err = reader.Read(buffer[:])
			if err != nil {
				if err == io.EOF {
					break
				}
				return "", err
			}
			if n == 0 {
				continue
			}
			c := buffer[0]
			if c == '\n' {
				break
			}
			line = append(line, c)
		}
		raw = bytes.TrimSuffix(line, []byte{'\r'})
	}
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
