package modbus

type transportType uint
const (
	modbusRTU        transportType   = 1
	modbusRTUOverTCP transportType   = 2
	modbusTCP        transportType   = 3
)

type transport interface {
	Close()              (error)
	ExecuteRequest(*pdu) (*pdu, error)
	ReadRequest()        (*pdu, error)
	WriteResponse(*pdu)  (error)
}
