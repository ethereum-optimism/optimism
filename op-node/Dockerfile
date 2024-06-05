# Use the official Golang image as the base image
FROM golang:1.21-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install git and other dependencies
RUN apk update && apk add --no-cache git make

# Copy the rest of the application source code
COPY . .

# Build the op-node binary
RUN cd op-node && make op-node

# Use a minimal image for the runtime environment
FROM golang:1.21-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/op-node/bin/op-node /usr/local/bin/op-node

# Expose the necessary port (change as needed)
EXPOSE 8080

# Set the entrypoint to the built binary
ENTRYPOINT ["op-node"]
