FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN false
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app .

FROM alpine:3
COPY --from=build /app /app
USER 10001
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app"]
