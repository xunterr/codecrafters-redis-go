package parser

import "log"

type DataType int

const (
	StringData DataType = iota
	ArrayData
	CommandData
)

type Data struct {
	dataType DataType
	value    any
}

type HandlerFunc func(params *Data)

type Parser struct {
	tokens   []Token
	handlers map[string]HandlerFunc
	current  int
}

func (p *Parser) Parse() *Data {
	dataType, length := p.parseType() //0
	var value any

	token := p.advance() //2
	switch dataType {
	case StringData:
		if token.TokenType == Command {
			p.parseCommand()
			break
		}
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
	typeToken := p.advance() //0
	var dataType DataType
	log.Printf("Current: %d, token: %d", p.current, typeToken.TokenType)
	switch typeToken.TokenType {
	case Plus:
	case Dollar:
		dataType = StringData
		break
	case Asterisk:
		dataType = ArrayData
		break
	default:
		log.Printf("Unknown type: %d", typeToken.TokenType)
		return -1, -1
	}
	length := p.parseLength()
	return dataType, length
}

func (p *Parser) parseLength() int {
	length := p.advance() //1
	if length.TokenType != Number {
		log.Printf("Wrong length value, want Number, have: %d", length.TokenType)
		return -1
	}

	return length.Literal.(int)
}

func (p *Parser) parseCommand() {
	name := p.tokens[p.current-1].Literal
	params := p.Parse()
	if f, ok := p.handlers[name.(string)]; ok {
		f(params)
	}
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
