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
			"+valid_single",
			args{"1"},
			[]string{"1"},
			false,
			false,
		},
		{
			"+valid_same",
			args{"1.1"},
			[]string{"1", "1"},
			false,
			false,
		},
		{
			"+valid_id",
			args{"foo.bar.test"},
			[]string{"foo", "bar", "test"},
			true,
			false,
		},
		{
			"+valid_id_single",
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
			"-empty_field",
			args{"1."},
			nil,
			false,
			true,
		},
		{
			"-empty_spaced_field",
			args{"1. "},
			nil,
			false,
			true,
		},
		{
			"-empty_id_field",
			args{"foo."},
			nil,
			false,
			true,
		},
		{
			"-empty_space_id_field",
			args{"foo. "},
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
