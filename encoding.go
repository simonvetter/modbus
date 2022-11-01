package mbserver

import (
	"encoding/binary"
	"math"
)

func asBytes(bo binary.ByteOrder, in uint16) []byte {
	out := make([]byte, 2)
	bo.PutUint16(out, in)
	return out
}

func uint16ToBytes(bo binary.ByteOrder, in []uint16) (out []byte) {
	for i := range in {
		out = append(out, asBytes(bo, in[i])...)
	}

	return
}

func bytesToUint16(bo binary.ByteOrder, in []byte) (out []uint16) {
	for i := 0; i < len(in); i += 2 {
		out = append(out, bo.Uint16(in[i:i+2]))
	}

	return
}

func bytesToUint32(bo binary.ByteOrder, wordOrder WordOrder, in []byte) (out []uint32) {
	var u32 uint32

	for i := 0; i < len(in); i += 4 {
		switch bo {
		case binary.BigEndian:
			if wordOrder == HIGH_WORD_FIRST {
				u32 = binary.BigEndian.Uint32(in[i : i+4])
			} else {
				u32 = binary.BigEndian.Uint32(
					[]byte{in[i+2], in[i+3], in[i+0], in[i+1]})
			}
		case binary.LittleEndian:
			if wordOrder == LOW_WORD_FIRST {
				u32 = binary.LittleEndian.Uint32(in[i : i+4])
			} else {
				u32 = binary.LittleEndian.Uint32(
					[]byte{in[i+2], in[i+3], in[i+0], in[i+1]})
			}
		}

		out = append(out, u32)
	}

	return
}

func uint32ToBytes(bo binary.ByteOrder, wordOrder WordOrder, in uint32) (out []byte) {
	out = make([]byte, 4)
	bo.PutUint32(out, in)

	switch bo {
	case binary.BigEndian:
		// swap words if needed
		if wordOrder == LOW_WORD_FIRST {
			out[0], out[1], out[2], out[3] = out[2], out[3], out[0], out[1]
		}
	case binary.LittleEndian:
		// swap words if needed
		if wordOrder == HIGH_WORD_FIRST {
			out[0], out[1], out[2], out[3] = out[2], out[3], out[0], out[1]
		}
	}

	return
}

func bytesToFloat32(bo binary.ByteOrder, wordOrder WordOrder, in []byte) (out []float32) {
	for _, u32 := range bytesToUint32(bo, wordOrder, in) {
		out = append(out, math.Float32frombits(u32))
	}

	return
}

func float32ToBytes(bo binary.ByteOrder, wordOrder WordOrder, in float32) []byte {
	return uint32ToBytes(bo, wordOrder, math.Float32bits(in))
}

func bytesToUint64(bo binary.ByteOrder, wordOrder WordOrder, in []byte) (out []uint64) {
	var u64 uint64

	for i := 0; i < len(in); i += 8 {
		switch bo {
		case binary.BigEndian:
			if wordOrder == HIGH_WORD_FIRST {
				u64 = binary.BigEndian.Uint64(in[i : i+8])
			} else {
				u64 = binary.BigEndian.Uint64(
					[]byte{in[i+6], in[i+7], in[i+4], in[i+5],
						in[i+2], in[i+3], in[i+0], in[i+1]})
			}
		case binary.LittleEndian:
			if wordOrder == LOW_WORD_FIRST {
				u64 = binary.LittleEndian.Uint64(in[i : i+8])
			} else {
				u64 = binary.LittleEndian.Uint64(
					[]byte{in[i+6], in[i+7], in[i+4], in[i+5],
						in[i+2], in[i+3], in[i+0], in[i+1]})
			}
		}

		out = append(out, u64)
	}

	return
}

func uint64ToBytes(bo binary.ByteOrder, wordOrder WordOrder, in uint64) (out []byte) {
	out = make([]byte, 8)
	bo.PutUint64(out, in)

	switch bo {
	case binary.BigEndian:
		// swap words if needed
		if wordOrder == LOW_WORD_FIRST {
			out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7] =
				out[6], out[7], out[4], out[5], out[2], out[3], out[0], out[1]
		}
	case binary.LittleEndian:
		// swap words if needed
		if wordOrder == HIGH_WORD_FIRST {
			out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7] =
				out[6], out[7], out[4], out[5], out[2], out[3], out[0], out[1]
		}
	}

	return
}

func bytesToFloat64(bo binary.ByteOrder, wordOrder WordOrder, in []byte) (out []float64) {
	for _, u64 := range bytesToUint64(bo, wordOrder, in) {
		out = append(out, math.Float64frombits(u64))
	}

	return
}

func float64ToBytes(bo binary.ByteOrder, wordOrder WordOrder, in float64) (out []byte) {
	return uint64ToBytes(bo, wordOrder, math.Float64bits(in))
}

func encodeBools(in []bool) (out []byte) {
	var byteCount uint
	var i uint

	byteCount = uint(len(in)) / 8
	if len(in)%8 != 0 {
		byteCount++
	}

	out = make([]byte, byteCount)
	for i = 0; i < uint(len(in)); i++ {
		if in[i] {
			out[i/8] |= (0x01 << (i % 8))
		}
	}

	return
}

func decodeBools(quantity uint16, in []byte) (out []bool) {
	for i := uint(0); i < uint(quantity); i++ {
		out = append(out, (((in[i/8] >> (i % 8)) & 0x01) == 0x01))
	}

	return
}
