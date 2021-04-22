package rest

import (
	"fmt"
	"os"

	yaa "github.com/hhio618/ya-go-client/ya-activity"
	yam "github.com/hhio618/ya-go-client/ya-market"
	yap "github.com/hhio618/ya-go-client/ya-payment"

	"github.com/pkg/errors"
)

const (
	DefaultYagnaApiUrl = "http://127.0.0.1:7465"
)

type MissingConfiguration struct {
	key         string
	description string
}

func (e *MissingConfiguration) Error() string {
	return fmt.Sprintf("missing configuration for %v, Please set env var %v",
		e.description, e.key)
}

type Configuration struct {
	appKey      string
	url         string
	marketUrl   string
	paymentUrl  string
	activityUrl string
}

func NewConfiguration(appKey string,
	url string,
	marketUrl string,
	paymentUrl string,
	activityUrl string) (*Configuration, error) {
	if appKey == "" {
		appKey = os.Getenv("YAGNA_APPKEY")
		if appKey == "" {
			return nil, errors.New("missing API authentication token")
		}
	}
	if url == "" {
		url = DefaultYagnaApiUrl
	}
	return &Configuration{
		appKey:      appKey,
		url:         url,
		marketUrl:   resolveUrl(url, marketUrl, "YAGNA_MARKET_URL", "/market-api/v1"),
		paymentUrl:  resolveUrl(url, paymentUrl, "YAGNA_PAYMENT_URL", "/payment-api/v1"),
		activityUrl: resolveUrl(url, activityUrl, "YAGNA_ACTIVITY_URL", "/activity-api/v1"),
	}, nil

}

func (c *Configuration) AppKey() string {
	return c.appKey
}

func (c *Configuration) Url() string {
	return c.url
}

func (c *Configuration) MarketUrl() string {
	return c.marketUrl
}

func (c *Configuration) PaymentUrl() string {
	return c.paymentUrl
}

func (c *Configuration) ActivityUrl() string {
	return c.activityUrl
}

func (c *Configuration) Market() *yam.APIClient {
	cfg := yam.NewConfiguration()
	cfg.Host = c.marketUrl
	cfg.DefaultHeader["authorization"] =
		fmt.Sprintf("Bearer %v", c.appKey)
	return yam.NewAPIClient(cfg)

}

func (c *Configuration) Activity() *yaa.APIClient {
	cfg := yaa.NewConfiguration()
	cfg.Host = c.activityUrl
	cfg.DefaultHeader["authorization"] =
		fmt.Sprintf("Bearer %v", c.appKey)
	return yaa.NewAPIClient(cfg)
}

func (c *Configuration) Payment() *yap.APIClient {
	cfg := yap.NewConfiguration()
	cfg.Host = c.paymentUrl
	cfg.DefaultHeader["authorization"] =
		fmt.Sprintf("Bearer %v", c.appKey)
	return yap.NewAPIClient(cfg)

}

func resolveUrl(url, givenUrl, envVar, prefix string) string {
	if givenUrl != "" {
		return givenUrl
	}
	if env := os.Getenv(envVar); env != "" {
		return env
	}
	return fmt.Sprintf("%v%v", url, prefix)
}
