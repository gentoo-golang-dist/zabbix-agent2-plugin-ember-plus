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

package plugin

import (
	"testing"

	"git.zabbix.com/ap/ember-plus/ember"
	"github.com/google/go-cmp/cmp"
)

func Test_withJSONResponse(t *testing.T) {
	t.Parallel()

	type args struct {
		handler handlerFunc
	}

	tests := []struct {
		name    string
		args    args
		want    string
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
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := withJSONResponse(tt.args.handler)

			got, err := handler(nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("withJSONResponse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("withJSONResponse() = %s", diff)
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
				t.Fatalf("parsePathString() error = %v, wantErr %v", err, tt.wantErr)
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
