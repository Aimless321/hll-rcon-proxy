FROM golang:1.20-alpine AS build_base

RUN apk add --no-cache git build-base

# Set the Current Working Directory inside the container
WORKDIR /build

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Build the Go app
RUN go build -o ./out/hll-stats-backend .

# Start fresh from a smaller image
FROM alpine:3.9
RUN apk add ca-certificates

COPY --from=build_base /build/out/hll-stats-backend /app/hll-stats-backend

WORKDIR /app

# Run the binary program produced by `go install`
CMD ["/app/hll-stats-backend"]