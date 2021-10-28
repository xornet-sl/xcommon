package xcommon

import (
	flag "github.com/spf13/pflag"
)

// ConfigurePlan struct is a plan to how to configure your application
// It envolves commandline parsing, config parsing, binding commandline and config parameters together
// And it contains all your defaults for Configure function to work
type ConfigurePlan struct {
	CommandlineMap CmdlineMap // A map with cmdline parsing rules
	// BeforeConfigParsingHook func(parsingRules *ViperConfig) // Hook that is run right before config parsing phase. Can alter parsing rules (ex. specify concrete filepaths)
	ConfigParsingRules    ViperConfig            // Parsing rules for configs
	DefaultConfig         map[string]interface{} // Default configuration that is loaded on empty places in config files
	DontBindFlagsToConfig bool                   // Do not bind pflags to Viper config
	ConfigOverrideFlags   []string               // Flags that should override normal configuration file searching. Such as "--config" (without dashes). These file will be used for config reading
}

//ConfigurationResult result of Configure function
type ConfigurationResult struct {
	ParsedConfigs []string // A list of loaded config files
}

// Configure is the main configuration function in your application
// It takes a config plan and parses everything into your cfgStruct structure that you can use in runtime
func Configure(configPlan *ConfigurePlan, cfgStruct interface{}) (*ConfigurationResult, error) {
	subcommandName, subcommandHandler, err := parseCmdLine(configPlan.CommandlineMap)
	if err != nil {
		return nil, err // Mostly UnknownCommandlineCommandError I hope
	}

	if len(configPlan.ConfigOverrideFlags) > 0 {
		concreeteFiles := []string{}
		for _, checkFlag := range configPlan.ConfigOverrideFlags {
			if cfgFlag := flag.Lookup(checkFlag); cfgFlag != nil && cfgFlag.Changed {
				concreeteFiles = append(concreeteFiles, cfgFlag.Value.String())
			}
		}
		if len(concreeteFiles) > 0 {
			configPlan.ConfigParsingRules.ConcreeteFilePaths = concreeteFiles
		}
	}

	// if configPlan.BeforeConfigParsingHook != nil {
	// 	configPlan.BeforeConfigParsingHook(&configPlan.ConfigParsingRules)
	// }

	vp, parsedConfigs, err := parseConfigFiles(&configPlan.ConfigParsingRules)
	if err != nil {
		return nil, err
	}

	if !configPlan.DontBindFlagsToConfig {
		flag.Visit(func(f *flag.Flag) {
			_ = vp.BindPFlag(f.Name, f)
		})
	}

	// Set default config
	for key, value := range configPlan.DefaultConfig {
		vp.SetDefault(key, value)
	}

	if err := vp.Unmarshal(&cfgStruct); err != nil {
		return nil, err
	}

	State.subcommandName.Store(subcommandName)
	State.subcommandHandler.Store(subcommandHandler)
	return &ConfigurationResult{
		ParsedConfigs: parsedConfigs,
	}, nil
}
