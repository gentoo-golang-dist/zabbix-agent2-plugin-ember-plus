package ember

import (
	"bytes"
	"encoding/asn1"

	"git.zabbix.com/ap/plugin-support/errs"
)

// ASN1Encoder encoder ASN1 glow data.
type ASN1Encoder struct {
	data *bytes.Buffer
}

// GetData returns all data contained in the encoder.
func (c *ASN1Encoder) GetData() []byte {
	return c.data.Bytes()
}

// WriteRequest writes a request into the encoder buffer, for the provided element type, currently supports parameters,
// qualified parameters, nodes qualified nodes and functions.
func (c *ASN1Encoder) WriteRequest(path []int, tag ElementType) error {
	c.openSequence(application(rootElementCollectionTag))
	defer c.closeSequence()

	c.openSequence(application(rootElementTag))
	defer c.closeSequence()

	c.openSequence(context(0))
	defer c.closeSequence()

	switch tag {
	case ParameterType, QualifiedParameterType:
		c.openSequence(application(qualifiedParameterTag))
	case NodeType, QualifiedNodeType:
		c.openSequence(application(qualifiedNodeTag))
	case FunctionType:
		c.openSequence(application(functionTag))
	default:
		return errs.Errorf("unknown application tag %s", tag)
	}

	defer c.closeSequence()

	c.openSequence(context(0))
	defer c.closeSequence()

	c.WriteUniversal(path)

	c.openSequence(context(2))
	defer c.closeSequence()

	c.openSequence(application(elementCollectionTag))
	defer c.closeSequence()

	err := c.WriteGetDirCommand()
	if err != nil {
		return errs.Wrap(err, "failed to writer dir command")
	}

	return nil
}

// WriteUniversal writes the provided integer into the buffer as an glow encoded universal value.
func (c *ASN1Encoder) WriteUniversal(path []int) {
	c.data.WriteByte(universalObjectTag)
	c.data.WriteByte(uint8(len(path)))

	for _, p := range path {
		c.data.WriteByte(uint8(p))
	}
}

// WriteRootTreeRequest writes a request for root element collection into the buffer.
func (c *ASN1Encoder) WriteRootTreeRequest() error {
	c.openSequence(application(rootElementCollectionTag))
	defer c.closeSequence()

	c.openSequence(application(rootElementTag))
	defer c.closeSequence()

	err := c.WriteGetDirCommand()
	if err != nil {
		return errs.Wrap(err, "failed to write command request")
	}

	return nil
}

// WriteGetDirCommand writes a get dir command request into the buffer.
func (c *ASN1Encoder) WriteGetDirCommand() error {
	c.openSequence(context(0))
	defer c.closeSequence()

	c.openSequence(application(commandApplicationTag))
	defer c.closeSequence()

	err := c.writeInt(emberGetDirCommand, 0)
	if err != nil {
		return errs.Wrap(err, "failed dir write int")
	}

	err = c.writeInt(dirFieldMaskAll, 1)
	if err != nil {
		return errs.Wrap(err, "failed to write dir field mask int")
	}

	return nil
}

// NewASN1Encoder creates a new encoder with an initialized data buffer, but no actual data.
func NewASN1Encoder() *ASN1Encoder {
	return &ASN1Encoder{bytes.NewBuffer(nil)}
}

// writeInt writes integer to the buffer, wraps native go asn1 marshal, but adds context.
func (c *ASN1Encoder) writeInt(i int, cont uint8) error {
	c.data.WriteByte(context(cont))

	b, err := asn1.Marshal(i)
	if err != nil {
		return errs.Wrap(err, "failed native go int asn1 marshal")
	}

	c.data.WriteByte(uint8(len(b)))
	c.data.Write(b)

	return nil
}

// openSequence writes provided application byte together with a context byte (0x80) into the buffer.
func (c *ASN1Encoder) openSequence(appl byte) {
	c.data.WriteByte(appl)
	c.data.WriteByte(contextByte)
}

// closeSequence writes two '0' bytes into the buffer, used to identify end of a sequence.
func (c *ASN1Encoder) closeSequence() {
	c.data.Write([]byte{0, 0})
}
