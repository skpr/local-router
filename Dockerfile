FROM golang:1.24 AS build

WORKDIR /go/src/github.com/skpr/local-router
COPY . /go/src/github.com/skpr/local-router

ENV CGO_ENABLED=0

RUN go build -o bin/local-router -ldflags='-extldflags "-static"' github.com/skpr/local-router

FROM alpine:3.20

COPY --from=build /go/src/github.com/skpr/local-router/bin/local-router /usr/local/bin/local-router

ENTRYPOINT ["/usr/local/bin/local-router"]
