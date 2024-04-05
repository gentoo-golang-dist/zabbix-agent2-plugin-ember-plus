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
	// tag for defining glow qualified parameter tag.
	QualifiedParameterTag = 9
	// tag for defining glow qualified node tag.
	QualifiedNodeTag = 10

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
	// // tag for defining glow node tag.
	// nodeTag = 3
	// tag for defining glow function tag.
	functionTag = 20
	// tag for defining glow set tag.
	setByte = 49
)

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
func DecodeString(in []byte) (string, error) {
	var out string

	_, err := asn1.Unmarshal(in, &out)
	if err != nil {
		return "", errs.Wrap(err, "failed to unmarshal go native asn1 string")
	}

	return out, nil
}

// DecodeBool decodes asn1 bool to readable boolean
func DecodeBool(in []byte) (bool, error) {
	var out bool

	_, err := asn1.Unmarshal(in, &out)
	if err != nil {
		return false, errs.Wrap(err, "failed to unmarshal go native asn1 bool")
	}

	return out, nil
}
