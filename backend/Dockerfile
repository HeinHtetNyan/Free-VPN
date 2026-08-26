# Builds the control plane into a small, deployable image. Note: this
# containerizes the API only — the actual WireGuard interface (see
# ../infra/scripts/setup-central-server.sh) must exist on the HOST, since
# wgctrl (internal/servers/wireguard_manager.go) talks to it via netlink.
# If the backend itself runs inside this container on the central server,
# run it with --network host and mount /var/run/wireguard (or run the
# binary directly on the host instead — either works, see ../docs/INFRA.md).
FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /vpnapp-server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=build /vpnapp-server /vpnapp-server
EXPOSE 8080
ENTRYPOINT ["/vpnapp-server"]
