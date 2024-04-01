package ember

import (
	"bytes"
	"encoding/asn1"
	"errors"
	"fmt"

	"git.zabbix.com/ap/plugin-support/errs"
)

// these constants are based on ember+ protocol documentation
const (
	contextByte   = 0x80
	contextOR     = 0xa0
	applicationOR = 0x60
	lenByte       = 0x7F //no clue why

	setByte = 49 //HEX set is 31 I am not sure if this is universal

	universalObjectTag = 0x0D //13
	intObjectTag       = 0xA0 //13

	emberGetDirCommand = 32
	dirFieldMaskAll    = -1

	emberIntTag = 0x02

	maxLengthBytes = 4

	commandApplicationTag = 2

	rootElementTag           = 11
	elementCollectionTag     = 4
	qualifiedParameterTag    = 9
	qualifiedNodeTag         = 10
	nodeTag                  = 3
	functionTag              = 20
	rootElementCollectionTag = 0

	pathContextTag       = 0
	propertiesContextTag = 1
	childrenContextTag   = 2
)

type ASN1Encoder interface {
	Encode()
}

type cntxt struct {
	data *DefaultASN1Codec
	tag  int
}

type DefaultASN1Codec struct {
	emBER bytes.Buffer
	glow  *bytes.Buffer
}

func NewASN1Decoder(b []byte) *DefaultASN1Codec {
	dec := &DefaultASN1Codec{}

	dec.glow = bytes.NewBuffer(b)

	return dec
}

func (c *DefaultASN1Codec) GetData() []byte {
	return c.emBER.Bytes()
}

func (c *DefaultASN1Codec) Read(tag uint8, compareByte func(num uint8) uint8) (*DefaultASN1Codec, error) {
	b, err := c.glow.ReadByte()
	if err != nil {
		return nil, err
	}

	if b != compareByte(tag) {
		return nil, fmt.Errorf("is not correct byte: %x got %x", compareByte(tag), b)
	}

	lenB, _, err := c.ReadLength()
	if err != nil {
		return nil, err
	}

	var out []byte

	for i := 0; i < int(lenB); i++ {
		b, err := c.glow.ReadByte()
		if err != nil {
			return nil, err
		}

		out = append(out, b)
	}

	return NewASN1Decoder(out), nil
}

func (c *DefaultASN1Codec) ReadLength() (int, int, error) {
	var offset int

	lenB, err := c.glow.ReadByte()
	if err != nil {
		return 0, 0, errors.New("failed to read length")
	}

	offset++

	if lenB&contextByte != contextByte {
		return int(lenB), offset, nil
	}

	lenB &= lenByte

	if lenB > maxLengthBytes {
		return 0, 0, errors.New("length higher than 4")
	}

	var len int
	for i := 0; i < int(lenB); i++ {
		val, err := c.glow.ReadByte()
		if err != nil {
			return 0, 0, errors.New("failed to read length")
		}
		len = len<<8 + int(val)
		offset++
	}

	return len, offset, nil
}

func (c *DefaultASN1Codec) ReadAllContext() ([]cntxt, error) {

	var out []cntxt

	for {
		contInt, err := c.Peek()
		if err != nil {
			return nil, err
		}

		d, err := c.Read(uint8(contInt), context)
		if err != nil {
			return nil, err
		}

		switch contInt {
		case context(0):
			out = append(out, cntxt{data: d, tag: 0})
		case context(1):
			out = append(out, cntxt{data: d, tag: 1})
		case context(2):
			out = append(out, cntxt{data: d, tag: 2})
		case context(3):
			out = append(out, cntxt{data: d, tag: 3})
		case context(4):
			out = append(out, cntxt{data: d, tag: 4})
		case context(5):
			out = append(out, cntxt{data: d, tag: 5})
		case context(6):
			out = append(out, cntxt{data: d, tag: 6})
		case context(7):
			out = append(out, cntxt{data: d, tag: 7})
		case context(8):
			out = append(out, cntxt{data: d, tag: 8})
		case context(9):
			out = append(out, cntxt{data: d, tag: 9})
		case context(10):
			out = append(out, cntxt{data: d, tag: 10})
		case context(11):
			out = append(out, cntxt{data: d, tag: 11})
		case context(12):
			out = append(out, cntxt{data: d, tag: 12})
		case context(13):
			out = append(out, cntxt{data: d, tag: 13})
		case context(14):
			out = append(out, cntxt{data: d, tag: 14})
		case context(15):
			out = append(out, cntxt{data: d, tag: 15})
		case context(16):
			out = append(out, cntxt{data: d, tag: 16})
		case context(17):
			out = append(out, cntxt{data: d, tag: 17})
		default:
			return nil, fmt.Errorf("unknown context")
		}

		if c.glow.Len() == 0 {
			return out, nil
		}
	}
}

func (c *DefaultASN1Codec) Peek() (byte, error) {
	b, err := c.glow.ReadByte()
	if err != nil {
		return 0, err
	}

	err = c.glow.UnreadByte()
	if err != nil {
		return 0, err
	}

	return b, nil
}

func (c *DefaultASN1Codec) PeekX(count int) ([]byte, error) {
	var out []byte

	for i := 1; i <= count; i++ {
		b, err := c.glow.ReadByte()
		if err != nil {
			return nil, err
		}

		out = append(out, b)
	}

	for i := 1; i <= count; i++ {
		c.glow.UnreadByte()
	}

	return out, nil
}

func (c *DefaultASN1Codec) ReadSet() ([]cntxt, error) {
	cont, err := c.Read(setByte, universal)
	if err != nil {
		return nil, err
	}

	conts, err := cont.ReadAllContext()
	if err != nil {
		return nil, err
	}

	return conts, nil
}

func (c *DefaultASN1Codec) ReadApplicationByte(tag uint8) (byte, error) {
	rootElemCollByte, err := c.glow.ReadByte()
	if err != nil {
		return 0, err
	}

	if rootElemCollByte != application(tag) {
		return 0, err
	}

	_, err = c.glow.ReadByte()
	if err != nil {
		return 0, err
	}

	return rootElemCollByte, nil
}

func (c *DefaultASN1Codec) ReadContextByte(tag uint8) (byte, error) {
	rootElemCollByte, err := c.glow.ReadByte()
	if err != nil {
		return 0, err
	}

	if rootElemCollByte != application(tag) {
		return 0, err
	}

	_, err = c.glow.ReadByte()
	if err != nil {
		return 0, err
	}

	return rootElemCollByte, nil
}

func (c *DefaultASN1Codec) GetRequest(path []int, tag ElementType) error {
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

	c.EncodeUniversal(path)

	c.openSequence(context(2))
	defer c.closeSequence()

	c.openSequence(application(elementCollectionTag))
	defer c.closeSequence()

	c.WriteGetDirCommand()

	return nil
}

func (c *DefaultASN1Codec) EncodeUniversal(path []int) {
	c.emBER.WriteByte(universalObjectTag)
	c.emBER.WriteByte(uint8(len(path)))

	for _, p := range path {
		c.emBER.WriteByte(uint8(p))
	}
}

func (c *DefaultASN1Codec) DecodeUniversal() ([]int, error) {
	b, err := c.glow.ReadByte()
	if err != nil {
		return nil, err
	}

	if b != universalObjectTag {
		return nil, errors.New("incorrect universal byte")
	}

	lenB, _, err := c.ReadLength()
	if err != nil {
		return nil, err
	}

	var out []int

	for i := 1; i <= lenB; i++ {
		b, err := c.glow.ReadByte()
		if err != nil {
			return nil, err
		}

		out = append(out, int(b))
	}

	return out, nil
}

func (c *DefaultASN1Codec) DecodeInteger() (int, error) {
	t, err := c.glow.ReadByte()
	if err != nil {
		return 0, err
	}

	if t != emberIntTag {
		return 0, errors.New("incorrect integer byte")
	}

	lenB, _, err := c.ReadLength()
	if err != nil {
		return 0, err
	}

	var out int

	for ; lenB > 0; lenB-- {
		b, err := c.glow.ReadByte()
		if err != nil {
			return 0, err
		}

		out = (out << 8) | int(b)
	}

	return out, nil
}

func (c *DefaultASN1Codec) GetRootTreeRequest() {
	c.openSequence(application(rootElementCollectionTag))
	defer c.closeSequence()

	c.openSequence(application(rootElementTag))
	defer c.closeSequence()

	c.WriteGetDirCommand()
}

func (c *DefaultASN1Codec) WriteGetDirCommand() {
	c.openSequence(context(0))
	defer c.closeSequence()

	c.openSequence(application(commandApplicationTag))
	defer c.closeSequence()

	c.writeInt(emberGetDirCommand, 0)
	c.writeInt(dirFieldMaskAll, 1)
}

func (c *DefaultASN1Codec) Encode() {}

func NewAsn1Encoder() ASN1Encoder {
	return &DefaultASN1Codec{}
}

func (c *DefaultASN1Codec) writeInt(i int, cont uint8) error {
	c.emBER.WriteByte(context(cont))

	b, err := asn1.Marshal(i)
	if err != nil {
		return err
	}

	c.emBER.WriteByte(uint8(len(b)))
	c.emBER.Write(b)

	return nil
}

func (c *DefaultASN1Codec) openSequence(appl byte) {
	c.emBER.WriteByte(appl)
	c.emBER.WriteByte(contextByte)
}

func (c *DefaultASN1Codec) closeSequence() error {
	_, err := c.emBER.Write([]byte{0, 0})
	if err != nil {
		return err
	}

	return nil
}

func application(num byte) byte {
	return applicationOR | num
}

func context(num uint8) uint8 {
	return contextOR | num
}

func universal(num uint8) uint8 {
	return num
}

func decodeString(in []byte) (string, error) {
	var out string

	_, err := asn1.Unmarshal(in, &out)
	if err != nil {
		return "", err
	}

	return out, nil
}

func decodeBool(in []byte) (bool, error) {
	var out bool

	_, err := asn1.Unmarshal(in, &out)
	if err != nil {
		return false, err
	}

	return out, nil
}
