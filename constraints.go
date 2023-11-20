package main

var (
	supportedLevels = map[string]bool{
		"dev":  true,
		"stg":  true,
		"prod": true,
	}

	supportedProviders = map[string]bool{
		"gcp": true,
	}

	supportedRegions = map[string]map[string]bool{
		"gcp": {
			"usce1": true,
		},
	}
)
