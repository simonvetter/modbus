package mbserver

type DummyHandler struct{}

func (h *DummyHandler) HandleCoils(req *CoilsRequest) ([]bool, error) {
	return nil, ErrIllegalFunction
}

func (h *DummyHandler) HandleDiscreteInputs(req *DiscreteInputsRequest) ([]bool, error) {
	return nil, ErrIllegalFunction
}

func (h *DummyHandler) HandleInputRegisters(req *InputRegistersRequest) ([]uint16, error) {
	return nil, ErrIllegalFunction
}

func (h *DummyHandler) HandleHoldingRegisters(req *HoldingRegistersRequest) ([]uint16, error) {
	return nil, ErrIllegalFunction
}
