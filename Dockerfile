FROM golang:1.27-alpine3.24 AS build

ENV CGO_ENABLED=0
COPY . /src

RUN cd /src && \
  go build -ldflags="-s -w" -trimpath -o /cfgkit ./cmd/cfgkit

FROM alpine:3.24
COPY --from=build /cfgkit /cfgkit
USER 1337

ENTRYPOINT ["/cfgkit"]
