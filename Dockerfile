# Build on the runner's own platform and cross-compile, rather than building
# each platform under QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.27.3@sha256:898838c455eecb68ed7b0b6f0635630ebdd657e732f07a23d0ab475007a52c36 AS builder

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
