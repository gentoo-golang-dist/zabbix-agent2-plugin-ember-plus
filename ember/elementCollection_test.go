/*
** Copyright (C) 2001-2025 Zabbix SIA
**
** This program is free software: you can redistribute it and/or modify it under the terms of
** the GNU Affero General Public License as published by the Free Software Foundation, version 3.
**
** This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
** without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
** See the GNU Affero General Public License for more details.
**
** You should have received a copy of the GNU Affero General Public License along with this program.
** If not, see <https://www.gnu.org/licenses/>.
**/

package ember

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestElementCollection_GetElementByPath(t *testing.T) {
	t.Parallel()

	type args struct {
		currentPath string
	}

	tests := []struct {
		name    string
		ec      ElementCollection
		args    args
		want    *Element
		wantErr bool
	}{
		{
			"+valid",
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "test",
				}: &Element{
					Path:        "1",
					ElementType: "node",
					Identifier:  "test",
					Description: "foobar",
				},
			},
			args{
				"1",
			},
			&Element{
				Path:        "1",
				ElementType: "node",
				Identifier:  "test",
				Description: "foobar",
			},
			false,
		},
		{
			"+multiple",
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "test",
				}: &Element{
					Path:        "1",
					ElementType: "node",
					Identifier:  "test",
					Description: "foobar",
				},
				ElementKey{
					Path: "2",
					ID:   "test2",
				}: &Element{
					Path:        "2",
					ElementType: "node",
					Identifier:  "test2",
					Description: "foobar",
				},
			},
			args{
				"1",
			},
			&Element{
				Path:        "1",
				ElementType: "node",
				Identifier:  "test",
				Description: "foobar",
			},
			false,
		},
		{
			"+child",
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "foo",
				}: &Element{
					Path:        "1",
					ElementType: "node",
					Identifier:  "foo",
					Description: "foobar",
					Children: []*Element{
						{
							Path:        "2",
							ElementType: "node",
							Identifier:  "bar",
							Description: "foobar",
						},
					},
				},
			},
			args{
				"1.2",
			},
			&Element{
				Path:        "2",
				ElementType: "node",
				Identifier:  "bar",
				Description: "foobar",
			},
			false,
		},
		{
			"-notFound",
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "test",
				}: &Element{
					Path:        "1",
					ElementType: "node",
					Identifier:  "test",
					Description: "foobar",
				},
				ElementKey{
					Path: "2",
					ID:   "test2",
				}: &Element{
					Path:        "2",
					ElementType: "node",
					Identifier:  "test2",
					Description: "foobar",
				},
			},
			args{
				"12",
			},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.ec.GetElementByPath(tt.args.currentPath)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ElementCollection.GetElementByPath() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("ElementCollection.GetElementByPath() = %s", diff)
			}
		})
	}
}

func TestElementCollection_GetElementByID(t *testing.T) {
	t.Parallel()

	type args struct {
		id string
	}

	tests := []struct {
		name     string
		ec       ElementCollection
		args     args
		wantEl   *Element
		wantPath string
		wantErr  bool
	}{
		{
			"+valid",
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "test",
				}: &Element{
					Path:        "1",
					ElementType: "node",
					Identifier:  "test",
					Description: "foobar",
				},
			},
			args{
				"test",
			},
			&Element{
				Path:        "1",
				ElementType: "node",
				Identifier:  "test",
				Description: "foobar",
			},
			"1",
			false,
		},
		{
			"+multiple",
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "test",
				}: &Element{
					Path:        "1",
					ElementType: "node",
					Identifier:  "test",
					Description: "foobar",
				},
				ElementKey{
					Path: "2",
					ID:   "test2",
				}: &Element{
					Path:        "2",
					ElementType: "node",
					Identifier:  "test2",
					Description: "foobar",
				},
			},
			args{
				"test",
			},
			&Element{
				Path:        "1",
				ElementType: "node",
				Identifier:  "test",
				Description: "foobar",
			},
			"1",
			false,
		},
		{
			"+child",
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "foo",
				}: &Element{
					Path:        "1",
					ElementType: "node",
					Identifier:  "foo",
					Description: "foobar",
					Children: []*Element{
						{
							Path:        "2",
							ElementType: "node",
							Identifier:  "bar",
							Description: "foobar",
						},
					},
				},
			},
			args{
				"bar",
			},
			&Element{
				Path:        "2",
				ElementType: "node",
				Identifier:  "bar",
				Description: "foobar",
			},
			"1.2",
			false,
		},
		{
			"-notFound",
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "test",
				}: &Element{
					Path:        "1",
					ElementType: "node",
					Identifier:  "test",
					Description: "foobar",
				},
				ElementKey{
					Path: "2",
					ID:   "test2",
				}: &Element{
					Path:        "2",
					ElementType: "node",
					Identifier:  "test2",
					Description: "foobar",
				},
			},
			args{
				"foobar",
			},
			nil,
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, got1, err := tt.ec.GetElementByID(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ElementCollection.GetElementByID() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.wantEl, got); diff != "" {
				t.Fatalf("ElementCollection.GetElementByID() element = %s", diff)
			}

			if diff := cmp.Diff(tt.wantPath, got1); diff != "" {
				t.Fatalf("ElementCollection.GetElementByID() path = %s", diff)
			}
		})
	}
}

func TestElementCollection_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ec      ElementCollection
		want    []byte
		wantErr bool
	}{
		{
			"+node",
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "test",
				}: &Element{
					Path:        "1",
					ElementType: "node",
					Identifier:  "test",
					Description: "foobar",
				},
			},
			[]byte{
				0x7b, 0x22, 0x31, 0x22, 0x3a, 0x7b, 0x22, 0x70, 0x61, 0x74, 0x68, 0x22, 0x3a, 0x22, 0x31, 0x22, 0x2c,
				0x22, 0x65, 0x6c, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x22, 0x3a, 0x22, 0x6e,
				0x6f, 0x64, 0x65, 0x22, 0x2c, 0x22, 0x63, 0x68, 0x69, 0x6c, 0x64, 0x72, 0x65, 0x6e, 0x22, 0x3a, 0x6e,
				0x75, 0x6c, 0x6c, 0x2c, 0x22, 0x69, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x65, 0x72, 0x22, 0x3a,
				0x22, 0x74, 0x65, 0x73, 0x74, 0x22, 0x2c, 0x22, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69,
				0x6f, 0x6e, 0x22, 0x3a, 0x22, 0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72, 0x22, 0x2c, 0x22, 0x69, 0x73, 0x5f,
				0x6f, 0x6e, 0x6c, 0x69, 0x6e, 0x65, 0x22, 0x3a, 0x66, 0x61, 0x6c, 0x73, 0x65, 0x2c, 0x22, 0x69, 0x73,
				0x5f, 0x72, 0x6f, 0x6f, 0x74, 0x22, 0x3a, 0x66, 0x61, 0x6c, 0x73, 0x65, 0x7d, 0x7d,
			},
			false,
		},
		{
			"+parameter",
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "test",
				}: &Element{
					Path:        "1",
					ElementType: "parameter",
					Identifier:  "test",
					Description: "foobar",
					Value:       true,
					ValueType:   4,
				},
			},
			[]byte{
				0x7b, 0x22, 0x31, 0x22, 0x3a, 0x7b, 0x22, 0x70, 0x61, 0x74, 0x68, 0x22, 0x3a, 0x22, 0x31, 0x22, 0x2c,
				0x22, 0x65, 0x6c, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x22, 0x3a, 0x22, 0x70,
				0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x22, 0x2c, 0x22, 0x69, 0x64, 0x65, 0x6e, 0x74, 0x69,
				0x66, 0x69, 0x65, 0x72, 0x22, 0x3a, 0x22, 0x74, 0x65, 0x73, 0x74, 0x22, 0x2c, 0x22, 0x64, 0x65, 0x73,
				0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x3a, 0x22, 0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
				0x22, 0x2c, 0x22, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x3a, 0x74, 0x72, 0x75, 0x65, 0x2c, 0x22, 0x74,
				0x79, 0x70, 0x65, 0x22, 0x3a, 0x34, 0x7d, 0x7d,
			},
			false,
		},
		{
			"+function",
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "test",
				}: &Element{
					Path:        "1",
					ElementType: "function",
					Identifier:  "test",
					Description: "foobar",
				},
			},
			[]byte{
				0x7b, 0x22, 0x31, 0x22, 0x3a, 0x7b, 0x22, 0x70, 0x61, 0x74, 0x68, 0x22, 0x3a, 0x22, 0x31, 0x22,
				0x2c, 0x22, 0x65, 0x6c, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x22, 0x3a,
				0x22, 0x66, 0x75, 0x6e, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x2c, 0x22, 0x63, 0x68, 0x69, 0x6c,
				0x64, 0x72, 0x65, 0x6e, 0x22, 0x3a, 0x6e, 0x75, 0x6c, 0x6c, 0x2c, 0x22, 0x69, 0x64, 0x65, 0x6e,
				0x74, 0x69, 0x66, 0x69, 0x65, 0x72, 0x22, 0x3a, 0x22, 0x74, 0x65, 0x73, 0x74, 0x22, 0x2c, 0x22,
				0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x3a, 0x22, 0x66, 0x6f,
				0x6f, 0x62, 0x61, 0x72, 0x22, 0x7d, 0x7d},
			false,
		},
		{
			"+empty",
			ElementCollection{},
			[]byte{0x7b, 0x7d},
			false,
		},
		{
			"-unknownType",
			ElementCollection{
				ElementKey{
					Path: "1",
					ID:   "test",
				}: &Element{
					Path:        "1",
					ElementType: "foobar",
					Identifier:  "test",
					Description: "foobar",
				},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.ec.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ElementCollection.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("ElementCollection.MarshalJSON() = %s", diff)
			}
		})
	}
}

func TestNewElementCollection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want ElementCollection
	}{
		{
			"+valid",
			make(ElementCollection),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewElementCollection()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("NewElementCollection() = %s", diff)
			}
		})
	}
}
