package controller

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"

	"github.com/hidenari-yuda/go-trade/app/models"
	"github.com/hidenari-yuda/go-trade/config"
)

var templates = template.Must(template.ParseFiles("app/views/chart.html"))

func viewChartHandler(w http.ResponseWriter, r *http.Request) {
	// limit := 100
	// duration := "1m"
	// durationTime := config.Config.Durations[duration]
	// product_code := "BTC_JPY"

	// df, err := models.GetAllCandle(product_code, durationTime, limit)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	err := templates.ExecuteTemplate(w, "chart.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type JSONError struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

func APIError(w http.ResponseWriter, errMessage string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	JSONError, err := json.Marshal(JSONError{Error: errMessage, Code: code})
	if err != nil {
		fmt.Println(err)
	}
	w.Write(JSONError)
}

var apiValidPath = regexp.MustCompile("^/api/candle/$")

func apiMakeHandler(fn func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := apiValidPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			APIError(w, "Not Found", http.StatusNotFound)
			return
		}
		fn(w, r)
	}
}

func apiCandleHandler(w http.ResponseWriter, r *http.Request) {
	productCode := r.URL.Query().Get("product_code")
	if productCode == "" {
		APIError(w, "product_code is required", http.StatusBadRequest)
		return
	}
	strLimit := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(strLimit)
	if strLimit == "" || err != nil || limit < 0 || limit > 1000 {
		APIError(w, "limit is required", http.StatusBadRequest)
		return
	}

	duration := r.URL.Query().Get("duration")
	if duration == "" {
		duration = "1m"
	}
	durationTime := config.Config.Durations[duration]

	df, err := models.GetAllCandle(productCode, durationTime, limit)
	if err != nil {
		fmt.Println(err)
	}

	sma := r.URL.Query().Get("sma")
	if sma != "" {
		strSmaPeriod1 := r.URL.Query().Get("sma_period1")
		strSmaPeriod2 := r.URL.Query().Get("sma_period2")
		strSmaPeriod3 := r.URL.Query().Get("sma_period3")
		period1, err := strconv.Atoi(strSmaPeriod1)
		if strSmaPeriod1 == "" || err != nil || period1 < 0 {
			period1 = 7
		}
		period2, err := strconv.Atoi(strSmaPeriod2)
		if strSmaPeriod2 == "" || err != nil || period2 < 0 {
			period2 = 14
		}
		period3, err := strconv.Atoi(strSmaPeriod3)
		if strSmaPeriod3 == "" || err != nil || period3 < 0 {
			period3 = 50
		}
		df.AddSma(period1)
		df.AddSma(period2)
		df.AddSma(period3)

	}

	js, err := json.Marshal(df)
	if err != nil {
		APIError(w, "limit is required", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

}

func StartWebServer() error {
	http.HandleFunc("/api/candle/", apiMakeHandler(apiCandleHandler))
	http.HandleFunc("/chart/", viewChartHandler)
	return http.ListenAndServe(fmt.Sprintf(":%d", config.Config.Port), nil)
}
