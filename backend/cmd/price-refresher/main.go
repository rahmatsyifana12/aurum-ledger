package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"time"

	"precious-metal-dashboard/backend/internal/config"
	"precious-metal-dashboard/backend/internal/database"
	"precious-metal-dashboard/backend/internal/prices"
)

type metal struct {
	ID     int64
	Type   string
	Brand  string
	Weight float64
}

type refreshSummary struct {
	Total, Refreshed, Failed int
}

var jakartaOffset = time.FixedZone("+07:00", 7*60*60)

func main() {
	once := flag.Bool("once", false, "refresh prices once and exit")
	flag.Parse()

	cfg := config.Load()
	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	registry := prices.NewRegistry()
	if *once {
		refreshAll(context.Background(), db, registry)
		return
	}

	log.Println("price refresher scheduled daily at 10:00 +07:00")
	for {
		next := nextRun(time.Now(), jakartaOffset)
		log.Printf("next price refresh at %s", next.Format(time.RFC3339))
		time.Sleep(time.Until(next))
		refreshAll(context.Background(), db, registry)
	}
}

func nextRun(now time.Time, location *time.Location) time.Time {
	local := now.In(location)
	next := time.Date(local.Year(), local.Month(), local.Day(), 10, 0, 0, 0, location)
	if !next.After(local) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func refreshAll(ctx context.Context, db *sql.DB, registry *prices.Registry) refreshSummary {
	started := time.Now()
	rows, err := db.QueryContext(ctx, `SELECT id,type,brand,weight FROM precious_metals ORDER BY id`)
	if err != nil {
		log.Printf("load metals: %v", err)
		return refreshSummary{}
	}
	defer rows.Close()

	metals := []metal{}
	for rows.Next() {
		var item metal
		if err := rows.Scan(&item.ID, &item.Type, &item.Brand, &item.Weight); err != nil {
			log.Printf("scan metal: %v", err)
			continue
		}
		metals = append(metals, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("read metals: %v", err)
	}

	summary := refreshSummary{Total: len(metals)}
	for _, item := range metals {
		quote, err := registry.Quote(ctx, item.Brand, item.Type, item.Weight)
		if err != nil {
			summary.Failed++
			log.Printf("refresh metal %d (%s %s %.3fg): %v", item.ID, item.Brand, item.Type, item.Weight, err)
			continue
		}
		_, err = db.ExecContext(ctx, `UPDATE precious_metals SET buy_price=?,buyback_price=?,price_updated_at=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, quote.BuyPrice, quote.BuybackPrice, quote.RetrievedAt, item.ID)
		if err != nil {
			summary.Failed++
			log.Printf("save metal %d: %v", item.ID, err)
			continue
		}
		summary.Refreshed++
	}

	log.Printf("price refresh finished in %s: %d total, %d refreshed, %d failed", time.Since(started).Round(time.Millisecond), summary.Total, summary.Refreshed, summary.Failed)
	return summary
}
