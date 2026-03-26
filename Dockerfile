# Build gogents server
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . ./
RUN go build -o gogents ./cmd/gogents

# Run gogents in server mode
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /app/gogents .
EXPOSE 8080
ENTRYPOINT ["./gogents", "--serve"]
