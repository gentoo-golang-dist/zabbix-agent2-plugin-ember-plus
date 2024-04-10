package plugin

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_parsePathString(t *testing.T) {
	t.Parallel()

	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		want1   bool
		wantErr bool
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

			got, got1, err := parsePathString(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parsePathString() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("parsePathString() got = %s", diff)
			}

			if diff := cmp.Diff(tt.want1, got1); diff != "" {
				t.Fatalf("parsePathString() got1 = %s", diff)
			}
		})
	}
}
