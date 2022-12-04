FROM golang:1.19-alpine AS builder

WORKDIR /build

RUN apk update && apk add --no-cache make

COPY ./ /build
RUN go mod download

RUN make bin/cohab-server

# Create a new release build stage
FROM alpine:3.17

WORKDIR /

COPY --from=builder /build/bin/cohab-server /cohab-server

EXPOSE 8080

ENTRYPOINT ["/cohab-server"]
