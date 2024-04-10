/*
** Zabbix
** Copyright 2001-2024 Zabbix SIA
**
** Licensed under the Apache License, Version 2.0 (the "License");
** you may not use this file except in compliance with the License.
** You may obtain a copy of the License at
**
**     http://www.apache.org/licenses/LICENSE-2.0
**
** Unless required by applicable law or agreed to in writing, software
** distributed under the License is distributed on an "AS IS" BASIS,
** WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
** See the License for the specific language governing permissions and
** limitations under the License.
**/

package asn1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

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

func TestApplicationByte(t *testing.T) {
	t.Parallel()

	type args struct {
		num uint8
	}
	tests := []struct {
		name string
		args args
		want uint8
	}{
		{"+valid", args{0x30}, 0x70},
		{"+zero", args{0x00}, 0x60},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ApplicationByte(tt.args.num)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("ApplicationByte() = %s", diff)
			}
		})
	}
}

func TestContextByte(t *testing.T) {
	t.Parallel()

	type args struct {
		num uint8
	}
	tests := []struct {
		name string
		args args
		want uint8
	}{
		{"+valid", args{0x30}, 0xb0},
		{"+zero", args{0x00}, 0xa0},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ContextByte(tt.args.num)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("ContextByte() = %s", diff)
			}
		})
	}
}

func TestUniversalByte(t *testing.T) {
	t.Parallel()

	type args struct {
		num uint8
	}
	tests := []struct {
		name string
		args args
		want uint8
	}{
		{"+valid", args{0x30}, 0x30},
		{"+zero", args{0x00}, 0x00},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := UniversalByte(tt.args.num)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("UniversalByte() = %s", diff)
			}
		})
	}
}
