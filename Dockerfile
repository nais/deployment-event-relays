FROM golang:1.12-alpine as builder
RUN apk add --no-cache git make
ENV GOOS=linux
ENV CGO_ENABLED=0
ENV GO111MODULE=on
COPY . /src
WORKDIR /src
RUN rm -f go.sum
RUN make test
RUN make alpine

FROM alpine:3.5
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /src/deploys2stdout /app/deploys2stdout
COPY --from=builder /src/deploys2influx /app/deploys2influx
COPY --from=builder /src/deploys2vera /app/deploys2vera
COPY --from=builder /src/deploys2nora /app/deploys2nora
