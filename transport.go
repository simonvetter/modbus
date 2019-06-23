package modbus

type transport interface {
	Open()				(error)
	Close()				(error)
	WriteRequest(*request)		(error)
	ReadResponse()			(*response, error)
}
