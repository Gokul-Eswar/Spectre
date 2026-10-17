package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/Gokul-Eswar/Spectre/internal/ethics"
	"github.com/spf13/viper"
)

// InitConfig reads config using the full initialization path.
// Kept for backward compatibility with existing call sites.
func InitConfig(cfgFile string) {
	_ = InitConfigFull(cfgFile)
}

// InitConfigLight performs low-overhead config initialization.
// It only binds environment variables and optionally reads an explicit config file.
func InitConfigLight(cfgFile string) error {
	viper.AutomaticEnv()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config file %q: %w", cfgFile, err)
		}
	}

	return nil
}

// InitConfigFull performs full config initialization including default
// config-file discovery and ethics policy hydration.
func InitConfigFull(cfgFile string) error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to resolve user home directory: %w", err)
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.AddConfigPath("configs")
		viper.SetConfigType("yaml")
		viper.SetConfigName("default")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	ApplyEthicsConfig()
	return nil
}

// ApplyEthicsConfig loads settings from viper into the ethics package.
func ApplyEthicsConfig() {
	// Apply Blacklist
	bl := viper.GetStringSlice("ethics.blacklist")
	if len(bl) > 0 {
		ethics.SetBlacklist(bl)
	}

	// Apply Whitelist
	wl := viper.GetStringSlice("ethics.whitelist")
	if len(wl) > 0 {
		ethics.SetWhitelist(wl)
	}

	// Apply Rate Limits
	// We check for collectors.<name>.rate_limit
	collectors := []string{"dns", "whois", "github", "geo", "ports"}
	for _, name := range collectors {
		key := fmt.Sprintf("collectors.%s.rate_limit", name)
		if viper.IsSet(key) {
			limit := viper.GetFloat64(key)
			ethics.SetLimit(name, limit)
		}
	}
}

// GetAPIKey retrieves an API key from configuration or environment.
func GetAPIKey(name string) string {
	// Checks keys.<name> in config or SPECTRE_KEYS_<NAME> in env
	return viper.GetString("keys." + name)
}

// SaveConfig writes the current Viper configuration to disk safely.
func SaveConfig() error {
	err := viper.WriteConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return viper.SafeWriteConfig()
		}
		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			configFile = "configs/default.yaml"
			return viper.WriteConfigAs(configFile)
		}
		return err
	}
	return nil
}
