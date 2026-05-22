FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -o /goproxy .

FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=build /goproxy /goproxy

EXPOSE 8090

ENTRYPOINT ["/goproxy"]
