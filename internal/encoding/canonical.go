package encoding

import (
	"encoding/binary"
	"math"
	"sort"

	"golang.org/x/text/unicode/norm"
)

// Encode returns the canonical byte representation of v.
func Encode(v Value) []byte {
	var buf []byte
	encodeInto(v, &buf)
	return buf
}

func encodeInto(v Value, buf *[]byte) {
	switch v.Kind {
	case KindUndefined:
		*buf = append(*buf, byte(KindUndefined))

	case KindBool:
		*buf = append(*buf, byte(KindBool))
		if v.Bool {
			*buf = append(*buf, 1)
		} else {
			*buf = append(*buf, 0)
		}

	case KindU8:
		*buf = append(*buf, byte(KindU8), v.U8)

	case KindU16:
		*buf = append(*buf, byte(KindU16))
		b := [2]byte{}
		binary.LittleEndian.PutUint16(b[:], v.U16)
		*buf = append(*buf, b[:]...)

	case KindU32:
		*buf = append(*buf, byte(KindU32))
		b := [4]byte{}
		binary.LittleEndian.PutUint32(b[:], v.U32)
		*buf = append(*buf, b[:]...)

	case KindU64:
		*buf = append(*buf, byte(KindU64))
		b := [8]byte{}
		binary.LittleEndian.PutUint64(b[:], v.U64)
		*buf = append(*buf, b[:]...)

	case KindI8:
		*buf = append(*buf, byte(KindI8), byte(v.I8))

	case KindI16:
		*buf = append(*buf, byte(KindI16))
		b := [2]byte{}
		binary.LittleEndian.PutUint16(b[:], uint16(v.I16))
		*buf = append(*buf, b[:]...)

	case KindI32:
		*buf = append(*buf, byte(KindI32))
		b := [4]byte{}
		binary.LittleEndian.PutUint32(b[:], uint32(v.I32))
		*buf = append(*buf, b[:]...)

	case KindI64:
		*buf = append(*buf, byte(KindI64))
		b := [8]byte{}
		binary.LittleEndian.PutUint64(b[:], uint64(v.I64))
		*buf = append(*buf, b[:]...)

	case KindF32:
		*buf = append(*buf, byte(KindF32))
		// Normalize negative zero
		f := v.F32
		if f == 0 {
			f = 0
		}
		bits := math.Float32bits(f)
		b := [4]byte{}
		binary.LittleEndian.PutUint32(b[:], bits)
		*buf = append(*buf, b[:]...)

	case KindStr:
		normalized := norm.NFC.String(v.Str)
		strBytes := []byte(normalized)
		*buf = append(*buf, byte(KindStr))
		b := [4]byte{}
		binary.LittleEndian.PutUint32(b[:], uint32(len(strBytes)))
		*buf = append(*buf, b[:]...)
		*buf = append(*buf, strBytes...)

	case KindBytes:
		*buf = append(*buf, byte(KindBytes))
		b := [4]byte{}
		binary.LittleEndian.PutUint32(b[:], uint32(len(v.Bytes)))
		*buf = append(*buf, b[:]...)
		*buf = append(*buf, v.Bytes...)

	case KindRef:
		*buf = append(*buf, byte(KindRef))
		*buf = append(*buf, v.Ref.Bytes[:]...)

	case KindArray:
		*buf = append(*buf, byte(KindArray))
		b := [4]byte{}
		binary.LittleEndian.PutUint32(b[:], uint32(len(v.Elements)))
		*buf = append(*buf, b[:]...)
		for _, el := range v.Elements {
			encodeInto(el, buf)
		}

	case KindObject:
		// Sort keys for canonical encoding
		keys := make([]string, 0, len(v.Fields))
		for k := range v.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		*buf = append(*buf, byte(KindObject))
		b := [4]byte{}
		binary.LittleEndian.PutUint32(b[:], uint32(len(keys)))
		*buf = append(*buf, b[:]...)
		for _, k := range keys {
			encodeInto(StrVal(k), buf)
			encodeInto(v.Fields[k], buf)
		}

	case KindTable:
		*buf = append(*buf, byte(KindTable))
		*buf = append(*buf, v.Schema.Bytes[:]...)
		b := [4]byte{}
		binary.LittleEndian.PutUint32(b[:], uint32(len(v.Rows)))
		*buf = append(*buf, b[:]...)
		*buf = append(*buf, v.Rows...)
	}
}
