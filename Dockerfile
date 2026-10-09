FROM golang:1.27.2@sha256:5bc7f572bbaa98885a3a1fd9c0aa76b59e3e14e8628bfc316bbfd0c701e4818c AS builder

WORKDIR /app

COPY src/go.mod .
COPY src/go.sum .
RUN go mod download

COPY src/ .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o modbus-eth-controller "./cmd"

FROM scratch

COPY --from=builder /app/modbus-eth-controller /modbus-eth-controller

COPY /sample-programs/*.json /etc/modbus/

VOLUME ["/etc/modbus"]

EXPOSE 8080

# Run the binary
ENTRYPOINT ["/modbus-eth-controller", "--server"]
