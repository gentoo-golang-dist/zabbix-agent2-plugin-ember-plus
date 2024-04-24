/*
** Zabbix
** Copyright (C) 2001-2024 Zabbix SIA
**
** This program is free software; you can redistribute it and/or modify
** it under the terms of the GNU General Public License as published by
** the Free Software Foundation; either version 2 of the License, or
** (at your option) any later version.
**
** This program is distributed in the hope that it will be useful,
** but WITHOUT ANY WARRANTY; without even the implied warranty of
** MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
** GNU General Public License for more details.
**
** You should have received a copy of the GNU General Public License
** along with this program; if not, write to the Free Software
** Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
**/

package s101

import (
	"bytes"

	"git.zabbix.com/ap/ember-plus/ember/asn1"
	"git.zabbix.com/ap/plugin-support/errs"
)

// Encode creates a 101 packet from the message adding all the required S101 bytes based on the S101 protocol, if
// package type is multi packet message, adds an empty packet to the end as require by the protocol.
func Encode(message []byte, packetType uint8) []uint8 {
	out := createS101(message, packetType)
	if packetType == FirstMultiPacket {
		out = append(out, createS101([]byte{}, lastMultiPacket)...)
	}

	return out
}

// Decode removes all the S101 addons from the packet returning only glow data, currently does not check CRC.
func Decode(message []byte) ([]uint8, error) {
	s101 := getS101(message)
	if len(s101) < s101LenTilGlow+byteSkip+1 {
		return nil, errs.Errorf("malformed s101 packet, malformed s101 data: %x", s101)
	}

	d := asn1.NewDecoder(s101[s101LenTilGlow+byteSkip:])

	l, offset, err := d.ReadLength()
	if err != nil {
		return nil, errs.Wrap(err, "failed to get glow length")
	}

	var ceFound bool

	//nolint:prealloc
	var out []uint8

	for _, b := range s101 {
		if b == ce {
			ceFound = true

			continue
		}

		if ceFound {
			ceFound = false

			out = append(out, xorce^b)

			continue
		}

		out = append(out, b)
	}

	// needs CRC check

	if len(out)+1 < s101LenTilGlow || len(out)+1 < s101LenTilGlow+offset+l+byteSkip {
		return nil, errs.Errorf("failed to get glow data, malformed payload %x", out)
	}

	return out[s101LenTilGlow : s101LenTilGlow+offset+l+byteSkip], nil
}

// getS101 reads the last entry in the byte array start starts with BOF byte and ends with EOF byte.
func getS101(in []uint8) []uint8 {
	var start, end int

	var sFound, eFound bool

	for i, b := range in {
		if b == bof {
			start = i
			sFound = true

			continue
		}

		if b == eof {
			end = i
			eFound = true

			continue
		}
	}

	if !sFound || !eFound {
		return []byte{}
	}

	return in[start : end+1]
}

// createS101 creates a S101 packet from the provided payload and packet type.
func createS101(payload []byte, pType uint8) []byte {
	escaped := escapeBytesAboveBOFNE(payload)
	s101Info := []byte{slot, messageType, commandType, version, pType, dtdType, appBytes, minorVersion, majorVersion}
	tmp := make([]byte, 0, len(s101Info)+len(payload))

	tmp = append(tmp, s101Info...)
	crc := getCRC(append(tmp, payload...))

	s101 := make([]byte, 0, len(s101Info)+len(escaped)+len(crc)+2)
	s101 = append(s101, bof)
	s101 = append(s101, s101Info...)
	s101 = append(s101, escaped...)
	s101 = append(s101, crc...)
	s101 = append(s101, eof)

	return s101
}

// escapeBytesAboveBOFNE parses the message as based on Glow protocol all the bytes with bigger value then 0xf8 must
// preceded with and 0xfd byte and XORed with 0x20 byte.
func escapeBytesAboveBOFNE(message []byte) []byte {
	//nolint:prealloc
	var out []byte

	for _, b := range message {
		if b >= bofne {
			out = append(out, ce, xorce^b)

			continue
		}

		out = append(out, b)
	}

	return out
}

// getCRC prepares and returns CRC for S101 packet based on the payload, crc is generated based on the S101 protocol
// requirements.
func getCRC(data []byte) []uint8 {
	var crc uint16 = eof16

	reader := bytes.NewReader(data)

	for {
		b, err := reader.ReadByte()
		if err != nil {
			break
		}

		if b == ce {
			next, err := reader.ReadByte()
			if err != nil {
				break
			}

			b = xorce ^ next
		}

		crc = computeCRCByte(crc, b)
	}

	crc = (^crc) & eof16

	return parseCRC([]uint8{uint8(crc & eof), uint8(crc >> checkSumSecondDeviation)})
}

// parseCRC bytes above 0xf8 must be preceded with and 0xfd byte and XORed with 0x20 byte.
func parseCRC(in []uint8) []uint8 {
	//nolint:prealloc
	var out []uint8

	for _, v := range in {
		if v < bofne {
			out = append(out, v)

			continue
		}

		out = append(out, ce, v^xorce)
	}

	return out
}

// computeCRCByte creates crc double byte value bases on the S101 crc table against current crc byte using the
// provided byte.
func computeCRCByte(crc uint16, b uint8) uint16 {
	return ((crc >> 8) ^ crcTable[(crc^uint16(b))&0xFF]) & 0xFFFF
}
