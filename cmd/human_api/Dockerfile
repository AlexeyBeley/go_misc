FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy the Go module files to leverage Docker's cache
COPY go.mod go.sum ./

# Download all project dependencies
RUN go mod download

# Copy the rest of your application's source code
COPY . .

# Build the Go application into a single, static binary
# The -o flag names the output binary.
# The CGO_ENABLED=0 flag creates a statically linked binary without external dependencies.

RUN cd /app/cmd/human_api && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o my-app .

FROM alpine:latest

# Set the working directory for the final image
WORKDIR /root/

RUN mkdir -p /opt/human_api
COPY --from=builder /app/cmd/human_api/my-app .
COPY --from=builder /app/cmd/human_api/slack_block_kit /opt/human_api/slack_block_kit


# Expose the port that your Go application listens on
EXPOSE 8080

# The command to run your Go application
CMD ["./my-app"]