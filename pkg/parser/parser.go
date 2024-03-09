package parser

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/utils"
	"github.com/mitchellh/mapstructure"
)

type DataType byte

const (
	String     DataType = '+'
	Array      DataType = '*'
	BulkString DataType = '$'
	Error      DataType = '-'
)

type Data struct {
	dataType DataType
	string   string
	array    []Data
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
	switch data.dataType {
	case Array:
		arrData := data.array
		for _, d := range arrData {
			res = append(res, d.Flat()...)
		}
	case BulkString, String:
		strData := data.string
		res = append(res, strData)
	}
	return res
}

func IsSimple(t DataType) bool {
	return t == String || t == Error
}

func (p *Parser) Parse() (*Data, error) {
	typeHeader, err := p.parseTypeHeader()
	if err != nil {
		return nil, err
	}

	var value any
	data := Data{dataType: typeHeader.dataType}
	switch typeHeader.dataType {
	case Array:
		value, err = p.parseArray(typeHeader.length)
		data.array = value.([]Data)
	case BulkString, String, Error:
		value = p.scanString()
		err = p.consume("Expecting CRLF after value", '\r', '\n')
		data.string = value.(string)
	default:
		err = errors.New(fmt.Sprintf("Unknown type: %s", string(typeHeader.dataType)))
	}
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (p *Parser) parseTypeHeader() (TypeHeader, error) {
	typeChar := p.peek()
	length := 0
	p.current++
	p.start = p.current

	if !IsSimple(DataType(typeChar)) {
		length = p.parseNumber()
		err := p.consume("Expecting CRLF after type header", '\r', '\n')
		if err != nil {
			return TypeHeader{}, err
		}
	}

	return TypeHeader{DataType(typeChar), length}, nil
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

func StringData(str string) Data {
	return Data{dataType: String, string: str}
}

func BulkStringData(str string) Data {
	return Data{dataType: BulkString, string: str}
}

func ArrayData(arr []Data) Data {
	return Data{dataType: Array, array: arr}
}

func ErrorData(str string) Data {
	return Data{dataType: Error, string: str}
}

func (d Data) Marshal() []byte {
	switch d.dataType {
	case String, Error:
		return d.marshalSimple()
	case BulkString:
		return d.marshalBulk()
	case Array:
		return d.marshalArray()
	default:
		return nil
	}
}

func (d Data) marshalSimple() (res []byte) {
	res = append(res, byte(d.dataType))
	value := d.string
	for _, b := range value {
		res = append(res, byte(b))
	}
	res = append(res, '\r', '\n')
	return
}

func (d Data) marshalBulk() (res []byte) {
	res = append(res, '$')
	value := d.string
	res = append(res, []byte(strconv.FormatInt(int64(len(value)), 10))...)
	res = append(res, '\r', '\n')
	res = append(res, []byte(value)...)
	res = append(res, '\r', '\n')
	return
}

func (d Data) marshalArray() (res []byte) {
	res = append(res, '*')
	value := d.array
	res = append(res, []byte(strconv.FormatInt(int64(len(value)), 10))...)
	res = append(res, '\r', '\n')
	for _, v := range value {
		res = append(res, v.Marshal()...)
	}
	return
}
