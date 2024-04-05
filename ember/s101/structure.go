package s101

const (
	// FirstMultiPacket is byte to identify a multilayer packet in S101.
	FirstMultiPacket = 0x80

	// LastMultiPacket is byte to identify end of a multilayer packet in S101.
	lastMultiPacket = 0x40
	// BOF S101 packet opening byte.
	bof = 0xfe
	// Slot S101 packet Slot byte byte.
	slot = 0x00
	// messageType S101 byte defines our required message type.
	messageType = 0x0e
	// commandType defines that our S101 message is a command.
	commandType = 0x00
	// version defines version plugin uses for S101.
	version = 1
	// dtdType defines that plugin uses glow for S101 packet payload.
	dtdType = 1
	// appBytes defines how many application (version) byte will be used before payload.
	appBytes = 2
	// minorVersion Glow protocol minor version.
	minorVersion = 40
	// majorVersion Glow protocol major version.
	majorVersion = 2
	// eof is S101 packet end byte.
	eof = 0xff
	// ce is S101 packet biggest byte before the byte needs to be XOR.
	ce = 0xfd
	// eof16 is CRC calculation start byte as crc is calculated from biggest byte to lowest.
	eof16 = 0xffff
	// xorce is glow escape byte for bytes that are over an allowed threshold.
	xorce = 0x20
	// bofne is biggest byte before the byte needs to be XOR.
	bofne = 0xf8

	// s101LenTilGlow is offset for how many S101 bytes it takes to get till Glow payload.
	s101LenTilGlow = 10
	// checkSumSecondDeviation is byte used in CRC calculations based on ember+ documentation.
	checkSumSecondDeviation = 8
	// byteSkip used to indicate when byte is required to be skipped in different ways.
	byteSkip = 1
)
