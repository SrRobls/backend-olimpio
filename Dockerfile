# Etapa 1: build
FROM golang:1.23-alpine AS builder

# Instalar dependencias del sistema necesarias
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copiar archivos de dependencias primero para aprovechar el cache de Docker
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar el código fuente
COPY . .

# Build estático optimizado
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o backend

# Etapa 2: imagen final
FROM alpine:latest

# Instalar ca-certificates para HTTPS
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copiar el binario desde la etapa de build
COPY --from=builder /app/backend .

# Asegurar permisos de ejecución
RUN chmod +x backend

# Crear usuario no-root para seguridad
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Cambiar propiedad del archivo
RUN chown appuser:appgroup backend

# Cambiar al usuario no-root
USER appuser

EXPOSE 8080

CMD ["./backend"] 