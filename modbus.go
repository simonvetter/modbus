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
	fcReadCoils                  uint8 = 0x01
	fcWriteSingleCoil            uint8 = 0x05
	fcWriteMultipleCoils         uint8 = 0x0f

	// discrete inputs
	fcReadDiscreteInputs         uint8 = 0x02

	// 16-bit input/holding registers
	fcReadHoldingRegisters       uint8 = 0x03
	fcReadInputRegisters         uint8 = 0x04
	fcWriteSingleRegister        uint8 = 0x06
	fcWriteMultipleRegisters     uint8 = 0x10
	fcMaskWriteRegister          uint8 = 0x16
	fcReadWriteMultipleRegisters uint8 = 0x17
	fcReadFifoQueue              uint8 = 0x18

	// file access
	fcReadFileRecord             uint8 = 0x14
	fcWriteFileRecord            uint8 = 0x15

	// exception codes
	exIllegalFunction            uint8 = 0x01
	exIllegalDataAddress         uint8 = 0x02
	exIllegalDataValue           uint8 = 0x03
	exServerDeviceFailure        uint8 = 0x04
	exAcknowledge                uint8 = 0x05
	exServerDeviceBusy           uint8 = 0x06
	exMemoryParityError          uint8 = 0x08
	exGWPathUnavailable          uint8 = 0x0a
	exGWTargetFailedToRespond    uint8 = 0x0b

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
	case exIllegalFunction:         err = ErrIllegalFunction
	case exIllegalDataAddress:      err = ErrIllegalDataAddress
	case exIllegalDataValue:        err = ErrIllegalDataValue
	case exServerDeviceFailure:     err = ErrServerDeviceFailure
	case exAcknowledge:             err = ErrAcknowledge
	case exMemoryParityError:       err = ErrMemoryParityError
	case exServerDeviceBusy:        err = ErrServerDeviceBusy
	case exGWPathUnavailable:       err = ErrGWPathUnavailable
	case exGWTargetFailedToRespond: err = ErrGWTargetFailedToRespond
	default:
		err = fmt.Errorf("unknown exception code (%v)", exceptionCode)
	}

	return
}

// mapErrorToExceptionCode turns an Error object into a modbus exception code.
func mapErrorToExceptionCode(err error) (exceptionCode uint8) {
	switch err {
	case ErrIllegalFunction:         exceptionCode = exIllegalFunction
	case ErrIllegalDataAddress:      exceptionCode = exIllegalDataAddress
	case ErrIllegalDataValue:        exceptionCode = exIllegalDataValue
	case ErrServerDeviceFailure:     exceptionCode = exServerDeviceFailure
	case ErrAcknowledge:             exceptionCode = exAcknowledge
	case ErrMemoryParityError:       exceptionCode = exMemoryParityError
	case ErrServerDeviceBusy:        exceptionCode = exServerDeviceBusy
	case ErrGWPathUnavailable:       exceptionCode = exGWPathUnavailable
	case ErrGWTargetFailedToRespond: exceptionCode = exGWTargetFailedToRespond
	default:
		exceptionCode = exServerDeviceFailure
	}

	return
}
