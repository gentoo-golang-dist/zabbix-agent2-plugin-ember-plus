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
 
func Test_parsePath(t *testing.T) {
	t.Parallel()

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
		{
			"-nonIntPart",
			args{
				"1.abc.3",
			},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parsePath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parsePath() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("parsePath() = %s", diff)
			}
		})
	}
}
