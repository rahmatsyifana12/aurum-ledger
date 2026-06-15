package prices

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGaleri24QuoteSelectsBrandAndWeight(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":[
			{"material":"gold","materialType":"GALERI 24","weight":1,"sellPrice":2718000,"buybackPrice":2549000,"recordedDate":"2026-06-15"},
			{"material":"gold","materialType":"ANTAM","weight":1,"sellPrice":2800000,"buybackPrice":2600000,"recordedDate":"2026-06-15"},
			{"material":"gold","materialType":"ANTAM","weight":5,"sellPrice":13957000,"buybackPrice":12707000,"recordedDate":"2026-06-15"}
		]}`))
	}))
	defer server.Close()
	t.Setenv("GALERI24_PRICE_URL", server.URL)

	quote, err := NewRegistry().Quote(context.Background(), "ANTAM", "Gold", 5)
	if err != nil {
		t.Fatal(err)
	}
	if quote.BuyPrice != 13957000 || quote.BuybackPrice != 12707000 {
		t.Fatalf("unexpected quote: %#v", quote)
	}
}

func TestGaleri24QuoteRejectsUnavailablePrice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":[{"material":"gold","materialType":"ANTAM","weight":1,"sellPrice":0,"buybackPrice":0}]}`))
	}))
	defer server.Close()
	t.Setenv("GALERI24_PRICE_URL", server.URL)

	if _, err := NewRegistry().Quote(context.Background(), "ANTAM", "Gold", 1); err == nil {
		t.Fatal("expected unavailable price error")
	}
}
