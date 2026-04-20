/*
** Copyright (C) 2001-2026 Zabbix SIA
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

package conn

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.zabbix.com/plugin/ember-plus/ember"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/log"
)

func Test_connHandler_readExpected(t *testing.T) {
	t.Parallel()

	type args struct {
		path    string
		timeout int
		message []readResponse
	}

	tests := []struct {
		name    string
		args    args
		want    ember.ElementCollection
		wantErr bool
	}{
		{
			"+valid",
			args{
				"1",
				3,
				[]readResponse{
					{
						collection: ember.ElementCollection{
							ember.ElementKey{
								ID: "R3LAYVirtualPatchBay", Path: "1",
							}: &ember.Element{
								Path:        "1",
								ElementType: "node",
								Identifier:  "R3LAYVirtualPatchBay",
								IsOnline:    true,
							},
						},
						err: nil,
					},
				},
			},
			ember.ElementCollection{
				ember.ElementKey{
					ID: "R3LAYVirtualPatchBay", Path: "1",
				}: &ember.Element{
					Path:        "1",
					ElementType: "node",
					Identifier:  "R3LAYVirtualPatchBay",
					IsOnline:    true,
				},
			},
			false,
		},
		{
			"+multiFirstCorrect",
			args{
				"1",
				3,
				[]readResponse{
					{
						collection: ember.ElementCollection{
							ember.ElementKey{
								ID: "R3LAYVirtualPatchBay", Path: "1",
							}: &ember.Element{
								Path:        "1",
								ElementType: "node",
								Identifier:  "R3LAYVirtualPatchBay",
								IsOnline:    true,
							},
						},
						err: nil,
					},
					{
						collection: ember.ElementCollection{
							ember.ElementKey{
								ID: "R3LAYVirtualPatchBay", Path: "2",
							}: &ember.Element{
								Path:        "2",
								ElementType: "node",
								Identifier:  "R3LAYVirtualPatchBay",
								IsOnline:    true,
							},
						},
						err: nil,
					},
				},
			},
			ember.ElementCollection{
				ember.ElementKey{
					ID: "R3LAYVirtualPatchBay", Path: "1",
				}: &ember.Element{
					Path:        "1",
					ElementType: "node",
					Identifier:  "R3LAYVirtualPatchBay",
					IsOnline:    true,
				},
			},
			false,
		},
		{
			"+multiSecondCorrect",
			args{
				"2",
				3,
				[]readResponse{
					{
						collection: ember.ElementCollection{
							ember.ElementKey{
								ID: "R3LAYVirtualPatchBay", Path: "1",
							}: &ember.Element{
								Path:        "1",
								ElementType: "node",
								Identifier:  "R3LAYVirtualPatchBay",
								IsOnline:    true,
							},
						},
						err: nil,
					},
					{
						collection: ember.ElementCollection{
							ember.ElementKey{
								ID: "R3LAYVirtualPatchBay", Path: "2",
							}: &ember.Element{
								Path:        "2",
								ElementType: "node",
								Identifier:  "R3LAYVirtualPatchBay",
								IsOnline:    true,
							},
						},
						err: nil,
					},
				},
			},
			ember.ElementCollection{
				ember.ElementKey{
					ID: "R3LAYVirtualPatchBay", Path: "2",
				}: &ember.Element{
					Path:        "2",
					ElementType: "node",
					Identifier:  "R3LAYVirtualPatchBay",
					IsOnline:    true,
				},
			},
			false,
		},
		{
			"+multiFirstCorrectWithInvalidData",
			args{
				"1",
				3,
				[]readResponse{
					{
						collection: ember.ElementCollection{
							ember.ElementKey{
								ID: "R3LAYVirtualPatchBay", Path: "1",
							}: &ember.Element{
								Path:        "1",
								ElementType: "node",
								Identifier:  "R3LAYVirtualPatchBay",
								IsOnline:    true,
							},
						},
						err: nil,
					},
					{
						collection: ember.ElementCollection{},
						err:        nil,
					},
				},
			},
			ember.ElementCollection{
				ember.ElementKey{
					ID: "R3LAYVirtualPatchBay", Path: "1",
				}: &ember.Element{
					Path:        "1",
					ElementType: "node",
					Identifier:  "R3LAYVirtualPatchBay",
					IsOnline:    true,
				},
			},
			false,
		},
		{
			"+multiSecondCorrectWithInvalidData",
			args{
				"2",
				3,
				[]readResponse{
					{
						collection: ember.ElementCollection{},
						err:        nil,
					},
					{
						collection: ember.ElementCollection{
							ember.ElementKey{
								ID: "R3LAYVirtualPatchBay", Path: "2",
							}: &ember.Element{
								Path:        "2",
								ElementType: "node",
								Identifier:  "R3LAYVirtualPatchBay",
								IsOnline:    true,
							},
						},
						err: nil,
					},
				},
			},
			ember.ElementCollection{
				ember.ElementKey{
					ID: "R3LAYVirtualPatchBay", Path: "2",
				}: &ember.Element{
					Path:        "2",
					ElementType: "node",
					Identifier:  "R3LAYVirtualPatchBay",
					IsOnline:    true,
				},
			},
			false,
		},
		{
			"-readErr",
			args{
				"1",
				3,
				[]readResponse{
					{
						err: errs.New("fail"),
					},
				},
			},
			nil,
			true,
		},
		{
			"-timeout",
			args{
				"1",
				1,
				nil,
			},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ch := &connHandler{
				logr:       log.New("Test"),
				readData:   make(chan readResponse),
				parsedData: make(chan parsedResponse),
			}

			go func() {
				for _, rr := range tt.args.message {
					ch.readData <- rr
				}
			}()

			ctx, cancel := context.WithTimeout(t.Context(), time.Duration(tt.args.timeout)*time.Second)
			defer cancel()

			got := ch.readExpected(ctx, tt.args.path)
			if (got.err != nil) != tt.wantErr {
				t.Fatalf("connHandler.readExpected() error = %v, wantErr %v", got.err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want, got.element); diff != "" {
				t.Fatalf("connHandler.readExpected() = %s", diff)
			}
		})
	}
}
