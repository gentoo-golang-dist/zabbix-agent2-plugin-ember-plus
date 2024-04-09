package ember

import (
	"reflect"
	"testing"

	"git.zabbix.com/ap/ember-plus/ember/asn1"
	"github.com/google/go-cmp/cmp"
)

func Test_parsePath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    []int
		wantErr bool
	}{
		{
			"+valid",
			args{
				"1.2.3",
			},
			[]int{1, 2, 3},
			false,
		},
		{
			"+empty",
			args{
				"",
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestElement_handleContent(t *testing.T) {
	t.Parallel()

	type fields struct {
		IsOnline    bool
		IsRoot      bool
		Identifier  string
		Description string
		Path        string
		ElementType ElementType
		Enumeration string
	}
	type args struct {
		c *asn1.Decoder
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantDecoder  []*asn1.Decoder
		wantElements *Element
		wantErr      bool
	}{
		{
			"+valid",
			fields{
				ElementType: asn1.NodeType,
			},
			args{
				asn1.NewDecoder(
					[]byte{
						0xa1, 0x27, 0x31, 0x25, 0xA0, 0x16, 0x0C, 0x14, 0x52, 0x33, 0x4C, 0x41, 0x59, 0x56, 0x69, 0x72,
						0x74, 0x75, 0x61, 0x6C, 0x50, 0x61, 0x74, 0x63, 0x68, 0x42, 0x61, 0x79, 0xA1, 0x02, 0x0C, 0x00,
						0xA4, 0x02, 0x0C, 0x00, 0xA3, 0x03, 0x01, 0x01, 0xFF,
					},
				),
			},
			[]*asn1.Decoder{asn1.NewDecoder([]byte{}), asn1.NewDecoder([]byte{}), asn1.NewDecoder([]byte{})},
			&Element{
				IsOnline:    true,
				ElementType: "node",
				Identifier:  "R3LAYVirtualPatchBay",
			},
			false,
		},
		{
			"+validEndless",
			fields{
				ElementType: asn1.NodeType,
			},
			args{
				asn1.NewDecoder(
					[]byte{
						0xa1, 0x80, 0x31, 0x80, 0xA0, 0x0C, 0x0C, 0x0A, 0x43, 0x6F, 0x6D, 0x70, 0x72, 0x65, 0x73,
						0x73, 0x6F, 0x72, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00,
					},
				),
			},
			[]*asn1.Decoder{asn1.NewDecoder([]byte{}), asn1.NewDecoder([]byte{}), asn1.NewDecoder([]byte{})},
			&Element{
				IsOnline:    true,
				ElementType: "node",
				Identifier:  "Compressor",
			},
			false,
		},
		{
			"+validEndlessWithLeftover",
			fields{
				ElementType: asn1.NodeType,
			},
			args{
				asn1.NewDecoder(
					[]byte{
						0xa1, 0x80, 0x31, 0x80, 0xA0, 0x0C, 0x0C, 0x0A, 0x43, 0x6F, 0x6D, 0x70, 0x72,
						0x65, 0x73, 0x73, 0x6F, 0x72, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x31,
						0x80, 0xA0, 0x0C, 0x0C, 0x0A, 0x43, 0x6F, 0x6D, 0x70, 0x72, 0x65, 0x73, 0x73,
						0x6F, 0x72, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00,
					},
				),
			},
			[]*asn1.Decoder{
				asn1.NewDecoder([]byte{}),
				asn1.NewDecoder([]byte{}),
				asn1.NewDecoder(
					[]byte{
						0x31, 0x80, 0xA0, 0x0C, 0x0C, 0x0A, 0x43, 0x6F, 0x6D, 0x70, 0x72, 0x65, 0x73,
						0x73, 0x6F, 0x72, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00,
					},
				),
			},
			&Element{
				IsOnline:    true,
				ElementType: "node",
				Identifier:  "Compressor",
			},
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			el := &Element{
				IsOnline:    tt.fields.IsOnline,
				IsRoot:      tt.fields.IsRoot,
				Identifier:  tt.fields.Identifier,
				Description: tt.fields.Description,
				Path:        tt.fields.Path,
				ElementType: tt.fields.ElementType,
				Enumeration: tt.fields.Enumeration,
			}
			got, err := el.handleContent(tt.args.c)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Element.handleContent() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.wantElements, el); diff != "" {
				t.Fatalf("Decoder.handleContent() = %s", diff)
			}

			if len(tt.wantDecoder) != len(got) {
				t.Fatalf(
					"Decoder.handleContent() = incorrect got %d and expected slice len %d",
					len(got), len(tt.wantDecoder),
				)
			}

			for i, g := range got {
				if diff := cmp.Diff(tt.wantDecoder[i].Bytes(), g.Bytes()); diff != "" {
					t.Fatalf("Decoder.handleContent() out decoders do not match at %d = %s", i, diff)
				}
			}
		})
	}
}

func TestElement_handleContext(t *testing.T) {
	t.Parallel()

	type fields struct {
		Path        string
		ElementType ElementType
		Identifier  string
		Description string
		Children    []*Element
		IsOnline    bool
		IsRoot      bool
		Maximum     any
		Minimum     any
		Value       any
		Access      int
		Format      string
		Enumeration string
		Factor      int
		Default     any
		ValueType   int
		Qualified   bool
	}
	type args struct {
		decoder *asn1.Decoder
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantDecoder []*asn1.Decoder
		wantElement *Element
		wantErr     bool
	}{
		{
			"+valid",
			fields{
				ElementType: asn1.NodeType,
			},
			args{
				asn1.NewDecoder(
					[]byte{
						0xA0, 0x16, 0x0C, 0x14, 0x52, 0x33, 0x4C, 0x41, 0x59, 0x56, 0x69, 0x72, 0x74, 0x75, 0x61,
						0x6C, 0x50, 0x61, 0x74, 0x63, 0x68, 0x42, 0x61, 0x79, 0xA1, 0x02, 0x0C, 0x00, 0xA4, 0x02, 0x0C,
						0x00, 0xA3, 0x03, 0x01, 0x01, 0xFF,
					},
				),
			},
			[]*asn1.Decoder{
				asn1.NewDecoder([]byte{0xa1, 0x02, 0x0c, 0x00, 0xa4, 0x02, 0x0c, 0x00, 0xa3, 0x03, 0x01, 0x01, 0xff}),
				asn1.NewDecoder([]byte{}),
			},
			&Element{
				ElementType: "node",
				Identifier:  "R3LAYVirtualPatchBay",
			},
			false,
		},
		{
			"+NoFieldsSet",
			fields{
				ElementType: asn1.NodeType,
			},
			args{
				asn1.NewDecoder(
					[]byte{
						0xa1, 0x02, 0x0c, 0x00, 0xa4, 0x02, 0x0c, 0x00, 0xa3, 0x03, 0x01, 0x01, 0xff,
					},
				),
			},
			[]*asn1.Decoder{
				asn1.NewDecoder([]byte{0xa4, 0x02, 0x0c, 0x00, 0xa3, 0x03, 0x01, 0x01, 0xff}),
				asn1.NewDecoder([]byte{}),
			},
			&Element{
				ElementType: "node",
			},
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			el := &Element{
				Path:        tt.fields.Path,
				ElementType: tt.fields.ElementType,
				Identifier:  tt.fields.Identifier,
				Description: tt.fields.Description,
				Children:    tt.fields.Children,
				IsOnline:    tt.fields.IsOnline,
				IsRoot:      tt.fields.IsRoot,
				Maximum:     tt.fields.Maximum,
				Minimum:     tt.fields.Minimum,
				Value:       tt.fields.Value,
				Access:      tt.fields.Access,
				Format:      tt.fields.Format,
				Enumeration: tt.fields.Enumeration,
				Factor:      tt.fields.Factor,
				Default:     tt.fields.Default,
				ValueType:   tt.fields.ValueType,
				Qualified:   tt.fields.Qualified,
			}
			got, err := el.handleContext(tt.args.decoder)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Element.handleContext() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.wantElement, el); diff != "" {
				t.Fatalf("Decoder.handleContext() = %s", diff)
			}

			if len(tt.wantDecoder) != len(got) {
				t.Fatalf(
					"Decoder.handleContext() = incorrect got %d and expected slice len %d",
					len(got), len(tt.wantDecoder),
				)
			}

			for i, g := range got {
				if diff := cmp.Diff(tt.wantDecoder[i].Bytes(), g.Bytes()); diff != "" {
					t.Fatalf("Decoder.handleContext() out decoders do not match at %d = %s", i, diff)
				}
			}
		})
	}
}

func TestElement_handleApplication(t *testing.T) {
	t.Parallel()

	type fields struct {
		IsOnline    bool
		IsRoot      bool
		Identifier  string
		Description string
		Path        string
		ElementType ElementType
		Enumeration string
	}
	type args struct {
		decoder *asn1.Decoder
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantDecoder  *asn1.Decoder
		wantElements *Element
		wantErr      bool
	}{
		{
			"+node",
			fields{
				ElementType: "node",
			},
			args{
				asn1.NewDecoder(
					[]byte{
						0xA0, 0x03, 0x02, 0x01, 0x01, 0xA1, 0x27, 0x31, 0x25, 0xA0, 0x16, 0x0C, 0x14, 0x52, 0x33, 0x4C,
						0x41, 0x59, 0x56, 0x69, 0x72, 0x74, 0x75, 0x61, 0x6C, 0x50, 0x61, 0x74, 0x63, 0x68, 0x42, 0x61,
						0x79, 0xA1, 0x02, 0x0C, 0x00, 0xA4, 0x02, 0x0C, 0x00, 0xA3, 0x03, 0x01, 0x01, 0xFF,
					},
				),
			},
			asn1.NewDecoder([]byte{}),
			&Element{
				IsOnline:    true,
				Identifier:  "R3LAYVirtualPatchBay",
				Path:        "1",
				ElementType: "node",
			},
			false,
		},
		{
			"+nodeEndless",
			fields{
				ElementType: "node",
			},
			args{
				asn1.NewDecoder(
					[]byte{
						0xA0, 0x03, 0x02, 0x01, 0x03, 0xA1, 0x80, 0x31, 0x80, 0xA0, 0x0C, 0x0C, 0x0A, 0x43, 0x6F, 0x6D,
						0x70, 0x72, 0x65, 0x73, 0x73, 0x6F, 0x72, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00,
					},
				),
			},
			asn1.NewDecoder([]byte{}),
			&Element{
				IsOnline:    true,
				Identifier:  "Compressor",
				Path:        "3",
				ElementType: "node",
			},
			false,
		},
		{
			"+endlessAdditional",
			fields{
				ElementType: "node",
			},
			args{
				asn1.NewDecoder(
					[]byte{
						0xA0, 0x03, 0x02, 0x01, 0x03, 0xA1, 0x80, 0x31, 0x80, 0xA0, 0x0C, 0x0C, 0x0A, 0x43, 0x6F, 0x6D,
						0x70, 0x72, 0x65, 0x73, 0x73, 0x6F, 0x72, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00,
						0xA0, 0x03, 0x02, 0x01, 0x03, 0xA1, 0x80, 0x31, 0x80, 0xA0, 0x0C, 0x0C, 0x0A, 0x43, 0x6F, 0x6D,
						0x70, 0x72, 0x65, 0x73, 0x73, 0x6F, 0x72, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00,
					},
				),
			},
			asn1.NewDecoder(
				[]byte{
					0xA0, 0x03, 0x02, 0x01, 0x03, 0xA1, 0x80, 0x31, 0x80, 0xA0, 0x0C, 0x0C, 0x0A, 0x43, 0x6F, 0x6D,
					0x70, 0x72, 0x65, 0x73, 0x73, 0x6F, 0x72, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00,
				},
			),
			&Element{
				IsOnline:    true,
				Identifier:  "Compressor",
				Path:        "3",
				ElementType: "node",
			},
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			el := &Element{
				IsOnline:    tt.fields.IsOnline,
				IsRoot:      tt.fields.IsRoot,
				Identifier:  tt.fields.Identifier,
				Description: tt.fields.Description,
				Path:        tt.fields.Path,
				ElementType: tt.fields.ElementType,
				Enumeration: tt.fields.Enumeration,
			}
			got, err := el.handleApplication(tt.args.decoder)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Element.handleApplication() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.wantElements, el); diff != "" {
				t.Fatalf("Decoder.handleApplication() Element = %s", diff)
			}

			if diff := cmp.Diff(tt.wantDecoder.Bytes(), got.Bytes()); diff != "" {
				t.Fatalf("Decoder.handleApplication() Decoder = %s", diff)
			}
		})
	}
}

func TestElement_handlePath(t *testing.T) {
	t.Parallel()

	type fields struct {
		Path        string
		ElementType ElementType
		IsOnline    bool
		IsRoot      bool
		Identifier  string
		Description string
		Enumeration string
		Qualified   bool
	}
	type args struct {
		decoder *asn1.Decoder
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantDecoders []*asn1.Decoder
		wantPath     string
		wantErr      bool
	}{
		{
			"integer",
			fields{},
			args{
				asn1.NewDecoder(
					[]byte{0xA0, 0x03, 0x02, 0x01, 0x01},
				),
			},
			[]*asn1.Decoder{
				asn1.NewDecoder([]byte{}),
				asn1.NewDecoder([]byte{}),
			},
			"1",
			false,
		},
		{
			"qualified",
			fields{
				Qualified: true,
			},
			args{
				asn1.NewDecoder(
					[]byte{0xA0, 0x07, 0x0D, 0x05, 0x01, 0x01, 0x02, 0x04, 0x03},
				),
			},
			[]*asn1.Decoder{
				asn1.NewDecoder([]byte{}),
				asn1.NewDecoder([]byte{}),
			},
			"1.1.2.4.3",
			false,
		},
		{
			"qualifiedLeftover",
			fields{
				Qualified: true,
			},
			args{
				asn1.NewDecoder(
					[]byte{
						0xA0, 0x07, 0x0D, 0x05, 0x01, 0x01, 0x02, 0x04, 0x03,
						0x01, 0x01, 0x02, 0x04, 0x03,
					},
				),
			},
			[]*asn1.Decoder{
				asn1.NewDecoder([]byte{0x01, 0x01, 0x02, 0x04, 0x03}),
				asn1.NewDecoder([]byte{}),
			},
			"1.1.2.4.3",
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			el := &Element{
				Path:        tt.fields.Path,
				ElementType: tt.fields.ElementType,
				IsOnline:    tt.fields.IsOnline,
				IsRoot:      tt.fields.IsRoot,
				Identifier:  tt.fields.Identifier,
				Description: tt.fields.Description,
				Enumeration: tt.fields.Enumeration,
				Qualified:   tt.fields.Qualified,
			}
			got, err := el.handlePath(tt.args.decoder)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Element.handlePath() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.wantPath, el.Path); diff != "" {
				t.Fatalf("Element.handlePath() Path = %s", diff)
			}

			if len(tt.wantDecoders) != len(got) {
				t.Fatalf(
					"Decoder.handlePath() = incorrect got %d and expected slice len %d",
					len(got), len(tt.wantDecoders),
				)
			}

			for i, g := range got {
				if diff := cmp.Diff(tt.wantDecoders[i].Bytes(), g.Bytes()); diff != "" {
					t.Fatalf("Decoder.handlePath() out decoders do not match at %d = %s", i, diff)
				}
			}
		})
	}
}

func TestElement_handleChildren(t *testing.T) {
	t.Parallel()

	type fields struct {
		Path        string
		ElementType ElementType
		IsOnline    bool
		IsRoot      bool
		Identifier  string
		Description string
		Enumeration string
		Children    []*Element
		Qualified   bool
	}
	type args struct {
		decoder *asn1.Decoder
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantDecoders []*asn1.Decoder
		wantElement  *Element
		wantErr      bool
	}{
		{
			"+valid",
			fields{},
			args{
				asn1.NewDecoder([]byte{
					0xA2, 0x80, 0x64, 0x80, 0xA0, 0x80, 0x61, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x01, 0xA1, 0x80, 0x31,
					0x80, 0xA0, 0x04, 0x0C, 0x02, 0x4F, 0x6E, 0xAD, 0x03, 0x02, 0x01, 0x04, 0xA2, 0x03, 0x01, 0x01,
					0x00, 0xA5, 0x03, 0x02, 0x01, 0x03, 0xA9, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0xA0, 0x80, 0x61, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x02, 0xA1, 0x80, 0x31, 0x80,
					0xA0, 0x0F, 0x0C, 0x0D, 0x43, 0x6F, 0x72, 0x72, 0x20, 0x47, 0x61, 0x69, 0x6E, 0x5B, 0x64, 0x42,
					0x5D, 0xAD, 0x03, 0x02, 0x01, 0x01, 0xA2, 0x03, 0x02, 0x01, 0x00, 0xA5, 0x03, 0x02, 0x01, 0x03,
					0xA9, 0x03, 0x01, 0x01, 0xFF, 0xA4, 0x03, 0x02, 0x01, 0x00, 0xA3, 0x03, 0x02, 0x01, 0xF4, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA0, 0x80, 0x63, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x03,
					0xA1, 0x80, 0x31, 0x80, 0xA0, 0x0C, 0x0C, 0x0A, 0x43, 0x6F, 0x6D, 0x70, 0x72, 0x65, 0x73, 0x73,
					0x6F, 0x72, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA0,
					0x80, 0x63, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x04, 0xA1, 0x80, 0x31, 0x80, 0xA0, 0x0A, 0x0C, 0x08,
					0x45, 0x78, 0x70, 0x61, 0x6E, 0x64, 0x65, 0x72, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0xA0, 0x80, 0x63, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x05, 0xA1, 0x80,
					0x31, 0x80, 0xA0, 0x06, 0x0C, 0x04, 0x47, 0x61, 0x74, 0x65, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA0, 0x80, 0x63, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x06,
					0xA1, 0x80, 0x31, 0x80, 0xA0, 0x09, 0x0C, 0x07, 0x44, 0x65, 0x45, 0x73, 0x73, 0x65, 0x72, 0xA3,
					0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				}),
			},
			[]*asn1.Decoder{
				asn1.NewDecoder([]byte{}),
				asn1.NewDecoder([]byte{}),
				asn1.NewDecoder([]byte{}),
			},
			&Element{
				Children: []*Element{
					{
						Path:        "1",
						ElementType: "parameter",
						Identifier:  "On",
						IsOnline:    true,
						Access:      3,
						ValueType:   4,
						Value:       false,
					},
					{
						Path:        "2",
						ElementType: "parameter",
						Identifier:  "Corr Gain[dB]",
						Maximum:     int64(0),
						Minimum:     int64(-12),
						Value:       int64(0),
						IsOnline:    true,
						Access:      3,
						ValueType:   1,
					},
					{
						Path:        "3",
						ElementType: "node",
						Identifier:  "Compressor",
						IsOnline:    true,
					},
					{
						Path:        "4",
						ElementType: "node",
						Identifier:  "Expander",
						IsOnline:    true,
					},
					{
						Path:        "5",
						ElementType: "node",
						Identifier:  "Gate",
						IsOnline:    true,
					},
					{
						Path:        "6",
						ElementType: "node",
						Identifier:  "DeEsser",
						IsOnline:    true,
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			el := &Element{
				Path:        tt.fields.Path,
				ElementType: tt.fields.ElementType,
				IsOnline:    tt.fields.IsOnline,
				IsRoot:      tt.fields.IsRoot,
				Identifier:  tt.fields.Identifier,
				Description: tt.fields.Description,
				Enumeration: tt.fields.Enumeration,
				Children:    tt.fields.Children,
				Qualified:   tt.fields.Qualified,
			}
			got, err := el.handleChildren(tt.args.decoder)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Element.handleChildren() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.wantElement, el); diff != "" {
				t.Fatalf("Element.handleChildren() got = %s", diff)
			}

			if len(tt.wantDecoders) != len(got) {
				t.Fatalf(
					"Element.handleChildren() = incorrect got %d and expected slice len %d",
					len(got), len(tt.wantDecoders),
				)
			}

			for i, g := range got {
				if diff := cmp.Diff(tt.wantDecoders[i].Bytes(), g.Bytes()); diff != "" {
					t.Fatalf("Element.handleChildren() out decoders do not match at %d = %s", i, diff)
				}
			}
		})
	}
}

func TestGetElement(t *testing.T) {
	t.Parallel()

	type args struct {
		d *asn1.Decoder
	}
	tests := []struct {
		name        string
		args        args
		wantElement *Element
		wantDecoder *asn1.Decoder
		wantErr     bool
	}{
		{
			"+node",
			args{
				asn1.NewDecoder(
					[]byte{
						0x63, 0x2E, 0xA0, 0x03, 0x02, 0x01, 0x01, 0xA1, 0x27, 0x31,
						0x25, 0xA0, 0x16, 0x0C, 0x14, 0x52, 0x33, 0x4C, 0x41, 0x59, 0x56, 0x69, 0x72, 0x74, 0x75, 0x61,
						0x6C, 0x50, 0x61, 0x74, 0x63, 0x68, 0x42, 0x61, 0x79, 0xA1, 0x02, 0x0C, 0x00, 0xA4, 0x02, 0x0C,
						0x00, 0xA3, 0x03, 0x01, 0x01, 0xFF,
					},
				),
			},
			&Element{
				Path:        "1",
				ElementType: "node",
				IsOnline:    true,
				Identifier:  "R3LAYVirtualPatchBay",
			},
			asn1.NewDecoder([]byte{}),
			false,
		},
		{
			"+parameter",
			args{
				asn1.NewDecoder(
					[]byte{
						0x61, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x01, 0xA1, 0x80, 0x31, 0x80, 0xA0, 0x04, 0x0C, 0x02, 0x4F,
						0x6E, 0xAD, 0x03, 0x02, 0x01, 0x04, 0xA2, 0x03, 0x01, 0x01, 0x00, 0xA5, 0x03, 0x02, 0x01, 0x03,
						0xA9, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00,
					},
				),
			},
			&Element{
				Path:        "1",
				ElementType: "parameter",
				Identifier:  "On",
				IsOnline:    true,
				Access:      3,
				ValueType:   4,
				Value:       false,
			},
			asn1.NewDecoder([]byte{}),
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, got1, err := GetElement(tt.args.d)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetElement() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.wantElement, got); diff != "" {
				t.Fatalf("GetElement() got = %s", diff)
			}

			if diff := cmp.Diff(tt.wantDecoder.Bytes(), got1.Bytes()); diff != "" {
				t.Fatalf("GetElement() got1 = %s", diff)
			}
		})
	}
}

func TestElement_setChild(t *testing.T) {
	t.Parallel()

	type fields struct {
		Path        string
		ElementType ElementType
		IsOnline    bool
		IsRoot      bool
		Identifier  string
		Description string
		Enumeration string
		Children    []*Element
		Qualified   bool
	}
	type args struct {
		childrenDecoder *asn1.Decoder
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantDecoders []*asn1.Decoder
		wantElement  *Element
		wantErr      bool
	}{
		{
			"+valid",
			fields{},
			args{
				asn1.NewDecoder(
					[]byte{
						0xA0, 0x80, 0x61, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x01, 0xA1, 0x80, 0x31, 0x80, 0xA0, 0x04, 0x0C,
						0x02, 0x4F, 0x6E, 0xAD, 0x03, 0x02, 0x01, 0x04, 0xA2, 0x03, 0x01, 0x01, 0x00, 0xA5, 0x03, 0x02,
						0x01, 0x03, 0xA9, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00,
					},
				),
			},
			[]*asn1.Decoder{
				asn1.NewDecoder([]byte{}),
				asn1.NewDecoder([]byte{}),
				asn1.NewDecoder([]byte{}),
			},
			&Element{
				Children: []*Element{
					{
						Path:        "1",
						ElementType: "parameter",
						Identifier:  "On",
						IsOnline:    true,
						Access:      3,
						ValueType:   4,
						Value:       false,
					},
				},
			},
			false,
		},
		{
			"+additionalData",
			fields{},
			args{
				asn1.NewDecoder(
					[]byte{
						0xA0, 0x80, 0x61, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x01, 0xA1, 0x80,
						0x31, 0x80, 0xA0, 0x04, 0x0C, 0x02, 0x4F, 0x6E, 0xAD, 0x03, 0x02, 0x01, 0x04, 0xA2, 0x03, 0x01,
						0x01, 0x00, 0xA5, 0x03, 0x02, 0x01, 0x03, 0xA9, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00,
						0xA0, 0x80, 0x61, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x02, 0xA1, 0x80, 0x31, 0x80, 0xA0, 0x0F, 0x0C,
						0x0D, 0x43, 0x6F, 0x72, 0x72, 0x20, 0x47, 0x61, 0x69, 0x6E, 0x5B, 0x64, 0x42, 0x5D, 0xAD, 0x03,
						0x02, 0x01, 0x01, 0xA2, 0x03, 0x02, 0x01, 0x00, 0xA5, 0x03, 0x02, 0x01, 0x03, 0xA9, 0x03, 0x01,
						0x01, 0xFF, 0xA4, 0x03, 0x02, 0x01, 0x00, 0xA3, 0x03, 0x02, 0x01, 0xF4, 0x00, 0x00, 0x00, 0x00,
					},
				),
			},
			[]*asn1.Decoder{
				asn1.NewDecoder([]byte{}),
				asn1.NewDecoder([]byte{}),
				asn1.NewDecoder([]byte{
					0xA0, 0x80, 0x61, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x02, 0xA1, 0x80, 0x31,
					0x80, 0xA0, 0x0F, 0x0C, 0x0D, 0x43, 0x6F, 0x72, 0x72, 0x20, 0x47, 0x61, 0x69, 0x6E, 0x5B, 0x64,
					0x42, 0x5D, 0xAD, 0x03, 0x02, 0x01, 0x01, 0xA2, 0x03, 0x02, 0x01, 0x00, 0xA5, 0x03, 0x02, 0x01,
					0x03, 0xA9, 0x03, 0x01, 0x01, 0xFF, 0xA4, 0x03, 0x02, 0x01, 0x00, 0xA3, 0x03, 0x02, 0x01, 0xF4,
					0x00, 0x00, 0x00, 0x00,
				}),
			},
			&Element{
				Children: []*Element{
					{
						Path:        "1",
						ElementType: "parameter",
						Identifier:  "On",
						IsOnline:    true,
						Access:      3,
						ValueType:   4,
						Value:       false,
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			el := &Element{
				Path:        tt.fields.Path,
				ElementType: tt.fields.ElementType,
				IsOnline:    tt.fields.IsOnline,
				IsRoot:      tt.fields.IsRoot,
				Identifier:  tt.fields.Identifier,
				Description: tt.fields.Description,
				Enumeration: tt.fields.Enumeration,
				Children:    tt.fields.Children,
				Qualified:   tt.fields.Qualified,
			}
			got, err := el.setChild(tt.args.childrenDecoder)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Element.setChild() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.wantElement, el); diff != "" {
				t.Fatalf("Element.setChild() got = %s", diff)
			}

			if len(tt.wantDecoders) != len(got) {
				t.Fatalf(
					"Element.setChild() = incorrect got %d and expected slice len %d",
					len(got), len(tt.wantDecoders),
				)
			}

			for i, g := range got {
				if diff := cmp.Diff(tt.wantDecoders[i].Bytes(), g.Bytes()); diff != "" {
					t.Fatalf("Element.setChild() out decoders do not match at %d = %s", i, diff)
				}
			}
		})
	}
}

func TestElementCollection_Populate(t *testing.T) {
	t.Parallel()

	type args struct {
		data *asn1.Decoder
	}
	tests := []struct {
		name    string
		ec      ElementCollection
		args    args
		want    ElementCollection
		wantErr bool
	}{
		{
			"+valid",
			ElementCollection{},
			args{
				asn1.NewDecoder(
					[]byte{
						0x60, 0x34, 0x6B, 0x32, 0xA0, 0x30, 0x63, 0x2E, 0xA0, 0x03, 0x02, 0x01, 0x01, 0xA1, 0x27, 0x31,
						0x25, 0xA0, 0x16, 0x0C, 0x14, 0x52, 0x33, 0x4C, 0x41, 0x59, 0x56, 0x69, 0x72, 0x74, 0x75, 0x61,
						0x6C, 0x50, 0x61, 0x74, 0x63, 0x68, 0x42, 0x61, 0x79, 0xA1, 0x02, 0x0C, 0x00, 0xA4, 0x02, 0x0C,
						0x00, 0xA3, 0x03, 0x01, 0x01, 0xFF,
					},
				),
			},
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "R3LAYVirtualPatchBay",
				}: &Element{
					Path:        "1",
					ElementType: asn1.NodeType,
					Identifier:  "R3LAYVirtualPatchBay",
					IsOnline:    true,
				},
			},
			false,
		},
		{
			"+children",
			ElementCollection{},
			args{
				asn1.NewDecoder(
					[]byte{
						0x60, 0x80, 0x6B, 0x80, 0xA0, 0x80, 0x6A, 0x80, 0xA0, 0x07, 0x0D, 0x05, 0x01, 0x01, 0x02, 0x04,
						0x03, 0xA2, 0x80, 0x64, 0x80, 0xA0, 0x80, 0x61, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x01, 0xA1, 0x80,
						0x31, 0x80, 0xA0, 0x04, 0x0C, 0x02, 0x4F, 0x6E, 0xAD, 0x03, 0x02, 0x01, 0x04, 0xA2, 0x03, 0x01,
						0x01, 0x00, 0xA5, 0x03, 0x02, 0x01, 0x03, 0xA9, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0xA0, 0x80, 0x61, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x02, 0xA1, 0x80, 0x31,
						0x80, 0xA0, 0x0F, 0x0C, 0x0D, 0x43, 0x6F, 0x72, 0x72, 0x20, 0x47, 0x61, 0x69, 0x6E, 0x5B, 0x64,
						0x42, 0x5D, 0xAD, 0x03, 0x02, 0x01, 0x01, 0xA2, 0x03, 0x02, 0x01, 0x00, 0xA5, 0x03, 0x02, 0x01,
						0x03, 0xA9, 0x03, 0x01, 0x01, 0xFF, 0xA4, 0x03, 0x02, 0x01, 0x00, 0xA3, 0x03, 0x02, 0x01, 0xF4,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA0, 0x80, 0x63, 0x80, 0xA0, 0x03, 0x02, 0x01,
						0x03, 0xA1, 0x80, 0x31, 0x80, 0xA0, 0x0C, 0x0C, 0x0A, 0x43, 0x6F, 0x6D, 0x70, 0x72, 0x65, 0x73,
						0x73, 0x6F, 0x72, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0xA0, 0x80, 0x63, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x04, 0xA1, 0x80, 0x31, 0x80, 0xA0, 0x0A, 0x0C,
						0x08, 0x45, 0x78, 0x70, 0x61, 0x6E, 0x64, 0x65, 0x72, 0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA0, 0x80, 0x63, 0x80, 0xA0, 0x03, 0x02, 0x01, 0x05, 0xA1,
						0x80, 0x31, 0x80, 0xA0, 0x06, 0x0C, 0x04, 0x47, 0x61, 0x74, 0x65, 0xA3, 0x03, 0x01, 0x01, 0xFF,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xA0, 0x80, 0x63, 0x80, 0xA0, 0x03, 0x02, 0x01,
						0x06, 0xA1, 0x80, 0x31, 0x80, 0xA0, 0x09, 0x0C, 0x07, 0x44, 0x65, 0x45, 0x73, 0x73, 0x65, 0x72,
						0xA3, 0x03, 0x01, 0x01, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					},
				),
			},
			ElementCollection{
				ElementKey{
					Path: "1.1.2.4.3",
				}: &Element{
					Path:        "1.1.2.4.3",
					ElementType: asn1.QualifiedNodeType,
					Qualified:   true,
					Children: []*Element{
						{
							Path:        "1",
							ElementType: "parameter",
							Identifier:  "On",
							IsOnline:    true,
							Access:      3,
							ValueType:   4,
							Value:       false,
						},
						{
							Path:        "2",
							ElementType: "parameter",
							Identifier:  "Corr Gain[dB]",
							Maximum:     int64(0),
							Minimum:     int64(-12),
							Value:       int64(0),
							IsOnline:    true,
							Access:      3,
							ValueType:   1,
						},
						{
							Path:        "3",
							ElementType: "node",
							Identifier:  "Compressor",
							IsOnline:    true,
						},
						{
							Path:        "4",
							ElementType: "node",
							Identifier:  "Expander",
							IsOnline:    true,
						},
						{
							Path:        "5",
							ElementType: "node",
							Identifier:  "Gate",
							IsOnline:    true,
						},
						{
							Path:        "6",
							ElementType: "node",
							Identifier:  "DeEsser",
							IsOnline:    true,
						},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.ec.Populate(tt.args.data); (err != nil) != tt.wantErr {
				t.Fatalf("ElementCollection.Populate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.ec, tt.want); diff != "" {
				t.Fatalf("ElementCollection.Populate() = %s", diff)
			}
		})
	}
}

func TestReadOverElement(t *testing.T) {
	t.Parallel()

	type args struct {
		decoder *asn1.Decoder
	}
	tests := []struct {
		name    string
		args    args
		want    *asn1.Decoder
		wantErr bool
	}{
		{
			"+valid",
			args{
				asn1.NewDecoder([]byte{0xa1, 0x02, 0x0c, 0x00, 0xa4, 0x02, 0x0c, 0x00, 0xa3, 0x03, 0x01, 0x01, 0xff}),
			},
			asn1.NewDecoder([]byte{0xa4, 0x02, 0x0c, 0x00, 0xa3, 0x03, 0x01, 0x01, 0xff}),
			false,
		},
		{
			"+noLeft",
			args{
				asn1.NewDecoder([]byte{0xa3, 0x03, 0x01, 0x01, 0xff}),
			},
			asn1.NewDecoder([]byte{}),
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ReadOverElement(tt.args.decoder)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReadOverElement() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want.Bytes(), got.Bytes()); diff != "" {
				t.Fatalf("ReadOverElement() = %s", diff)
			}
		})
	}
}
