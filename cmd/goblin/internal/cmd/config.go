package cmd

import "github.com/spf13/viper"

// ProviderConfig holds the upstream provider connection settings.
type ProviderConfig struct {
	BaseURL string `mapstructure:"base_url"`
	EnvKey  string `mapstructure:"env_key"`
}

// ModelConfig holds per-model overrides.
type ModelConfig struct {
	MaxTokens      int     `mapstructure:"max_tokens"`
	Temperature    float64 `mapstructure:"temperature"`
	ResponseFormat string  `mapstructure:"response_format"`
}

func getLogLevel() string {
	return viper.GetString("log_level")
}

func getTitleModel() string {
	return viper.GetString("title_model")
}

func getServerHost() string {
	return viper.GetString("server.host")
}

func getServerPort() int {
	return viper.GetInt("server.port")
}

func getProviderConfig() ProviderConfig { //nolint:unused
	var cfg ProviderConfig
	if err := viper.UnmarshalKey("provider", &cfg); err != nil {
		return ProviderConfig{}
	}
	return cfg
}

func getModelConfig(modelID string) ModelConfig { //nolint:unused
	var cfg ModelConfig
	if err := viper.UnmarshalKey("models."+modelID, &cfg); err != nil {
		return ModelConfig{}
	}
	return cfg
}

func getAllModelConfigs() map[string]ModelConfig {
	var models map[string]ModelConfig
	if err := viper.UnmarshalKey("models", &models); err != nil {
		return nil
	}
	return models
}

func setDefaults() {
	viper.SetDefault("log_level", "info")
	viper.SetDefault("server.host", "127.0.0.1")
	viper.SetDefault("server.port", 8090)
}
