package modbus

import (
	"context"
	"fmt"
	"net"

	. "github.com/jakerobb/modbus-eth-controller/pkg"
)

func Connect(ctx context.Context, addr string) (net.Conn, error) {
	LogDebug(ctx, "Connecting to %s\n", addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	LogDebug(ctx, "Connected to %s\n", addr)
	return conn, nil
}

func GetStatus(ctx context.Context, addr string) (*CoilStates, error) {
	conn, err := Connect(ctx, addr)
	if err != nil {
		return nil, err
	}
	defer CloseQuietly(conn)

	coilCount, err := DiscoverRelayCount(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to discover relay count for %s: %w", addr, err)
	}

	msgData := NewReadCoils(0, coilCount)
	_, response, err := Send(ctx, conn, msgData)
	if err != nil {
		return nil, fmt.Errorf("failed to read relay states for %s: %w", addr, err)
	}
	relayStates := response.(*CoilStates)
	return relayStates, nil
}
