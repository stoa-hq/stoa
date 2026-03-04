# Stoa

Eine leichtgewichtige, Open-Source Headless-E-Commerce-Plattform, gebaut mit Go. Das System wird als einzelnes Binary ausgeliefert, in dem sowohl das Admin-Panel als auch der Storefront eingebettet sind.

## Features

- **Headless Architecture** – REST API (JSON)
- **Single Binary** – Go-Backend mit eingebetteten SvelteKit-Frontends (Admin + Storefront)
- **Plugin-System** – Erweiterbar über Hooks und eigene API-Endpunkte
- **Mehrsprachig** – Übersetzungstabellen mit Locale-basierter API
- **Eigenschaftsgruppen & Varianten** – Farbe, Größe etc. mit automatischer Kombinationsgenerierung
- **Volltextsuche** – PostgreSQL-basiert
- **RBAC** – Rollenbasierte Zugriffskontrolle

## Voraussetzungen

| Tool | Version | Wofür |
|------|---------|-------|
| Docker + Docker Compose | aktuell | Datenbank (und optionaler App-Container) |
| Go | 1.23+ | Backend bauen (nur bei lokaler Entwicklung) |
| Node.js | 20+ | Frontends bauen (nur bei lokaler Entwicklung) |
| PostgreSQL | 16+ | Datenbank (wird per Docker bereitgestellt) |

---

## Schnellstart mit Docker (empfohlen)

Dies ist der einfachste Weg, um die gesamte Plattform lokal zu starten. Du brauchst nur Docker.

### 1. Repository klonen

```bash
git clone <repository-url>
cd stoa
```

### 2. Konfiguration anlegen

```bash
cp config.example.yaml config.yaml
```

Die Standardwerte funktionieren sofort mit Docker Compose — du musst nichts ändern.

### 3. Alles starten

```bash
docker compose up -d
```

Das startet PostgreSQL und die Stoa-Anwendung. Beim ersten Mal wird das Docker-Image gebaut (inkl. Admin- und Storefront-Frontend). Das dauert beim ersten Mal einige Minuten.

### 4. Datenbank einrichten

```bash
# Migrationen ausführen (Tabellen anlegen)
docker compose exec stoa ./stoa migrate up

# Admin-Benutzer anlegen
docker compose exec stoa ./stoa admin create --email admin@example.com --password dein-passwort

# Optional: Demo-Daten laden (Produkte, Kategorien, etc.)
docker compose exec stoa ./stoa seed --demo
```

### 5. Fertig!

| Was | URL |
|-----|-----|
| Storefront | http://localhost:8080 |
| Admin-Panel | http://localhost:8080/admin |
| API Health-Check | http://localhost:8080/api/v1/health |

Melde dich im Admin-Panel mit den Zugangsdaten aus Schritt 4 an.

### Stoppen und Neustarten

```bash
# Stoppen (Daten bleiben erhalten)
docker compose down

# Stoppen und alle Daten löschen
docker compose down -v

# Neustarten
docker compose up -d
```

---

## Lokale Entwicklung (ohne Docker für die App)

Für die Entwicklung am Code ist es praktischer, nur PostgreSQL per Docker laufen zu lassen und die App direkt auszuführen.

### 1. PostgreSQL starten

```bash
docker compose up -d postgres
```

### 2. Konfiguration anlegen

```bash
cp config.example.yaml config.yaml
```

### 3. Datenbank einrichten

```bash
go run ./cmd/stoa migrate up
go run ./cmd/stoa admin create --email admin@example.com --password dein-passwort
go run ./cmd/stoa seed --demo   # optional
```

### 4. Frontends bauen

Sowohl Admin als auch Storefront sind SvelteKit-Anwendungen, die per `//go:embed` in das Go-Binary eingebettet werden. Vor dem ersten Start müssen sie gebaut werden:

```bash
# Admin-Panel
cd admin && npm install && npm run build && cd ..

# Storefront
cd storefront && npm install && npm run build && cd ..
```

> **Wichtig:** Nach jeder Änderung an den Frontends muss `npm run build` UND danach das Go-Binary neu gebaut werden, weil die Frontends statisch in das Binary eingebettet sind.

### 5. Backend starten

```bash
go run ./cmd/stoa serve
```

Oder als kompiliertes Binary:

```bash
go build -o stoa ./cmd/stoa
./stoa serve
```

### Frontend-Entwicklung mit Hot-Reload

Für die Entwicklung an den Frontends kannst du die Vite-Dev-Server starten, die Hot-Reload bieten:

```bash
# Admin-Panel (Port 5174)
cd admin && npm run dev

# Storefront (Port 5173)
cd storefront && npm run dev
```

Die Dev-Server kommunizieren über die API mit dem Go-Backend auf Port 8080. Stelle sicher, dass das Backend läuft.

---

## Makefile-Befehle

```bash
make build              # Frontends bauen + Go-Binary kompilieren
make run                # build + starten
make test               # Go-Tests ausführen
make test-race          # Tests mit Race-Detector
make lint               # Linter ausführen (golangci-lint + go vet)
make docker-up          # docker compose up -d
make docker-down        # docker compose down
make admin-dev          # Admin-Frontend Dev-Server
make storefront-dev     # Storefront Dev-Server
make seed               # Demo-Daten laden
```

---

## Konfiguration

Alle Einstellungen stehen in `config.yaml`. Alternativ können sie per Umgebungsvariable mit dem Prefix `STOA_` überschrieben werden:

```bash
STOA_DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=disable"
STOA_AUTH_JWT_SECRET="ein-sicherer-schlüssel"
STOA_SERVER_PORT=8080
```

### Wichtige Einstellungen

| Einstellung | Standard | Beschreibung |
|-------------|----------|--------------|
| `server.port` | `8080` | HTTP-Port |
| `database.url` | `postgres://stoa:secret@localhost:5432/stoa` | PostgreSQL-Connection-String |
| `auth.jwt_secret` | `change-me-in-production` | JWT-Signierungsschlüssel |
| `media.storage` | `local` | Medien-Speicher (`local` oder `s3`) |
| `media.local_path` | `./uploads` | Lokaler Upload-Pfad |
| `i18n.default_locale` | `de-DE` | Standard-Sprache |

---

## API-Übersicht

| Bereich | Pfad | Authentifizierung |
|---------|------|-------------------|
| Admin-API | `/api/v1/admin/*` | JWT (Admin-Rolle) |
| Store-API | `/api/v1/store/*` | Öffentlich / Kunden-JWT |
| Auth | `/api/v1/auth/*` | Keine |
| Health | `/api/v1/health` | Keine |

### Authentifizierung

```bash
# Admin-Login
curl -X POST http://localhost:8080/api/v1/auth/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"email": "admin@example.com", "password": "dein-passwort"}'

# Antwort enthält access_token und refresh_token
# access_token in Authorization-Header verwenden:
curl http://localhost:8080/api/v1/admin/products \
  -H 'Authorization: Bearer <access_token>'
```

---

## CLI-Befehle

```bash
stoa serve                  # HTTP-Server starten
stoa migrate up             # Migrationen ausführen
stoa migrate down           # Letzte Migration zurückrollen
stoa admin create           # Admin-Benutzer anlegen
  --email admin@example.com
  --password dein-passwort
stoa seed --demo            # Demo-Daten laden
stoa plugin list            # Installierte Plugins anzeigen
stoa version                # Version ausgeben
```

---

## Projektstruktur

```
stoa/
├── cmd/stoa/           # CLI-Einstiegspunkt (main.go)
├── internal/
│   ├── app/                # Application-Bootstrapping
│   ├── config/             # Konfiguration laden
│   ├── server/             # HTTP-Server, Router, Middleware
│   ├── auth/               # JWT, RBAC, Berechtigungen
│   ├── database/           # DB-Verbindung, Migrations-Runner
│   ├── domain/             # Business-Logik (DDD-artig)
│   │   ├── product/        # Produkte, Varianten, Eigenschaftsgruppen
│   │   ├── category/       # Kategorien (Baumstruktur)
│   │   ├── order/          # Bestellungen
│   │   ├── cart/           # Warenkorb
│   │   ├── customer/       # Kundenverwaltung
│   │   ├── media/          # Medien-Upload
│   │   ├── discount/       # Rabatte
│   │   ├── shipping/       # Versandmethoden
│   │   ├── payment/        # Zahlungsmethoden
│   │   ├── tax/            # Steuerregeln
│   │   ├── tag/            # Tags
│   │   └── audit/          # Audit-Log
│   ├── admin/              # Eingebettetes Admin-Frontend (//go:embed)
│   ├── storefront/         # Eingebettetes Storefront (//go:embed)
│   ├── plugin/             # Plugin-Registry
│   └── search/             # Suchindex
├── admin/                  # Admin-Frontend (SvelteKit)
├── storefront/             # Storefront (SvelteKit)
├── migrations/             # SQL-Migrationen
├── pkg/sdk/                # Plugin-SDK
├── Dockerfile
├── docker-compose.yaml
├── Makefile
└── config.example.yaml
```

Jede Domain folgt dem gleichen Muster:
- `entity.go` – Datenstrukturen
- `repository.go` – Interface
- `postgres.go` – Implementierung
- `service.go` – Business-Logik
- `handler.go` – HTTP-Handler
- `dto.go` – Request/Response-Typen

---

## Plugins entwickeln

Stoa hat ein eingebautes Plugin-System, mit dem du die Plattform erweitern kannst, ohne den Kerncode zu ändern. Plugins können:

- **Auf Ereignisse reagieren** (z. B. E-Mail versenden nach Bestellung)
- **Operationen verhindern** (z. B. Validierung vor Warenkorb-Änderung)
- **Eigene API-Endpunkte** bereitstellen
- **Direkt auf die Datenbank** zugreifen

### Plugin-Interface

Jedes Plugin implementiert das `sdk.Plugin`-Interface aus `pkg/sdk`:

```go
package sdk

type Plugin interface {
    Name() string        // Eindeutiger Name, z. B. "order-email"
    Version() string     // Semver, z. B. "1.0.0"
    Description() string // Kurzbeschreibung
    Init(app *AppContext) error   // Wird beim Start aufgerufen
    Shutdown() error              // Wird beim Herunterfahren aufgerufen
}
```

In der `Init`-Methode bekommt das Plugin einen `AppContext` mit allem, was es braucht:

```go
type AppContext struct {
    DB     *pgxpool.Pool       // PostgreSQL-Verbindung
    Router chi.Router           // HTTP-Router für eigene Endpunkte
    Hooks  *HookRegistry        // Event-System
    Config map[string]interface{} // Plugin-spezifische Konfiguration
    Logger zerolog.Logger        // Strukturiertes Logging
}
```

### Beispiel: E-Mail bei neuer Bestellung

Erstelle eine neue Datei, z. B. `plugins/orderemail/plugin.go`:

```go
package orderemail

import (
    "context"
    "fmt"

    "github.com/epoxx-arch/stoa/internal/domain/order"
    "github.com/epoxx-arch/stoa/pkg/sdk"
)

type Plugin struct {
    logger zerolog.Logger
}

func New() *Plugin {
    return &Plugin{}
}

func (p *Plugin) Name() string        { return "order-email" }
func (p *Plugin) Version() string     { return "1.0.0" }
func (p *Plugin) Description() string { return "Sendet Bestätigungs-E-Mails nach Bestellungen" }

func (p *Plugin) Init(app *sdk.AppContext) error {
    p.logger = app.Logger

    // Nach jeder neuen Bestellung eine E-Mail versenden
    app.Hooks.On(sdk.HookAfterOrderCreate, func(ctx context.Context, event *sdk.HookEvent) error {
        o := event.Entity.(*order.Order)
        p.logger.Info().
            Str("order", o.OrderNumber).
            Msg("Bestätigungsmail versenden")

        // Hier: SMTP-Versand, externer Service, etc.
        return nil
    })

    return nil
}

func (p *Plugin) Shutdown() error {
    return nil
}
```

### Beispiel: Mindestbestellwert prüfen

Before-Hooks können Operationen **verhindern**, indem sie einen Fehler zurückgeben:

```go
func (p *Plugin) Init(app *sdk.AppContext) error {
    app.Hooks.On(sdk.HookBeforeCheckout, func(ctx context.Context, event *sdk.HookEvent) error {
        o := event.Entity.(*order.Order)
        if o.Total < 1000 { // Preise in Cent
            return fmt.Errorf("Mindestbestellwert: 10,00 €")
        }
        return nil
    })
    return nil
}
```

### Beispiel: Eigene API-Endpunkte

Plugins können über den Chi-Router eigene Endpunkte registrieren:

```go
func (p *Plugin) Init(app *sdk.AppContext) error {
    app.Router.Route("/api/v1/wishlist", func(r chi.Router) {
        r.Get("/", p.handleList)
        r.Post("/", p.handleAdd)
        r.Delete("/{id}", p.handleRemove)
    })

    return nil
}

func (p *Plugin) handleList(w http.ResponseWriter, r *http.Request) {
    // Direkter DB-Zugriff über p.db (im Init gespeichert)
    rows, err := p.db.Query(r.Context(), "SELECT * FROM wishlists WHERE customer_id = $1", customerID)
    // ...
}
```

### Plugin registrieren

Um ein Plugin zu aktivieren, registriere es in `internal/app/app.go` nach dem Erstellen der `App`:

```go
import "github.com/epoxx-arch/stoa/plugins/orderemail"

// In New() oder einer eigenen Methode:
func (a *App) RegisterPlugins() error {
    appCtx := &plugin.AppContext{
        DB:     a.DB.Pool,
        Router: a.Server.Router(),
        Config: nil, // oder aus config.yaml laden
        Logger: a.Logger,
    }

    return a.PluginRegistry.Register(orderemail.New(), appCtx)
}
```

### Verfügbare Hooks

| Hook | Zeitpunkt | Kann abbrechen? |
|------|-----------|-----------------|
| `product.before_create` | Vor Produkt-Erstellung | Ja |
| `product.after_create` | Nach Produkt-Erstellung | Nein |
| `product.before_update` | Vor Produkt-Update | Ja |
| `product.after_update` | Nach Produkt-Update | Nein |
| `product.before_delete` | Vor Produkt-Löschung | Ja |
| `product.after_delete` | Nach Produkt-Löschung | Nein |
| `order.before_create` | Vor Bestellerstellung | Ja |
| `order.after_create` | Nach Bestellerstellung | Nein |
| `order.before_update` | Vor Status-Änderung | Ja |
| `order.after_update` | Nach Status-Änderung | Nein |
| `cart.before_add_item` | Vor Warenkorb-Hinzufügen | Ja |
| `cart.after_add_item` | Nach Warenkorb-Hinzufügen | Nein |
| `cart.before_update_item` | Vor Mengen-Änderung | Ja |
| `cart.after_update_item` | Nach Mengen-Änderung | Nein |
| `cart.before_remove_item` | Vor Artikel-Entfernung | Ja |
| `cart.after_remove_item` | Nach Artikel-Entfernung | Nein |
| `customer.before_create` | Vor Kundenregistrierung | Ja |
| `customer.after_create` | Nach Kundenregistrierung | Nein |
| `customer.before_update` | Vor Kunden-Update | Ja |
| `customer.after_update` | Nach Kunden-Update | Nein |
| `category.before_create` | Vor Kategorie-Erstellung | Ja |
| `category.after_create` | Nach Kategorie-Erstellung | Nein |
| `category.before_update` | Vor Kategorie-Update | Ja |
| `category.after_update` | Nach Kategorie-Update | Nein |
| `category.before_delete` | Vor Kategorie-Löschung | Ja |
| `category.after_delete` | Nach Kategorie-Löschung | Nein |
| `checkout.before` | Vor Checkout-Abschluss | Ja |
| `checkout.after` | Nach Checkout-Abschluss | Nein |
| `payment.after_complete` | Nach erfolgreicher Zahlung | Nein |
| `payment.after_failed` | Nach fehlgeschlagener Zahlung | Nein |

**Before-Hooks** werden vor der Datenbankoperation ausgeführt und können die Operation durch Rückgabe eines Fehlers abbrechen. **After-Hooks** werden danach ausgeführt — Fehler werden nur geloggt, brechen die Operation aber nicht ab.

---

## Lizenz

Apache 2.0 – siehe [LICENSE](LICENSE).
