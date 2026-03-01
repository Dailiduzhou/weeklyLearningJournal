package main

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"

	"github.com/PuerkitoBio/goquery"
)

const libURL = "http://kjyy.ccnu.edu.cn/clientweb/xcus/ic2/Default.aspx"

func main() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	client := &http.Client{
		Jar: jar,
	}

	req, err := http.NewRequest("GET", libURL, nil)
	if err != nil {
		panic(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		panic(err)
	}

	seatHeader := doc.Find("h5:contains('座位')").First()
	if seatHeader.Length() == 0 {
		fmt.Printf("未找到座位位置")
		return
	}
}
