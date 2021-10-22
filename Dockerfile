FROM golang:1.17 AS build_base

WORKDIR /tmp/fan

COPY . .

RUN GOOS=linux GOARCH=arm GOARM=7 go build -o ./out/fan .

FROM alpine:3.13

RUN apk add ca-certificates

COPY --from=build_base /tmp/fan/out/fan /app/fan

ENTRYPOINT ["/app/fan"]
