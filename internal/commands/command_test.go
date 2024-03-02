package commands

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/utils"
	"github.com/google/go-cmp/cmp"
)

func TestParseOptions(t *testing.T) {
	tests := []utils.Test[[]string, map[string][]string]{
		{
			Name:  "Parse one option with one argument",
			Input: []string{"-OPTION", "arg1"},
			Want: map[string][]string{
				"OPTION": {"arg1"},
			},
		},
		{
			Name:  "Parse one option with multiple arguments",
			Input: []string{"-OPTION", "arg1", "arg2"},
			Want: map[string][]string{
				"OPTION": {"arg1", "arg2"},
			},
		},
		{
			Name:  "Parse multiple options",
			Input: []string{"-OPTION", "arg1", "-OPTION2", "arg2"},
			Want: map[string][]string{
				"OPTION":  {"arg1"},
				"OPTION2": {"arg2"},
			},
		},
	}

	for _, test := range tests {
		res := parseOptions(test.Input)
		if !cmp.Equal(res, test.Want) {
			t.Errorf(test.ToString(res))
		}
		t.Log(res)
	}
}
