package s101

const (
	// Parameter types for Glow parameters.

	// BOF S101 packet opening byte.
	BOF = 0xfe
	// Slot S101 packet Slot byte byte.
	Slot = 0x00
	// MessageType S101 byte defines our required message type.
	MessageType = 0x0e
	// CommandType defines that our S101 message is a command.
	CommandType = 0x00
	// Version defines version plugin uses for S101.
	Version = 1
	// DTDType defines that plugin uses glow for S101 packet payload.
	DTDType = 1
	// AppBytes defines how many application (version) byte will be used before payload.
	AppBytes = 2
	// MinorVersion Glow protocol minor version.
	MinorVersion = 40
	// MajorVersion Glow protocol major version.
	MajorVersion = 2

	// EOF is S101 packet end byte.
	EOF = 0xff
	// CE is S101 packet biggest byte before the byte needs to be XOR.
	CE = 0xfd
	// FirstMultiPacket is byte to identify a multilayer packet in S101.
	FirstMultiPacket = 0x80
	// LastMultiPacket is byte to identify end of a multilayer packet in S101.
	LastMultiPacket = 0x40

	// EOF16 is CRC calculation start byte as crc is calculated from biggest byte to lowest.
	EOF16 = 0xffff

	// XORCE is glow escape byte for bytes that are over an allowed threshold.
	XORCE = 0x20

	// BOFNE is biggest byte before the byte needs to be XOR.
	BOFNE = 0xf8

	// S101LenTilGlow is offset for how many S101 bytes it takes to get till Glow payload.
	S101LenTilGlow = 10
	// CheckSumSecondDeviation is byte used in CRC calculations based on ember+ documentation.
	CheckSumSecondDeviation = 8
	// used to indicate when byte is required to be skipped in different ways.
	byteSkip = 1
)
