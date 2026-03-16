package encoding

import (
	"encoding/hex"
	"fmt"

	"lukechampine.com/blake3"
)

// Fingerprint is a 32-byte BLAKE3 hash represented as a 64-char hex string.
// In source code it is written with a '#' prefix: #a3f9c2...
type Fingerprint struct {
	Bytes [32]byte
}

// String returns the '#'-prefixed hex representation.
func (f Fingerprint) String() string {
	return "#" + hex.EncodeToString(f.Bytes[:])
}

// FingerprintOf computes the BLAKE3 fingerprint of an Amber value.
func FingerprintOf(v Value) Fingerprint {
	return FingerprintBytes(Encode(v))
}

// FingerprintBytes computes the BLAKE3 fingerprint of raw bytes.
func FingerprintBytes(data []byte) Fingerprint {
	digest := blake3.Sum256(data)
	return Fingerprint{Bytes: digest}
}

// ParseFingerprint parses a '#'-prefixed hex fingerprint string.
func ParseFingerprint(s string) (Fingerprint, error) {
	if len(s) == 0 || s[0] != '#' {
		return Fingerprint{}, fmt.Errorf("fingerprint must start with '#': %q", s)
	}
	raw := s[1:]
	if len(raw) != 64 {
		return Fingerprint{}, fmt.Errorf("fingerprint must be 64 hex chars, got %d: %q", len(raw), s)
	}
	b, err := hex.DecodeString(raw)
	if err != nil {
		return Fingerprint{}, fmt.Errorf("invalid fingerprint hex: %w", err)
	}
	var fp Fingerprint
	copy(fp.Bytes[:], b)
	return fp, nil
}
