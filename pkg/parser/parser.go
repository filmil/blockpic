// SPDX-License-Identifier: Apache-2.0
package parser

import (
	"fmt"
	"strconv"
	"github.com/filmil/hodos/pkg/ast"
	"github.com/filmil/hodos/pkg/scanner"
)

type Parser struct {
	s         *scanner.Scanner
	curToken  scanner.Token
	peekToken scanner.Token
	errors    []string
}

func New(s *scanner.Scanner) *Parser {
	p := &Parser{s: s}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.s.NextToken()
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) ParseDiagram() *ast.Diagram {
	diagram := &ast.Diagram{}

	if !p.curTokenIs(scanner.DIAGRAM) {
		p.addError(fmt.Sprintf("expected diagram keyword, got %d", p.curToken.Type))
		return nil
	}

	if !p.expectPeek(scanner.IDENT) {
		return nil
	}
	diagram.Name = p.curToken.Literal

	if !p.expectPeek(scanner.LBRACE) {
		return nil
	}
	p.nextToken() // skip {

	for !p.curTokenIs(scanner.RBRACE) && !p.curTokenIs(scanner.EOF) {
		switch p.curToken.Type {
		case scanner.BLOCK:
			block := p.parseBlock()
			if block != nil {
				diagram.Blocks = append(diagram.Blocks, *block)
			}
		case scanner.WIRE:
			wire := p.parseWire()
			if wire != nil {
				diagram.Wires = append(diagram.Wires, *wire)
			}
		case scanner.LAYOUT:
			layout := p.parseLayout()
			if layout != nil {
				diagram.Layout = append(diagram.Layout, layout...)
			}
		default:
			p.nextToken()
		}
	}

	return diagram
}

func (p *Parser) parseBlock() *ast.Block {
	if !p.expectPeek(scanner.IDENT) {
		return nil
	}
	block := &ast.Block{Name: p.curToken.Literal, Params: make(map[string]string)}

	if !p.expectPeek(scanner.LBRACE) {
		return nil
	}
	p.nextToken() // skip {

	for !p.curTokenIs(scanner.RBRACE) && !p.curTokenIs(scanner.EOF) {
		switch p.curToken.Type {
		case scanner.KIND:
			if p.expectPeek(scanner.COLON) {
				if p.expectPeek(scanner.STRING) {
					block.Kind = p.curToken.Literal
				}
			}
		case scanner.INPUTS:
			if p.expectPeek(scanner.COLON) {
				block.Inputs = p.parsePins()
			}
		case scanner.OUTPUTS:
			if p.expectPeek(scanner.COLON) {
				block.Outputs = p.parsePins()
			}
		case scanner.PARAMS:
			if p.expectPeek(scanner.COLON) {
				block.Params = p.parseParams()
			}
		case scanner.SIZE:
			if p.expectPeek(scanner.COLON) {
				block.Size = p.parseSize()
			}
		}
		p.nextToken()
	}
	p.nextToken()
	return block
}

func (p *Parser) parsePins() []ast.Pin {
	var pins []ast.Pin
	p.nextToken() // skip COLON
	for !p.curTokenIs(scanner.SEMICOLON) && !p.curTokenIs(scanner.RBRACE) && !p.curTokenIs(scanner.EOF) && !p.curTokenIs(scanner.KIND) && !p.curTokenIs(scanner.INPUTS) && !p.curTokenIs(scanner.OUTPUTS) && !p.curTokenIs(scanner.PARAMS) && !p.curTokenIs(scanner.SIZE) {
		if p.curTokenIs(scanner.IDENT) {
			pin := ast.Pin{Name: p.curToken.Literal}
			if p.peekTokenIs(scanner.LBRACKET) {
				p.nextToken() // move to LBRACKET
				p.nextToken() // move to IDENT inside bracket
				pin.Position = p.curToken.Literal
				p.expectPeek(scanner.RBRACKET) // move to RBRACKET
			}
			pins = append(pins, pin)
		}
		if p.peekTokenIs(scanner.COMMA) {
			p.nextToken() // move to COMMA
			p.nextToken() // move to next IDENT
		} else {
			break
		}
	}
	return pins
}

func (p *Parser) parseParams() map[string]string {
	params := make(map[string]string)
	if !p.expectPeek(scanner.LBRACE) {
		return params
	}
	p.nextToken() // skip {
	for !p.curTokenIs(scanner.RBRACE) && !p.curTokenIs(scanner.EOF) {
		if p.curTokenIs(scanner.IDENT) {
			key := p.curToken.Literal
			if p.expectPeek(scanner.COLON) {
				if p.expectPeek(scanner.STRING) {
					params[key] = p.curToken.Literal
				}
			}
		}
		if p.peekTokenIs(scanner.COMMA) {
			p.nextToken() // move to COMMA
			p.nextToken() // move to next key
		} else {
			p.nextToken() // move to RBRACE
		}
	}
	return params
}

func (p *Parser) parseSize() *ast.Size {
	if !p.expectPeek(scanner.LPAREN) {
		return nil
	}
	p.nextToken() // skip (
	w, _ := strconv.ParseFloat(p.curToken.Literal, 64)
	if !p.expectPeek(scanner.COMMA) {
		return nil
	}
	p.nextToken() // skip ,
	h, _ := strconv.ParseFloat(p.curToken.Literal, 64)
	p.expectPeek(scanner.RPAREN) // move to )
	return &ast.Size{W: w, H: h}
}

func (p *Parser) parseWire() *ast.Wire {
	wire := &ast.Wire{}
	p.nextToken() // skip WIRE token
	wire.Source = p.parsePinRef()
	if !p.expectPeek(scanner.ARROW) {
		return nil
	}
	p.nextToken() // skip ->
	wire.Destination = p.parsePinRef()

	if p.peekTokenIs(scanner.LBRACE) {
		p.nextToken() // move to {
		p.nextToken() // move past {
		for !p.curTokenIs(scanner.RBRACE) && !p.curTokenIs(scanner.EOF) {
			switch p.curToken.Type {
			case scanner.HINT:
				if p.expectPeek(scanner.COLON) {
					if p.expectPeek(scanner.STRING) {
						wire.Hint = p.curToken.Literal
					}
				}
			case scanner.THICKNESS:
				if p.expectPeek(scanner.COLON) {
					if p.expectPeek(scanner.NUMBER) {
						wire.Thickness, _ = strconv.Atoi(p.curToken.Literal)
					}
				}
			case scanner.CROSSLINE:
				if p.expectPeek(scanner.COLON) {
					if p.expectPeek(scanner.NUMBER) {
						wire.Crossline, _ = strconv.Atoi(p.curToken.Literal)
					}
				}
			}
			p.nextToken()
		}
	}
	p.nextToken()
	return wire
}

func (p *Parser) parsePinRef() ast.PinRef {
	if !p.curTokenIs(scanner.IDENT) {
		return ast.PinRef{}
	}
	block := p.curToken.Literal
	if !p.expectPeek(scanner.DOT) {
		return ast.PinRef{Block: block}
	}
	if !p.expectPeek(scanner.IDENT) {
		return ast.PinRef{Block: block}
	}
	return ast.PinRef{Block: block, Pin: p.curToken.Literal}
}

func (p *Parser) parseLayout() []ast.LayoutEntry {
	var entries []ast.LayoutEntry
	if !p.expectPeek(scanner.LBRACE) {
		return nil
	}
	p.nextToken() // skip {
	for !p.curTokenIs(scanner.RBRACE) && !p.curTokenIs(scanner.EOF) {
		if p.curTokenIs(scanner.IDENT) {
			block := p.curToken.Literal
			if p.expectPeek(scanner.AT) {
				if p.expectPeek(scanner.LPAREN) {
					if p.expectPeek(scanner.NUMBER) {
						x, _ := strconv.ParseFloat(p.curToken.Literal, 64)
						if p.expectPeek(scanner.COMMA) {
							if p.expectPeek(scanner.NUMBER) {
								y, _ := strconv.ParseFloat(p.curToken.Literal, 64)
								if p.expectPeek(scanner.RPAREN) {
									entries = append(entries, ast.LayoutEntry{Block: block, X: x, Y: y})
								}
							}
						}
					}
				}
			}
		}
		p.nextToken()
	}
	p.nextToken()
	return entries
}

func (p *Parser) curTokenIs(t scanner.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t scanner.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t scanner.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.addError(fmt.Sprintf("expected next token to be %d, got %d instead", t, p.peekToken.Type))
	return false
}

func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)
}
