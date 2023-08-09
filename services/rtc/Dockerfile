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
COPY ./services/rtc services/rtc
COPY go.mod go.sum .

RUN go build -o main ./services/rtc

# Start a new stage from scratch
FROM golang:1.20
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    pkg-config \
    libopus-dev \
    libopusfile-dev \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/main ./main
COPY ./services/rtc/config.toml ./config.toml
COPY ./services/rtc/web/. ./web/.
EXPOSE 8088
CMD ["/app/main"]
