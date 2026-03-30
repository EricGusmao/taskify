FROM docker.io/library/golang:1.26.1 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./api ./api
COPY ./cmd ./cmd
COPY ./internal ./internal
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/bin/server ./cmd/server/main.go
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/bin/migrate ./cmd/migrate/main.go
RUN mkdir -p /app/uploads && chown 65532:65532 /app/uploads

FROM gcr.io/distroless/static-debian13:nonroot
WORKDIR /
COPY --from=builder /app/bin/server /server
COPY --from=builder /app/bin/migrate /migrate
COPY --from=builder --chown=nonroot:nonroot /app/uploads /uploads
EXPOSE 8080
ENTRYPOINT ["/server"]
