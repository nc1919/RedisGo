package parser

import (
	"bufio"
	"errors"
	"log"
	"net"
	"strconv"
)

type Parser struct {
	conn net.Conn
	r    *bufio.Reader
	line []byte
	pos  int
}

func NewParser(conn net.Conn) *Parser {
	return &Parser{
		conn: conn,
		r:    bufio.NewReader(conn),
		line: make([]byte, 0),
		pos:  0,
	}
}

func (p *Parser) current() byte {
	if p.atEnd() {
		return '\r'
	}
	return p.line[p.pos]
}

func (p *Parser) advance() {
	p.pos++
}

func (p *Parser) atEnd() bool {
	return p.pos >= len(p.line)
}

func (p *Parser) readLine() ([]byte, error) {
	for {
		b, err := p.r.ReadByte()
		if err != nil {
			return nil, err
		}

		p.line = append(p.line, b)

		if b == '\n' {
			line := p.line[:len(p.line)-1]
			p.line = p.line[:0]
			return line, nil
		}
	}
}

func (p *Parser) consumeString() (s []byte, err error) {
	for p.current() != '"' && !p.atEnd() {
		cur := p.current()
		p.advance()
		next := p.current()
		if cur == '\\' && next == '"' {
			s = append(s, '"')
			p.advance()
		} else {
			s = append(s, cur)
		}
	}
	if p.current() != '"' {
		return nil, errors.New("unbalanced quotes in request")
	}
	p.advance()
	return
}

// consumeArg reads an argument from the current line.
func (p *Parser) consumeArg() (s string, err error) {
	for p.current() == ' ' {
		p.advance()
	}
	if p.current() == '"' {
		p.advance()
		buf, err := p.consumeString()
		return string(buf), err
	}
	for !p.atEnd() && p.current() != ' ' && p.current() != '\r' {
		s += string(p.current())
		p.advance()
	}
	return
}

// Command reads and parses a command from the connection.
func (p *Parser) Command() (Command, error) {
	tp, err := p.r.ReadByte()
	if err != nil {
		log.Println("Error reading byte:", err)
		return Command{}, err
	}

	if tp == '*' {
		log.Println("Detected RESP array")
		cmd, err := p.respArray()
		if err != nil {
			return Command{}, err
		}
		p.line = p.line[:0]
		return cmd, nil

	} else {
		line, err := p.readLine()
		if err != nil {
			log.Println("Error reading line:", err)
			return Command{}, err
		}

		p.pos = 0
		p.line = append([]byte{tp}, line...)
		log.Println("Detected inline command:", string(p.line))
		cmd, err := p.inline()
		if err != nil {
			return Command{}, err
		}
		p.line = p.line[:0]
		return cmd, nil
	}
}

// respArray parses a RESP array and returns a Command. Returns an error when there's
// a problem reading from the connection.
func (p *Parser) respArray() (Command, error) {
	cmd := Command{}
	elementsStr, err := p.readLine()
	if err != nil {
		return cmd, err
	}
	elements, _ := strconv.Atoi(string(elementsStr))
	log.Println("Elements", elements)
	for i := 0; i < elements; i++ {
		tp, err := p.r.ReadByte()
		if err != nil {
			return cmd, err
		}
		switch tp {
		case ':':
			arg, err := p.readLine()
			if err != nil {
				return cmd, err
			}
			cmd.args = append(cmd.args, string(arg))
		case '$':
			arg, err := p.readLine()
			if err != nil {
				return cmd, err
			}
			length, _ := strconv.Atoi(string(arg))
			text := make([]byte, 0)
			for i := 0; len(text) <= length; i++ {
				line, err := p.readLine()
				if err != nil {
					return cmd, err
				}
				text = append(text, line...)
			}
			cmd.args = append(cmd.args, string(text[:length]))
		case '*':
			next, err := p.respArray()
			if err != nil {
				return cmd, err
			}
			cmd.args = append(cmd.args, next.args...)
		}
	}
	return cmd, nil
}
