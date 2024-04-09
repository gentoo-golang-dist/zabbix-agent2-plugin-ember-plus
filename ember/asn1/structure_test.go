package asn1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// func TestDecodeAny(t *testing.T) {
// 	t.Parallel()

// 	type args struct {
// 		in []byte
// 	}
// 	tests := []struct {
// 		name    string
// 		args    args
// 		want    any
// 		wantErr bool
// 	}{
// 		// {
// 		// 	"+valid",
// 		// 	args{
// 		// 		[]byte{0x01, 0x01, 0xff},
// 		// 	},
// 		// 	true,
// 		// 	false,
// 		// },
// 		{
// 			"+valid",
// 			args{
// 				[]byte{0x0C, 0x04, 0x52, 0x75, 0x62, 0x79},
// 			},
// 			"Ruby",
// 			false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		tt := tt
// 		t.Run(tt.name, func(t *testing.T) {
// 			t.Parallel()

// 			got, err := DecodeAny(tt.args.in)
// 			if (err != nil) != tt.wantErr {
// 				t.Fatalf("DecodeAny() error = %v, wantErr %v", err, tt.wantErr)
// 			}

// 			if diff := cmp.Diff(tt.want, got); diff != "" {
// 				t.Fatalf("DecodeAny() = %s", diff)
// 			}
// 		})
// 	}
// }

func TestDecodeAny(t *testing.T) {
	t.Parallel()

	var boolVar bool
	boolVarTrue := true

	var stringVar string
	stringVarSet := "Ruby"

	type args struct {
		in  []byte
		val any
	}
	tests := []struct {
		name      string
		args      args
		wantValue any
		wantLen   int
		wantErr   bool
	}{
		{
			"+bool",
			args{
				[]byte{0x01, 0x01, 0xff},
				&boolVar,
			},
			&boolVarTrue,
			3,
			false,
		},
		{
			"+string",
			args{
				[]byte{0x0C, 0x04, 0x52, 0x75, 0x62, 0x79},
				&stringVar,
			},
			&stringVarSet,
			6,
			false,
		},
		{
			"+stringAdditional",
			args{
				[]byte{0x0C, 0x04, 0x52, 0x75, 0x62, 0x79, 0x01, 0x01, 0xff},
				&stringVar,
			},
			&stringVarSet,
			6,
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := DecodeAny(tt.args.in, tt.args.val)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DecodeAny() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.wantValue, tt.args.val); diff != "" {
				t.Fatalf("DecodeAny() val = %s", diff)
			}

			if diff := cmp.Diff(tt.wantLen, got); diff != "" {
				t.Fatalf("DecodeAny() len = %s", diff)
			}
		})
	}
}
