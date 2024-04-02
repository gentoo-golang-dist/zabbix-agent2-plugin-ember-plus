package ember

import (
	"bytes"
	"encoding/asn1"

	"git.zabbix.com/ap/plugin-support/errs"
)

// ASN1Decoder decoder for ASN1 glow data.
type ASN1Decoder struct {
	data *bytes.Buffer
}

// Context contains decoder for context and it's tag for easy of use.
type Context struct {
	value *ASN1Decoder
	tag   int
}

// NewASN1Decoder creates a new ASN1 Decoder.
func NewASN1Decoder(b []byte) *ASN1Decoder {
	return &ASN1Decoder{bytes.NewBuffer(b)}
}

// Read reads the next glow data block of the appropriate type, it checks the glow tag against the provided compare
// function and if they match it reads the glow data block and returns it as it's own decoded, original decoder might
// have more data left, THIS DOES NOT READ ALL THE DATA.
func (c *ASN1Decoder) Read(tag uint8, compareByte func(num uint8) uint8) (*ASN1Decoder, error) {
	b, err := c.data.ReadByte()
	if err != nil {
		return nil, errs.Wrap(err, "failed to read tag byte")
	}

	if b != compareByte(tag) {
		return nil, errs.Errorf("is not correct byte: %x got %x", compareByte(tag), b)
	}

	lenB, _, err := c.ReadLength()
	if err != nil {
		return nil, errs.Wrap(err, "failed to read length byte")
	}

	var out []byte

	for i := 0; i < lenB; i++ {
		b, err := c.data.ReadByte()
		if err != nil {
			return nil, errs.Wrap(err, "failed to read extra length bytes")
		}

		out = append(out, b)
	}

	return NewASN1Decoder(out), nil
}

// ReadLength reads next in line data blocks length and returns it as well ass how many bytes the data
// length was written in.
func (c *ASN1Decoder) ReadLength() (int, int, error) {
	var offset int

	lenB, err := c.data.ReadByte()
	if err != nil {
		return 0, 0, errs.Wrap(err, "failed to read length")
	}

	offset++

	if lenB&contextByte != contextByte {
		return int(lenB), offset, nil
	}

	lenB &= lenByte

	if lenB > maxLengthBytes {
		return 0, 0, errs.New("length higher than 4")
	}

	var out int

	for i := 0; i < int(lenB); i++ {
		val, err := c.data.ReadByte()
		if err != nil {
			return 0, 0, errs.Wrap(err, "failed to read length")
		}

		out = out<<8 + int(val)

		offset++
	}

	return out, offset, nil
}

// ReadAllContext returns all the following contexts as new decoders.
//
//nolint:gocyclo,cyclop
func (c *ASN1Decoder) ReadAllContext() ([]Context, error) {
	var out []Context

	for {
		contInt, err := c.Peek()
		if err != nil {
			return nil, err
		}

		d, err := c.Read(contInt, context)
		if err != nil {
			return nil, err
		}

		// currently bases on ember+ documentation there are 0-17 contexts

		switch contInt {
		case context(0):
			out = append(out, Context{value: d, tag: 0})
		case context(1):
			out = append(out, Context{value: d, tag: 1})
		case context(2):
			out = append(out, Context{value: d, tag: 2})
		case context(3):
			out = append(out, Context{value: d, tag: 3})
		case context(4):
			out = append(out, Context{value: d, tag: 4})
		case context(5):
			out = append(out, Context{value: d, tag: 5})
		case context(6):
			out = append(out, Context{value: d, tag: 6})
		case context(7):
			out = append(out, Context{value: d, tag: 7})
		case context(8):
			out = append(out, Context{value: d, tag: 8})
		case context(9):
			out = append(out, Context{value: d, tag: 9})
		case context(10):
			out = append(out, Context{value: d, tag: 10})
		case context(11):
			out = append(out, Context{value: d, tag: 11})
		case context(12):
			out = append(out, Context{value: d, tag: 12})
		case context(13):
			out = append(out, Context{value: d, tag: 13})
		case context(14):
			out = append(out, Context{value: d, tag: 14})
		case context(15):
			out = append(out, Context{value: d, tag: 15})
		case context(16):
			out = append(out, Context{value: d, tag: 16})
		case context(17):
			out = append(out, Context{value: d, tag: 17})
		default:
			return nil, errs.New("unknown context")
		}

		if c.data.Len() == 0 {
			return out, nil
		}
	}
}

// Peek returns the next byte, but does not remove it from the buffer.
func (c *ASN1Decoder) Peek() (byte, error) {
	b, err := c.data.ReadByte()
	if err != nil {
		return 0, errs.Wrap(err, "failed to read a byte")
	}

	err = c.data.UnreadByte()
	if err != nil {
		return 0, errs.Wrap(err, "failed to unread a byte")
	}

	return b, nil
}

// ReadSet returns all following contexts contained in a Set.
func (c *ASN1Decoder) ReadSet() ([]Context, error) {
	set, err := c.Read(setByte, universal)
	if err != nil {
		return nil, errs.Wrap(err, "failed to set")
	}

	conts, err := set.ReadAllContext()
	if err != nil {
		return nil, errs.Wrap(err, "failed to contexts")
	}

	return conts, nil
}

// DecodeUniversal decoded the following universal data type of glow, currently only used for universal path decoding,
// witch is an array of integers.
func (c *ASN1Decoder) DecodeUniversal() ([]int, error) {
	b, err := c.data.ReadByte()
	if err != nil {
		return nil, errs.Wrap(err, "failed to read tag byte")
	}

	if b != universalObjectTag {
		return nil, errs.New("incorrect universal byte")
	}

	lenB, _, err := c.ReadLength()
	if err != nil {
		return nil, errs.Wrap(err, "failed to read len byte")
	}

	var out []int

	for i := 1; i <= lenB; i++ {
		b, err := c.data.ReadByte()
		if err != nil {
			return nil, errs.Wrap(err, "failed to read extra len bytes")
		}

		out = append(out, int(b))
	}

	return out, nil
}

// DecodeInteger decodes the following integer.
func (c *ASN1Decoder) DecodeInteger() (int, error) {
	t, err := c.data.ReadByte()
	if err != nil {
		return 0, errs.Wrap(err, "failed to read tag byte")
	}

	if t != emberIntTag {
		return 0, errs.New("incorrect integer byte")
	}

	lenB, _, err := c.ReadLength()
	if err != nil {
		return 0, errs.Wrap(err, "failed to read len byte")
	}

	var out int

	for ; lenB > 0; lenB-- {
		b, err := c.data.ReadByte()
		if err != nil {
			return 0, errs.Wrap(err, "failed to read extra len bytes")
		}

		out = (out << 8) | int(b)
	}

	return out, nil
}

// tag types used in decoder.Read.
func application(num uint8) uint8 {
	return applicationOR | num
}

func context(num uint8) uint8 {
	return contextOR | num
}

func universal(num uint8) uint8 {
	return num
}

// wrappers on overlapping asn1 functionality from native go ASN1 package.

func decodeString(in []byte) (string, error) {
	var out string

	_, err := asn1.Unmarshal(in, &out)
	if err != nil {
		return "", errs.Wrap(err, "failed to unmarshal go native asn1 string")
	}

	return out, nil
}

func decodeBool(in []byte) (bool, error) {
	var out bool

	_, err := asn1.Unmarshal(in, &out)
	if err != nil {
		return false, errs.Wrap(err, "failed to unmarshal go native asn1 bool")
	}

	return out, nil
}
