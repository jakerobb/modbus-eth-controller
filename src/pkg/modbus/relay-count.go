package modbus

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/jakerobb/modbus-eth-controller/pkg/util"
)

var RelayCountCache = make(map[string]uint16)

func GetRelayCount(ctx context.Context, addr string, conn net.Conn) (uint16, error) {
	if count, ok := RelayCountCache[addr]; ok {
		return count, nil
	} else {
		count, err := DiscoverRelayCount(ctx, addr, conn)
		if err != nil {
			return 0, err
		}
		RelayCountCache[addr] = count
		return count, nil
	}
}

// DiscoverRelayCount attempts to discover the number of relays (coils) on a Modbus device
// by performing a binary search over the valid coil addresses (0 to 65535).
// It sends Read Coils requests and checks for Illegal Data Address errors to determine
// the highest valid coil address, thus inferring the total number of relays.
func DiscoverRelayCount(ctx context.Context, addr string, conn net.Conn) (uint16, error) {
	low := uint16(0)
	high := uint16(0xFFFF)

	util.LogDebug(ctx, "Starting relay count discovery", "address", addr)
	pass := 0
	for low <= high {
		mid := (low + high) / 2

		pass++
		util.LogDebug(ctx, "Checking presence of relay", "address", addr, "relayIndex", mid)
		req := NewReadCoils(mid, 1)

		if _, _, err := Send(ctx, conn, req); err == nil {
			// mid is valid, so continue the search in the upper half
			low = mid + 1
		} else {
			// Check if it's an Illegal Data Address error
			if IsIllegalDataAddress(err) {
				// mid was not valid, so continue the search in the lower half
				high = mid - 1
			} else {
				return 0, fmt.Errorf("error during relay count discovery at address %d (pass %d): %w", mid, pass, err)
			}
		}
	}

	util.LogDebug(ctx, "Discovered relay count", "actualCount", high+1, "requestsMade", pass)
	return high + 1, nil
}

func IsIllegalDataAddress(err error) bool {
	var me *ModbusError
	return errors.As(err, &me) && me.Code == 0x02
}
