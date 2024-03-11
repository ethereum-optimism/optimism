package config

import (
	"encoding/json"
	"io"
	"os"
)

type Config struct {
	Seed   string `json:"seed"`
	ApiURL string `json:"api_url"`
	AppID  int    `json:"app_id"`
}

func (c *Config) GetConfig() error {
	jsonFile, err := os.Open("./avail/config/config.json")
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(byteValue, c)
	if err != nil {
		return err
	}

	return nil
}
