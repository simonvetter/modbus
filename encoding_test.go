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

func TestUint32ToBytes(t *testing.T) {
	var out []byte

	out = uint32ToBytes(BIG_ENDIAN, HIGH_WORD_FIRST, 0x87654321)
	if len(out) != 4 {
		t.Errorf("expected 4 bytes, got %v", len(out))
	}
	if out[0] != 0x87 || out[1] != 0x65 || // first word
	   out[2] != 0x43 || out[3] != 0x21 {  // second word
		t.Errorf("expected {0x87, 0x65, 0x43, 0x21}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			 out[0], out[1], out[2], out[3])
	}

	out = uint32ToBytes(BIG_ENDIAN, LOW_WORD_FIRST, 0x87654321)
	if len(out) != 4 {
		t.Errorf("expected 4 bytes, got %v", len(out))
	}
	if out[0] != 0x43 || out[1] != 0x21 || out[2] != 0x87 || out[3] != 0x65 {
		t.Errorf("expected {0x43, 0x21, 0x87, 0x65}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			 out[0], out[1], out[2], out[3])
	}

	out = uint32ToBytes(LITTLE_ENDIAN, LOW_WORD_FIRST, 0x87654321)
	if len(out) != 4 {
		t.Errorf("expected 4 bytes, got %v", len(out))
	}
	if out[0] != 0x21 || out[1] != 0x43 || out[2] != 0x65 || out[3] != 0x87 {
		t.Errorf("expected {0x21, 0x43, 0x65, 0x87}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			 out[0], out[1], out[2], out[3])
	}

	out = uint32ToBytes(LITTLE_ENDIAN, HIGH_WORD_FIRST, 0x87654321)
	if len(out) != 4 {
		t.Errorf("expected 4 bytes, got %v", len(out))
	}
	if out[0] != 0x65 || out[1] != 0x87 || out[2] != 0x21 || out[3] != 0x43 {
		t.Errorf("expected {0x65, 0x87, 0x21, 0x43}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			 out[0], out[1], out[2], out[3])
	}

	return
}

func TestBytesToUint32s(t *testing.T) {
	var results []uint32

	results	= bytesToUint32s(BIG_ENDIAN, HIGH_WORD_FIRST, []byte{
		0x87, 0x65, 0x43, 0x21,
		0x00, 0x11, 0x22, 0x33,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 0x87654321 {
		t.Errorf("expected 0x87654321, got 0x%08x", results[0])
	}
	if results[1] != 0x00112233 {
		t.Errorf("expected 0x00112233, got 0x%08x", results[1])
	}

	results	= bytesToUint32s(BIG_ENDIAN, LOW_WORD_FIRST, []byte{
		0x87, 0x65, 0x43, 0x21,
		0x00, 0x11, 0x22, 0x33,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 0x43218765 {
		t.Errorf("expected 0x43218765, got 0x%08x", results[0])
	}
	if results[1] != 0x22330011 {
		t.Errorf("expected 0x22330011, got 0x%08x", results[1])
	}

	results	= bytesToUint32s(LITTLE_ENDIAN, LOW_WORD_FIRST, []byte{
		0x87, 0x65, 0x43, 0x21,
		0x00, 0x11, 0x22, 0x33,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 0x21436587 {
		t.Errorf("expected 0x21436587, got 0x%08x", results[0])
	}
	if results[1] != 0x33221100 {
		t.Errorf("expected 0x33221100, got 0x%08x", results[1])
	}

	results	= bytesToUint32s(LITTLE_ENDIAN, HIGH_WORD_FIRST, []byte{
		0x87, 0x65, 0x43, 0x21,
		0x00, 0x11, 0x22, 0x33,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 0x65872143 {
		t.Errorf("expected 0x65872143, got 0x%08x", results[0])
	}
	if results[1] != 0x11003322 {
		t.Errorf("expected 0x11003322, got 0x%08x", results[1])
	}

	return
}

func TestFloat32ToBytes(t *testing.T) {
	var out	[]byte

	out = float32ToBytes(BIG_ENDIAN, HIGH_WORD_FIRST, 1.234)
	if len(out) != 4 {
		t.Errorf("expected 4 bytes, got %v", len(out))
	}
	if out[0] != 0x3f || out[1] != 0x9d || out[2] != 0xf3 || out[3] != 0xb6 {
		t.Errorf("expected {0x3f, 0x9d, 0xf3, 0xb6}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			 out[0], out[1], out[2], out[3])
	}

	out = float32ToBytes(BIG_ENDIAN, LOW_WORD_FIRST, 1.234)
	if len(out) != 4 {
		t.Errorf("expected 4 bytes, got %v", len(out))
	}
	if out[0] != 0xf3 || out[1] != 0xb6 || out[2] != 0x3f || out[3] != 0x9d {
		t.Errorf("expected {0xf3, 0xb6, 0x3f, 0x9d}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			 out[0], out[1], out[2], out[3])
	}

	out = float32ToBytes(LITTLE_ENDIAN, LOW_WORD_FIRST, 1.234)
	if len(out) != 4 {
		t.Errorf("expected 4 bytes, got %v", len(out))
	}
	if out[0] != 0xb6 || out[1] != 0xf3 || out[2] != 0x9d || out[3] != 0x3f {
		t.Errorf("expected {0xb6, 0xf3, 0x9d, 0x3f}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			 out[0], out[1], out[2], out[3])
	}

	out = float32ToBytes(LITTLE_ENDIAN, HIGH_WORD_FIRST, 1.234)
	if len(out) != 4 {
		t.Errorf("expected 4 bytes, got %v", len(out))
	}
	if out[0] != 0x9d || out[1] != 0x3f || out[2] != 0xb6 || out[3] != 0xf3 {
		t.Errorf("expected {0x9d, 0x3f, 0xb6, 0xf3}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			 out[0], out[1], out[2], out[3])
	}

	return
}

func TestBytesToFloat32s(t *testing.T) {
	var results []float32

	results	= bytesToFloat32s(BIG_ENDIAN, HIGH_WORD_FIRST, []byte{
		0x3f, 0x9d, 0xf3, 0xb6,
		0x40, 0x49, 0x0f, 0xdb,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 1.234 {
		t.Errorf("expected 1.234, got %.04f", results[0])
	}
	if results[1] != 3.14159274101 {
		t.Errorf("expected 3.14159274101, got %.09f", results[1])
	}

	results	= bytesToFloat32s(BIG_ENDIAN, LOW_WORD_FIRST, []byte{
		0xf3, 0xb6, 0x3f, 0x9d,
		0x0f, 0xdb, 0x40, 0x49,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 1.234 {
		t.Errorf("expected 1.234, got %.04f", results[0])
	}
	if results[1] != 3.14159274101 {
		t.Errorf("expected 3.14159274101, got %.09f", results[1])
	}

	results	= bytesToFloat32s(LITTLE_ENDIAN, LOW_WORD_FIRST, []byte{
		0xb6, 0xf3, 0x9d, 0x3f,
		0xdb, 0x0f, 0x49, 0x40,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 1.234 {
		t.Errorf("expected 1.234, got %.04f", results[0])
	}
	if results[1] != 3.14159274101 {
		t.Errorf("expected 3.14159274101, got %.09f", results[1])
	}

	results	= bytesToFloat32s(LITTLE_ENDIAN, HIGH_WORD_FIRST, []byte{
		0x9d, 0x3f, 0xb6, 0xf3,
		0x49, 0x40, 0xdb, 0x0f,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 1.234 {
		t.Errorf("expected 1.234, got %.04f", results[0])
	}
	if results[1] != 3.14159274101 {
		t.Errorf("expected 3.14159274101, got %.09f", results[1])
	}

	return
}

func TestUint64ToBytes(t *testing.T) {
	var out []byte

	out = uint64ToBytes(BIG_ENDIAN, HIGH_WORD_FIRST, 0x0fedcba987654321)
	if len(out) != 8 {
		t.Errorf("expected 8 bytes, got %v", len(out))
	}

	if out[0] != 0x0f || out[1] != 0xed || //  1st word
		out[2] != 0xcb || out[3] != 0xa9 || // 2nd word
		out[4] != 0x87 || out[5] != 0x65 || // 3rd word
		out[6] != 0x43 || out[7] != 0x21 {  // 4th word
		t.Errorf("expected {0x0f, 0xed, 0xcb, 0xa9, 0x87, 0x65, 0x43, 0x21}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7])
	}

	out = uint64ToBytes(BIG_ENDIAN, LOW_WORD_FIRST, 0x0fedcba987654321)
	if len(out) != 8 {
		t.Errorf("expected 8 bytes, got %v", len(out))
	}
	if out[0] != 0x43 || out[1] != 0x21 || //  1st word
		out[2] != 0x87 || out[3] != 0x65 || // 2nd word
		out[4] != 0xcb || out[5] != 0xa9 || // 3rd word
		out[6] != 0x0f || out[7] != 0xed {  // 4th word
		t.Errorf("expected {0x43, 0x21, 0x87, 0x65, 0xcb, 0xa9, 0x0f, 0xed}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7])
	}

	out = uint64ToBytes(LITTLE_ENDIAN, LOW_WORD_FIRST, 0x0fedcba987654321)
	if len(out) != 8 {
		t.Errorf("expected 8 bytes, got %v", len(out))
	}
	if out[0] != 0x21 || out[1] != 0x43 || //  1st word
		out[2] != 0x65 || out[3] != 0x87 || // 2nd word
		out[4] != 0xa9 || out[5] != 0xcb || // 3rd word
		out[6] != 0xed || out[7] != 0x0f {  // 4th word
		t.Errorf("expected {0x21, 0x43, 0x65, 0x87, 0xa9, 0xcb, 0xed, 0x0f}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7])
	}

	out = uint64ToBytes(LITTLE_ENDIAN, HIGH_WORD_FIRST, 0x0fedcba987654321)
	if len(out) != 8 {
		t.Errorf("expected 8 bytes, got %v", len(out))
	}
	if out[0] != 0xed || out[1] != 0x0f ||  //  1st word
		out[2] != 0xa9 || out[3] != 0xcb || // 2nd word
		out[4] != 0x65 || out[5] != 0x87 || // 3rd word
		out[6] != 0x21 || out[7] != 0x43 {  // 4th word
		t.Errorf("expected {0xed, 0x0f, 0xa9, 0xcb, 0x65, 0x87, 0x21, 0x43}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7])
	}

	return
}

func TestBytesToUint64s(t *testing.T) {
	var results []uint64

	results	= bytesToUint64s(BIG_ENDIAN, HIGH_WORD_FIRST, []byte{
		0x0f, 0xed, 0xcb, 0xa9, 0x87, 0x65, 0x43, 0x21,
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 0x0fedcba987654321 {
		t.Errorf("expected 0x0fedcba987654321, got 0x%016x", results[0])
	}
	if results[1] != 0x0011223344556677 {
		t.Errorf("expected 0x0011223344556677, got 0x%016x", results[1])
	}

	results	= bytesToUint64s(BIG_ENDIAN, LOW_WORD_FIRST, []byte{
		0x0f, 0xed, 0xcb, 0xa9, 0x87, 0x65, 0x43, 0x21,
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}

	if results[0] != 0x43218765cba90fed {
		t.Errorf("expected 0x43218765cba90fed, got 0x%016x", results[0])
	}

	if results[1] != 0x6677445522330011 {
		t.Errorf("expected 0x6677445522330011, got 0x%016x", results[1])
	}

	results	= bytesToUint64s(LITTLE_ENDIAN, LOW_WORD_FIRST, []byte{
		0x0f, 0xed, 0xcb, 0xa9, 0x87, 0x65, 0x43, 0x21,
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 0x21436587a9cbed0f {
		t.Errorf("expected 0x21436587a9cbed0f, got 0x%016x", results[0])
	}
	if results[1] != 0x7766554433221100 {
		t.Errorf("expected 0x7766554433221100, got 0x%016x", results[1])
	}

	results	= bytesToUint64s(LITTLE_ENDIAN, HIGH_WORD_FIRST, []byte{
		0x0f, 0xed, 0xcb, 0xa9, 0x87, 0x65, 0x43, 0x21,
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 0xed0fa9cb65872143 {
		t.Errorf("expected 0xed0fa9cb65872143, got 0x%016x", results[0])
	}
	if results[1] != 0x1100332255447766 {
		t.Errorf("expected 0x1100332255447766, got 0x%016x", results[1])
	}

	return
}

func TestFloat64ToBytes(t *testing.T) {
	var out	[]byte

	out = float64ToBytes(BIG_ENDIAN, HIGH_WORD_FIRST, 1.2345678)
	if len(out) != 8 {
		t.Errorf("expected 8 bytes, got %v", len(out))
	}
	if out[0] != 0x3f || out[1] != 0xf3 || out[2] != 0xc0 || out[3] != 0xca ||
		out[4] != 0x2a || out[5] != 0x5b || out[6] != 0x1d || out[7] != 0x5d {
		t.Errorf("expected {0x3f, 0xf3, 0xc0, 0xca, 0x2a, 0x5b, 0x1d, 0x5d}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7])
	}

	out = float64ToBytes(BIG_ENDIAN, LOW_WORD_FIRST, 1.2345678)
	if len(out) != 8 {
		t.Errorf("expected 8 bytes, got %v", len(out))
	}
	if out[0] != 0x1d || out[1] != 0x5d || out[2] != 0x2a || out[3] != 0x5b ||
		out[4] != 0xc0 || out[5] != 0xca || out[6] != 0x3f || out[7] != 0xf3 {
		t.Errorf("expected {0x1d, 0x5d, 0x2a, 0x5b, 0xc0, 0xca, 0x3f, 0xf3}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7])
	}

	out = float64ToBytes(LITTLE_ENDIAN, LOW_WORD_FIRST, 1.2345678)
	if len(out) != 8 {
		t.Errorf("expected 8 bytes, got %v", len(out))
	}

	if out[0] != 0x5d || out[1] != 0x1d || out[2] != 0x5b || out[3] != 0x2a ||
		out[4] != 0xca || out[5] != 0xc0 || out[6] != 0xf3 || out[7] != 0x3f {
		t.Errorf("expected {0x5d, 0x1d, 0x5b, 0x2a, 0xca, 0xc0, 0xf3, 0x3f}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7])
	}

	out = float64ToBytes(LITTLE_ENDIAN, HIGH_WORD_FIRST, 1.2345678)
	if len(out) != 8 {
		t.Errorf("expected 8 bytes, got %v", len(out))
	}
	if out[0] != 0xf3 || out[1] != 0x3f || out[2] != 0xca || out[3] != 0xc0 ||
		out[4] != 0x5b || out[5] != 0x2a || out[6] != 0x5d || out[7] != 0x1d {
		t.Errorf("expected {0xf3, 0x3f, 0xca, 0xc0, 0x5b, 0x2a, 0x5d, 0x1d}, got {0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x, 0x%02x}",
			out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7])
	}

	return
}

func TestBytesToFloat64s(t *testing.T) {
	var results []float64

	results	= bytesToFloat64s(BIG_ENDIAN, HIGH_WORD_FIRST, []byte{
		0x3f, 0xf3, 0xc0, 0xca, 0x2a, 0x5b, 0x1d, 0x5d,
		0x40, 0x09, 0x21, 0xfb, 0x5f, 0xff, 0xe9, 0x5e,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 1.2345678 {
		t.Errorf("expected 1.2345678, got %.08f", results[0])
	}
	if results[1] != 3.14159274101 {
		t.Errorf("expected 3.14159274101, got %.09f", results[1])
	}

	results	= bytesToFloat64s(BIG_ENDIAN, LOW_WORD_FIRST, []byte{
		0x1d, 0x5d, 0x2a, 0x5b, 0xc0, 0xca, 0x3f, 0xf3,
		0xe9, 0x5e, 0x5f, 0xff, 0x21, 0xfb, 0x40, 0x09,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 1.2345678 {
		t.Errorf("expected 1.234, got %.08f", results[0])
	}
	if results[1] != 3.14159274101 {
		t.Errorf("expected 3.14159274101, got %.09f", results[1])
	}

	results	= bytesToFloat64s(LITTLE_ENDIAN, LOW_WORD_FIRST, []byte{
		0x5d, 0x1d, 0x5b, 0x2a, 0xca, 0xc0, 0xf3, 0x3f,
		0x5e, 0xe9, 0xff, 0x5f, 0xfb, 0x21, 0x09, 0x40,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 1.2345678 {
		t.Errorf("expected 1.234, got %.08f", results[0])
	}
	if results[1] != 3.14159274101 {
		t.Errorf("expected 3.14159274101, got %.09f", results[1])
	}

	results	= bytesToFloat64s(LITTLE_ENDIAN, HIGH_WORD_FIRST, []byte{
		0xf3, 0x3f, 0xca, 0xc0, 0x5b, 0x2a, 0x5d, 0x1d,
		0x09, 0x40, 0xfb, 0x21, 0xff, 0x5f, 0x5e, 0xe9,
	})
	if len(results) != 2 {
		t.Errorf("expected 2 values, got %v", len(results))
	}
	if results[0] != 1.2345678 {
		t.Errorf("expected 1.234, got %.08f", results[0])
	}
	if results[1] != 3.14159274101 {
		t.Errorf("expected 3.14159274101, got %.09f", results[1])
	}

	return
}

func TestDecodeBools(t *testing.T) {
	var results	[]bool

	results	= decodeBools(1, []byte{0x01})
	if len(results) != 1 {
		t.Errorf("expected 1 value, got %v", len(results))
	}
	if results[0] != true {
		t.Errorf("expected true, got false")
	}

	results	= decodeBools(1, []byte{0x0f})
	if len(results) != 1 {
		t.Errorf("expected 1 value, got %v", len(results))
	}
	if results[0] != true {
		t.Errorf("expected true, got false")
	}

	results	= decodeBools(9, []byte{0x75, 0x03})
	if len(results) != 9 {
		t.Errorf("expected 9 values, got %v", len(results))
	}
	for i, b := range []bool{
			true, false, true, false,	// 0x05
			true, true, true, false,	// 0x07
			true, } {			// 0x01
		if b != results[i] {
			t.Errorf("expected %v at %v, got %v", b, i, results[i])
		}
	}

	return
}

func TestEncodeBools(t *testing.T) {
	var results	[]byte

	results	= encodeBools([]bool{false, true, false, true, })
	if len(results) != 1 {
		t.Errorf("expected 1 byte, got %v", len(results))
	}
	if results[0] != 0x0a {
		t.Errorf("expected 0x0a, got 0x%02x", results[0])
	}

	results	= encodeBools([]bool{true, false, true, })
	if len(results) != 1 {
		t.Errorf("expected 1 byte, got %v", len(results))
	}
	if results[0] != 0x05 {
		t.Errorf("expected 0x05, got 0x%02x", results[0])
	}

	results	= encodeBools([]bool{true, false, false, true, false, true, true, false,
			             true, true, true, false, true, true, true, false,
				     false, true})
	if len(results) != 3 {
		t.Errorf("expected 3 bytes, got %v", len(results))
	}
	if results[0] != 0x69 || results[1] != 0x77 || results[2] != 0x02 {
		t.Errorf("expected {0x69, 0x77, 0x02}, got {0x%02x, 0x%02x, 0x%02x}",
			 results[0], results[1], results[2])
	}

	return
}
