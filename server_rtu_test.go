package modbus

import (
	"testing"
)

func TestCRCReturnNilIfMessageIsValid(t *testing.T) {
	message := []byte{0x11, 0x01, 0x00, 0x13, 0x00, 0x25, 0x0E, 0x84}

	var result = messageIsValid(message)
	if result != nil {
		t.Errorf("Expecting nil, sice CRC is correct!")
	}
}

func TestCRCReturnErrorIfMessageIsInvalid(t *testing.T) {
	message := []byte{0x21, 0x01, 0x00, 0x13, 0x00, 0x25, 0x0E, 0x84}

	var result = messageIsValid(message)
	if result == nil {
		t.Errorf("Expecting error, sice CRC is correct!")
	}
}

func TestReadMultipleCoilRegister(t *testing.T) {
	message := []byte{0x11, 0x01, 0x00, 0x13, 0x00, 0x25, 0x0E, 0x84}

	raw, err := createRequestFromBytes(message)
	if err != nil {
		t.Errorf("Request should not be an error!")
	}

	response := raw.(CoilsRequest)
	var expected = CoilsRequest{
		ClientAddr: "127.0.0.1",
		ClientRole: "",
		UnitId:     0x11,
		Addr:       0x13,
		Quantity:   37,
		IsWrite:    false,
		Args:       nil,
	}

	if response.ClientAddr != expected.ClientAddr {
		t.Errorf("ClientAddr incorrect! Expected %v, Got %v", expected.ClientAddr, response.ClientAddr)
	}
	if response.ClientRole != expected.ClientRole {
		t.Errorf("ClientRole incorrect! Expected %v, Got %v", expected.ClientRole, response.ClientRole)
	}
	if response.UnitId != expected.UnitId {
		t.Errorf("UnitId incorrect! Expected %v, Got %v", expected.UnitId, response.UnitId)
	}
	if response.Addr != expected.Addr {
		t.Errorf("Addr incorrect! Expected %v, Got %v", expected.Addr, response.Addr)
	}
	if response.Quantity != expected.Quantity {
		t.Errorf("Quantity incorrect! Expected %v, Got %v", expected.Quantity, response.Quantity)
	}
	if response.IsWrite != expected.IsWrite {
		t.Errorf("IsWrite incorrect! Expected %v, Got %v", expected.IsWrite, response.IsWrite)
	}
	if response.Args != nil {
		t.Errorf("Args incorrect! Expected %v, Got %v", expected.Args, response.Args)
	}
}

func TestReadMultipleDiscreteInputRegister(t *testing.T) {
	message := []byte{0x11, 0x02, 0x00, 0xC4, 0x00, 0x16, 0xBA, 0xA9}

	raw, err := createRequestFromBytes(message)
	if err != nil {
		t.Errorf("Request should not be an error!")
	}

	response := raw.(DiscreteInputsRequest)
	var expected = DiscreteInputsRequest{
		ClientAddr: "127.0.0.1",
		ClientRole: "",
		UnitId:     0x11,
		Addr:       0xC4,
		Quantity:   22,
	}

	if response.ClientAddr != expected.ClientAddr {
		t.Errorf("ClientAddr incorrect! Expected %v, Got %v", expected.ClientAddr, response.ClientAddr)
	}
	if response.ClientRole != expected.ClientRole {
		t.Errorf("ClientRole incorrect! Expected %v, Got %v", expected.ClientRole, response.ClientRole)
	}
	if response.UnitId != expected.UnitId {
		t.Errorf("UnitId incorrect! Expected %v, Got %v", expected.UnitId, response.UnitId)
	}
	if response.Addr != expected.Addr {
		t.Errorf("Addr incorrect! Expected %v, Got %v", expected.Addr, response.Addr)
	}
	if response.Quantity != expected.Quantity {
		t.Errorf("Quantity incorrect! Expected %v, Got %v", expected.Quantity, response.Quantity)
	}
}

func TestReadMultipleHoldingRegister(t *testing.T) {
	message := []byte{0x11, 0x03, 0x00, 0x6B, 0x00, 0x03, 0x76, 0x87}

	raw, err := createRequestFromBytes(message)
	if err != nil {
		t.Errorf("Request should not be an error!")
	}

	response := raw.(HoldingRegistersRequest)
	var expected = HoldingRegistersRequest{
		ClientAddr: "127.0.0.1",
		ClientRole: "",
		UnitId:     0x11,
		Addr:       0x6B,
		Quantity:   3,
		IsWrite:    false,
		Args:       nil,
	}

	if response.ClientAddr != expected.ClientAddr {
		t.Errorf("ClientAddr incorrect! Expected %v, Got %v", expected.ClientAddr, response.ClientAddr)
	}
	if response.ClientRole != expected.ClientRole {
		t.Errorf("ClientRole incorrect! Expected %v, Got %v", expected.ClientRole, response.ClientRole)
	}
	if response.UnitId != expected.UnitId {
		t.Errorf("UnitId incorrect! Expected %v, Got %v", expected.UnitId, response.UnitId)
	}
	if response.Addr != expected.Addr {
		t.Errorf("Addr incorrect! Expected %v, Got %v", expected.Addr, response.Addr)
	}
	if response.Quantity != expected.Quantity {
		t.Errorf("Quantity incorrect! Expected %v, Got %v", expected.Quantity, response.Quantity)
	}
	if response.IsWrite != expected.IsWrite {
		t.Errorf("IsWrite incorrect! Expected %v, Got %v", expected.IsWrite, response.IsWrite)
	}
	if response.Args != nil {
		t.Errorf("Args incorrect! Expected %v, Got %v", expected.Args, response.Args)
	}
}

func TestReadMultipleInputRegister(t *testing.T) {
	message := []byte{0x11, 0x04, 0x00, 0x08, 0x00, 0x01, 0xB2, 0x98}

	raw, err := createRequestFromBytes(message)
	if err != nil {
		t.Errorf("Request should not be an error!")
	}

	response := raw.(InputRegistersRequest)
	var expected = InputRegistersRequest{
		ClientAddr: "127.0.0.1",
		ClientRole: "",
		UnitId:     0x11,
		Addr:       0x08,
		Quantity:   1,
	}

	if response.ClientAddr != expected.ClientAddr {
		t.Errorf("ClientAddr incorrect! Expected %v, Got %v", expected.ClientAddr, response.ClientAddr)
	}
	if response.ClientRole != expected.ClientRole {
		t.Errorf("ClientRole incorrect! Expected %v, Got %v", expected.ClientRole, response.ClientRole)
	}
	if response.UnitId != expected.UnitId {
		t.Errorf("UnitId incorrect! Expected %v, Got %v", expected.UnitId, response.UnitId)
	}
	if response.Addr != expected.Addr {
		t.Errorf("Addr incorrect! Expected %v, Got %v", expected.Addr, response.Addr)
	}
	if response.Quantity != expected.Quantity {
		t.Errorf("Quantity incorrect! Expected %v, Got %v", expected.Quantity, response.Quantity)
	}
}

func TestWriteSingleCoilRegister(t *testing.T) {
	message := []byte{0x11, 0x05, 0x00, 0xAC, 0xFF, 0x00, 0x4E, 0x8B}

	raw, err := createRequestFromBytes(message)
	if err != nil {
		t.Errorf("Request should not be an error!")
	}

	response := raw.(CoilsRequest)
	var expected = CoilsRequest{
		ClientAddr: "127.0.0.1",
		ClientRole: "",
		UnitId:     0x11,
		Addr:       0xAC,
		Quantity:   1,
		IsWrite:    true,
		Args:       []bool{true},
	}

	if response.ClientAddr != expected.ClientAddr {
		t.Errorf("ClientAddr incorrect! Expected %v, Got %v", expected.ClientAddr, response.ClientAddr)
	}
	if response.ClientRole != expected.ClientRole {
		t.Errorf("ClientRole incorrect! Expected %v, Got %v", expected.ClientRole, response.ClientRole)
	}
	if response.UnitId != expected.UnitId {
		t.Errorf("UnitId incorrect! Expected %v, Got %v", expected.UnitId, response.UnitId)
	}
	if response.Addr != expected.Addr {
		t.Errorf("Addr incorrect! Expected %v, Got %v", expected.Addr, response.Addr)
	}
	if response.Quantity != expected.Quantity {
		t.Errorf("Quantity incorrect! Expected %v, Got %v", expected.Quantity, response.Quantity)
	}
	if response.IsWrite != expected.IsWrite {
		t.Errorf("IsWrite incorrect! Expected %v, Got %v", expected.IsWrite, response.IsWrite)
	}
	for i := range response.Args {
		if response.Args[i] != expected.Args[i] {
			t.Errorf("Args incorrect! Expected %v, Got %v", expected.Args[i], response.Args[i])
		}
	}
}

func TestWriteSingleHoldingRegister(t *testing.T) {
	message := []byte{0x11, 0x06, 0x00, 0x01, 0x00, 0x03, 0x9A, 0x9B}

	raw, err := createRequestFromBytes(message)
	if err != nil {
		t.Errorf("Request should not be an error!")
	}

	response := raw.(HoldingRegistersRequest)
	var expected = HoldingRegistersRequest{
		ClientAddr: "127.0.0.1",
		ClientRole: "",
		UnitId:     0x11,
		Addr:       0x01,
		Quantity:   1,
		IsWrite:    true,
		Args:       []uint16{0x03},
	}

	if response.ClientAddr != expected.ClientAddr {
		t.Errorf("ClientAddr incorrect! Expected %v, Got %v", expected.ClientAddr, response.ClientAddr)
	}
	if response.ClientRole != expected.ClientRole {
		t.Errorf("ClientRole incorrect! Expected %v, Got %v", expected.ClientRole, response.ClientRole)
	}
	if response.UnitId != expected.UnitId {
		t.Errorf("UnitId incorrect! Expected %v, Got %v", expected.UnitId, response.UnitId)
	}
	if response.Addr != expected.Addr {
		t.Errorf("Addr incorrect! Expected %v, Got %v", expected.Addr, response.Addr)
	}
	if response.Quantity != expected.Quantity {
		t.Errorf("Quantity incorrect! Expected %v, Got %v", expected.Quantity, response.Quantity)
	}
	if response.IsWrite != expected.IsWrite {
		t.Errorf("IsWrite incorrect! Expected %v, Got %v", expected.IsWrite, response.IsWrite)
	}
	for i := range response.Args {
		if response.Args[i] != expected.Args[i] {
			t.Errorf("Args incorrect! Expected %v, Got %v", expected.Args[i], response.Args[i])
		}
	}
}

func TestWriteMultipleCoilRegister(t *testing.T) {
	message := []byte{0x11, 0x0F, 0x00, 0x13, 0x00, 0x0A, 0x02, 0xCD, 0x01, 0xBF, 0x0D}

	raw, err := createRequestFromBytes(message)
	if err != nil {
		t.Errorf("Request should not be an error!")
	}

	response := raw.(CoilsRequest)
	var expected = CoilsRequest{
		ClientAddr: "127.0.0.1",
		ClientRole: "",
		UnitId:     0x11,
		Addr:       0x13,
		Quantity:   10,
		IsWrite:    true,
		// 0xCD	Byte-Wert DO 27-20 (1100 1101)
		// 0x01	Byte-Wert DO 29-28 (0000 0001)
		Args: []bool{true, false, true, true, false, false, true, true, true, false},
	}

	if response.ClientAddr != expected.ClientAddr {
		t.Errorf("ClientAddr incorrect! Expected %v, Got %v", expected.ClientAddr, response.ClientAddr)
	}
	if response.ClientRole != expected.ClientRole {
		t.Errorf("ClientRole incorrect! Expected %v, Got %v", expected.ClientRole, response.ClientRole)
	}
	if response.UnitId != expected.UnitId {
		t.Errorf("UnitId incorrect! Expected %v, Got %v", expected.UnitId, response.UnitId)
	}
	if response.Addr != expected.Addr {
		t.Errorf("Addr incorrect! Expected %v, Got %v", expected.Addr, response.Addr)
	}
	if response.Quantity != expected.Quantity {
		t.Errorf("Quantity incorrect! Expected %v, Got %v", expected.Quantity, response.Quantity)
	}
	if response.IsWrite != expected.IsWrite {
		t.Errorf("IsWrite incorrect! Expected %v, Got %v", expected.IsWrite, response.IsWrite)
	}
	for i := range response.Args {
		if response.Args[i] != expected.Args[i] {
			t.Errorf("Args[%v] incorrect! Expected %v, Got %v", i, expected.Args[i], response.Args[i])
		}
	}
}

func TestWriteMultipleHoldingRegister(t *testing.T) {
	message := []byte{0x11, 0x10, 0x00, 0x01, 0x00, 0x02, 0x04, 0x00, 0x0A, 0x01, 0x02, 0xC6, 0xF0}

	raw, err := createRequestFromBytes(message)
	if err != nil {
		t.Errorf("Request should not be an error!")
	}

	response := raw.(HoldingRegistersRequest)
	var expected = HoldingRegistersRequest{
		ClientAddr: "127.0.0.1",
		ClientRole: "",
		UnitId:     0x11,
		Addr:       0x01,
		Quantity:   2,
		IsWrite:    true,
		Args:       []uint16{0x000A, 0x0102},
	}

	if response.ClientAddr != expected.ClientAddr {
		t.Errorf("ClientAddr incorrect! Expected %v, Got %v", expected.ClientAddr, response.ClientAddr)
	}
	if response.ClientRole != expected.ClientRole {
		t.Errorf("ClientRole incorrect! Expected %v, Got %v", expected.ClientRole, response.ClientRole)
	}
	if response.UnitId != expected.UnitId {
		t.Errorf("UnitId incorrect! Expected %v, Got %v", expected.UnitId, response.UnitId)
	}
	if response.Addr != expected.Addr {
		t.Errorf("Addr incorrect! Expected %v, Got %v", expected.Addr, response.Addr)
	}
	if response.Quantity != expected.Quantity {
		t.Errorf("Quantity incorrect! Expected %v, Got %v", expected.Quantity, response.Quantity)
	}
	if response.IsWrite != expected.IsWrite {
		t.Errorf("IsWrite incorrect! Expected %v, Got %v", expected.IsWrite, response.IsWrite)
	}
	for i := range response.Args {
		if response.Args[i] != expected.Args[i] {
			t.Errorf("Args incorrect! Expected %v, Got %v", expected.Args[i], response.Args[i])
		}
	}
}

func TestReadMultipleCoilRegisterByteAnswer(t *testing.T) {
	message := []byte{0x11, 0x01, 0x00, 0x13, 0x00, 0x25, 0x0E, 0x84}
	expected := []byte{0x11, 0x01, 0x05, 0xCD, 0x6B, 0xB2, 0x0E, 0x1B, 0x45, 0xE6}

	var requestResult = []bool{
		true, false, true, true, false, false, true, true,
		true, true, false, true, false, true, true, false,
		false, true, false, false, true, true, false, true,
		false, true, true, true, false, false, false, false,
		true, true, false, true, true,
	}

	result, err := createBytesFromRequest(message, requestResult)

	if err != nil {
		t.Errorf("Expecting a result, not an error!")
		return
	}
	for i := 0; i < len(expected); i++ {
		if expected[i] != result[i] {
			t.Errorf("Byte value incorrect! Pos %v of %v Expected 0x%X, Got 0x%X", i, len(expected), expected[i], result[i])
		}
	}
}

func TestReadMultipleDiscreteInputRegisterByteAnswer(t *testing.T) {
	message := []byte{0x11, 0x02, 0x00, 0xC4, 0x00, 0x16, 0xBA, 0xA9}
	expected := []byte{0x11, 0x02, 0x03, 0xAC, 0xDB, 0x35, 0x20, 0x18}

	var requestResult = []bool{
		false, false, true, true, false, true, false, true,
		true, true, false, true, true, false, true, true,
		true, false, true, false, true, true,
	}

	result, err := createBytesFromRequest(message, requestResult)

	if err != nil {
		t.Errorf("Expecting a result, not an error!")
		return
	}
	for i := 0; i < len(expected); i++ {
		if expected[i] != result[i] {
			t.Errorf("Byte value incorrect! Pos %v of %v Expected 0x%X, Got 0x%X", i, len(expected), expected[i], result[i])
		}
	}
}

func TestReadMultipleHoldingRegisterByteAnswer(t *testing.T) {
	message := []byte{0x11, 0x03, 0x00, 0x6B, 0x00, 0x03, 0x76, 0x87}
	expected := []byte{0x11, 0x03, 0x06, 0xAE, 0x41, 0x56, 0x52, 0x43, 0x40, 0x49, 0xAD}

	var requestResult = []uint16{0xAE41, 0x5652, 0x4340}

	result, err := createBytesFromRequest(message, requestResult)

	if err != nil {
		t.Errorf("Expecting a result, not an error!")
		return
	}
	for i := 0; i < len(expected); i++ {
		if expected[i] != result[i] {
			t.Errorf("Byte value incorrect! Pos %v of %v Expected 0x%X, Got 0x%X", i, len(expected), expected[i], result[i])
		}
	}
}

func TestReadMultipleInputRegisterByteAnswer(t *testing.T) {
	message := []byte{0x11, 0x04, 0x00, 0x08, 0x00, 0x01, 0xB2, 0x98}
	expected := []byte{0x11, 0x04, 0x02, 0x00, 0x0A, 0xF8, 0xF4}

	var requestResult = []uint16{0x000A}

	result, err := createBytesFromRequest(message, requestResult)

	if err != nil {
		t.Errorf("Expecting a result, not an error!")
		return
	}
	for i := 0; i < len(expected); i++ {
		if expected[i] != result[i] {
			t.Errorf("Byte value incorrect! Pos %v of %v Expected 0x%X, Got 0x%X", i, len(expected), expected[i], result[i])
		}
	}
}

func TestWriteSingleCoilRegisterByteAnswer(t *testing.T) {
	message := []byte{0x11, 0x05, 0x00, 0xAC, 0xFF, 0x00, 0x4E, 0x8B}
	expected := []byte{0x11, 0x05, 0x00, 0xAC, 0xFF, 0x00, 0x4E, 0x8B}

	var requestResult = []bool{true}

	result, err := createBytesFromRequest(message, requestResult)

	if err != nil {
		t.Errorf("Expecting a result, not an error!")
		return
	}
	for i := 0; i < len(expected); i++ {
		if expected[i] != result[i] {
			t.Errorf("Byte value incorrect! Pos %v of %v Expected 0x%X, Got 0x%X", i, len(expected), expected[i], result[i])
		}
	}
}

func TestWriteSingleHoldingRegisterByteAnswer(t *testing.T) {
	message := []byte{0x11, 0x06, 0x00, 0x01, 0x00, 0x03, 0x9A, 0x9B}
	expected := []byte{0x11, 0x06, 0x00, 0x01, 0x00, 0x03, 0x9A, 0x9B}

	var requestResult = []uint16{0x0003}

	result, err := createBytesFromRequest(message, requestResult)

	if err != nil {
		t.Errorf("Expecting a result, not an error!")
		return
	}
	for i := 0; i < len(expected); i++ {
		if expected[i] != result[i] {
			t.Errorf("Byte value incorrect! Pos %v of %v Expected 0x%X, Got 0x%X", i, len(expected), expected[i], result[i])
		}
	}
}

func TestWriteMultipleCoilRegisterByteAnswer(t *testing.T) {
	message := []byte{0x11, 0x0F, 0x00, 0x13, 0x00, 0x0A, 0x02, 0xCD, 0x01, 0xBF, 0x0D}
	expected := []byte{0x11, 0x0F, 0x00, 0x13, 0x00, 0x0A, 0x26, 0x99}

	var requestResult = []bool{true, false, true, true, false, false, true, true, true, false}

	result, err := createBytesFromRequest(message, requestResult)

	if err != nil {
		t.Errorf("Expecting a result, not an error!")
		return
	}
	for i := 0; i < len(expected); i++ {
		if expected[i] != result[i] {
			t.Errorf("Byte value incorrect! Pos %v of %v Expected 0x%X, Got 0x%X", i, len(expected), expected[i], result[i])
		}
	}
}

func TestWriteMultipleHoldingRegisterByteAnswer(t *testing.T) {
	message := []byte{0x11, 0x10, 0x00, 0x01, 0x00, 0x02, 0x04, 0x00, 0x0A, 0x01, 0x02, 0xC6, 0xF0}
	expected := []byte{0x11, 0x10, 0x00, 0x01, 0x00, 0x02, 0x12, 0x98}

	var requestResult = []uint16{0x000A, 0x0102}

	result, err := createBytesFromRequest(message, requestResult)

	if err != nil {
		t.Errorf("Expecting a result, not an error!")
		return
	}
	for i := 0; i < len(expected); i++ {
		if expected[i] != result[i] {
			t.Errorf("Byte value incorrect! Pos %v of %v Expected 0x%X, Got 0x%X", i, len(expected), expected[i], result[i])
		}
	}
}
