FROM golang:1.27.0 AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o ./rtctl ./cmd

FROM scratch
COPY --from=builder /app/rtctl /app/rtctl
CMD ["/app/rtctl"]
