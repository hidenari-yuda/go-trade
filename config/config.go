package config

import (
	"time"

	"gopkg.in/go-ini/ini.v1"
)

type ConfigList struct {
	ApiKey        string
	ApiSecret     string
	LogFile       string
	ProductCode   string
	TradeDuration time.Duration
	Durations     map[string]time.Duration
	DbName        string
	SQLDriver     string
	Port          int
}

var Config ConfigList

func init() {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		println(err)
	}
	durations := map[string]time.Duration{
		"1s": time.Second,
		"1m": time.Minute,
		"1h": time.Hour,
	}

	Config = ConfigList{
		ApiKey:        cfg.Section("bitflyer").Key("api_key").String(),
		ApiSecret:     cfg.Section("bitflyer").Key("api_secret").String(),
		LogFile:       cfg.Section("gotrade").Key("log_file").String(),
		ProductCode:   cfg.Section("gotrade").Key("product_code").String(),
		Durations:     durations,
		TradeDuration: durations[cfg.Section("gotrade").Key("trade_duration").String()],
		DbName:        cfg.Section("db").Key("name").String(),
		SQLDriver:     cfg.Section("db").Key("sql_driver").String(),
		Port:          cfg.Section("web").Key("port").MustInt(),
	}

}
