package helpers

import (
	"encoding/json"
	"os"
)

type Config struct {
	ApiEndpoint				string 		`json:"api_endpoint"`
	AdminUser					string 		`json:"cf_admin_user"`
	AdminPassword			string 		`json:"cf_admin_password"`
	User							string 		`json:"cf_user"`
	Password					string 		`json:"cf_user_password"`
	Org								string 		`json:"cf_org"`
	Space							string 		`json:"cf_space"`
	AppsDomain        string 		`json:"apps_domain"`
	PersistentAppHost string 		`json:"persistent_app_host"`
	LoginFlags				string 	`json:"login_flags"`
}

var loadedConfig *Config

func LoadConfig() Config {
	if loadedConfig == nil {
		loadedConfig = loadConfigJsonFromPath()
	}

	if loadedConfig.PersistentAppHost == "" {
		loadedConfig.PersistentAppHost = "persistent-app"
	}

	return *loadedConfig
}

func loadConfigJsonFromPath() *Config {
	var config *Config = &Config{}

	path := loadConfigPathFromEnv()

	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(config)
	if err != nil {
		panic(err)
	}

	return config
}

func loadConfigPathFromEnv() string {
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("Must set $CONFIG to point to an integration config .json file.")
	}

	return path
}
