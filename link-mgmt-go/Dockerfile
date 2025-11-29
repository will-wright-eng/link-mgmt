# Development Dockerfile with hot reloading using Air
FROM golang:1.25-alpine

# Install Air for hot reloading and add to PATH
RUN go install github.com/air-verse/air@latest
ENV PATH="${PATH}:/root/go/bin"

# Install git (needed for some Go modules) and wget (for healthcheck)
RUN apk add --no-cache git wget

# Set working directory
WORKDIR /app

# Expose port
EXPOSE 8080

# Run Air for hot reloading
CMD ["air", "-c", ".air.toml"]
