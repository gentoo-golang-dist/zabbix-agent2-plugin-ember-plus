package asn1

import (
	"bytes"
	"encoding/asn1"

	"git.zabbix.com/ap/plugin-support/errs"
)

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

	// RootElementCollectionTag tag for defining glow root element collection encoding command.
	RootElementCollectionTag = 0
	// RootElementTag tag for defining glow root element collection.
	RootElementTag = 11
	// ElementCollectionTag tag for element collection
	ElementCollectionTag = 4
	// ContextZeroTag tag for top level context
	ContextZeroTag = 0
	// ContextTagOne tag for context
	// Node means data contains content.
	ContextTagOne = 1
	// ContextTagOne tag for context
	// Node means data contains children.
	ContextTagTwo = 2
	// tag for defining glow qualified parameter tag.
	QualifiedParameterTag = 9
	// tag for defining glow qualified node tag.
	QualifiedNodeTag = 10

	// SetTag tag for defining Ember set structure.
	SetTag = 49

	// GLOW encoding.

	// UniversalObjectTag context universal object tag.
	UniversalObjectTag = 0x0D
	// IntObjectTag context integer tag.
	IntObjectTag = 0xA0
	// byte used for writing context.
	contextByte = 0x80
	// byte used for byte OR check for context tag.
	contextOR = 0xa0
	// byte used for byte OR check for application tag.
	applicationOR = 0x60
	// byte used in glow data len decoding.
	lenByte = 0x7F

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
	// tag for defining glow element collection tag.
	elementCollectionTag = 4
	// tag for defining glow function tag.
	functionTag = 20

	// tag for defining glow offset when reading all values.
	closingOffset = 2
	// closingByte defines the closing byte of any glow element.
	closingByte = 0x00
)

var noLenByteErr = errs.New("can not determine length")

// Decoder decoder for ASN1 glow data.
type Decoder struct {
	data *bytes.Buffer
}

// Encoder encoder ASN1 glow data.
type Encoder struct {
	data *bytes.Buffer
}

// Context contains decoder for context and it's tag for easy of use.
type Context struct {
	Value *Decoder
	Tag   int
}

type ByteCheckFunc func(byte) byte

// Bytes wrapper to containing decoders data bytes
func (c *Decoder) Bytes() []byte {
	if c == nil || c.data == nil {
		return nil
	}

	return c.data.Bytes()
}

// Bytes wrapper to containing decoders data bytes
func (c *Decoder) Len() int {
	if c == nil || c.data == nil {
		return 0
	}

	return c.data.Len()
}

// Bytes wrapper to containing decoders data bytes
func (c *Context) Bytes() []byte {
	if c == nil || c.Value == nil {
		return nil
	}

	return c.Value.data.Bytes()
}

// NewDecoder creates a new ASN1 Decoder.
func NewDecoder(b []byte) *Decoder {
	return &Decoder{bytes.NewBuffer(b)}
}

// NewEncoder creates a new encoder with an initialized data buffer, but no actual data.
func NewEncoder() *Encoder {
	return &Encoder{bytes.NewBuffer(nil)}
}

// wrappers on overlapping asn1 functionality from native go ASN1 package.

// DecodeString decodes asn1 string to readable string
func DecodeString(in []byte) (string, int, error) {
	var out string

	r, err := asn1.Unmarshal(in, &out)
	if err != nil {
		return "", 0, errs.Wrap(err, "failed to unmarshal go native asn1 string")
	}

	return out, len(r), nil
}

// DecodeAny decodes native asn1 value
func DecodeAny(in []byte, val any) (int, error) {
	slen := len(in)

	r, err := asn1.Unmarshal(in, val)
	if err != nil {
		return 0, errs.Wrap(err, "failed to unmarshal go native asn1 value")
	}

	return slen - len(r), nil
}

// DecodeBool decodes asn1 bool to readable boolean
func DecodeBool(in []byte) (bool, int, error) {
	var out bool

	r, err := asn1.Unmarshal(in, &out)
	if err != nil {
		return false, 0, errs.Wrap(err, "failed to unmarshal go native asn1 bool")
	}

	return out, len(r), nil
}

// tag types used in ember+ glow protocol.

// ApplicationByte have the same meaning wherever they are seen and used.
func ApplicationByte(num uint8) uint8 {
	return applicationOR | num
}

// ContextByte context-specific tags depends on the location where they are seen.
func ContextByte(num uint8) uint8 {
	return contextOR | num
}

// UniversalByte predefined types, the value is returned unchanged.
func UniversalByte(num uint8) uint8 {
	return num
}
