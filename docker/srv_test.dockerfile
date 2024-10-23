# Stage 1: Build the Go binary
FROM golang:1.23 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY ../srv/go.mod ../srv/go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY ../srv/. .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o adit-srv

# Stage 2: Create the minimal final image
FROM alpine:latest

# Set the working directory in the minimal image
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/adit-srv .

# Set permissions and default command
RUN chmod +x ./adit-srv
CMD ["./adit-srv"]
