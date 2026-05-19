package cmd

import "github.com/spf13/viper"

func getLogLevel() string {
	return viper.GetString("log_level")
}

func getServerPort() int {
	return viper.GetInt("server.port")
}

func setDefaults() {
	viper.SetDefault("log_level", "info")
	viper.SetDefault("server.port", 8090)
}
