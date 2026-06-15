package prices

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Quote struct {
	BuyPrice, BuybackPrice float64
	RetrievedAt            time.Time
}
type Provider interface {
	Quote(context.Context, string, float64) (Quote, error)
}
type Registry struct{ client *http.Client }

func NewRegistry() *Registry { return &Registry{client: &http.Client{Timeout: 10 * time.Second}} }

const defaultGaleri24URL = "https://logam-mulia-api.iamutaki.workers.dev/api/prices/galeri24"

type galeri24Response struct {
	Success bool            `json:"success"`
	Data    []galeri24Price `json:"data"`
}

type galeri24Price struct {
	Material     string  `json:"material"`
	MaterialType string  `json:"materialType"`
	Weight       float64 `json:"weight"`
	SellPrice    float64 `json:"sellPrice"`
	BuybackPrice float64 `json:"buybackPrice"`
	RecordedDate string  `json:"recordedDate"`
}

func (r *Registry) Quote(ctx context.Context, brand, metal string, weight float64) (Quote, error) {
	if materialType, ok := galeri24MaterialType(brand); ok {
		return r.galeri24Quote(ctx, materialType, metal, weight)
	}

	key := strings.ToUpper(strings.NewReplacer(" ", "", "-", "", "_", "").Replace(brand))
	url := os.Getenv(key + "_PRICE_URL")
	if url == "" {
		return Quote{}, fmt.Errorf("no price feed configured for %s", brand)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Quote{}, err
	}
	q := req.URL.Query()
	q.Set("metal", metal)
	q.Set("weight", strconv.FormatFloat(weight, 'f', -1, 64))
	req.URL.RawQuery = q.Encode()
	res, err := r.client.Do(req)
	if err != nil {
		return Quote{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Quote{}, fmt.Errorf("provider returned HTTP %d", res.StatusCode)
	}
	var payload any
	if err = json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return Quote{}, err
	}
	buy, err := numberAt(payload, env(key+"_BUY_PRICE_PATH", "buy_price"))
	if err != nil {
		return Quote{}, fmt.Errorf("buy price: %w", err)
	}
	buyback, err := numberAt(payload, env(key+"_BUYBACK_PRICE_PATH", "buyback_price"))
	if err != nil {
		return Quote{}, fmt.Errorf("buyback price: %w", err)
	}
	return Quote{BuyPrice: buy, BuybackPrice: buyback, RetrievedAt: time.Now().UTC()}, nil
}

func (r *Registry) galeri24Quote(ctx context.Context, materialType, metal string, weight float64) (Quote, error) {
	if !strings.EqualFold(strings.TrimSpace(metal), "Gold") {
		return Quote{}, fmt.Errorf("Galeri24 feed only provides gold prices, not %s", metal)
	}
	url := env("GALERI24_PRICE_URL", defaultGaleri24URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Quote{}, err
	}
	res, err := r.client.Do(req)
	if err != nil {
		return Quote{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Quote{}, fmt.Errorf("Galeri24 provider returned HTTP %d", res.StatusCode)
	}
	var payload galeri24Response
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return Quote{}, fmt.Errorf("decode Galeri24 response: %w", err)
	}
	if !payload.Success {
		return Quote{}, errors.New("Galeri24 provider reported an unsuccessful response")
	}
	for _, price := range payload.Data {
		if !strings.EqualFold(price.Material, "gold") || !strings.EqualFold(price.MaterialType, materialType) || math.Abs(price.Weight-weight) > 0.000001 {
			continue
		}
		if price.SellPrice <= 0 || price.BuybackPrice <= 0 {
			return Quote{}, fmt.Errorf("%s price for %g gram is currently unavailable", materialType, weight)
		}
		retrievedAt := time.Now().UTC()
		if recordedAt, err := time.Parse("2006-01-02", price.RecordedDate); err == nil {
			retrievedAt = recordedAt.UTC()
		}
		return Quote{BuyPrice: price.SellPrice, BuybackPrice: price.BuybackPrice, RetrievedAt: retrievedAt}, nil
	}
	return Quote{}, fmt.Errorf("no %s price found for %g gram", materialType, weight)
}

func galeri24MaterialType(brand string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(brand)) {
	case "GALERI24", "GALERI 24":
		return "GALERI 24", true
	case "ANTAM":
		return "ANTAM", true
	default:
		return "", false
	}
}

func numberAt(payload any, path string) (float64, error) {
	current := payload
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return 0, errors.New("invalid JSON path")
		}
		current, ok = object[part]
		if !ok {
			return 0, fmt.Errorf("key %q missing", part)
		}
	}
	switch value := current.(type) {
	case float64:
		return value, nil
	case string:
		return strconv.ParseFloat(strings.ReplaceAll(value, ",", ""), 64)
	default:
		return 0, errors.New("value is not a number")
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
