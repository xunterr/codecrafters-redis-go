package commands

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"testing"

	//	"github.com/codecrafters-io/redis-starter-go/utils"
	"github.com/google/go-cmp/cmp"
)

var table map[string]CommandInfo

var content embed.FS

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func setup() {
	path, err := filepath.Abs("../../cmds.json")
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	table, err = LoadJSON(path)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}
}

func TestGetCommand(t *testing.T) {
	expected := Command{
		Name:      "SET",
		Arguments: []string{"heheh", "asdasd"},
		Options: map[string][]string{
			"PX": {"123"},
		},
	}
	cmdArr := []string{"SET", "heheh", "asdasd", "PX", "123"}

	cmdParser := NewCommandParser(table)
	parsedCmd, err := cmdParser.GetCommand(cmdArr)
	if err != nil {
		t.Errorf("Error: %s", err.Error())
		return
	}
	if !cmp.Equal(expected, parsedCmd) {
		t.Errorf("Wrong parsed command. Have: %v, want: %v", parsedCmd, expected)
	}
}

//	func TestParseOptions(t *testing.T) {
//		tests := []utils.Test[[]string, map[string][]string]{
//			{
//				Name:  "Parse one option with one argument",
//				Input: []string{"-OPTION", "arg1"},
//				Want: map[string][]string{
//					"OPTION": {"arg1"},
//				},
//			},
//			{
//				Name:  "Parse one option with multiple arguments",
//				Input: []string{"-OPTION", "arg1", "arg2"},
//				Want: map[string][]string{
//					"OPTION": {"arg1", "arg2"},
//				},
//			},
//			{
//				Name:  "Parse multiple options",
//				Input: []string{"-OPTION", "arg1", "-OPTION2", "arg2"},
//				Want: map[string][]string{
//					"OPTION":  {"arg1"},
//					"OPTION2": {"arg2"},
//				},
//			},
//		}
//
//		table, err := LoadJSON("cmds.json")
//		if err != nil{
//			t.Errorf(err.Error())
//			return
//		}
//		cmdParser := NewCommandParser(table)
//
//		for _, test := range tests {
//			res, _ := cmdParser.parseOptions(test.Input)
//			if !cmp.Equal(res, test.Want) {
//				t.Errorf(test.ToString(res))
//			}
//			t.Log(res)
//		}
//	}
