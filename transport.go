package modbus

type transport interface {
	Open()				(error)
	Close()				(error)
	ExecuteRequest(*pdu)		(*pdu, error)
	ReadRequest()			(*pdu, error)
	WriteResponse(*pdu)		(error)
}
