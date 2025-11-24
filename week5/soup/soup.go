package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/anaskhan96/soup"
)

const reqURL = "https://guthib.com/"

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

	content := string(body)

	// // 在代码中添加：
	// fmt.Println("Response Status:", resp.Status)
	// fmt.Println("First 200 chars of body:", string(body)[:200])

	// 下面主要是解析标签
	doc := soup.HTMLParse(content)
	root := doc.Find("h1")
	fmt.Println(root.Text())
	// 	subDocs := doc.FindAll("div", "class", "home-book-list")
	// 	for _, subDoc := range subDocs {
	// 		link := subDoc.Find("div", "home-book-list-item")
	// 		fmt.Println(link.Text())
	// 	}
}
