FROM golang:1.16-alpine as builder
RUN apk add --no-cache git make
ENV GOOS=linux
ENV CGO_ENABLED=0
ENV GO111MODULE=on
COPY . /src
WORKDIR /src
RUN make test
RUN make alpine

FROM alpine:3.13
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /src/bin/deployment-event-relays /app/deployment-event-relays
CMD ["/app/deployment-event-relays"]
