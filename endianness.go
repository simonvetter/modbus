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
