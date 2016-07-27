package main

import(
	"os"
	"bufio"
	"strings"
	"fmt"
)

type Contact interface {
}

type EmailContact struct {
	Address string
}

type Parser struct {
}

func (p *Parser) Parse(path string) ([]*Contact, error){ 
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
		fmt.Println("ABC: " + input[i])
	}
	for i := 0; i < len(input); i++{
		split := strings.Split(input[i], ";")
		for j := 0; j < len(split); j++{
			fmt.Println("abc")
			fmt.Println(split[j])
		}
	}
	return []*Contact{}, nil
}
