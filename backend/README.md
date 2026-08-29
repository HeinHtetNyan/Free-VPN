# backend

Go control plane. See [`../docs/BACKEND.md`](../docs/BACKEND.md) for responsibilities, planned structure, and open decisions (DB choice, auth strategy) before adding real logic here.

Deployed to production at `https://sy-api.malmah.fyi` (Shared VPS, `/home/appbox/SY/backend/`). Pushes to `main` touching this directory auto-deploy via `.github/workflows/deploy.yml`.

## Run

Requires Go 1.22+. (This dev environment didn't have Go installed system-wide — it was installed to `~/go-sdk` and added to `PATH` via `~/.bashrc`/`~/.profile`; open a new shell or `export PATH=$HOME/go-sdk/bin:$PATH` if `go` isn't found.)

```
cd backend
PORT=8090 go run ./cmd/server   # default port 8080 may already be in use on your machine
curl localhost:8090/health
```

Or via Docker (verified working — built and hit the real endpoints in this container; note it containerizes the API only, not a WireGuard interface — see `Dockerfile`'s header comment and `../docs/INFRA.md`):
```
docker build -t sy-vpn-backend backend/
docker run -d --rm -p 8090:8080 sy-vpn-backend
curl localhost:8090/health
```

Full flow, either way of running it:
```
# register (anonymous, device-bound — see ../docs/BACKEND.md)
curl -X POST localhost:8090/auth/register -d '{"device_id":"some-locally-generated-id"}'
# -> {"token":"...","tier":"free"}

curl localhost:8090/locations -H "Authorization: Bearer <token>"
curl -X POST localhost:8090/connect -H "Authorization: Bearer <token>" -d '{"location_id":"singapore"}'
```

## Tests

```
go test ./...
```

## Current state

Auth, locations, WireGuard key generation, SQLite persistence, and best-effort live peer registration are all implemented and tested — see `../docs/BACKEND.md` "Persistence and WireGuard integration." What's still a placeholder: the actual central WireGuard server (`WG_INTERFACE`/`DB_PATH` are configurable but nothing real is deployed at `../infra/` yet) and the LocalToNet relay addresses in `internal/servers/locations.json`.
