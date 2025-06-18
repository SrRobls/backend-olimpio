# Etapa 1: build
FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Fuerza build estático para Linux x86_64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o backend

# Etapa 2: imagen final
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/backend .

# Asegura permisos de ejecución
RUN chmod +x backend

EXPOSE 8080

CMD ["./backend"] 