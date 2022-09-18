package bitflyer

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const baseURL = "https://api.bitflyer.com/v1/"

type APIClient struct {
	key        string
	secret     string
	httpClient *http.Client
}

//APIキーを送信する
func New(key, secret string) *APIClient {
	APIClient := &APIClient{key, secret, &http.Client{}}
	return APIClient
}

//httpのヘッダーを作成する
func (api *APIClient) header(method, endpoint string, body []byte) map[string]string {
	timeStamp := strconv.FormatInt(time.Now().Unix(), 10)

	message := timeStamp + method + endpoint + string(body)

	mac := hmac.New(sha256.New, []byte(api.secret))
	mac.Write([]byte(message))

	sign := hex.EncodeToString(mac.Sum(nil))

	return map[string]string{
		"ACCESS-KEY":       api.key,
		"ACCESS-TIMESTAMP": timeStamp,
		"ACCESS-SIGN":      sign,
		"Content-Type":     "application/json",
	}
}

//httpにリクエストを送る
func (api *APIClient) doRequest(method, urlPath string, query map[string]string, data []byte) (body []byte, err error) {
	baseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	apiURL, err := baseURL.Parse(urlPath)
	if err != nil {
		return nil, err
	}

	endpoint := baseURL.ResolveReference(apiURL).String()
	log.Printf("action=doRequest, endpoint=%s", endpoint)

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	for key, value := range query {
		q.Add(key, value)
	}

	req.URL.RawQuery = q.Encode()

	for key, value := range api.header(method, req.URL.RequestURI(), data) {
		req.Header.Add(key, value)
	}
	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

type Balance struct {
	CurrencyCode string  `json:"currency_code"`
	Amount       float64 `json:"amount"`
	Available    float64 `json:"available"`
}

//利用情報を取得する
func (api *APIClient) GetBalance() ([]Balance, error) {
	url := "me/getbalance"
	resp, err := api.doRequest("GET", url, map[string]string{}, nil)
	log.Printf("ur;=%s, resp=%s", url, string(resp))
	if err != nil {
		return nil, err
	}

	var balance []Balance
	err = json.Unmarshal(resp, &balance)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

//ティッカーを定義する
type Ticker struct {
	ProductCode     string  `json:"product_code"`
	State           string  `json:"state"`
	Timestamp       string  `json:"timestamp"`
	TickID          float64 `json:"tick_id"`
	BestBid         float64 `json:"best_bid"`
	BestAsk         float64 `json:"best_ask"`
	BestBidSize     float64 `json:"best_bid_size"`
	BestAskSize     float64 `json:"best_ask_size"`
	TotalBidDepth   float64 `json:"total_bid_depth"`
	TotalAskDepth   float64 `json:"total_ask_depth"`
	MarketBidSize   float64 `json:"market_bid_size"`
	MarketAskSize   float64 `json:"market_ask_size"`
	Ltp             float64 `json:"ltp"`
	Volume          float64 `json:"volume"`
	VolumeByProduct float64 `json:"volume_by_product"`
}

//中央値を取得する
func (t *Ticker) GetMidPrice() float64 {
	return (t.BestBid + t.BestAsk) / 2
}

//日付をSQL指定の型に変換する
func (t *Ticker) DateTime() time.Time {
	datetime, _ := time.Parse("time.RFC3339", t.Timestamp)
	return datetime
}

//設定した間隔ごとに日付を切り捨てる
func (t *Ticker) TruncateDateTime(duration time.Duration) time.Time {
	return t.DateTime().Truncate(duration)
}

//ティッカーを取得する
func (api *APIClient) GetTicker(product_code string) (*Ticker, error) {
	url := "ticker"
	resp, err := api.doRequest("GET", url, map[string]string{"product_code": product_code}, nil)
	log.Printf("ur;=%s, resp=%s", url, string(resp))
	if err != nil {
		return nil, err
	}

	var ticker Ticker
	err = json.Unmarshal(resp, &ticker)
	if err != nil {
		return nil, err
	}
	return &ticker, nil
}

// リアルタイムのティッカーを取得する

type JsonRPC2 struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Result  interface{} `json:"result,omitempty"`
	Id      *int        `json:"id,omitempty"`
}

type SubscribeParams struct {
	Channel string `json:"channel"`
}

func (api *APIClient) GetRealTimeTicker(symbol string, ch chan<- Ticker) {
	u := url.URL{Scheme: "wss", Host: "ws.lightstream.bitflyer.com", Path: "/json-rpc"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	channel := fmt.Sprintf("lightning_ticker_%s", symbol)
	if err := c.WriteJSON(&JsonRPC2{Version: "2.0", Method: "subscribe", Params: &SubscribeParams{channel}}); err != nil {
		log.Fatal("subscribe:", err)
		return
	}

OUTER:
	for {
		message := new(JsonRPC2)
		if err := c.ReadJSON(message); err != nil {
			log.Println("read:", err)
			return
		}

		if message.Method == "channelMessage" {
			switch v := message.Params.(type) {
			case map[string]interface{}:
				for key, binary := range v {
					if key == "message" {
						marshaTic, err := json.Marshal(binary)
						if err != nil {
							continue OUTER
						}
						var ticker Ticker
						if err := json.Unmarshal(marshaTic, &ticker); err != nil {
							continue OUTER
						}
						ch <- ticker
					}
				}
			}
		}
	}
}

// 注文する
type Order struct {
	ID                     int     `json:"id"`
	ChildOrderAcceptanceID string  `json:"child_order_acceptance_id"`
	ProductCode            string  `json:"product_code"`
	ChildOrderType         string  `json:"child_order_type"`
	Side                   string  `json:"side"`
	Price                  float64 `json:"price"`
	Size                   float64 `json:"size"`
	MinuteToExpires        int     `json:"minute_to_expire"`
	TimeInForce            string  `json:"time_in_force"`
	Status                 string  `json:"status"`
	ErrorMessage           string  `json:"error_message"`
	AveragePrice           float64 `json:"average_price"`
	ChildOrderState        string  `json:"child_order_state"`
	ExpireDate             string  `json:"expire_date"`
	ChildOrderDate         string  `json:"child_order_date"`
	OutstandingSize        float64 `json:"outstanding_size"`
	CancelSize             float64 `json:"cancel_size"`
	ExecutedSize           float64 `json:"executed_size"`
	TotalCommission        float64 `json:"total_commission"`
	Count                  int     `json:"count"`
	Before                 int     `json:"before"`
	After                  int     `json:"after"`
}

type ResponseSendChildOrder struct {
	ChildOrderAcceptanceID string `json:"child_order_acceptance_id"`
}

func (api *APIClient) SendOrder(order *Order) (*ResponseSendChildOrder, error) {
	data, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}

	url := "me/sendchildorder"
	resp, err := api.doRequest("POST", url, map[string]string{}, data)
	if err != nil {
		return nil, err
	}

	var response ResponseSendChildOrder
	err = json.Unmarshal(resp, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (api *APIClient) ListOrder(query map[string]string) ([]Order, error) {
	resp, err := api.doRequest("GET", "me/getchildorders", query, nil)
	if err != nil {
		return nil, err
	}

	var responseListOrder []Order
	err = json.Unmarshal(resp, &responseListOrder)
	if err != nil {
		return nil, err
	}

	return responseListOrder, nil
}
