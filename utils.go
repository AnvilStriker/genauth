package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func LogCLIFlagSummary(c *cli.Context, flags []cli.Flag) {
	flagMap := map[string]interface{}{}

	for _, f := range flags {
		name := f.Names()[0]
		switch f.(type) {
		case *cli.BoolFlag:
			flagMap[name] = c.Bool(name)
		case *cli.PathFlag:
			flagMap[name] = c.Path(name)
		case *cli.StringFlag:
			flagMap[name] = c.String(name)
		case *cli.StringSliceFlag:
			flagMap[name] = c.StringSlice(name)
		}
	}

	log.WithField("flags", flagMap).Debug("CLI flag summary")
}

type logrusLogLevel struct {
	level log.Level
}

func (l *logrusLogLevel) Set(value string) error {
	level, err := log.ParseLevel(value)
	if err != nil {
		return err
	}
	l.level = level
	log.SetLevel(level)
	return nil
}

func (l *logrusLogLevel) String() string {
	return l.level.String()
}

var LogLevelFlag = cli.GenericFlag{
	Name:    "log-level",
	Usage:   "One of: panic, fatal, error, warn, info, debug, or trace",
	EnvVars: []string{"LOG_LEVEL"},
	// Default to INFO (which is also the default logrus logging level)
	Value: &logrusLogLevel{level: log.InfoLevel},
}
