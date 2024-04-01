//nolint:gofmt
package ember

const (
	// Parameter types for Glow parameters.

	// ParameterType glow data field parameter type.
	ParameterType = "parameter"
	// QualifiedParameterType glow data field qualified parameter type.
	QualifiedParameterType = "qualified_parameter"
	// QualifiedNodeType glow data field qualified node type.
	QualifiedNodeType = "qualified_node"
	// NodeType glow data field node type.
	NodeType = "node"
	// FunctionType glow data field function type.
	FunctionType = "function"

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

	// GLOW encoding.

	// byte used for writing context.
	contextByte = 0x80
	// byte used for byte OR check for context tag.
	contextOR = 0xa0
	// byte used for byte OR check for application tag.
	applicationOR = 0x60
	// byte used in glow data len decoding.
	lenByte = 0x7F

	// context universal object tag.
	universalObjectTag = 0x0D
	// context integer tag.
	intObjectTag = 0xA0

	// integer to request dir command, based on S101 and glow protocol.
	emberGetDirCommand = 32
	// additional option for dir command, based on S101 and glow protocol.
	dirFieldMaskAll = -1
	// ember encoding int tag.
	emberIntTag = 0x02
	// maximum length of the bytes that describe the data blocks length in glow encoding.
	maxLengthBytes = 4
	// application tag that describes that the glow message is a application command.
	commandApplicationTag = 2
	// tag for defining glow root element collection.
	rootElementTag = 11
	// tag for defining glow element collection tag.
	elementCollectionTag = 4
	// tag for defining glow qualified parameter tag.
	qualifiedParameterTag = 9
	// tag for defining glow qualified node tag.
	qualifiedNodeTag = 10
	// tag for defining glow node tag.
	nodeTag = 3
	// tag for defining glow function tag.
	functionTag = 20
	// tag for defining glow root element collection encoding command.
	rootElementCollectionTag = 0
	// tag for defining glow set tag.
	setByte = 49

	// tag for path value in context.
	pathContextTag = 0
	// tag for properties value in context.
	propertiesContextTag = 1
	// tag for children value in context.
	childrenContextTag = 2
)
