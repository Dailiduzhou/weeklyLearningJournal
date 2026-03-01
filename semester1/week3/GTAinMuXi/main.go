package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	client := &http.Client{}

	req, err := http.NewRequest("HEAD", "https://gtainmuxi.muxixyz.com/api/v1/organization/code", nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Set("code", "400")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	fmt.Println(resp.Header)
	header, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	js, err := json.Marshal(header)
	if err != nil {
		fmt.Println(err)
	}
	file, _ := os.OpenFile("output.json", os.O_CREATE|os.O_WRONLY, 0666)
	defer file.Close()
	enc := json.NewEncoder(file)
	err = enc.Encode(js)
	if err != nil {
		fmt.Println()
	}
}
