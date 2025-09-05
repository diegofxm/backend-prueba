# Dockerfile simple para backend SECOP blockchain
FROM golang:1.21

# Crear directorio de trabajo
WORKDIR /app

# Copiar archivos Go
COPY go.mod go.sum ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Descargar dependencias
RUN go mod download

# Compilar aplicación
RUN go build -o main ./cmd/server

# Exponer puerto
EXPOSE 8080

# Ejecutar aplicación
CMD ["./main"]
