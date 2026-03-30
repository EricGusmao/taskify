FROM docker.io/library/golang:1.26.1 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./cmd ./cmd
COPY ./internal ./internal
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/bin/server ./cmd/server/main.go

FROM gcr.io/distroless/static-debian13:nonroot
WORKDIR /
COPY --from=builder /app/bin/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
