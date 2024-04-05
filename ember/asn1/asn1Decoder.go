package asn1

import (
	"git.zabbix.com/ap/plugin-support/errs"
)

// Read reads the next glow data block of the appropriate type, it checks the glow tag against the provided compare
// function and if they match it reads the glow data block and returns it as it's own decoded, original decoder might
// have more data left, THIS DOES NOT READ ALL THE DATA.
func (c *Decoder) Read(tag uint8, compareByte func(num uint8) uint8) (*Decoder, error) {
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

	return NewDecoder(out), nil
}

// ReadLength reads next in line data blocks length and returns it as well as how many bytes the data
// length was written in.
func (c *Decoder) ReadLength() (int, int, error) {
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
func (c *Decoder) ReadAllContext() ([]Context, error) {
	var out []Context

	for c.data.Len() > 0 {
		contInt, err := c.Peek()
		if err != nil {
			return nil, err
		}

		d, err := c.Read(contInt, ContextByte)
		if err != nil {
			return nil, err
		}

		// currently bases on ember+ documentation there are 0-17 contexts

		switch contInt {
		case ContextByte(0):
			out = append(out, Context{Value: d, Tag: 0})
		case ContextByte(1):
			out = append(out, Context{Value: d, Tag: 1})
		case ContextByte(2):
			out = append(out, Context{Value: d, Tag: 2})
		case ContextByte(3):
			out = append(out, Context{Value: d, Tag: 3})
		case ContextByte(4):
			out = append(out, Context{Value: d, Tag: 4})
		case ContextByte(5):
			out = append(out, Context{Value: d, Tag: 5})
		case ContextByte(6):
			out = append(out, Context{Value: d, Tag: 6})
		case ContextByte(7):
			out = append(out, Context{Value: d, Tag: 7})
		case ContextByte(8):
			out = append(out, Context{Value: d, Tag: 8})
		case ContextByte(9):
			out = append(out, Context{Value: d, Tag: 9})
		case ContextByte(10):
			out = append(out, Context{Value: d, Tag: 10})
		case ContextByte(11):
			out = append(out, Context{Value: d, Tag: 11})
		case ContextByte(12):
			out = append(out, Context{Value: d, Tag: 12})
		case ContextByte(13):
			out = append(out, Context{Value: d, Tag: 13})
		case ContextByte(14):
			out = append(out, Context{Value: d, Tag: 14})
		case ContextByte(15):
			out = append(out, Context{Value: d, Tag: 15})
		case ContextByte(16):
			out = append(out, Context{Value: d, Tag: 16})
		case ContextByte(17):
			out = append(out, Context{Value: d, Tag: 17})
		default:
			return nil, errs.Errorf("unknown context: %d", contInt)
		}
	}

	return out, nil
}

// Peek returns the next byte, but does not remove it from the buffer.
func (c *Decoder) Peek() (byte, error) {
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
func (c *Decoder) ReadSet() ([]Context, error) {
	set, err := c.Read(setByte, UniversalByte)
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
func (c *Decoder) DecodeUniversal() ([]int, error) {
	b, err := c.data.ReadByte()
	if err != nil {
		return nil, errs.Wrap(err, "failed to read tag byte")
	}

	if b != UniversalObjectTag {
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
func (c *Decoder) DecodeInteger() (int, error) {
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
func ApplicationByte(num uint8) uint8 {
	return applicationOR | num
}

func ContextByte(num uint8) uint8 {
	return contextOR | num
}

func UniversalByte(num uint8) uint8 {
	return num
}
