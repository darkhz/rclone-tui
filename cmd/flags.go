package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/darkhz/rclone-tui/rclone"
	"github.com/jnovack/flag"
)

const appVersion = "v0.0.2"

// CmdOptions stores command-line options.
type CmdOptions struct {
	Page             string
	Host, User, Pass string
	Version          bool
}

var cmdOptions CmdOptions

// ParseFlags parses the command-line flags.
func ParseFlags() error {
	configFile, err := ConfigPath("config")
	if err != nil {
		return err
	}

	fs := flag.NewFlagSetWithEnvPrefix("rclone-tui", "RCLONETUI", flag.ExitOnError)

	fs.StringVar(
		&cmdOptions.Page,
		"page",
		"",
		"Load the specified page (one of dashboard, configuration, explorer, mounts).",
	)
	fs.StringVar(
		&cmdOptions.Host,
		"host",
		"",
		"Specify a rclone host to connect to.",
	)
	fs.StringVar(
		&cmdOptions.User,
		"user",
		"",
		"Specify a login username.",
	)
	fs.StringVar(
		&cmdOptions.Pass,
		"password",
		"",
		"Specify a login password.",
	)
	fs.BoolVar(
		&cmdOptions.Version,
		"version",
		false,
		"Show version.",
	)

	fs.Usage = func() {
		fmt.Fprintf(
			flag.CommandLine.Output(),
			"rclone-tui [<flags>]\n\nConfig file is %s\n\nFlags:\n",
			configFile,
		)

		fs.VisitAll(func(f *flag.Flag) {
			s := fmt.Sprintf("  --%s", f.Name)

			if len(s) <= 4 {
				s += "\t"
			} else {
				s += "\n    \t"
			}

			s += strings.ReplaceAll(f.Usage, "\n", "\n    \t")

			fmt.Fprint(flag.CommandLine.Output(), s, "\n\n")
		})
	}

	fs.ParseFile(configFile)
	fs.Parse(os.Args[1:])

	cmdLogin()
	cmdPage()
	cmdVersion()

	return nil
}

func cmdLogin() {
	if cmdOptions.Host == "" {
		if cmdOptions.User != "" || cmdOptions.Pass != "" {
			fmt.Println("Error: Specify a host")
			goto Exit
		}

		return
	}

	if userInfo, err := rclone.Login(cmdOptions.Host, cmdOptions.User, cmdOptions.Pass); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	} else {
		AddConfigProperty("userInfo", userInfo)
		return
	}

Exit:
	os.Exit(0)
}

func cmdPage() {
	if cmdOptions.Page == "" {
		return
	}

	for _, page := range []string{
		"Dashboard",
		"Configuration",
		"Explorer",
		"Mounts",
	} {
		if strings.Title(cmdOptions.Page) == page {
			AddConfigProperty("page", page)
			return
		}
	}

	fmt.Printf("Error: %s: No such page\n", cmdOptions.Page)

	os.Exit(0)
}

func cmdVersion() {
	if !cmdOptions.Version {
		return
	}

	fmt.Printf("rclone-tui %s\n", appVersion)

	os.Exit(0)
}
