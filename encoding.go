package modbus

import (
	"encoding/binary"
	"math"
)

func uint16ToBytes(endianness Endianness, in uint16) (out []byte) {
	out	= make([]byte, 2)
	switch endianness {
	case BIG_ENDIAN:	binary.BigEndian.PutUint16(out, in)
	case LITTLE_ENDIAN:	binary.LittleEndian.PutUint16(out, in)
	}

	return
}

func uint16sToBytes(endianness Endianness, in []uint16) (out []byte) {
	for i := range in {
		out = append(out, uint16ToBytes(endianness, in[i])...)
	}

	return
}

func bytesToUint16(endianness Endianness, in []byte) (out uint16) {
	switch endianness {
	case BIG_ENDIAN:	out = binary.BigEndian.Uint16(in)
	case LITTLE_ENDIAN:	out = binary.LittleEndian.Uint16(in)
	}

	return
}

func bytesToUint16s(endianness Endianness, in []byte) (out []uint16) {
	for i := 0; i < len(in); i += 2 {
		out = append(out, bytesToUint16(endianness, in[i:i+2]))
	}

	return
}

func bytesToUint32s(endianness Endianness, wordOrder WordOrder, in []byte) (out []uint32) {
	var u32		uint32

	for i := 0; i < len(in); i += 4 {
		switch endianness {
		case BIG_ENDIAN:
			if wordOrder == HIGH_WORD_FIRST {
				u32 = binary.BigEndian.Uint32(in[i:i+4])
			} else {
				u32 = binary.BigEndian.Uint32(
					[]byte{in[i+2], in[i+3], in[i+0], in[i+1]})
			}
		case LITTLE_ENDIAN:
			if wordOrder == LOW_WORD_FIRST {
				u32 = binary.LittleEndian.Uint32(in[i:i+4])
			} else {
				u32 = binary.LittleEndian.Uint32(
					[]byte{in[i+2], in[i+3], in[i+0], in[i+1]})
			}
		}

		out = append(out, u32)
	}

	return
}

func uint32ToBytes(endianness Endianness, wordOrder WordOrder, in uint32) (out []byte) {
	out = make([]byte, 4)

	switch endianness {
	case BIG_ENDIAN:
		binary.BigEndian.PutUint32(out, in)

		// swap words if needed
		if wordOrder == LOW_WORD_FIRST {
			out[0], out[1], out[2], out[3] = out[2], out[3], out[0], out[1]
		}
	case LITTLE_ENDIAN:
		binary.LittleEndian.PutUint32(out, in)

		// swap words if needed
		if wordOrder == HIGH_WORD_FIRST {
			out[0], out[1], out[2], out[3] = out[2], out[3], out[0], out[1]
		}
	}

	return
}

func bytesToFloat32s(endianness Endianness, wordOrder WordOrder, in []byte) (out []float32) {
	var u32s	[]uint32

	u32s	= bytesToUint32s(endianness, wordOrder, in)

	for _, u32 := range u32s {
		out = append(out, math.Float32frombits(u32))
	}

	return
}

func float32ToBytes(endianness Endianness, wordOrder WordOrder, in float32) (out []byte) {
	out = uint32ToBytes(endianness, wordOrder, math.Float32bits(in))

	return
}

func bytesToUint64s(endianness Endianness, wordOrder WordOrder, in []byte) (out []uint64) {
	var u64 uint64

	for i := 0; i < len(in); i += 8 {
		switch endianness {
		case BIG_ENDIAN:
			if wordOrder == HIGH_WORD_FIRST {
				u64 = binary.BigEndian.Uint64(in[i:i+8])
			} else {
				u64 = binary.BigEndian.Uint64(
					[]byte{in[i+6], in[i+7], in[i+4], in[i+5],
					       in[i+2], in[i+3], in[i+0], in[i+1]})
			}
		case LITTLE_ENDIAN:
			if wordOrder == LOW_WORD_FIRST {
				u64 = binary.LittleEndian.Uint64(in[i:i+8])
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

func uint64ToBytes(endianness Endianness, wordOrder WordOrder, in uint64) (out []byte) {
	out = make([]byte, 8)

	switch endianness {
	case BIG_ENDIAN:
		binary.BigEndian.PutUint64(out, in)

		// swap words if needed
		if wordOrder == LOW_WORD_FIRST {
			out[0], out[1], out[2], out[3],out[4], out[5], out[6], out[7] =
				out[6], out[7], out[4], out[5], out[2], out[3], out[0], out[1]
		}
	case LITTLE_ENDIAN:
		binary.LittleEndian.PutUint64(out, in)

		// swap words if needed
		if wordOrder == HIGH_WORD_FIRST {
			out[0], out[1], out[2], out[3],out[4], out[5], out[6], out[7] =
				out[6], out[7], out[4], out[5], out[2], out[3], out[0], out[1]
		}
	}

	return
}

func bytesToFloat64s(endianness Endianness, wordOrder WordOrder, in []byte) (out []float64) {
	var u64s []uint64

	u64s = bytesToUint64s(endianness, wordOrder, in)

	for _, u64 := range u64s {
		out = append(out, math.Float64frombits(u64))
	}

	return
}

func float64ToBytes(endianness Endianness, wordOrder WordOrder, in float64) (out []byte) {
	out = uint64ToBytes(endianness, wordOrder, math.Float64bits(in))

	return
}

func encodeBools(in []bool) (out []byte) {
	var byteCount	uint
	var i		uint

	byteCount	= uint(len(in)) / 8
	if len(in) % 8 != 0 {
		byteCount++
	}

	out		= make([]byte, byteCount)
	for i = 0; i < uint(len(in)); i++ {
		if in[i] {
			out[i/8] |= (0x01 << (i % 8))
		}
	}

	return
}

func decodeBools(quantity uint16, in []byte) (out []bool) {
	var i	uint
	for i = 0; i < uint(quantity); i++ {
		out = append(out, (((in[i/8] >> (i % 8)) & 0x01) == 0x01))
	}

	return
}
