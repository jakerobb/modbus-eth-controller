package modbus

import (
	"errors"
	"fmt"
)

func ValidateEchoResponse(requestBytes []byte, response *Response) error {
	responseBytes := response.ToBytes()

	validationErrors := make([]error, 0)

	if len(requestBytes) != len(responseBytes) {
		validationErrors = append(validationErrors, fmt.Errorf("response length %d does not match request length %d", len(responseBytes), len(requestBytes)))
	} else {
		for i := 0; i < len(requestBytes); i++ {
			if requestBytes[i] != responseBytes[i] {
				validationErrors = append(validationErrors, fmt.Errorf("byte %d mismatch: command 0x%X, response 0x%X\n", i, requestBytes[i], responseBytes[i]))
			}
		}
	}

	if len(validationErrors) == 0 {
		return nil
	}
	return errors.Join(validationErrors...)
}
