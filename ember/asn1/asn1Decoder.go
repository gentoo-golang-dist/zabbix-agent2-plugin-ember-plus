package asn1

import (
	"bytes"
	"errors"

	"git.zabbix.com/ap/plugin-support/errs"
)

// Read reads the next glow data block of the appropriate type, it checks the glow tag against the provided compare
// function and if they match it reads the glow data block and returns it as it's own decoded, original decoder might
// have more data left, THIS DOES NOT READ ALL THE DATA.
// If no length byte is found in data returns ALL remaining bytes.
// Returns True if next element length is unknown.
func (c *Decoder) Read(tag uint8, compareByte func(num uint8) uint8) (*Decoder, bool, error) {
	b, err := c.data.ReadByte()
	if err != nil {
		return nil, false, errs.Wrap(err, "failed to read tag byte")
	}

	if b != compareByte(tag) {
		return nil, false, errs.Errorf("is not correct byte: %x got %x", compareByte(tag), b)
	}

	lenB, _, err := c.ReadLength()

	if err != nil {
		if !errors.Is(err, noLenByteErr) {
			return nil, false, errs.Wrap(err, "failed to read length byte")
		}

		var out []byte

		out, err = c.readWithOutLength()
		if err != nil {
			return nil, false, errs.Wrap(err, "failed to read with out provided length")
		}

		return NewDecoder(out), true, nil
	}

	out, err := c.readWithLength(lenB)
	if err != nil {
		return nil, false, errs.Wrap(err, "failed to read with provided length")
	}

	return NewDecoder(out), false, nil
}

// ReadLength reads next in line data blocks length and returns it as well as how many bytes the data
// length was written in.
func (c *Decoder) ReadLength() (int, int, error) {
	var offset int

	lenB, err := c.data.ReadByte()
	if err != nil {
		return 0, 0, errs.Wrap(err, "incorrect length byte")
	}

	offset++

	if lenB&contextByte != contextByte {
		return int(lenB), offset, nil
	}

	lenB &= lenByte

	if lenB == 0 {
		return 0, offset, noLenByteErr
	}

	if lenB > maxLengthBytes {
		return 0, 0, errs.New("length higher than 4")
	}

	var out int

	for i := 0; i < int(lenB); i++ {
		val, err := c.data.ReadByte()
		if err != nil {
			return 0, 0, errs.Wrap(err, "incorrect additional length bytes")
		}

		out = out<<8 + int(val)

		offset++
	}

	return out, offset, nil
}

// ReadEnd checks if decoder is currently at the end of element and moves the reader over it, if at end.
func (c *Decoder) ReadEnd() (bool, error) {
	if c.data.Len() == 0 {
		return true, nil
	}

	if c.data.Len() < 2 {
		return false, errs.New("not enough bytes")
	}

	tmp := bytes.NewBuffer(c.Bytes())
	for i, b := range tmp.Bytes() {
		if b != closingByte {
			return false, nil
		}

		if i+1 == closingOffset {
			break
		}
	}

	for i := 0; i < closingOffset; i++ {
		_, err := c.data.ReadByte()
		if err != nil {
			return false, errs.Wrapf(err, "failed to read end bytes")
		}
	}

	return true, nil
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

// ReadByte reads one byte from the underlining bytes buffer in decoder.
func (c *Decoder) ReadByte() (byte, error) {
	b, err := c.data.ReadByte()
	if err != nil {
		return 0, errs.Wrap(err, "failed to read byte")
	}

	return b, nil
}

func (c *Decoder) readWithOutLength() ([]byte, error) {
	var out []byte

	for c.data.Len() > 0 {
		b, err := c.data.ReadByte()
		if err != nil {
			return nil, errs.Wrap(err, "failed to read byte")
		}

		out = append(out, b)
	}

	return out, nil
}

func (c *Decoder) readWithLength(length int) ([]byte, error) {
	var out []byte

	for i := 0; i < length; i++ {
		b, err := c.data.ReadByte()
		if err != nil {
			return nil, errs.Wrap(err, "failed to read extra bytes")
		}

		out = append(out, b)
	}

	return out, nil
}
