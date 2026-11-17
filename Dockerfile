# Build on the runner's own platform and cross-compile, rather than building
# each platform under QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.27.4@sha256:205c7636d74e0ffbe3d007162df7d74f859f5a8c8322dfb9f907977686c126cf AS builder

ARG TARGETOS
ARG TARGETARCH

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
