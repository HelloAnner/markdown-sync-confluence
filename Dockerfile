FROM golang:1.21-alpine

WORKDIR /app

# Install git and build dependencies
RUN apk add --no-cache git build-base

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the application
RUN cd cmd/web && go build -o main

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
CMD ["./cmd/web/main"] 