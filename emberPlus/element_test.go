package ember

import (
	"reflect"
	"testing"
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

func TestElementCollection_handleFunction(t *testing.T) {
	type args struct {
		values []Context
	}
	tests := []struct {
		name    string
		ec      ElementCollection
		args    args
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.ec.handleFunction(tt.args.values); (err != nil) != tt.wantErr {
				t.Errorf("ElementCollection.handleFunction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
