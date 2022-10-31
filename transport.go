package mbserver

type transport interface {
	Close() error
	ExecuteRequest(*pdu) (*pdu, error)
	ReadRequest() (*pdu, error)
	WriteResponse(*pdu) error
}
