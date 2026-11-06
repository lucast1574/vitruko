FROM node:24-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/vitruko ./cmd/api

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && addgroup -S vitruko && adduser -S -G vitruko vitruko
COPY --from=build /out/vitruko /usr/local/bin/vitruko
USER vitruko
ENV PORT=4000
EXPOSE 4000
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s CMD wget -q -O /dev/null http://127.0.0.1:4000/health || exit 1
ENTRYPOINT ["/usr/local/bin/vitruko"]
