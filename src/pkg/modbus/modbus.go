package modbus

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jakerobb/modbus-eth-controller/pkg/util"
)

func Connect(ctx context.Context, addr string) (net.Conn, error) {
	util.LogDebug(ctx, "Connecting to %s\n", addr)
	conn, err := net.DialTimeout("tcp", addr, time.Second*5)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	util.LogDebug(ctx, "Connected to %s\n", addr)
	return conn, nil
}

func GetStatus(ctx context.Context, addr string) (*CoilStates, error) {
	conn, err := Connect(ctx, addr)
	if err != nil {
		return nil, err
	}
	defer util.CloseQuietly(conn)

	coilCount, err := GetRelayCount(ctx, addr, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to get relay count for %s: %w", addr, err)
	}

	msgData := NewReadCoils(0, coilCount)
	_, response, err := Send(ctx, conn, msgData)
	if err != nil {
		return nil, fmt.Errorf("failed to read relay states for %s: %w", addr, err)
	}
	relayStates := response.(*CoilStates)
	return relayStates, nil
}
