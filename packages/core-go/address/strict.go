package address

import "strings"

// StrKeyEncodedLength is the exact encoded length, in characters, of each
// strkey kind:
//
//   - G and C: version byte + 32-byte key + 2-byte CRC16 = 35 bytes = 56 chars
//   - M: version byte + 32-byte key + 8-byte ID + 2-byte CRC16 = 43 bytes = 69 chars
var StrKeyEncodedLength = map[AddressKind]int{
	KindG: 56,
	KindM: 69,
	KindC: 56,
}

// DecodeStrKeyStrict decodes a G, M or C strkey and never panics.
//
// Checks run in a fixed order shared with core-ts and core-dart, so every
// implementation reports the same code for the same input:
//
//  1. empty input                          -> INVALID_LENGTH
//  2. first character not G/M/C            -> UNKNOWN_PREFIX
//  3. length differs from the kind's size  -> INVALID_LENGTH
//     (truncated or over-long input is rejected before any slicing)
//  4. character outside the base32 alphabet or non-zero trailing bits
//     -> INVALID_BASE32
//  5. CRC16 mismatch                       -> INVALID_CHECKSUM
//
// On success it returns the kind, the canonical (uppercase) address and the
// payload without version byte or checksum. Errors are RoutingError values.
func DecodeStrKeyStrict(input string) (kind AddressKind, canonical string, payload []byte, err error) {
	fail := func(code ErrorCode, msg string) (AddressKind, string, []byte, error) {
		return "", "", nil, RoutingError{Code: code, Input: input, Message: msg}
	}

	if len(input) == 0 {
		return fail(ErrInvalidLength, "address is empty")
	}

	canonical = strings.ToUpper(input)

	kind = AddressKind(canonical[:1])
	want, known := StrKeyEncodedLength[kind]
	if !known {
		return fail(ErrUnknownPrefix, "unknown address prefix")
	}
	if len(canonical) != want {
		return fail(ErrInvalidLength, "invalid address length")
	}

	versionByte, payload, _, err := decodeStrKey(canonical)
	if err != nil {
		switch {
		case err == errInvalidChecksum:
			return fail(ErrInvalidChecksum, "invalid checksum")
		case err == errInvalidBase32:
			return fail(ErrInvalidBase32, "invalid base32 encoding")
		case err == errInvalidLength:
			return fail(ErrInvalidLength, "invalid address length")
		default:
			return fail(ErrUnknownPrefix, "unknown address prefix")
		}
	}
	if k, kerr := kindForVersionByte(versionByte); kerr != nil || k != kind {
		return fail(ErrUnknownPrefix, "version byte does not match prefix")
	}

	return kind, canonical, payload, nil
}
