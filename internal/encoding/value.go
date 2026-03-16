// Package encoding defines the Amber value types and their canonical byte encoding.
//
// Canonical encoding rules:
//   - Object keys sorted alphabetically
//   - Floats JSON-normalized (no trailing zeros, negative zero → positive zero)
//   - Strings UTF-8 NFC normalized
//   - Integers fixed-width little-endian
//   - Absent optional fields represented explicitly
package encoding

import (
	"fmt"
	"math"
)

// Kind identifies the type of an Amber value.
type Kind uint8

const (
	KindUndefined Kind = 0x00
	KindBool      Kind = 0x01
	KindU8        Kind = 0x10
	KindU16       Kind = 0x11
	KindU32       Kind = 0x12
	KindU64       Kind = 0x13
	KindI8        Kind = 0x14
	KindI16       Kind = 0x15
	KindI32       Kind = 0x16
	KindI64       Kind = 0x17
	KindF32       Kind = 0x20
	KindStr       Kind = 0x30
	KindBytes     Kind = 0x31
	KindRef       Kind = 0x40 // fingerprint pointer
	KindArray     Kind = 0x50
	KindObject    Kind = 0x51
	KindTable     Kind = 0x52
)

// Value is an immutable Amber value.
type Value struct {
	Kind Kind

	// Scalar fields — only one is valid per Kind.
	Bool   bool
	U8     uint8
	U16    uint16
	U32    uint32
	U64    uint64
	I8     int8
	I16    int16
	I32    int32
	I64    int64
	F32    float32
	Str    string
	Bytes  []byte
	Ref    Fingerprint // KindRef

	// Compound fields
	Elements []Value            // KindArray
	Fields   map[string]Value   // KindObject (encode in sorted key order)

	// KindTable
	Schema Fingerprint
	Rows   []byte
}

// Undefined is the single absence value in Amber (no null).
var Undefined = Value{Kind: KindUndefined}

// Bool returns a boolean Amber value.
func Bool(b bool) Value { return Value{Kind: KindBool, Bool: b} }

// U32 returns a u32 Amber value.
func U32Val(v uint32) Value { return Value{Kind: KindU32, U32: v} }

// I64 returns an i64 Amber value.
func I64Val(v int64) Value { return Value{Kind: KindI64, I64: v} }

// F32 returns an f32 Amber value. Panics on NaN or Infinity.
func F32Val(v float32) Value {
	if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
		panic(fmt.Sprintf("NaN and Infinity are not valid Amber values: %v", v))
	}
	// Normalize negative zero to positive zero.
	if v == 0 {
		v = 0
	}
	return Value{Kind: KindF32, F32: v}
}

// Str returns a string Amber value (caller should NFC-normalize).
func StrVal(s string) Value { return Value{Kind: KindStr, Str: s} }

// Array returns an array Amber value.
func Array(elements ...Value) Value { return Value{Kind: KindArray, Elements: elements} }

// Object returns an object Amber value.
func Object(fields map[string]Value) Value { return Value{Kind: KindObject, Fields: fields} }

// RefVal returns a reference (fingerprint) Amber value.
func RefVal(fp Fingerprint) Value { return Value{Kind: KindRef, Ref: fp} }
