package modbus

type transportType uint
const (
	RTU_TRANSPORT		transportType	= 1
	RTU_OVER_TCP_TRANSPORT	transportType	= 2
	TCP_TRANSPORT		transportType	= 3
)

type transport interface {
	Close()				(error)
	ExecuteRequest(*pdu)		(*pdu, error)
	ReadRequest()			(*pdu, error)
	WriteResponse(*pdu)		(error)
}
