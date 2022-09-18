package main

import (
	"github.com/hidenari-yuda/go-trade/app/controller"
)

func main() {
	controller.StreamIngestionData()
	controller.StartWebServer()

	// apiClient := bitflyer.New(config.Config.ApiKey, config.Config.ApiSecret)
	// ticker, _ := apiClient.GetTicker("BTC_JPY")
	// fmt.Println(ticker)
	// fmt.Println(ticker.GetMidPrice())

	// order := &bitflyer.Order{
	// 	ProductCode:     "BTC_JPY",
	// 	ChildOrderType:  "MAKRKET",
	// 	Side:            "BUY",
	// 	Size:            0.0001,
	// 	MinuteToExpires: 1,
	// 	TimeInForce:     "GTC",
	// }
	// res, _ := apiClient.SendOrder(order)
	// fmt.Println(res.ChildOrderAcceptanceID)
}
