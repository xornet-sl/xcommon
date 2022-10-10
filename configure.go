package xcommon

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ConfigurePlan is a plan to how to configure your application
// It envolves config parsing, binding commandline, environment and config parameters together
// And it contains all your defaults for Configure function to work
type ConfigurePlan struct {
	ConfigParsingRules    ViperConfig // Parsing rules for configs
	DontBindFlagsToConfig bool        // Do not bind pflags to Viper config
	DontBindEnvToConfig   bool        // Do not bind environment variables to Viper config
	EnvVariablesPrefix    string      // Look up only prefixed ENV variables
	ConfigOverrideFlag    string      // Flag that should override normal configuration file searching. Such as "--config" (without dashes). These files will be used for config reading
}

// ConfigurationResult stores a result of Configure function
type ConfigurationResult struct {
	ConfigurePlan *ConfigurePlan // Used configuration plan
	ParsedConfigs []string       // A list of loaded config filepaths
	RootCmd       *cobra.Command // A pointer to root cobra command structure
	Viper         *viper.Viper   // Viper instance
}

type CobraBuilderFunc func() (*cobra.Command, map[string]*pflag.Flag, error)

func InitCobra(
	cobraBuilder CobraBuilderFunc,
	configPlan *ConfigurePlan,
	defaultConfig map[string]interface{},
	cfgStruct interface{},
	configurationResult **ConfigurationResult,
	initializers ...func(),
) (*cobra.Command, error) {
	rootCmd, bindFlags, err := cobraBuilder()
	if err != nil {
		return nil, err
	}

	if configPlan.ConfigOverrideFlag != "" {
		rootCmd.PersistentFlags().StringArray(configPlan.ConfigOverrideFlag, nil, "override configuration files")
	}

	initializer := func() {
		result, err := configure(rootCmd, configPlan, defaultConfig, cfgStruct)
		if err != nil {
			panic(err)
		}
		*configurationResult = result
		if bindFlags != nil && !configPlan.DontBindFlagsToConfig {
			for key, flag := range bindFlags {
				if err := result.Viper.BindPFlag(key, flag); err != nil {
					log.WithError(err).Debugf("Unable to bind pflag %s to config", key)
				}
			}
		}

		// Save configuration in struct
		if err := result.Viper.Unmarshal(&cfgStruct); err != nil {
			panic(err)
		}
	}
	cobra.OnInitialize(initializer)
	cobra.OnInitialize(initializers...)
	return rootCmd, nil
}

// configure is trying to be the main configuration function in application
// It takes a config plan and parses everything into your cfgStruct structure that application could use in runtime
// WARNING: this function should be called in cobra initializer
func configure(
	rootCmd *cobra.Command,
	configPlan *ConfigurePlan,
	defaultConfig map[string]interface{},
	cfgStruct interface{},
) (*ConfigurationResult, error) {
	if configPlan.ConfigOverrideFlag != "" {
		concreeteFiles := []string{}
		if flagFiles, err := rootCmd.Flags().GetStringArray(configPlan.ConfigOverrideFlag); err == nil {
			concreeteFiles = append(concreeteFiles, flagFiles...)
		}
		if len(concreeteFiles) > 0 {
			configPlan.ConfigParsingRules.ConcreeteFilePaths = concreeteFiles
		}
	}

	vp, parsedConfigs, err := parseConfigFiles(&configPlan.ConfigParsingRules)
	if err != nil {
		return nil, err
	}

	// Bind cmdline flags to config
	// if !configPlan.DontBindFlagsToConfig {
	// 	rootCmd.Flags().Visit(func(f *pflag.Flag) {
	// 		_ = vp.BindPFlag(f.Name, f)
	// 	})
	// }

	// Bind environment variables to config
	if !configPlan.DontBindEnvToConfig {
		vp.SetEnvPrefix(configPlan.EnvVariablesPrefix)
		vp.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		vp.AutomaticEnv()
	}

	// Set default config
	for key, value := range defaultConfig {
		vp.SetDefault(key, value)
	}

	return &ConfigurationResult{
		ConfigurePlan: configPlan,
		ParsedConfigs: parsedConfigs,
		RootCmd:       rootCmd,
		Viper:         vp,
	}, nil
}

// // ConfigurePlan is a plan to how to configure your application
// // It envolves commandline parsing, config parsing, binding commandline, environment and config parameters together
// // And it contains all your defaults for Configure function to work
// type ConfigurePlan struct {
// 	CommandlineMap        CmdlineMap  // A map with cmdline parsing rules
// 	ConfigParsingRules    ViperConfig // Parsing rules for configs
// 	DontBindFlagsToConfig bool        // Do not bind pflags to Viper config
// 	DontBindEnvToConfig   bool        // Do not bind environment variables to Viper config
// 	EnvVariablesPrefix    string      // Look up only prefixed ENV variables
// 	ConfigOverrideFlags   []string    // Flags that should override normal configuration file searching. Such as "--config" (without dashes). These files will be used for config reading
// 	// DefaultConfig         map[string]interface{} // Default configuration that is loaded on empty places in config files
// }
//
// // ConfigurationResult stores a result of Configure function
// type ConfigurationResult struct {
// 	ConfigurePlan     *ConfigurePlan // Used configuration plan
// 	ParsedConfigs     []string       // A list of loaded config filepaths
// 	SubcommandName    string
// 	SubcommandHandler CommandHandler
// }
//
// // Configure is trying to be the main configuration function in application
// // It takes a config plan and parses everything into your cfgStruct structure that application could use in runtime
// func Configure(configPlan *ConfigurePlan, defaultConfig map[string]interface{}, cfgStruct interface{}) (*ConfigurationResult, error) {
// 	subcommandName, subcommandHandler, err := parseCmdLine(&configPlan.CommandlineMap)
// 	if err != nil {
// 		return nil, err // Mostly UnknownCommandlineCommandError I hope
// 	}
//
// 	// If there are config flag overrides in commandline, override ConcreeteFilePaths then
// 	if len(configPlan.ConfigOverrideFlags) > 0 {
// 		concreeteFiles := []string{}
// 		for _, checkFlag := range configPlan.ConfigOverrideFlags {
// 			if cfgFlag := pflag.Lookup(checkFlag); cfgFlag != nil && cfgFlag.Changed {
// 				concreeteFiles = append(concreeteFiles, cfgFlag.Value.String())
// 			}
// 		}
// 		if len(concreeteFiles) > 0 {
// 			configPlan.ConfigParsingRules.ConcreeteFilePaths = concreeteFiles
// 		}
// 	}
//
// 	// Parse configuration files
// 	vp, parsedConfigs, err := parseConfigFiles(&configPlan.ConfigParsingRules)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	// Bind cmdline flags to config
// 	if !configPlan.DontBindFlagsToConfig {
// 		pflag.Visit(func(f *pflag.Flag) {
// 			_ = vp.BindPFlag(f.Name, f)
// 		})
// 	}
//
// 	// Bind environment variables to config
// 	if !configPlan.DontBindEnvToConfig {
// 		viper.SetEnvPrefix(configPlan.EnvVariablesPrefix)
// 		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
// 		viper.AutomaticEnv()
// 	}
//
// 	// Set default config
// 	for key, value := range defaultConfig {
// 		vp.SetDefault(key, value)
// 	}
//
// 	if err := vp.Unmarshal(&cfgStruct); err != nil {
// 		return nil, err
// 	}
//
// 	return &ConfigurationResult{
// 		ConfigurePlan:     configPlan,
// 		ParsedConfigs:     parsedConfigs,
// 		SubcommandName:    subcommandName,
// 		SubcommandHandler: subcommandHandler,
// 	}, nil
// }
