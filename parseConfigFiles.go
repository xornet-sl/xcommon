package xcommon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// ViperConfig struct describes how config files will be searched and loaded
type ViperConfig struct {
	SearchDirs           []string // Dirs for automatic search, '.' always included
	SearchFiles          []string // Filenames (without paths) for automatic search (all found files will be merged)
	SearchAtLeastOneFile bool     // If true will fail if nothing was found in automatic mode
	ConcreeteFilePaths   []string // Concreete paths with configs. Auto searching is not working here. Only first config must exists
	ExtractSubtree       string   // Extract a subtree from parsed config
}

// parseConfigFiles parses one or more config files. Returns viper instance with parsed configs, list of parsed config or error
// If ExtractSubtree is specified then result will be a subtree or empty viper (never nil)
// If there are several configs found this function will try to merge them in single config
func parseConfigFiles(config *ViperConfig) (*viper.Viper, []string, error) {
	var configsLoaded []string

	if len(config.ConcreeteFilePaths) > 0 {
		for _, cfgPath := range config.ConcreeteFilePaths {
			_, cfgPathErr := os.Stat(cfgPath)
			if os.IsNotExist(cfgPathErr) && len(configsLoaded) == 0 {
				return nil, configsLoaded, fmt.Errorf("specified configuration file '%s' doesn't exist", cfgPath)
			}
			viper.SetConfigFile(cfgPath)
			if len(configsLoaded) == 0 {
				if err := viper.ReadInConfig(); err != nil {
					return nil, configsLoaded, err
				}
			} else {
				if err := viper.MergeInConfig(); err != nil {
					return nil, configsLoaded, err
				}
			}
			configsLoaded = append(configsLoaded, cfgPath)
		}
	} else {
		viper.AddConfigPath(".")
		for _, searchDir := range config.SearchDirs {
			viper.AddConfigPath(searchDir)
		}
		for _, fileName := range config.SearchFiles {
			ext := filepath.Ext(fileName)
			viper.SetConfigType(strings.TrimLeft(ext, "."))
			viper.SetConfigName(strings.TrimSuffix(fileName, ext))
			if len(configsLoaded) == 0 {
				if err := viper.ReadInConfig(); err != nil {
					continue // In auto mode first file is not mandatory
				}
			} else {
				if err := viper.MergeInConfig(); err != nil {
					continue // In automode all files are not mandatory
				}
			}
			configsLoaded = append(configsLoaded, fileName)
		}
		if len(configsLoaded) == 0 && config.SearchAtLeastOneFile {
			return nil, configsLoaded, fmt.Errorf("No configuration files found")
		}
	}

	outViper := viper.GetViper()
	if config.ExtractSubtree != "" {
		outViper = viper.Sub(config.ExtractSubtree)
		if outViper == nil {
			outViper = viper.New()
		}
	}
	return outViper, configsLoaded, nil
}
