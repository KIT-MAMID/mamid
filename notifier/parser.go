package main

import(
	"os"
	"bufio"
	"strings"
//	"fmt"
//	"text/scanner"
)

type Contact interface { 
}

type EmailContact struct {
	Address string
}

type Parser struct {
}

func (p *Parser) Parse(path string) ([]Contact, error){ 
	var contacts []Contact
	var input []string
	file, err := os.Open(path)
	if err != nil{
		panic(err)
	}
	defer file.Close()
	scan := bufio.NewScanner(file)
	for i := 0; scan.Scan(); i++{
		input = append(input, scan.Text())
	}
	for i := 0; i < len(input); i++ {
                split := strings.Split(input[i], ";")
                for j := 0; j < len(split); j++ {
			x := strings.Split(split[j], ":")
			switch x[0]{
			case "email":
				var newContact EmailContact
				newContact.Address = x[1]
				contacts = append(contacts, newContact)
			default:
				panic("unrecknoized input")
			}
                }
        }
	return contacts, nil
}

