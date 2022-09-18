package controller

import (
	"github.com/hidenari-yuda/go-trade/app/models"
	"github.com/hidenari-yuda/go-trade/bitflyer"
	"github.com/hidenari-yuda/go-trade/config"
)

func StreamIngestionData() {
	var tickerChannel = make(chan bitflyer.Ticker)

	apiClient := bitflyer.New(config.Config.ApiKey, config.Config.ApiSecret)
	go apiClient.GetRealTimeTicker(config.Config.ProductCode, tickerChannel)

	go func() {
		for ticker := range tickerChannel {
			for _, duration := range config.Config.Durations {
				isCreated := models.CreateCandleWithDuration(ticker, ticker.ProductCode, duration)
				if isCreated == true && duration == config.Config.TradeDuration {
					// TODO:
				}
			}
		}
	}()
}
