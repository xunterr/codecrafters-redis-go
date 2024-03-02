package parser

import (
	"log"

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

type Parser struct {
	tokens  []Token
	current int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens:  tokens,
		current: 0,
	}
}

func (d Data) ToMap() (out map[string]any, err error) {
	err = mapstructure.Decode(d, &out)
	return
}

func (p *Parser) Parse() *Data {
	dataType, length := p.parseType()
	var value any

	switch dataType {
	case StringData:
		token := p.advance()
		if token.TokenType != String {
			log.Printf("Type %d is not assignable to type 'StringData'", token.TokenType)
			break
		}
		value = token.Literal
	case ArrayData:
		value = p.parseArray(length)
	}

	return &Data{dataType, value}
}

func (p *Parser) parseType() (DataType, int) {
	typeToken := p.advance()
	var dataType DataType
	switch typeToken.TokenType {
	case Plus:
	case Dollar:
		dataType = StringData
	case Asterisk:
		dataType = ArrayData
	default:
		log.Printf("Unknown type: %d", typeToken.TokenType)
		return -1, -1
	}
	length := p.parseLength()
	return dataType, length
}

func (p *Parser) parseLength() int {
	length := p.advance()
	if length.TokenType != Number {
		log.Printf("Wrong length value, want Number, have: %d", length.TokenType)
		return -1
	}

	return length.Literal.(int)
}

func (p *Parser) parseArray(length int) []Data {
	value := make([]Data, length)
	for i := 0; i < length; i++ {
		value[i] = *p.Parse()
	}
	return value
}

func (p *Parser) advance() Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.tokens[p.current-1]
}

func (p Parser) isAtEnd() bool {
	return p.tokens[p.current].TokenType == EOF
}
