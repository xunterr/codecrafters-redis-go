package parser

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/utils"
	"github.com/mitchellh/mapstructure"
)

type DataType int

const (
	StringData DataType = iota
	ArrayData
)

type Data struct {
	DataType DataType
	Value    any
}

type TypeHeader struct {
	dataType DataType
	length   int
}

type Parser struct {
	source  string
	start   int
	current int
}

func NewParser(source string) *Parser {
	return &Parser{
		source:  source,
		start:   0,
		current: 0,
	}
}

func (d Data) ToMap() (out map[string]any, err error) {
	err = mapstructure.Decode(d, &out)
	return
}

func (data Data) Flat() (res []string) {
	switch data.DataType {
	case ArrayData:
		arrData := data.Value.([]Data)
		for _, d := range arrData {
			res = append(res, d.Flat()...)
		}
	case StringData:
		strData := data.Value.(string)
		res = append(res, strData)
	}
	return res
}

func (p *Parser) Parse() (*Data, error) {
	typeHeader, err := p.parseTypeHeader()
	if err != nil {
		return nil, err
	}

	err = p.consume("Expecting CRLF after type definition", '\r', '\n')
	if err != nil {
		return nil, err
	}

	var value any
	switch typeHeader.dataType {
	case ArrayData:
		value, err = p.parseArray(typeHeader.length)
	case StringData:
		value = p.scanString()
		err = p.consume("Expecting CRLF after value", '\r', '\n')
	}
	if err != nil {
		return nil, err
	}
	return &Data{typeHeader.dataType, value}, nil
}

func (p *Parser) parseTypeHeader() (TypeHeader, error) {
	typeChar := p.peek()
	p.current++
	p.start = p.current
	var dataType DataType
	switch typeChar {
	case '*':
		dataType = ArrayData
		break
	case '$':
		dataType = StringData
		break
	default:
		return TypeHeader{}, errors.New(fmt.Sprintf("Unknown type: %c", typeChar))
	}
	length := p.parseNumber()
	return TypeHeader{dataType, length}, nil
}

func (p *Parser) parseArray(length int) ([]Data, error) {
	value := make([]Data, length)
	for i := 0; i < length; i++ {
		parsed, err := p.Parse()
		if err != nil {
			return value, err
		}
		value[i] = *parsed
	}
	return value, nil
}

func (p *Parser) scanString() string {
	p.start = p.current
	for (utils.IsAlfa(p.peek()) || utils.IsSpecial(p.peek())) && !p.isAtEnd() {
		p.current++
	}

	literal := p.source[p.start:p.current]
	return literal
}

func (p *Parser) parseNumber() int {
	p.start = p.current
	for utils.IsDigit(p.peek()) {
		p.current++
	}

	value, err := strconv.Atoi(p.source[p.start:p.current])

	if err != nil {
		return -1
	}

	return value
}

func (p *Parser) consume(msg string, seq ...byte) error {
	for _, value := range seq {
		toConsume := p.peek()
		if toConsume != value {
			return errors.New(msg)
		}
		p.current++
	}
	return nil
}

func (p *Parser) peek() byte {
	if p.isAtEnd() {
		return '\000'
	}
	return p.source[p.current]
}

func (p *Parser) peekNext() byte {
	if p.current+1 > len(p.source) {
		return '\000'
	}
	return p.source[p.current+1]
}

func (p Parser) isAtEnd() bool {
	return p.current >= len(p.source)
}
