package plugin

import (
	"reflect"
	"testing"
)

func Test_parsePathString(t *testing.T) {
	t.Parallel()

	type args struct {
		path string
	}

	tests := []struct {
		name     string
		args     args
		want     []string
		wantBool bool
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
			"+empty",
			args{},
			nil,
			false,
			false,
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, got1, err := parsePathString(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePathString() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePathString() got response = %v, want %v", got, tt.want)
			}

			if got1 != tt.wantBool {
				t.Errorf("parsePathString() got identifies = %v, want %v", got1, tt.wantBool)
			}
		})
	}
}
