package main

type Contact interface {
}

type EmailContact struct {
	Address string
}

type Parser struct {

}

func (p *Parser) Parse(path string) ([]*Contact, error) {
	return []*Contact{}, nil
}
