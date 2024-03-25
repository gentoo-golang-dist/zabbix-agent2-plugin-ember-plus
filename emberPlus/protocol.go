package ember

var _ Encoder = &DefaultCodec{}
var _ Decoder = &DefaultCodec{}

const (
	BOF              = 0xfe
	Slot             = 0x00
	MessageType      = 0x0e
	CommandType      = 0x00
	Version          = 1
	DTDType          = 1
	AppBytes         = 2
	MinorVersion     = 40
	MajorVersion     = 2
	EOF              = 0xff
	EOF16            = 0xffff
	CE               = 0xfd
	FirstMultiPacket = 0x80
	LastMultiPacket  = 0x40
	XORCE            = 0x20
	BOFNE            = 0xf8

	S101LenTilGlow          = 10
	CheckSumLen             = 2
	CheckSumSecondDeviation = 8
	byteSkip                = 1
)

type Decoder interface {
	Decode(message []byte) ([]uint8, error)
}

type Encoder interface {
	Encode(message []byte, packetType uint8) []uint8
}
