package commands

import (
	"strings"
)

const OPTIONS_PREFIX string = "-"

var commands map[string]int = map[string]int{ //key is a name, value is a min count of arguments
	"ECHO": 1,
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

func GetCommand(req []string) (Command, error) {
	args := parseArguments(req[1:])
	options := parseOptions(req[len(args)+1:])
	return Command{req[0], options, args}, nil
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

func parseArguments(input []string) (args []string) {
	for _, arg := range input {
		if !isArgument(arg) {
			break
		}
		args = append(args, arg)
	}
	return args
}

func isArgument(str string) bool {
	return !strings.HasPrefix(str, "-")
}
