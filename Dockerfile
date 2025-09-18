# Imagen base para Go >= 1.24.2
FROM golang:1.24.4-alpine AS builder

WORKDIR /app

# Copiar dependencias primero
COPY go.mod go.sum ./
RUN go mod download

# Copiar el resto del código
COPY . .

# Compilar el binario
RUN go build -o formula-sim main.go

# ---- Imagen final ----
FROM alpine:3.20

WORKDIR /app

# Copiar binario
COPY --from=builder /app/formula-sim .

# Puerto
EXPOSE 8080

# Ejecutar el binario
CMD ["./formula-sim"]
