package ember

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_defaultASN1Encoder_encode(t *testing.T) {
	tests := []struct {
		name string
		c    *ASN1Encoder
		want []byte
	}{
		{
			"+valid",
			NewASN1Encoder(),
			[]byte{
				0x60, 0x80, 0x6B, 0x80, 0xA0, 0x80, 0x62, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x20, 0xA1, 0x03, 0x02, 0x01,
				0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.c.WriteRootTreeRequest()

			if diff := cmp.Diff(tt.want, tt.c.data.Bytes()); diff != "" {
				t.Fatalf("defaultASN1Codec.encode() = %s", diff)
			}
		})
	}
}

func TestDefaultASN1Encoder_EncodeUniversal(t *testing.T) {
	type args struct {
		path []int
	}
	tests := []struct {
		name string
		c    *ASN1Encoder
		args args
		want *ASN1Encoder
	}{
		{
			"+valid",
			NewASN1Encoder(),
			args{
				[]int{1},
			},
			&ASN1Encoder{
				data: bytes.NewBuffer([]byte{
					0x0D, 0x01, 0x01,
				}),
			},
		},
		{
			"+multiple",
			NewASN1Encoder(),
			args{
				[]int{1, 2},
			},
			&ASN1Encoder{
				data: bytes.NewBuffer([]byte{
					0x0D, 0x02, 0x01, 0x02,
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.c.WriteUniversal(tt.args.path)
		})

		if diff := cmp.Diff(tt.c.data.Bytes(), tt.want.data.Bytes()); diff != "" {
			t.Fatalf("DefaultASN1Codec.EncodeUniversal() = %s", diff)
		}
	}
}

func TestDefaultASN1Encoder_writeInt(t *testing.T) {
	type args struct {
		i    int
		cont uint8
	}
	tests := []struct {
		name    string
		c       *ASN1Encoder
		args    args
		want    *ASN1Encoder
		wantErr bool
	}{
		{
			"+valid",
			NewASN1Encoder(),
			args{
				1,
				0xa0,
			},
			&ASN1Encoder{
				data: bytes.NewBuffer([]byte{0xa0, 0x03, 0x02, 0x01, 0x01}),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c.writeInt(tt.args.i, tt.args.cont); (err != nil) != tt.wantErr {
				t.Errorf("DefaultASN1Codec.writeInt() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want.data.Bytes(), tt.c.data.Bytes()); diff != "" {
				t.Fatalf("defaultASN1Codec.encode() = %s", diff)
			}
		})
	}
}
