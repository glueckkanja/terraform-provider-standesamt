package tools

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"testing"
)

func TestGetBaseString(t *testing.T) {
	type args struct {
		s types.String
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty",
			args: args{
				s: types.StringValue(""),
			},
			want: "",
		},
		{
			name: "string",
			args: args{
				s: types.StringValue("test"),
			},
			want: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBaseString(tt.args.s); got != tt.want {
				t.Errorf("GetBaseString() = %v, want %v", got, tt.want)
			}
		})
	}
}
