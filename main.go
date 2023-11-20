package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	StageFlag = cli.StringFlag{
		Name:    "stage",
		Aliases: []string{"s"},
		Usage:   `Deployment stage; one of "dev", "stg" or "prod"`,
		Value:   "dev",
		Action: func(ctx *cli.Context, v string) error {
			if !supportedLevels[v] {
				return fmt.Errorf(`stage %v not supported`, v)
			}
			return nil
		},
	}
	RegionFlag = cli.StringFlag{
		Name:    "region",
		Aliases: []string{"r"},
		Usage:   "Short region code, without provider suffix",
		Value:   "usce1",
	}
	ProviderFlag = cli.StringFlag{
		Name:    "provider",
		Aliases: []string{"p"},
		Usage:   `Cloud provider (only "gcp" supported so far)`,
		Value:   "gcp",
		Action: func(ctx *cli.Context, v string) error {
			if !supportedProviders[v] {
				return fmt.Errorf(`provider %v not supported`, v)
			}
			return nil
		},
	}
	// TODO: Rather than requiring locator values to be specified/overridden from the command
	// line, (other than "provider", "region" and "stage") they should be loaded from a file.
	// And other than those three, and perhaps "company", their keys should not be known to
	// the source code at all.
	LocatorFlag = cli.StringSliceFlag{
		Name:    "locator",
		Aliases: []string{"l"},
		Usage:   "A \"`name=value`\" binding for template substitution",
		Action: func(ctx *cli.Context, v []string) error {
			for _, s := range v {
				parts := strings.SplitN(s, "=", 2)
				if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
					return fmt.Errorf(`locator argument "%v" malformed`, s)
				}
			}
			return nil
		},
	}
	AppsFileFlag = cli.PathFlag{
		Name:    "apps-file",
		Aliases: []string{"a"},
		Usage:   "Path to apps YAML file",
		Value:   "./apps.yaml",
	}
	UsageFileFlag = cli.PathFlag{
		Name:    "usage-file",
		Aliases: []string{"u"},
		Usage:   "Path to resource usage YAML file",
		Value:   "./resource-usage.yaml",
	}
	OutputFileFlag = cli.PathFlag{
		Name:    "output-file",
		Aliases: []string{"o"},
		Usage:   "Path to newline-delimited JSON output file (default: stdout)",
	}

	flags = []cli.Flag{
		&LogLevelFlag, // utils.go
		&StageFlag,
		&RegionFlag,
		&ProviderFlag,
		&LocatorFlag,
		&AppsFileFlag,
		// TODO: probably load permissions from separate file
		&UsageFileFlag,
		// Add a dry-run/validate-only mode?
		&OutputFileFlag,
	}
)

func genResourceAuth(c *cli.Context) error {
	LogCLIFlagSummary(c, flags)
	ac, err := NewAppContext(c)
	if err != nil {
		return err
	}

	// Load input files
	if err := ac.load(); err != nil {
		return err
	}

	// TODO: Validate inputs
	// ... FIXME ...

	// Derive SA names and full resource names
	// TODO: check/enforce length/syntax constraints
	if err := ac.deriveNames(); err != nil {
		return err
	}

	// Construct resource policies based on intended usage
	// TODO: Check/enforce that app/resource references match declarations
	if err := ac.derivePolicies(); err != nil {
		return err
	}

	// For each resource, output its computed policy.
	// NOTE: current output format is mainly for demo purposes.
	// The format that the platform team really needs is TBD.
	keys := make([]string, 0, len(ac.rpm))
	for r := range ac.rpm {
		keys = append(keys, string(r))
	}
	sort.Strings(keys)

	var f *os.File
	if len(ac.outputFilePath) == 0 {
		f = os.Stdout
	} else {
		f, err = os.OpenFile(ac.outputFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.WithError(err).Errorf("%s: error opening file", ac.outputFilePath)
			return err
		}
	}
	writer := bufio.NewWriter(f)

	for _, r := range keys {
		p := ac.rpm[RsrcFullName(r)]
		if len(p.roleBindingsMap) == 0 {
			// This is a bit of a hack. It avoids printing a policy
			// that names a resource but has no bindings; for example,
			// if only one of "publish" or "subscribe" was specified
			// for a queue. It would be better (but harder) to prevent
			// the unpopulated policy from getting into the resource
			// policy map in the first place.
			continue
		}
		b, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			log.WithError(err).Errorf("%s: marshal", r)
		} else {
			_, err := writer.WriteString(string(b) + "\n")
			if err != nil {
				log.WithError(err).Errorf("write error")
			}
		}
	}
	writer.Flush()
	if f != os.Stdout {
		f.Close()
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = os.Args[0]
	app.Usage = "generate auth config for infra resources"
	app.Flags = flags
	app.Action = genResourceAuth
	if err := app.Run(os.Args); err != nil {
		log.WithError(err).Fatal(os.Args[0])
	}
}
