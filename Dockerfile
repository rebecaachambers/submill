# ============================================================
# SubMill Dockerfile - Single Container (SubMill + Mihomo)
# ============================================================
# Build:  docker build -t submill .
# Run:    docker run -d --name submill --restart=always \
#           -p 7890:7890 -p 8199:8199 \
#           -v submill-config:/app/config \
#           -v submill-output:/app/output \
#           submill
# ============================================================

# --- Stage 1: Build -------------------------------------------------
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Build SubMill
COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY app/ app/
COPY check/ check/
COPY config/ config/
COPY proxy/ proxy/
COPY save/ save/
COPY utils/ utils/
COPY assets/ assets/
COPY main.go init.go ./

RUN CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags="-s -w" -o /out/submill .

# Build Mihomo
COPY mihomo/go.mod mihomo/go.sum mihomo/
COPY mihomo/vendor/ mihomo/vendor/

RUN cd mihomo && CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags="-s -w" -o /out/mihomo github.com/metacubex/mihomo

# --- Stage 2: Runtime -----------------------------------------------
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /out/submill /app/submill
COPY --from=builder /out/mihomo /app/mihomo

# Config and output directories (mounted as volumes at runtime)
RUN mkdir -p /app/config /app/output

# Config template for first-run generation
COPY --from=builder /src/config/config.example.yaml /app/config/config.example.yaml

# Entrypoint
COPY docker-entrypoint.sh /app/
RUN chmod +x /app/docker-entrypoint.sh

EXPOSE 7890 8199

ENTRYPOINT ["/app/docker-entrypoint.sh"]