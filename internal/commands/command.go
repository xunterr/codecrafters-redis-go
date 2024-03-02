package commands

import (
	"strings"

	"github.com/codecrafters-io/redis-starter-go/pkg/parser"
)

const OPTIONS_PREFIX string = "-"

var commands map[string]int = map[string]int{ //key is a name, value is a min count of arguments
	"ECHO": 0,
	"PING": 0,
	"PONG": 0,
	"SET":  2,
	"GET":  1,
}

type Command struct {
	Name      string
	Options   map[string][]string
	Arguments []string
}

func GetCommand(req []string) Command {
	argc, ok := commands[req[0]]
	if !ok {
		return Command{}
	}

	args := parseArguments(req[1:], argc)
	options := parseOptions(req[argc:])
	return Command{req[0], options, args}
}

func parseOptions(input []string) map[string][]string {
	res := make(map[string][]string)
	var currentOption string
	for _, arg := range input {
		if len(arg) == 0 {
			continue
		}

		if strings.HasPrefix(arg, OPTIONS_PREFIX) {
			currentOption = arg[1:]
			res[currentOption] = make([]string, 0)
		} else {
			res[currentOption] = append(res[currentOption], arg)
		}
	}
	return res
}

func parseArguments(input []string, argc int) []string {
	return input[:argc-1]
}

func flat(data parser.Data) (res []string) {
	switch data.DataType {
	case parser.ArrayData:
		arrData := data.Value.([]parser.Data)
		for _, d := range arrData {
			res = append(res, flat(d)...)
		}
	case parser.StringData:
		strData := data.Value.(string)
		res = append(res, strData)
	}
	return res
}
