# --- Build Stage ---
FROM golang:1.21.1 AS builder

# Set the current working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the entire directory to the container
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o mev-commit ./cmd/main.go

# --- Production Stage ---
FROM alpine:latest

# Copy the binary from the builder stage
COPY --from=builder /app/mev-commit /mev-commit

EXPOSE 13522 13523

CMD ["/mev-commit"]

