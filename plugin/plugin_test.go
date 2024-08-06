/*
** Copyright (C) 2001-2024 Zabbix SIA
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

package plugin

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.zabbix.com/plugin/ember-plus/ember"
	"golang.zabbix.com/sdk/errs"
)

func Test_withJSONResponse(t *testing.T) {
	t.Parallel()

	type args struct {
		handler handlerFunc
	}

	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			"+valid",
			args{
				func(metricParams map[string]string, extraParams ...string) (any, error) {
					coll := ember.ElementCollection{
						ember.ElementKey{
							ID:   "ID",
							Path: "1.2.3",
						}: &ember.Element{
							Path:        "1.2.3",
							ElementType: "node",
							IsOnline:    true,
							IsRoot:      false,
							Maximum:     1,
							ValueType:   2,
						},
					}

					return coll, nil
				},
			},
			`{"1.2.3":` + `{"path":"1.2.3","element_type":"node","children":null,"identifier":"","description":"",` +
				`"is_online":true,"is_root":false}}`,
			false,
		},
		{
			"-handlerErr",
			args{
				func(metricParams map[string]string, extraParams ...string) (any, error) {
					return nil, errs.New("failed")
				},
			},
			nil,
			true,
		},
		{
			"-marshalErr",
			args{
				func(metricParams map[string]string, extraParams ...string) (any, error) {
					coll := ember.ElementCollection{
						ember.ElementKey{
							ID:   "ID",
							Path: "1.2.3",
						}: &ember.Element{
							Path:        "1.2.3",
							ElementType: "foobar",
						},
					}

					return coll, nil
				},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := withJSONResponse(tt.args.handler)

			got, err := handler(nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"withJSONResponse() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("withJSONResponse() = %s", diff)
			}
		})
	}
}

func Test_pathJoin(t *testing.T) {
	t.Parallel()

	type args struct {
		currentPath string
		pathPart    string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"+valid",
			args{
				"foo",
				"bar",
			},
			"foo.bar",
		},
		{
			"+multiple",
			args{
				"foo.bar",
				"bar",
			},
			"foo.bar.bar",
		},
		{
			"+emptyCurrent",
			args{
				"",
				"bar",
			},
			"bar",
		},
		{
			"+emptyAdditional",
			args{
				"bar",
				"",
			},
			"bar.",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := pathJoin(tt.args.currentPath, tt.args.pathPart)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("pathJoin() = %s", diff)
			}
		})
	}
}

func Test_parsePathString(t *testing.T) {
	t.Parallel()

	type args struct {
		path string
	}

	tests := []struct {
		name     string
		args     args
		wantPath []string
		wantIsID bool
		wantErr  bool
	}{
		{
			"+valid",
			args{"1.2.3"},
			[]string{"1", "2", "3"},
			false,
			false,
		},
		{
			"+validSingle",
			args{"1"},
			[]string{"1"},
			false,
			false,
		},
		{
			"+validSame",
			args{"1.1"},
			[]string{"1", "1"},
			false,
			false,
		},
		{
			"+validID",
			args{"foo.bar.test"},
			[]string{"foo", "bar", "test"},
			true,
			false,
		},
		{
			"+validIDSingle",
			args{"foo"},
			[]string{"foo"},
			true,
			false,
		},
		{
			"-empty",
			args{},
			nil,
			false,
			true,
		},
		{
			"-emptyField",
			args{"1."},
			nil,
			false,
			true,
		},
		{
			"-emptySpacedField",
			args{"1. "},
			nil,
			false,
			true,
		},
		{
			"-emptyIdField",
			args{"foo."},
			nil,
			false,
			true,
		},
		{
			"-emptySpaceIdField",
			args{"foo. "},
			nil,
			false,
			true,
		},
		{
			"-mixedParameters",
			args{"foo.1.bar.2"},
			nil,
			false,
			true,
		},
		{
			"-negativeOID",
			args{"1.-1.2"},
			nil,
			false,
			true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, isId, err := parsePathString(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"parsePathString() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}

			if diff := cmp.Diff(tt.wantPath, got); diff != "" {
				t.Fatalf("parsePathString() got = %s", diff)
			}

			if diff := cmp.Diff(tt.wantIsID, isId); diff != "" {
				t.Fatalf("parsePathString() got1 = %s", diff)
			}
		})
	}
}
