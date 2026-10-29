FROM node:24-alpine AS web
WORKDIR /build/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS api
WORKDIR /build/server
COPY server/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /fieldnotes ./cmd/fieldnotes

FROM alpine:3.23
RUN addgroup -g 10001 app && adduser -D -u 10001 -G app app && mkdir -p /data && chown app:app /data
WORKDIR /app
COPY --from=api /fieldnotes /app/fieldnotes
COPY --from=web /build/web/dist /app/web
ENV ADDR=:8080 DATA_FILE=/data/issues.json WEB_DIR=/app/web
USER app
EXPOSE 8080
VOLUME ["/data"]
CMD ["/app/fieldnotes"]
