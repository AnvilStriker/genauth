package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/invopop/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type appContext struct {
	cliContext *cli.Context

	// Values from CLI context
	locators       map[string]string
	appsFilePath   string
	usageFilePath  string
	outputFilePath string

	// Values from loaded YAML files
	apps Apps
	ru   *ResourceUsage

	// Derived Values
	ksaNames      map[AppName]KSAName
	gsaNames      map[AppName]GSAName
	rsrcFullNames RsrcFullNameMap

	rpm ResourcePolicyMap
}

func NewAppContext(c *cli.Context) (*appContext, error) {
	provider := c.String(ProviderFlag.Name)
	region := c.String(RegionFlag.Name)
	if !supportedRegions[provider][region] {
		return nil, fmt.Errorf(`region %v not supported for provider %v`, region, provider)
	}
	locators := map[string]string{
		"company":  "yoyodyne",
		"provider": provider,
		"region":   region,
		"stage":    c.String(StageFlag.Name),
	}
	lvs := c.StringSlice(LocatorFlag.Name)
	for _, s := range lvs {
		kv := strings.SplitN(s, "=", 2)
		locators[kv[0]] = kv[1]
	}
	log.WithField("locators", locators).Debug("locator info")
	ac := &appContext{
		cliContext:     c,
		appsFilePath:   c.Path(AppsFileFlag.Name),
		usageFilePath:  c.Path(UsageFileFlag.Name),
		outputFilePath: c.Path(OutputFileFlag.Name),
		locators:       locators,
		ksaNames:       make(map[AppName]KSAName),
		gsaNames:       make(map[AppName]GSAName),
		rsrcFullNames:  newRsrcFullNameMap(),
		rpm:            make(ResourcePolicyMap),
	}
	return ac, nil
}

func (ac *appContext) loadAppsFile() error {
	y, err := os.ReadFile(ac.appsFilePath)
	if err == nil {
		err = yaml.Unmarshal(y, &ac.apps)
	}
	return errors.WithMessage(err, "loadAppsFile")
}

func (ac *appContext) loadResourceUsageFile() error {
	y, err := os.ReadFile(ac.usageFilePath)
	if err == nil {
		err = yaml.Unmarshal(y, &ac.ru)
	}
	return errors.WithMessage(err, "loadResourceUsageFile")
}

func (ac *appContext) load() error {
	if err := ac.loadAppsFile(); err != nil {
		return err
	}
	log.WithField("ac.apps", ac.apps).Debug("loaded apps file")

	if err := ac.loadResourceUsageFile(); err != nil {
		return err
	}
	log.WithField("ac.ru", ac.ru).Debug("loaded resource usage file")

	return nil
}
