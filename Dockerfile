FROM golang:1.22 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o ride-hail-system .

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=build /app/ride-hail-system /app/ride-hail-system
EXPOSE 3000 3001 3004
CMD ["/app/ride-hail-system"]
