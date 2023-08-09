# Start with a base image containing Go runtime
FROM golang:1.20 as builder
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    pkg-config \
    libopus-dev \
    libopusfile-dev \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY ./pkg pkg
COPY ./services/client services/client
COPY go.mod go.sum .

RUN go build -o main ./services/client

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
