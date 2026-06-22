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

Creating or updating a holding retrieves its current prices from the configured provider. Unsupported brands or weights return an error rather than saving an incorrect price.

## Scheduled price refresh

The backend includes a CLI worker that refreshes every holding's brand prices daily at 10:00 in fixed `+07:00` time.

Run it locally:

```bash
cd backend
set -a; source .env; set +a
go run ./cmd/price-refresher
```

Refresh once and exit, which is useful for cron or systemd timers:

```bash
go run ./cmd/price-refresher -once
```

## Production deployment

The following reference setup uses one Ubuntu server, systemd for the Go API, and Nginx for HTTPS, reverse proxying, and Vue static files. Replace `app.example.com` and `api.example.com` with your domains.

### 1. Server requirements

Install Git, Nginx, a C compiler for SQLite, Go 1.23+, Node 20+, npm, and Certbot:

```bash
sudo apt update
sudo apt install -y git nginx build-essential sqlite3 certbot python3-certbot-nginx
```

Install current Go and Node releases from their official distribution channels if the Ubuntu packages are older than the required versions.

Create a dedicated service user and application directories:

```bash
sudo useradd --system --home /opt/aurum --shell /usr/sbin/nologin aurum
sudo mkdir -p /opt/aurum /var/lib/aurum
sudo chown -R aurum:aurum /opt/aurum /var/lib/aurum
```

Clone or upload this repository to `/opt/aurum`.

Before configuring HTTPS, create DNS `A`/`AAAA` records for `app.example.com` and `api.example.com` pointing to the server and wait for them to resolve.

### 2. Deploy the backend

Build the Go binary on the server:

```bash
cd /opt/aurum/backend
CGO_ENABLED=1 go build -o aurum-api ./cmd/server
CGO_ENABLED=1 go build -o aurum-price-refresher ./cmd/price-refresher
sudo chown aurum:aurum aurum-api
sudo chown aurum:aurum aurum-price-refresher
sudo chmod 755 aurum-api aurum-price-refresher
```

Create `/opt/aurum/backend/.env`:

```env
PORT=8080
DATABASE_PATH=/var/lib/aurum/precious-metals.db
JWT_SECRET=replace-with-a-long-random-production-secret
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h
ALLOWED_ORIGIN=https://app.example.com
GALERI24_PRICE_URL=https://logam-mulia-api.iamutaki.workers.dev/api/prices/galeri24
```

Generate `JWT_SECRET` with a password generator or `openssl rand -base64 48`. Do not commit the production `.env` file.

Create `/etc/systemd/system/aurum-api.service`:

```ini
[Unit]
Description=Aurum Ledger API
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=aurum
Group=aurum
WorkingDirectory=/opt/aurum/backend
EnvironmentFile=/opt/aurum/backend/.env
ExecStart=/opt/aurum/backend/aurum-api
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Start the API:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now aurum-api
sudo systemctl status aurum-api
curl http://127.0.0.1:8080/health
```

Only Nginx needs access to port `8080`; do not expose that port publicly.

Create `/etc/systemd/system/aurum-price-refresher.service`:

```ini
[Unit]
Description=Aurum Ledger daily price refresher
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=aurum
Group=aurum
WorkingDirectory=/opt/aurum/backend
EnvironmentFile=/opt/aurum/backend/.env
ExecStart=/opt/aurum/backend/aurum-price-refresher
Restart=on-failure
RestartSec=30
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Start the scheduled refresher:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now aurum-price-refresher
sudo systemctl status aurum-price-refresher
```

### 3. Deploy the frontend

The API URL is embedded at build time. Create `/opt/aurum/frontend/.env.production`:

```env
VITE_API_URL=https://api.example.com/api
```

Install dependencies and build the production files:

```bash
cd /opt/aurum/frontend
npm ci
npm run build
sudo mkdir -p /var/www/aurum
sudo cp -R dist/. /var/www/aurum/
sudo chown -R www-data:www-data /var/www/aurum
```

### 4. Configure Nginx

Create `/etc/nginx/sites-available/aurum`:

```nginx
server {
    listen 80;
    server_name app.example.com;

    root /var/www/aurum;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location = /sw.js {
        add_header Cache-Control "no-cache";
    }
}

server {
    listen 80;
    server_name api.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Enable the site and HTTPS:

```bash
sudo ln -s /etc/nginx/sites-available/aurum /etc/nginx/sites-enabled/aurum
sudo nginx -t
sudo systemctl reload nginx
sudo certbot --nginx -d app.example.com -d api.example.com
```

HTTPS is required for production PWA installation and service workers. Confirm both `https://app.example.com` and `https://api.example.com/health` work after Certbot finishes.

### 5. Updating production

After pulling a new release, rebuild and restart the backend:

```bash
cd /opt/aurum/backend
CGO_ENABLED=1 go build -o aurum-api ./cmd/server
sudo systemctl restart aurum-api
```

Rebuild and publish the frontend:

```bash
cd /opt/aurum/frontend
npm ci
npm run build
sudo cp -R dist/. /var/www/aurum/
```

Back up `/var/lib/aurum/precious-metals.db` regularly. For a consistent live SQLite backup, use `sqlite3 /var/lib/aurum/precious-metals.db ".backup '/backup/precious-metals.db'"` instead of copying the file while writes are occurring.

## API routes

- `POST /api/auth/register`, `/login`, `/refresh`, `/logout`
- `GET /api/me`
- `GET|POST /api/metals`
- `GET|PUT|DELETE /api/metals/{id}`
- `POST /api/metals/{id}/refresh-price`

All metal routes are user-scoped and require an access token.
