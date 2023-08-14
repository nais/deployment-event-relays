FROM golang:1.20 as builder
ENV GOOS=linux
ENV CGO_ENABLED=0
COPY . /src
WORKDIR /src
RUN make test
RUN make static

FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /app
COPY --from=builder /src/bin/deployment-event-relays /app/deployment-event-relays
CMD ["/app/deployment-event-relays"]
