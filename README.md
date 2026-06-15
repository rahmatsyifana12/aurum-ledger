# Aurum Ledger

A mobile-first precious metal portfolio built with Vue 3 and Go. It supports user registration, login/logout, JWT access and rotating refresh tokens, private metal CRUD, portfolio valuation, and configurable price feeds per brand.

## Run locally

Requirements: Go 1.23+, Node 20+, npm, and a C compiler for SQLite.

```bash
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env

cd backend
set -a; source .env; set +a
go run ./cmd/server
```

In another terminal:

```bash
cd frontend
npm install
npm run dev
```

Open `http://localhost:5173`. On a phone, use the browser's **Add to Home Screen** action to install the PWA while keeping it browser-based.

## Brand price feeds

ANTAM and Galeri24 prices are retrieved from `https://logam-mulia-api.iamutaki.workers.dev/api/prices/galeri24`. The integration selects only `ANTAM` and `GALERI 24`, then matches the holding's exact weight. The API's `sellPrice` is stored as `buy_price`, and `buybackPrice` is stored as `buyback_price`.

Set `GALERI24_PRICE_URL` to override the endpoint. UBS can use the generic configurable JSON provider below.

```json
{
  "buy_price": 17000000,
  "buyback_price": 15500000
}
```

Nested values can be selected with dot paths:

```env
UBS_PRICE_URL=https://your-authorized-feed.example/ubs
UBS_BUY_PRICE_PATH=data.prices.buy
UBS_BUYBACK_PRICE_PATH=data.prices.buyback
```

When a brand does not expose a stable authorized JSON API, current prices remain editable manually. This avoids coupling the application to brittle HTML scraping or bypassing a provider's terms.

## API routes

- `POST /api/auth/register`, `/login`, `/refresh`, `/logout`
- `GET /api/me`
- `GET|POST /api/metals`
- `GET|PUT|DELETE /api/metals/{id}`
- `POST /api/metals/{id}/refresh-price`

All metal routes are user-scoped and require an access token.
