# The build stages run on the build machine's own platform and cross-compile,
# so ARM images build as fast as the x86-64 one.

# --- web app ---
FROM --platform=$BUILDPLATFORM node:26-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# --- backend ---
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS backend
ARG TARGETOS TARGETARCH TARGETVARIANT VERSION=dev
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GOARM=${TARGETVARIANT#v} \
    go build -trimpath -ldflags="-s -w -X github.com/selfhostit/plasto/app.Version=$VERSION" -o /plasto .

# --- runtime ---
FROM alpine:3.24 AS runtime
# Take Alpine's security fixes now rather than waiting for the base image to be rebuilt.
RUN apk upgrade --no-cache \
    && apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 1000 app \
    && mkdir -p /app/pb_data && chown app:app /app/pb_data
WORKDIR /app
COPY --from=backend /plasto /app/plasto
COPY --from=web /web/dist /app/pb_public
USER app
VOLUME /app/pb_data
EXPOSE 8090
HEALTHCHECK --interval=60s --timeout=5s CMD wget -qO- http://127.0.0.1:8090/api/health || exit 1
CMD ["/app/plasto", "serve", "--http=0.0.0.0:8090", "--dir=/app/pb_data", "--publicDir=/app/pb_public"]
