FROM golang:1.25 AS builder

WORKDIR /app

COPY *.go .
COPY *.mod .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o modbus-eth-controller

FROM scratch

COPY --from=builder /app/modbus-eth-controller /modbus-eth-controller

COPY /sample-programs/*.json /etc/modbus/

VOLUME ["/etc/modbus"]

EXPOSE 8008

# Run the binary
ENTRYPOINT ["/modbus-eth-controller", "--server"]
