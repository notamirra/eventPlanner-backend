# Build stage
FROM golang:1.23-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Make sure migrations are included in the build stage
COPY internal/database/migrations ./internal/database/migrations

RUN go build -o backend main.go

# Run stage
FROM alpine:latest

WORKDIR /app

# Copy backend binary
COPY --from=build /app/backend .

# Copy the database folder INCLUDING migrations
COPY --from=build /app/internal/database ./internal/database

EXPOSE 8080

CMD ["./backend"]
