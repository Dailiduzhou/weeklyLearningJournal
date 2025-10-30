// debog.go

package main

import (
	"encoding/gob"
	"fmt"
	"os"
)

type Address struct {
	Type    string
	City    string
	Country string
}
type VCard struct {
	FirstName string
	LastName  string
	Addresses []*Address
	Remark    string
}

func main() {
	file, err := os.Open("vcard.gob")
	if err != nil {
		fmt.Printf("error while opening file: %v\n", err)
		return
	}
	defer file.Close()

	dec := gob.NewDecoder(file)

	var vcard VCard
	err = dec.Decode(&vcard)
	if err != nil {
		fmt.Printf("error while decoding: %v\n", err)
		return
	}

	fmt.Printf("Decoded VCard: %+v\n", vcard)
}
