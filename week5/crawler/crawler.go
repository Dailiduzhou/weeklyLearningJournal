package main

import (
	"fmt"
	"io"
	"net/http"
)

const reqURL = "https://www.liaoxuefeng.com/"

func main() {
	client := &http.Client{}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		fmt.Printf("error new req: %q", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error sending req: %q", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error reading: %q", err)
	}
	defer resp.Body.Close()

	fmt.Printf("%v", string(body))
}
