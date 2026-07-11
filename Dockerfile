# Flagship — one image: the dashboard SPA is built, then baked into the Go binary (-tags dashboard).
# Final image is distroless nonroot (Kyverno-compliant: non-root, no shell).

# 1) Build the dashboard (Vite → web/dist).
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 2) Build the static Go binary, embedding the dashboard.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=web /web/dist ./web/dist
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -tags dashboard -ldflags="-s -w" -o /out/flagship ./cmd/flagship

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/flagship /flagship
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/flagship"]
