# Start with a base image containing Go runtime
FROM golang:1.20 as builder
RUN apt-get update 
RUN apt-get install pkg-config libopus-dev libopusfile-dev -y
# Set the Current Working Directory inside the container
WORKDIR /app

COPY ./. .

RUN go build -o client cmd/main.go

# Start a new stage from scratch
FROM golang:1.20

RUN apt-get update 
RUN apt-get install pkg-config libopus-dev libopusfile-dev -y
# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/client ./client
# COPY models models

# Run the binary program produced by `go install`
CMD ["./client"]
