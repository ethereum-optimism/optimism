// Package coinmarketcap Coin Market Cap API client for Go
package coinmarketcap

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Client the CoinMarketCap client
type Client struct {
	proAPIKey      string
	Cryptocurrency *CryptocurrencyService
	Exchange       *ExchangeService
	GlobalMetrics  *GlobalMetricsService
	Tools          *ToolsService
	common         service
}

// Config the client config structure
type Config struct {
	ProAPIKey string
}

// CryptocurrencyService ...
type CryptocurrencyService service

// ExchangeService ...
type ExchangeService service

// GlobalMetricsService ...
type GlobalMetricsService service

// ToolsService ...
type ToolsService service

// Status is the status structure
type Status struct {
	Timestamp    string  `json:"timestamp"`
	ErrorCode    int     `json:"error_code"`
	ErrorMessage *string `json:"error_message"`
	Elapsed      int     `json:"elapsed"`
	CreditCount  int     `json:"credit_count"`
}

// Response is the response structure
type Response struct {
	Status Status      `json:"status"`
	Data   interface{} `json:"data"`
}

// Listing is the listing structure
type Listing struct {
	ID                float64           `json:"id"`
	Name              string            `json:"name"`
	Symbol            string            `json:"symbol"`
	Slug              string            `json:"slug"`
	CirculatingSupply float64           `json:"circulating_supply"`
	TotalSupply       float64           `json:"total_supply"`
	MaxSupply         float64           `json:"max_supply"`
	DateAdded         string            `json:"date_added"`
	NumMarketPairs    float64           `json:"num_market_pairs"`
	CMCRank           float64           `json:"cmc_rank"`
	LastUpdated       string            `json:"last_updated"`
	Quote             map[string]*Quote `json:"quote"`
}

// MapListing is the structure of a map listing
type MapListing struct {
	ID                  float64 `json:"id"`
	Name                string  `json:"name"`
	Symbol              string  `json:"symbol"`
	Slug                string  `json:"slug"`
	IsActive            int     `json:"is_active"`
	FirstHistoricalData string  `json:"first_historical_data"`
	LastHistoricalData  string  `json:"last_historical_data"`
	Platform            *string
}

// ConvertListing is the converted listing structure
type ConvertListing struct {
	ID          int                      `json:"id"`
	Name        string                   `json:"name"`
	Symbol      string                   `json:"symbol"`
	Amount      float64                  `json:"amount"`
	LastUpdated string                   `json:"last_updated"`
	Quote       map[string]*ConvertQuote `json:"quote"`
}

// ConvertQuote is the converted listing structure
type ConvertQuote struct {
	Price       float64 `json:"price"`
	LastUpdated string  `json:"last_updated"`
}

// QuoteLatest is the quotes structure
type QuoteLatest struct {
	ID                float64           `json:"id"`
	Name              string            `json:"name"`
	Symbol            string            `json:"symbol"`
	Slug              string            `json:"slug"`
	CirculatingSupply float64           `json:"circulating_supply"`
	TotalSupply       float64           `json:"total_supply"`
	MaxSupply         float64           `json:"max_supply"`
	DateAdded         string            `json:"date_added"`
	NumMarketPairs    float64           `json:"num_market_pairs"`
	CMCRank           float64           `json:"cmc_rank"`
	LastUpdated       string            `json:"last_updated"`
	Quote             map[string]*Quote `json:"quote"`
}

// Quote is the quote structure
type Quote struct {
	Price            float64 `json:"price"`
	Volume24H        float64 `json:"volume_24h"`
	PercentChange1H  float64 `json:"percent_change_1h"`
	PercentChange24H float64 `json:"percent_change_24h"`
	PercentChange7D  float64 `json:"percent_change_7d"`
	MarketCap        float64 `json:"market_cap"`
	LastUpdated      string  `json:"last_updated"`
}

// MarketMetrics is the market metrics structure
type MarketMetrics struct {
	BTCDominance           float64                        `json:"btc_dominance"`
	ETHDominance           float64                        `json:"eth_dominance"`
	ActiveCryptocurrencies float64                        `json:"active_cryptocurrencies"`
	ActiveMarketPairs      float64                        `json:"active_market_pairs"`
	ActiveExchanges        float64                        `json:"active_exchanges"`
	LastUpdated            string                         `json:"last_updated"`
	Quote                  map[string]*MarketMetricsQuote `json:"quote"`
}

// MarketMetricsQuote is the quote structure
type MarketMetricsQuote struct {
	TotalMarketCap float64 `json:"total_market_cap"`
	TotalVolume24H float64 `json:"total_volume_24h"`
	LastUpdated    string  `json:"last_updated"`
}

// CryptocurrencyInfo options
type CryptocurrencyInfo struct {
	ID       float64                `json:"id"`
	Name     string                 `json:"name"`
	Symbol   string                 `json:"symbol"`
	Category string                 `json:"category"`
	Slug     string                 `json:"slug"`
	Logo     string                 `json:"logo"`
	Tags     []string               `json:"tags"`
	Urls     map[string]interface{} `json:"urls"`
}

// InfoOptions options
type InfoOptions struct {
	ID     string
	Symbol string
}

// ListingOptions options
type ListingOptions struct {
	Start   int
	Limit   int
	Convert string
	Sort    string
}

// MapOptions options
type MapOptions struct {
	ListingStatus string
	Start         int
	Limit         int
	Symbol        string
}

// QuoteOptions options
type QuoteOptions struct {
	// Covert suppots multiple currencies command separated. eg. "BRL,USD"
	Convert string
	// Symbols suppots multiple tickers command separated. eg. "BTC,ETH,XRP"
	Symbol string
}

// ConvertOptions options
type ConvertOptions struct {
	Amount  float64
	ID      string
	Symbol  string
	Time    int
	Convert string
}

// MarketPairOptions options
type MarketPairOptions struct {
	ID      int
	Symbol  string
	Start   int
	Limit   int
	Convert string
}

// service is abstraction for individual endpoint resources
type service struct {
	client *Client
}

// SortOptions sort options
var SortOptions sortOptions

type sortOptions struct {
	Name              string
	Symbol            string
	DateAdded         string
	MarketCap         string
	Price             string
	CirculatingSupply string
	TotalSupply       string
	MaxSupply         string
	NumMarketPairs    string
	Volume24H         string
	PercentChange1H   string
	PercentChange24H  string
	PercentChange7D   string
}

var (
	// ErrTypeAssertion is type assertion error
	ErrTypeAssertion = errors.New("type assertion error")
)

var (
	siteURL               = "https://coinmarketcap.com"
	baseURL               = "https://pro-api.coinmarketcap.com/v1"
	coinGraphURL          = "https://graphs2.coinmarketcap.com/currencies"
	globalMarketGraphURL  = "https://graphs2.coinmarketcap.com/global/marketcap-total"
	altcoinMarketGraphURL = "https://graphs2.coinmarketcap.com/global/marketcap-altcoin"
)

// NewClient initializes a new client
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = new(Config)
	}

	if cfg.ProAPIKey == "" {
		cfg.ProAPIKey = os.Getenv("CMC_PRO_API_KEY")
	}

	c := &Client{
		proAPIKey: cfg.ProAPIKey,
	}

	c.common.client = c
	c.Cryptocurrency = (*CryptocurrencyService)(&c.common)
	c.Exchange = (*ExchangeService)(&c.common)
	c.GlobalMetrics = (*GlobalMetricsService)(&c.common)
	c.Tools = (*ToolsService)(&c.common)

	return c
}

// Info returns all static metadata for one or more cryptocurrencies including name, symbol, logo, and its various registered URLs.
func (s *CryptocurrencyService) Info(options *InfoOptions) (map[string]*CryptocurrencyInfo, error) {
	var params []string
	if options == nil {
		options = new(InfoOptions)
	}
	if options.ID != "" {
		params = append(params, fmt.Sprintf("id=%s", options.ID))
	}
	if options.Symbol != "" {
		params = append(params, fmt.Sprintf("symbol=%s", options.Symbol))
	}

	url := fmt.Sprintf("%s/cryptocurrency/info?%s", baseURL, strings.Join(params, "&"))

	body, err := s.client.makeReq(url)
	resp := new(Response)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}

	var result = make(map[string]*CryptocurrencyInfo)
	ifcs, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, ErrTypeAssertion
	}

	for k, v := range ifcs {
		info := new(CryptocurrencyInfo)
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(b, info)
		if err != nil {
			return nil, err
		}
		result[k] = info
	}

	return result, nil
}

// LatestListings gets a paginated list of all cryptocurrencies with latest market data. You can configure this call to sort by market cap or another market ranking field. Use the "convert" option to return market values in multiple fiat and cryptocurrency conversions in the same call.
func (s *CryptocurrencyService) LatestListings(options *ListingOptions) ([]*Listing, error) {
	var params []string
	if options == nil {
		options = new(ListingOptions)
	}
	if options.Start != 0 {
		params = append(params, fmt.Sprintf("start=%v", options.Start))
	}
	if options.Limit != 0 {
		params = append(params, fmt.Sprintf("limit=%v", options.Limit))
	}
	if options.Convert != "" {
		params = append(params, fmt.Sprintf("convert=%s", options.Convert))
	}
	if options.Sort != "" {
		params = append(params, fmt.Sprintf("sort=%s", options.Sort))
	}

	url := fmt.Sprintf("%s/cryptocurrency/listings/latest?%s", baseURL, strings.Join(params, "&"))

	body, err := s.client.makeReq(url)
	resp := new(Response)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("JSON Error: [%s]. Response body: [%s]", err.Error(), string(body))
	}

	var listings []*Listing
	ifcs, ok := resp.Data.([]interface{})
	if !ok {
		return nil, ErrTypeAssertion
	}

	for i := range ifcs {
		ifc := ifcs[i]
		listing := new(Listing)
		b, err := json.Marshal(ifc)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(b, listing)
		if err != nil {
			return nil, err
		}
		listings = append(listings, listing)
	}

	return listings, nil
}

// Map returns a paginated list of all cryptocurrencies by CoinMarketCap ID.
func (s *CryptocurrencyService) Map(options *MapOptions) ([]*MapListing, error) {
	var params []string
	if options == nil {
		options = new(MapOptions)
	}

	if options.ListingStatus != "" {
		params = append(params, fmt.Sprintf("listing_status=%s", options.ListingStatus))
	}

	if options.Start != 0 {
		params = append(params, fmt.Sprintf("start=%d", options.Start))
	}

	if options.Limit != 0 {
		params = append(params, fmt.Sprintf("limit=%d", options.Limit))
	}

	if options.Symbol != "" {
		params = append(params, fmt.Sprintf("symbol=%s", options.Symbol))
	}

	url := fmt.Sprintf("%s/cryptocurrency/map?%s", baseURL, strings.Join(params, "&"))

	body, err := s.client.makeReq(url)
	resp := new(Response)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("JSON Error: [%s]. Response body: [%s]", err.Error(), string(body))
	}

	var result []*MapListing
	ifcs, ok := resp.Data.(interface{})
	if !ok {
		return nil, ErrTypeAssertion
	}

	for _, item := range ifcs.([]interface{}) {
		value := new(MapListing)
		b, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(b, value)
		if err != nil {
			return nil, err
		}

		result = append(result, value)
	}

	return result, nil
}

// Exchange ...
type Exchange struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// MarketPairBase ...
type MarketPairBase struct {
	CurrencyID     int    `json:"currency_id"`
	CurrencySymbol string `json:"currency_symbol"`
	CurrencyType   string `json:"currency_type"`
}

// MarketPairQuote ...
type MarketPairQuote struct {
	CurrencyID     int    `json:"currency_id"`
	CurrencySymbol string `json:"currency_symbol"`
	CurrencyType   string `json:"currency_type"`
}

// ExchangeQuote ...
type ExchangeQuote struct {
	Price          float64 `json:"price"`
	Volume24       float64 `json:"volume_24h"`
	Volume24HBase  float64 `json:"volume_24h_base"`  // for 'exchange_reported'
	Volume24HQuote float64 `json:"volume_24h_quote"` // for 'exchange_reported'
	LastUpdated    string  `json:"last_updated"`
}

// ExchangeQuotes ...
type ExchangeQuotes map[string]*ExchangeQuote

// ExchangeReported ...
type ExchangeReported struct {
	Price          float64 `json:"price"`
	Volume24HBase  float64 `json:"volume_24h_base"`
	Volume24HQuote float64 `json:"volume_24h_quote"`
	LastUpdated    string  `json:"last_updated"`
}

// MarketPairs ...
type MarketPairs struct {
	ID             int           `json:"id"`
	Name           string        `json:"name"`
	Symbol         string        `json:"symbol"`
	NumMarketPairs int           `json:"num_market_pairs"`
	MarketPairs    []*MarketPair `json:"market_pairs"`
}

// MarketPair ...
type MarketPair struct {
	Exchange         *Exchange
	MarketPair       string            `json:"market_pair"`
	MarketPairBase   *MarketPairBase   `json:"market_pair_base"`
	MarketPairQuote  *MarketPairQuote  `json:"market_pair_quote"`
	Quote            ExchangeQuotes    `json:"quote"`
	ExchangeReported *ExchangeReported `json:"exchange_reported"`
}

// LatestMarketPairs Lists all market pairs across all exchanges for the specified cryptocurrency with associated stats. Use the "convert" option to return market values in multiple fiat and cryptocurrency conversions in the same call.
func (s *CryptocurrencyService) LatestMarketPairs(options *MarketPairOptions) (*MarketPairs, error) {
	var params []string
	if options == nil {
		options = new(MarketPairOptions)
	}

	if options.ID != 0 {
		params = append(params, fmt.Sprintf("id=%v", options.ID))
	}

	if options.Symbol != "" {
		params = append(params, fmt.Sprintf("symbol=%s", options.Symbol))
	}

	if options.Start != 0 {
		params = append(params, fmt.Sprintf("start=%v", options.Start))
	}

	if options.Limit != 0 {
		params = append(params, fmt.Sprintf("limit=%v", options.Limit))
	}

	if options.Convert != "" {
		params = append(params, fmt.Sprintf("convert=%s", options.Convert))
	}

	url := fmt.Sprintf("%s/cryptocurrency/market-pairs/latest?%s", baseURL, strings.Join(params, "&"))

	body, err := s.client.makeReq(url)
	resp := new(Response)

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("JSON Error: [%s]. Response body: [%s]", err.Error(), string(body))
	}

	data, ok := resp.Data.(interface{})
	if !ok {
		return nil, ErrTypeAssertion
	}

	marketPairs := new(MarketPairs)
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, marketPairs)
	if err != nil {
		return nil, err
	}

	for _, pair := range marketPairs.MarketPairs {
		reported, ok := pair.Quote["exchange_reported"]
		if ok {
			pair.ExchangeReported = &ExchangeReported{
				Price:          reported.Price,
				Volume24HBase:  reported.Volume24HBase,
				Volume24HQuote: reported.Volume24HQuote,
				LastUpdated:    reported.LastUpdated,
			}

			delete(pair.Quote, "exchange_reported")
		}
	}

	return marketPairs, nil
}

// HistoricalOHLCV NOT IMPLEMENTED
func (s *CryptocurrencyService) HistoricalOHLCV() error {
	return nil
}

// LatestQuotes gets latest quote for each specified symbol. Use the "convert" option to return market values in multiple fiat and cryptocurrency conversions in the same call.
func (s *CryptocurrencyService) LatestQuotes(options *QuoteOptions) ([]*QuoteLatest, error) {
	var params []string
	if options == nil {
		options = new(QuoteOptions)
	}

	if options.Symbol != "" {
		params = append(params, fmt.Sprintf("symbol=%s", options.Symbol))
	}

	if options.Convert != "" {
		params = append(params, fmt.Sprintf("convert=%s", options.Convert))
	}

	url := fmt.Sprintf("%s/cryptocurrency/quotes/latest?%s", baseURL, strings.Join(params, "&"))

	body, err := s.client.makeReq(url)
	resp := new(Response)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("JSON Error: [%s]. Response body: [%s]", err.Error(), string(body))
	}

	var quotesLatest []*QuoteLatest
	ifcs, ok := resp.Data.(interface{})
	if !ok {
		return nil, ErrTypeAssertion
	}

	for _, coinObj := range ifcs.(map[string]interface{}) {
		quoteLatest := new(QuoteLatest)
		b, err := json.Marshal(coinObj)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(b, quoteLatest)
		if err != nil {
			return nil, err
		}

		quotesLatest = append(quotesLatest, quoteLatest)
	}
	return quotesLatest, nil
}

// HistoricalQuotes NOT IMPLEMENTED
func (s *CryptocurrencyService) HistoricalQuotes() error {
	return nil
}

// Info NOT IMPLEMENTED
func (s *ExchangeService) Info() error {
	return nil
}

// Map NOT IMPLEMENTED
func (s *ExchangeService) Map() error {
	return nil
}

// LatestListings NOT IMPLEMENTED
func (s *ExchangeService) LatestListings() error {
	return nil
}

// LatestMarketPairs NOT IMPLEMENTED
func (s *ExchangeService) LatestMarketPairs() error {
	return nil
}

// LatestQuotes NOT IMPLEMENTED
func (s *ExchangeService) LatestQuotes() error {
	return nil
}

// HistoricalQuotes NOT IMPLEMENTED
func (s *ExchangeService) HistoricalQuotes() error {
	return nil
}

// LatestQuotes Get the latest quote of aggregate market metrics. Use the "convert" option to return market values in multiple fiat and cryptocurrency conversions in the same call.
func (s *GlobalMetricsService) LatestQuotes(options *QuoteOptions) (*MarketMetrics, error) {
	var params []string
	if options == nil {
		options = new(QuoteOptions)
	}

	if options.Convert != "" {
		params = append(params, fmt.Sprintf("convert=%s", options.Convert))
	}

	url := fmt.Sprintf("%s/global-metrics/quotes/latest?%s", baseURL, strings.Join(params, "&"))

	body, err := s.client.makeReq(url)
	resp := new(Response)

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("JSON Error: [%s]. Response body: [%s]", err.Error(), string(body))
	}

	data, ok := resp.Data.(interface{})
	if !ok {
		return nil, ErrTypeAssertion
	}

	marketMetrics := new(MarketMetrics)
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, marketMetrics)
	if err != nil {
		return nil, err
	}

	return marketMetrics, nil
}

// HistoricalQuotes NOT IMPLEMENTED
func (s *GlobalMetricsService) HistoricalQuotes() error {
	return nil
}

// PriceConversion Convert an amount of one currency into multiple cryptocurrencies or fiat currencies at the same time using the latest market averages. Optionally pass a historical timestamp to convert values based on historic averages.
func (s *ToolsService) PriceConversion(options *ConvertOptions) (*ConvertListing, error) {
	var params []string
	if options == nil {
		options = new(ConvertOptions)
	}

	if options.Amount != 0 {
		params = append(params, fmt.Sprintf("amount=%f", options.Amount))
	}

	if options.ID != "" {
		params = append(params, fmt.Sprintf("id=%s", options.ID))
	}

	if options.Symbol != "" {
		params = append(params, fmt.Sprintf("symbol=%s", options.Symbol))
	}

	if options.Time != 0 {
		params = append(params, fmt.Sprintf("time=%d", options.Time))
	}

	if options.Convert != "" {
		params = append(params, fmt.Sprintf("convert=%s", options.Convert))
	}

	url := fmt.Sprintf("%s/tools/price-conversion?%s", baseURL, strings.Join(params, "&"))

	body, err := s.client.makeReq(url)

	resp := new(Response)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, fmt.Errorf("JSON Error: [%s]. Response body: [%s]", err.Error(), string(body))
	}

	ifc, ok := resp.Data.(interface{})
	if !ok {
		return nil, ErrTypeAssertion
	}

	listing := new(ConvertListing)
	b, err := json.Marshal(ifc)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, listing)
	if err != nil {
		return nil, err
	}

	return listing, nil
}

func init() {
	SortOptions = sortOptions{
		Name:              "name",
		Symbol:            "symbol",
		DateAdded:         "date_added",
		MarketCap:         "market_cap",
		Price:             "price",
		CirculatingSupply: "circulating_supply",
		TotalSupply:       "total_supply",
		MaxSupply:         "max_supply",
		NumMarketPairs:    "num_market_pairs",
		Volume24H:         "volume_24h",
		PercentChange1H:   "percent_change_1h",
		PercentChange24H:  "percent_change_24h",
		PercentChange7D:   "percent_change_7d",
	}
}
