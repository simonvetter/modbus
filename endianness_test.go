package modbus

import (
	"testing"
)

func TestUint16ToBytes(t *testing.T) {
	var out []byte

	out = uint16ToBytes(BIG_ENDIAN, 0x4321)
	if len(out) != 2 {
		t.Errorf("expected 2 bytes, got %v", len(out))
	}
	if out[0] != 0x43 || out[1] != 0x21 {
		t.Errorf("expected {0x43, 0x21}, got {0x%02x, 0x%02x}", out[0], out[1])
	}

	out = uint16ToBytes(LITTLE_ENDIAN, 0x4321)
	if len(out) != 2 {
		t.Errorf("expected 2 bytes, got %v", len(out))
	}
	if out[0] != 0x21 || out[1] != 0x43 {
		t.Errorf("expected {0x21, 0x43}, got {0x%02x, 0x%02x}", out[0], out[1])
	}

	return
}

func TestUint16sToBytes(t *testing.T) {
	var out []byte

	out = uint16sToBytes(BIG_ENDIAN, []uint16{0x4321, 0x8765, 0xcba9})
	if len(out) != 6 {
		t.Errorf("expected 6 bytes, got %v", len(out))
	}
	if out[0] != 0x43 || out[1] != 0x21 {
		t.Errorf("expected {0x43, 0x21}, got {0x%02x, 0x%02x}", out[0], out[1])
	}
	if out[2] != 0x87 || out[3] != 0x65 {
		t.Errorf("expected {0x87, 0x65}, got {0x%02x, 0x%02x}", out[0], out[1])
	}
	if out[4] != 0xcb || out[5] != 0xa9 {
		t.Errorf("expected {0xcb, 0xa9}, got {0x%02x, 0x%02x}", out[0], out[1])
	}

	out = uint16sToBytes(LITTLE_ENDIAN, []uint16{0x4321, 0x8765, 0xcba9})
	if len(out) != 6 {
		t.Errorf("expected 6 bytes, got %v", len(out))
	}
	if out[0] != 0x21 || out[1] != 0x43 {
		t.Errorf("expected {0x21, 0x43}, got {0x%02x, 0x%02x}", out[0], out[1])
	}
	if out[2] != 0x65 || out[3] != 0x87 {
		t.Errorf("expected {0x65, 0x87}, got {0x%02x, 0x%02x}", out[0], out[1])
	}
	if out[4] != 0xa9 || out[5] != 0xcb {
		t.Errorf("expected {0xa9, 0xcb}, got {0x%02x, 0x%02x}", out[0], out[1])
	}

	return
}

func TestBytesToUint16(t *testing.T) {
	var result	uint16

	result	= bytesToUint16(BIG_ENDIAN, []byte{0x43, 0x21})
	if result != 0x4321 {
		t.Errorf("expected 0x4321, got 0x%04x", result)
	}

	result	= bytesToUint16(LITTLE_ENDIAN, []byte{0x43, 0x21})
	if result != 0x2143 {
		t.Errorf("expected 0x2143, got 0x%04x", result)
	}

	return
}

func TestBytesToUint16s(t *testing.T) {
	var results	[]uint16

	results	= bytesToUint16s(BIG_ENDIAN, []byte{0x11, 0x22, 0x33, 0x44})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 0x1122 {
		t.Errorf("expected 0x1122, got 0x%04x", results[0])
	}
	if results[1] != 0x3344 {
		t.Errorf("expected 0x3344, got 0x%04x", results[1])
	}

	results	= bytesToUint16s(LITTLE_ENDIAN, []byte{0x11, 0x22, 0x33, 0x44})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 0x2211 {
		t.Errorf("expected 0x2211, got 0x%04x", results[0])
	}
	if results[1] != 0x4433 {
		t.Errorf("expected 0x4433, got 0x%04x", results[1])
	}

	return
}
