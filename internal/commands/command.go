package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/pkg/trie"
)

type Command struct {
	Name      string
	Options   map[string][]string
	Arguments []string
	Type      CommandType
}

type CommandInfo struct {
	Args    []string
	Options map[string][]string
	Type    CommandType
	Policy  CommandPolicy
}

type CommandType string
type CommandPolicy string

const (
	Write CommandType = "write"
	Read  CommandType = "read"
	Info  CommandType = "info"
	Repl  CommandType = "repl"
)

const (
	Match      CommandPolicy = "match"
	StartsWith CommandPolicy = "startsWith"
)

func LoadJSON(filename string) (map[string]CommandInfo, error) {
	jsonFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var table map[string]CommandInfo
	err = json.Unmarshal(byteValue, &table)
	return table, err
}

type CommandParser struct {
	cmdTable *trie.Trie[CommandInfo]
}

func NewCommandParser(cmdTable map[string]CommandInfo) CommandParser {
	return CommandParser{
		cmdTable: trie.FromMap[CommandInfo](cmdTable),
	}
}

func (p CommandParser) ParseCommand(req []string) (Command, error) {
	commandName := strings.ToUpper(req[0])

	commandName, cmdInfo, err := p.getCommandInfo(commandName)

	if err != nil {
		return Command{}, err
	}

	args, err := p.parseArguments(req[1:], cmdInfo)
	if err != nil {
		return Command{}, err
	}

	options, err := p.parseOptions(req[len(args)+1:], cmdInfo)
	if err != nil {
		return Command{}, err
	}
	return Command{commandName, options, args, cmdInfo.Type}, nil
}

func (p CommandParser) getCommandInfo(cmdName string) (string, CommandInfo, error) {
	k, v, err := p.cmdTable.GetBestMatch(cmdName)
	if err != nil {
		return "", CommandInfo{}, errors.New(fmt.Sprintf("Unknown command: %s", cmdName))
	}

	if k != cmdName && v.Policy != StartsWith {
		return "", CommandInfo{}, errors.New(fmt.Sprintf("Can't find matching command for %s", cmdName))
	}

	return k, *v, nil
}

func (p CommandParser) parseOptions(input []string, cmdInfo CommandInfo) (map[string][]string, error) {
	res := make(map[string][]string)
	var currentOption string
	for i := 0; i < len(input); i++ {
		if len(input[i]) == 0 {
			continue
		}

		option := strings.ToUpper(input[i])
		if args, ok := cmdInfo.Options[option]; ok {
			currentOption = option

			if len(input) < len(args)+i+1 {
				return res, errors.New(fmt.Sprintf("Too few argumets for option %s", currentOption))
			}
			res[currentOption] = input[i+1 : len(args)+1]
			if len(args) == 0 {
				continue
			}
			i += len(args) - 1
		} else if currentOption == "" {
			return res, errors.New(fmt.Sprintf("No such option: %s", option))
		}

	}
	return res, nil
}

func (p CommandParser) parseArguments(input []string, cmdInfo CommandInfo) ([]string, error) {
	if len(input) < len(cmdInfo.Args) {
		return nil, errors.New("Too few arguments")
	}

	return input[:len(cmdInfo.Args)], nil
}
