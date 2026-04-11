# fin-cascade-looker

Full-stack dashboard for visualizing the fin-cascade knowledge graph, news feed, cascade analysis, and trading signals. Read-only visualization layer — all analysis data is written by the `fin-cascade` project.

## Architecture

```
Go HTTP server (standard library) → serves React SPA + JSON API
  ├── Neo4j (knowledge graph: companies, sectors, supply chains)
  ├── PostgreSQL (scans, events, signals, cascade analysis, users)
  └── SQLite (news articles, scan logs — read-only)
```

## Tech Stack

- **Backend**: Go 1.26.1 (standard library `net/http`, no framework)
- **Frontend**: React 19.1 + React Router 7.6 + Vite 7 + D3.js 7.9
- **Auth**: Firebase (Google OAuth) with email allowlist
- **Icons**: Lucide React
- **Databases**: Same Neo4j + PostgreSQL + SQLite as fin-cascade

## Key Commands

```bash
docker compose -f ../fin-cascade/docker-compose.yml up -d   # Start Neo4j + PostgreSQL (shared with fin-cascade)
cd frontend && npm install && npm run dev                    # Frontend dev server (port 5173)
make dev                                                     # Backend dev server (port 8080)
make frontend                                                # Build frontend to frontend/dist/
make build                                                   # Build frontend + backend binary (bin/looker)
make clean                                                   # Remove bin/, frontend/dist/, node_modules/
```

Hot reload: `.air.toml` watches `.go` files, rebuilds to `./tmp/looker`

## Backend Structure

```
main.go                          # HTTP server, middleware (CORS, Firebase auth), route registration, SPA serving
internal/
  config/config.go               # Env-based config with .env support
  auth/firebase.go               # Firebase JWT verification (RS256), public key caching (1hr TTL), email allowlist
  db/
    neo4j.go                     # Neo4j client wrapper, generic query → map results
    postgres.go                  # PostgreSQL client, User model, UpsertUser (ON CONFLICT)
    sqlite.go                    # SQLite client (read-only mode)
  handlers/
    handlers.go                  # Handler struct (holds all 3 DB clients), helpers: isAdmin, writeJSON, writeError
    stats.go                     # GET /api/stats — Neo4j node/relationship counts + news stats
    companies.go                 # GET /api/companies (paginated, filtered), /api/companies/{ticker}, /{ticker}/graph
    sectors.go                   # GET /api/sectors — sector list with company counts
    news.go                      # GET /api/news (paginated, filtered), /api/news/stats, /api/scans
    analysis.go                  # GET /api/analysis/scans, /scans/{id}, /signals, /stats
    renko.go                     # GET /api/renko/{ticker}, /renko/signals, /renko/stats
```

## API Endpoints

### Admin-only (requires `is_admin` flag on user)
- `GET /api/stats` — Neo4j node counts, relationship counts, news summary
- `GET /api/companies?search=&sector=&cap=&page=` — paginated company list (20/page, max 100)
- `GET /api/companies/{ticker}` — full company profile (plants, competitors, suppliers, customers, materials, sector deps)
- `GET /api/companies/{ticker}/graph` — D3 knowledge graph data (nodes + links)
- `GET /api/sectors` — sectors ordered by company count
- `GET /api/scans` — recent SQLite scan logs

### Public
- `GET /api/me` — current user profile (id, email, name, avatar, is_admin)
- `GET /api/news?classification=&event_type=&page=` — paginated news articles
- `GET /api/news/stats` — article counts by classification and event type
- `GET /api/analysis/scans?page=` — paginated scan list with stats
- `GET /api/analysis/scans/{id}` — deep analysis: events, cascade sections, signals with reasoning chains, source articles
- `GET /api/analysis/signals?ticker=&signal=&page=` — global signals with event context
- `GET /api/analysis/stats` — totals by signal type and event severity
- `GET /api/renko/{ticker}?days=60` — daily prices + renko signals for a ticker
- `GET /api/renko/signals?trend=&direction=&limit=&offset=` — latest renko state per ticker (DISTINCT ON)
- `GET /api/renko/stats` — total tickers, counts by trend and direction

### Static
- `GET /` and all non-API paths → SPA fallback to `frontend/dist/index.html`

## Frontend Structure

```
frontend/src/
  main.jsx                       # Entry point
  App.jsx                        # Router setup, AuthProvider wrapper
  services/
    api.js                       # Centralized API client, auto Firebase token injection, 401 handling
    firebase.js                  # Firebase init, signInWithGoogle, logOut, onAuth, getIdToken
  components/
    AuthProvider.jsx             # Firebase auth context (Google OAuth popup)
    Sidebar.jsx                  # Responsive nav, admin/public route separation, user profile
    StatsCard.jsx                # Metric + label card component
    KnowledgeGraph.jsx           # D3 force-directed graph: zoom/pan/drag, node details, color-coded by type
  pages/
    Dashboard.jsx                # (Admin) Stats cards, sector breakdown, relationships, news summary, top signals
    Companies.jsx                # (Admin) Paginated table, search/sector/cap filters
    CompanyDetail.jsx            # Tabs: Overview, Plants, Competitors, Supply Chain, Raw Materials, Graph
    NewsFeed.jsx                 # (Public) Articles with classification badges, filters
    Analysis.jsx                 # (Public) AnalysisList + AnalysisDetail: events, cascade sections, reasoning chains
  css/                           # Page and component stylesheets
```

## Auth Flow

1. Firebase Google OAuth popup → ID token
2. Token sent as `Authorization: Bearer {token}` on all API calls
3. Backend verifies JWT (RS256, Firebase public keys with 1hr cache)
4. Optional email allowlist (comma-separated in `ALLOWED_EMAILS` env var)
5. User upserted into PostgreSQL on successful auth
6. Unauthenticated requests allowed (no token → no user context, for dev/mobile testing)

## Environment Variables (.env)

```
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=fincascade123
SQLITE_PATH=../fin-cascade/data/news.db        # Points to fin-cascade's SQLite
POSTGRES_DSN=postgres://fincascade:fincascade123@localhost:5432/fincascade?sslmode=disable
PORT=8080
CORS_ORIGIN=http://localhost:5173
FIREBASE_PROJECT_ID=fin-cascade
ALLOWED_EMAILS=                                  # Comma-separated, empty = all authenticated users
SERVER_KEY=                                       # Optional server key for API access
```

## Deployment

GitHub Actions (`.github/workflows/deploy.yml`):
1. Build frontend (Node 20: `npm ci && npm run build`)
2. Build backend (`CGO_ENABLED=0 GOOS=linux GOARCH=amd64`)
3. SCP artifacts to remote server
4. systemd service restart (`fin-cascade-looker`)

Secrets: `STT_HOST`, `STT_USERNAME`, `STT_SSH_KEY`, `STT_PORT`, `SERVER_KEY`

## Knowledge Graph Visualization (D3)

`KnowledgeGraph.jsx` renders a force-directed graph with:
- Node types: Company, Competitor, Plant, Sector, Supplier, Customer, RawMaterial
- Color and size coded by type
- Interactive: zoom, pan, drag, click for details panel
- Link labels show relationship types (COMPETES_WITH, SUPPLIES_TO, etc.)

## Related Project

**fin-cascade** — the analysis engine that writes data to these databases. Go CLI binaries + Claude skills for RSS fetching, news classification, cascade analysis, and signal generation.

## Coding Style

- Standard Go conventions, standard library HTTP (no framework)
- Frontend: functional React components with hooks, Context API for auth
- No comments unless logic is non-obvious
- JSON responses with `Content-Type: application/json`
- Errors: JSON `{"error": "message"}` with appropriate HTTP status codes
