# build frontend
FROM node:18-bullseye AS webbuild
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web .
RUN npm run build

# build backend
FROM golang:1.25-bookworm AS gobuild
WORKDIR /app
COPY server/go.mod server/go.sum ./server/
RUN cd server && go mod download
COPY server ./server
WORKDIR /app/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/wechat2rss ./cmd/server

# final runtime image
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=gobuild /app/wechat2rss ./wechat2rss
COPY --from=webbuild /app/web/dist ./static
ENV APP_PORT=8080 \
    WEB_STATIC_DIR=/app/static
EXPOSE 8080
CMD ["./wechat2rss"]
