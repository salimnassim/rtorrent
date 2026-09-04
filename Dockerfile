FROM golang:1.27.0 AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o ./rtctl ./cmd

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=builder /app/rtctl /app/rtctl
CMD ["/app/rtctl"]
