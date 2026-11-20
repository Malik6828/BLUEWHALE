import 'dart:typed_data';

import 'codes.dart';
import '../util/strkey.dart';

/// Exact encoded length, in characters, of each strkey kind.
///
/// G and C: version byte + 32-byte key + 2-byte CRC16 = 35 bytes = 56 chars.
/// M: version byte + 32-byte key + 8-byte ID + 2-byte CRC16 = 43 bytes =
/// 69 chars.
const Map<AddressKind, int> strKeyEncodedLength = {
  AddressKind.g: 56,
  AddressKind.m: 69,
  AddressKind.c: 56,
};

const Map<String, AddressKind> _kindByPrefix = {
  'G': AddressKind.g,
  'M': AddressKind.m,
  'C': AddressKind.c,
};

const Map<AddressKind, int> _versionByte = {
  AddressKind.g: 6 << 3,
  AddressKind.m: 12 << 3,
  AddressKind.c: 2 << 3,
};

/// Result of [decodeStrKey]. Exactly one of [kind] or [error] is non-null.
class StrKeyDecodeResult {
  /// The decoded address kind, or `null` on failure.
  final AddressKind? kind;

  /// The canonical (uppercase) address, or `null` on failure.
  final String? address;

  /// The payload without version byte or checksum, or `null` on failure.
  final Uint8List? payload;

  /// An [ErrorCode] value on failure, otherwise `null`.
  final String? error;

  const StrKeyDecodeResult._(
      {this.kind, this.address, this.payload, this.error});

  /// Whether decoding succeeded.
  bool get ok => error == null;

  @override
  String toString() => ok
      ? 'StrKeyDecodeResult(kind: $kind, address: $address)'
      : 'StrKeyDecodeResult(error: $error)';
}

StrKeyDecodeResult _fail(String code) => StrKeyDecodeResult._(error: code);

/// Strictly decodes a G, M or C strkey. Never throws.
///
/// Checks run in a fixed order shared with core-ts and core-go, so every
/// implementation reports the same [ErrorCode] for the same input:
///
/// 1. empty input                          -> `INVALID_LENGTH`
/// 2. first character not G/M/C            -> `UNKNOWN_PREFIX`
/// 3. length differs from the kind's size  -> `INVALID_LENGTH`
///    (truncated or over-long input is rejected before any slicing)
/// 4. character outside the base32 alphabet or non-zero trailing bits
///                                         -> `INVALID_BASE32`
/// 5. CRC16 mismatch                       -> `INVALID_CHECKSUM`
StrKeyDecodeResult decodeStrKey(String input) {
  if (input.isEmpty) return _fail(ErrorCode.invalidLength);

  final address = input.toUpperCase();
  final kind = _kindByPrefix[address[0]];
  if (kind == null) return _fail(ErrorCode.unknownPrefix);
  if (address.length != strKeyEncodedLength[kind]) {
    return _fail(ErrorCode.invalidLength);
  }

  final Uint8List bytes;
  try {
    bytes = StrKeyUtil.decodeBase32(address);
  } on FormatException {
    return _fail(ErrorCode.invalidBase32);
  }

  if (bytes[0] != _versionByte[kind]) return _fail(ErrorCode.unknownPrefix);

  final n = bytes.length;
  final crc =
      StrKeyUtil.calculateChecksum(Uint8List.sublistView(bytes, 0, n - 2));
  if (bytes[n - 2] != (crc & 0xFF) || bytes[n - 1] != (crc >> 8)) {
    return _fail(ErrorCode.invalidChecksum);
  }

  return StrKeyDecodeResult._(
    kind: kind,
    address: address,
    payload: Uint8List.fromList(bytes.sublist(1, n - 2)),
  );
}
