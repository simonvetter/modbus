package modbus

type transportType uint
const (
	modbusRTU        transportType   = 1
	modbusRTUOverTCP transportType   = 2
	modbusRTUOverUDP transportType   = 3
	modbusTCP        transportType   = 4
	modbusTCPOverTLS transportType   = 5
	modbusTCPOverUDP transportType   = 6
)

type transport interface {
	Close()              (error)
	ExecuteRequest(*pdu) (*pdu, error)
	ReadRequest()        (*pdu, error)
	WriteResponse(*pdu)  (error)
}
