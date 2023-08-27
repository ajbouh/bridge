# Start with a base image containing Go runtime
FROM golang:1.20 as builder
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    pkg-config \
    libopus-dev \
    libopusfile-dev \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum .
RUN go mod download

COPY ./pkg pkg
COPY ./services/rtc-peer services/rtc-peer

RUN go build -o main ./services/rtc-peer

# Start a new stage from scratch
FROM golang:1.20
WORKDIR /app

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    pkg-config \
    libopus-dev \
    libopusfile-dev \
  && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/main /app/main
CMD ["/app/main"]
