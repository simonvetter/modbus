package modbus

import (
	"fmt"
)

type pdu struct {
	unitId		uint8
	functionCode	uint8
	payload		[]byte
}

type Error string

// Error implements the error interface.
func (me Error) Error() (s string) {
	s = string(me)
	return
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

	// errors
	ErrConfigurationError        Error = "configuration error"
	ErrRequestTimedOut           Error = "request timed out"
	ErrIllegalFunction           Error = "illegal function"
	ErrIllegalDataAddress        Error = "illegal data address"
	ErrIllegalDataValue          Error = "illegal data value"
	ErrServerDeviceFailure       Error = "server device failure"
	ErrAcknowledge               Error = "request acknowledged"
	ErrServerDeviceBusy          Error = "server device busy"
	ErrMemoryParityError         Error = "memory parity error"
	ErrGWPathUnavailable         Error = "gateway path unavailable"
	ErrGWTargetFailedToRespond   Error = "gateway target device failed to respond"
	ErrBadCRC                    Error = "bad crc"
	ErrShortFrame                Error = "short frame"
	ErrProtocolError             Error = "protocol error"
	ErrBadUnitId                 Error = "bad unit id"
	ErrBadTransactionId          Error = "bad transaction id"
	ErrUnknownProtocolId         Error = "unknown protocol identifier"
	ErrUnexpectedParameters      Error = "unexpected parameters"
)

// mapExceptionCodeToError turns a modbus exception code into a higher level Error object.
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

// mapErrorToExceptionCode turns an Error object into a modbus exception code.
func mapErrorToExceptionCode(err error) (exceptionCode uint8) {
	switch err {
	case ErrIllegalFunction:	exceptionCode = EX_ILLEGAL_FUNCTION
	case ErrIllegalDataAddress:	exceptionCode = EX_ILLEGAL_DATA_ADDRESS
	case ErrIllegalDataValue:	exceptionCode = EX_ILLEGAL_DATA_VALUE
	case ErrServerDeviceFailure:    exceptionCode = EX_SERVER_DEVICE_FAILURE
	case ErrAcknowledge:		exceptionCode = EX_ACKNOWLEDGE
	case ErrMemoryParityError:      exceptionCode = EX_MEMORY_PARITY_ERROR
	case ErrServerDeviceBusy:	exceptionCode = EX_SERVER_DEVICE_BUSY
	case ErrGWPathUnavailable:	exceptionCode = EX_GW_PATH_UNAVAILABLE
	case ErrGWTargetFailedToRespond:
		exceptionCode = EX_GW_TARGET_FAILED_TO_RESPOND
	default:
		exceptionCode = EX_SERVER_DEVICE_FAILURE
	}

	return
}
