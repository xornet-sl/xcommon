package xcommon

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// ViperConfig describes how config files will be searched and loaded
type ViperConfig struct {
	SearchDirs           []string // Dirs for automatic search, '.' always included implicitly
	SearchFiles          []string // Filenames (without paths) for automatic search (all found files will be merged)
	SearchAtLeastOneFile bool     // If true and no files are found in automatic mode, it will fail
	ConcreeteFilePaths   []string // Concreete paths with configs. Auto searching is not working here. Only first config in this list must exist, others are optional
	ExtractSubtree       string   // Extract a subtree from parsed config
}

func parseConfigFiles(viperConfig *ViperConfig) (*viper.Viper, []string, error) {
	var configsLoaded []string // list of parsed config filepaths

	if len(viperConfig.ConcreeteFilePaths) > 0 {
		// Manual mode
		for _, cfgPath := range viperConfig.ConcreeteFilePaths {
			_, cfgPathErr := os.Stat(cfgPath)
			if errors.Is(cfgPathErr, fs.ErrNotExist) && len(configsLoaded) == 0 {
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
		// Auto mode
		viper.AddConfigPath(".")
		for _, searchDir := range viperConfig.SearchDirs {
			viper.AddConfigPath(searchDir)
		}
		for _, fileName := range viperConfig.SearchFiles {
			ext := filepath.Ext(fileName)
			viper.SetConfigType(strings.TrimLeft(ext, "."))
			viper.SetConfigName(strings.TrimSuffix(fileName, ext))
			if len(configsLoaded) == 0 {
				if err := viper.ReadInConfig(); err != nil {
					continue // In auto mode first file is not mandatory
				}
			} else {
				if err := viper.MergeInConfig(); err != nil {
					continue // In auto mode all files are not mandatory
				}
			}
			configsLoaded = append(configsLoaded, fileName)
		}
		if len(configsLoaded) == 0 && viperConfig.SearchAtLeastOneFile {
			return nil, configsLoaded, fmt.Errorf("no configuration files were found")
		}
	}

	outViper := viper.GetViper()
	if viperConfig.ExtractSubtree != "" {
		outViper = viper.Sub(viperConfig.ExtractSubtree)
		if outViper == nil {
			outViper = viper.New()
		}
	}
	return outViper, configsLoaded, nil
}

// // ViperConfig describes how config files will be searched and loaded
// type ViperConfig struct {
// 	SearchDirs           []string // Dirs for automatic search, '.' always included implicitly
// 	SearchFiles          []string // Filenames (without paths) for automatic search (all found files will be merged)
// 	SearchAtLeastOneFile bool     // If true and no files are found in automatic mode, it will fail
// 	ConcreeteFilePaths   []string // Concreete paths with configs. Auto searching is not working here. Only first config in this list must exist, others are optional
// 	ExtractSubtree       string   // Extract a subtree from parsed config
// }
//
// // parseConfigFiles parses one or more config files. Returns Viper instance with parsed configs, list of parsed config filepaths or error
// // If ExtractSubtree is specified then result will be a subtree or empty Viper instance (never nil unless err != nil)
// // If there are several configs found, this function will try to merge them in a single config
// func parseConfigFiles(config *ViperConfig) (*viper.Viper, []string, error) {
// 	var configsLoaded []string // list of parsed config filepaths
//
// 	if len(config.ConcreeteFilePaths) > 0 {
// 		// Manual mode
// 		for _, cfgPath := range config.ConcreeteFilePaths {
// 			_, cfgPathErr := os.Stat(cfgPath)
// 			if errors.Is(cfgPathErr, fs.ErrNotExist) && len(configsLoaded) == 0 {
// 				return nil, configsLoaded, fmt.Errorf("specified configuration file '%s' doesn't exist", cfgPath)
// 			}
// 			viper.SetConfigFile(cfgPath)
// 			if len(configsLoaded) == 0 {
// 				if err := viper.ReadInConfig(); err != nil {
// 					return nil, configsLoaded, err
// 				}
// 			} else {
// 				if err := viper.MergeInConfig(); err != nil {
// 					return nil, configsLoaded, err
// 				}
// 			}
// 			configsLoaded = append(configsLoaded, cfgPath)
// 		}
// 	} else {
// 		// Auto mode
// 		viper.AddConfigPath(".")
// 		for _, searchDir := range config.SearchDirs {
// 			viper.AddConfigPath(searchDir)
// 		}
// 		for _, fileName := range config.SearchFiles {
// 			ext := filepath.Ext(fileName)
// 			viper.SetConfigType(strings.TrimLeft(ext, "."))
// 			viper.SetConfigName(strings.TrimSuffix(fileName, ext))
// 			if len(configsLoaded) == 0 {
// 				if err := viper.ReadInConfig(); err != nil {
// 					continue // In auto mode first file is not mandatory
// 				}
// 			} else {
// 				if err := viper.MergeInConfig(); err != nil {
// 					continue // In auto mode all files are not mandatory
// 				}
// 			}
// 			configsLoaded = append(configsLoaded, fileName)
// 		}
// 		if len(configsLoaded) == 0 && config.SearchAtLeastOneFile {
// 			return nil, configsLoaded, fmt.Errorf("No configuration files were found")
// 		}
// 	}
//
// 	outViper := viper.GetViper()
// 	if config.ExtractSubtree != "" {
// 		outViper = viper.Sub(config.ExtractSubtree)
// 		if outViper == nil {
// 			outViper = viper.New()
// 		}
// 	}
// 	return outViper, configsLoaded, nil
// }
