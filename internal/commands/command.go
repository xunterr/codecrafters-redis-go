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
}

type CommandType string

const (
	WriteCommand CommandType = "write"
	ReadCommand  CommandType = "read"
	InfoCommand  CommandType = "info"
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

type CommandParser interface {
	GetCommand(in []string) (Command, error)
}

type DefaultCommandParser struct {
	cmdTable map[string]CommandInfo
}

type ReplicaCommandParser struct {
	cmdTable *trie.Trie[CommandInfo]
}

func NewDefaultCommandParser(cmdTable map[string]CommandInfo) CommandParser {
	return DefaultCommandParser{cmdTable}
}

func (p DefaultCommandParser) GetCommand(req []string) (Command, error) {
	commandName := strings.ToUpper(req[0])
	cmdInfo, ok := p.cmdTable[commandName]

	if !ok {
		return Command{}, errors.New(fmt.Sprintf("Unknown command: %s", req[0]))
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

func (p DefaultCommandParser) parseOptions(input []string, cmdInfo CommandInfo) (map[string][]string, error) {
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

func (p DefaultCommandParser) parseArguments(input []string, cmdInfo CommandInfo) ([]string, error) {
	if len(input) < len(cmdInfo.Args) {
		return nil, errors.New("Too few arguments")
	}

	return input[:len(cmdInfo.Args)], nil
}

func NewReplicaCommandParser(cmdTable map[string]CommandInfo) CommandParser {
	t := trie.NewTrie[CommandInfo]()
	for k, v := range cmdTable {
		t.Put(k, v)
	}

	return ReplicaCommandParser{
		cmdTable: &t,
	}
}

func (p ReplicaCommandParser) GetCommand(in []string) (Command, error) {
	return Command{}, nil //TODO
}
