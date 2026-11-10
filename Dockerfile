# Build on the runner's own platform and cross-compile, rather than building
# each platform under QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.27.4@sha256:08bcb99a38f85a265f9a93d1d3323e2440eec333380f3ea26ee0dec9264bb41c AS builder

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
