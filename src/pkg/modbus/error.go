package modbus

import "fmt"

type ModbusError struct {
	Function byte
	Code     byte
}

func (e *ModbusError) Error() string {
	return fmt.Sprintf("%s, function=0x%02X, code=0x%02X", exceptionMessage(e.Code), e.Function, e.Code)
}

func exceptionMessage(code byte) string {
	switch code {
	case 0x01:
		return "Illegal Function (unsupported operation for this device)"
	case 0x02:
		return "Illegal Data Address (invalid relay number)"
	case 0x03:
		return "Illegal Data Value (invalid command)"
	case 0x04:
		return "Slave Device Failure (device error, try again or reboot)"
	case 0x05:
		return "Acknowledge (command accepted, still processing)"
	case 0x06:
		return "Slave Device Busy (try again shortly)"
	case 0x08:
		return "Memory Parity Error (internal memory/firmware issue)"
	case 0x0A:
		return "Gateway Path Unavailable (network route failed)"
	case 0x0B:
		return "Gateway Target Device Failed to Respond"
	default:
		return fmt.Sprintf("Unknown error code 0x%02X", code)
	}
}
