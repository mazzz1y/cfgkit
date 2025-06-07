FROM golang:1.24-alpine3.22 AS build

ENV CGO_ENABLED=0
COPY . /src

RUN cd /src && \
  go build -ldflags="-s -w" -trimpath -o /cfgkit ./cmd/cfgkit

FROM alpine:3.22
COPY --from=build /cfgkit /cfgkit
USER 1337

ENTRYPOINT ["/cfgkit"]
