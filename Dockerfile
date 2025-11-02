# Stage 1: Build the Go application
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum to leverage Docker cache for dependencies
COPY go.mod ./
COPY go.sum ./

# Download Go modules
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
# -ldflags="-s -w" reduces binary size by omitting symbol table and DWARF debugging info
# -o specifies the output binary name
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o uploader .

# Stage 2: Create the final lean image
FROM alpine:latest

# Set up a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Copy the compiled binary from the builder stage
COPY --from=builder /app/uploader /usr/local/bin/uploader

# Expose the port your application listens on (e.g., 8080)
EXPOSE 8080

# Define the command to run the application
ENTRYPOINT ["/usr/local/bin/uploader"]