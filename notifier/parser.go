package main

import(
	"os"
	"bufio"
)

type Contact interface {
}

type EmailContact struct {
	Address string
}

type Parser struct {
}

func (p *Parser) Parse(path string) ([]*Contact, error) {
	file, err := os.Open(path)
	if err != nil{
		panic(err)
	}
	defer func() {
        	if err := file.Close(); err != nil {
        	    panic(err)
       		}
    	}()
	var input []string
	scanner := bufio.NewScanner(file)
	for i :=0; scanner.Scan(); i++{
		input[i] = scanner.Text()
	}
	return []*Contact{}, nil
}
