package modbus

import (
	"fmt"
	"errors"
)

type pdu struct {
	unitId		uint8
	functionCode	uint8
	payload		[]byte
}

const (
	// coils
	FC_READ_COILS			uint8	= 0x01
	FC_WRITE_SINGLE_COIL		uint8	= 0x05
	FC_WRITE_MULTIPLE_COILS		uint8	= 0x0f

	// discrete inputs
	FC_READ_DISCRETE_INPUTS		uint8	= 0x02

	// 16-bit input/holding registers
	FC_READ_HOLDING_REGISTERS	uint8	= 0x03
	FC_READ_INPUT_REGISTERS		uint8	= 0x04
	FC_WRITE_SINGLE_REGISTER	uint8	= 0x06
	FC_WRITE_MULTIPLE_REGISTERS	uint8	= 0x10
	FC_MASK_WRITE_REGISTER		uint8	= 0x16
	FC_READ_WRITE_MULTILE_REGISTERS	uint8	= 0x17
	FC_READ_FIFO_QUEUE		uint8	= 0x18

	// file access
	FC_READ_FILE_RECORD		uint8	= 0x14
	FC_WRITE_FILE_RECORD		uint8	= 0x15

	// exception codes
	EX_ILLEGAL_FUNCTION		uint8	= 0x01
	EX_ILLEGAL_DATA_ADDRESS		uint8	= 0x02
	EX_ILLEGAL_DATA_VALUE		uint8	= 0x03
	EX_SERVER_DEVICE_FAILURE	uint8	= 0x04
	EX_ACKNOWLEDGE			uint8	= 0x05
	EX_SERVER_DEVICE_BUSY		uint8	= 0x06
	EX_MEMORY_PARITY_ERROR		uint8	= 0x08
	EX_GW_PATH_UNAVAILABLE		uint8	= 0x0a
	EX_GW_TARGET_FAILED_TO_RESPOND	uint8	= 0x0b
)

var (
	ErrConfigurationError		error = errors.New("configuration error")
	ErrRequestTimedOut		error = errors.New("request timed out")
	ErrIllegalFunction		error = errors.New("illegal function")
	ErrIllegalDataAddress		error = errors.New("illegal data address")
	ErrIllegalDataValue		error = errors.New("illegal data value")
	ErrServerDeviceFailure		error = errors.New("server device failure")
	ErrAcknowledge			error = errors.New("request acknowledged")
	ErrServerDeviceBusy		error = errors.New("server device busy")
	ErrMemoryParityError		error = errors.New("memory parity error")
	ErrGWPathUnavailable		error = errors.New("gateway path unavailable")
	ErrGWTargetFailedToRespond	error = errors.New("gateway target device failed to respond")
	ErrBadCRC			error = errors.New("bad crc")
	ErrShortFrame			error = errors.New("short frame")
	ErrProtocolError		error = errors.New("protocol error")
	ErrBadUnitId			error = errors.New("bad unit id")
	ErrBadTransactionId		error = errors.New("bad transaction id")
	ErrUnknownProtocolId		error = errors.New("unknown protocol identifier")
	ErrUnexpectedParameters		error = errors.New("unexpected parameters")
)

func mapExceptionCodeToError(exceptionCode uint8) (err error) {
	switch exceptionCode {
	case EX_ILLEGAL_FUNCTION:		err = ErrIllegalFunction
	case EX_ILLEGAL_DATA_ADDRESS:		err = ErrIllegalDataAddress
	case EX_ILLEGAL_DATA_VALUE:		err = ErrIllegalDataValue
	case EX_SERVER_DEVICE_FAILURE:		err = ErrServerDeviceFailure
	case EX_ACKNOWLEDGE:			err = ErrAcknowledge
	case EX_MEMORY_PARITY_ERROR:		err = ErrMemoryParityError
	case EX_SERVER_DEVICE_BUSY:		err = ErrServerDeviceBusy
	case EX_GW_PATH_UNAVAILABLE:		err = ErrGWPathUnavailable
	case EX_GW_TARGET_FAILED_TO_RESPOND:	err = ErrGWTargetFailedToRespond
	default:
		err = fmt.Errorf("unsupported exception code (%v)", exceptionCode)
	}

	return
}
