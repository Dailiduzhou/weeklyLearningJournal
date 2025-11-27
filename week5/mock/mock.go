package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/term"
)

const loginURL = "https://account.ccnu.edu.cn/cas/login"
const libURL = "http://kjyy.ccnu.edu.cn/clientweb/xcus/ic2/Default.aspx"
const requestURL = loginURL + "?service=" + libURL

func main() {
	// params := url.Values{}
	// params.Add("service", libURL)
	// fullURL := loginURL + "?" + libURL
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	client := &http.Client{
		Jar: jar,
	}

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		fmt.Printf("failed at newing req: %q", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("failed at sending req: %q", err)
		return
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		panic(err)
	}

	lt, exists := doc.Find("input[name='lt']").Attr("value")
	if !exists {
		fmt.Println("lt doesn't exist")
		return
	}

	execution, exists := doc.Find("input[name='execution']").Attr("value")
	if !exists {
		fmt.Println("execution doesn't exist")
		return
	}

	var username, pswd string
	fmt.Println("请输入用户名:")
	fmt.Scanln(&username)
	fmt.Println("请输入密码:")

	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal("读取密码失败:", err)
	}
	pswd = string(bytePassword)
	fmt.Println()

	loginData := url.Values{}
	loginData.Add("username", username)
	loginData.Add("password", pswd)
	loginData.Add("lt", lt)
	loginData.Add("execution", execution)
	loginData.Add("_eventId", "submit")
	loginData.Add("submit", "登录")

	var jsessionid string
	// for key, items := range resp.Header {
	// 	if key == "Set-Cookie" {
	// 		jsessionid = items[0]
	// 	}
	// }

	// for key, items := range resp.Header {
	// 	fmt.Println(key)
	// 	for _, item := range items {
	// 		fmt.Printf("item: %s\n", item)
	// 	}
	// }

	// if len(jsessionid) == 0 {
	// 	fmt.Printf("failed at finding jsessionid")
	// 	return
	// }

	// jsessionid, err = extractJSessionID(jsessionid)
	// if err != nil {
	// 	fmt.Printf("failed at extracting: %q", err)
	// }
	// // fmt.Println(jsessionid)

	url, err := url.Parse(loginURL)
	if err != nil {
		panic(err)
	}
	for _, k := range jar.Cookies(url) {
		if k.Name == "JSESSIONID" {
			jsessionid = k.Value
			break
		}
	}

	// jar.SetCookies(url, []*http.Cookie{
	// 	{
	// 		Name:  "JSESSIONID",
	// 		Value: jsessionid,
	// 		Path:  "/",
	// 	},
	// })

	loginDataString := loginData.Encode()
	fmt.Println(loginDataString)
	req1, err := http.NewRequest("POST", loginURL+";jsessionid="+jsessionid, strings.NewReader(loginDataString))
	if err != nil {
		fmt.Printf("failed at newing req: %q", err)
		return
	}

	// 也没用
	// req1.Header.Set("Cookie", "JSESSIONID="+jsessionid)

	resp1, err := client.Do(req1)
	if err != nil {
		fmt.Printf("failed at sending req: %q", err)
		return
	}

	body, err := io.ReadAll(resp1.Body)
	if err != nil {
		fmt.Printf("failed at reading body: %q", err)
		return
	}
	defer resp1.Body.Close()

	for key, items := range resp1.Header {
		fmt.Println(key)
		for _, item := range items {
			fmt.Printf("item: %s\n", item)
		}
	}

	fmt.Println(string(body))

	// req2, err := http.NewRequest("GET", libURL, nil)
	// if err != nil {
	// 	panic(err)
	// }

	// resp2, err := client.Do(req2)
	// if err != nil {
	// 	panic(err)
	// }
	// defer resp2.Body.Close()

	// body, err := io.ReadAll(resp2.Body)
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println(string(body))

}

func extractJSessionID(cookieHeader string) (string, error) {

	lowerHeader := strings.ToLower(cookieHeader)
	target := "jsessionid="

	startIdx := strings.Index(lowerHeader, target)
	if startIdx == -1 {
		return "", fmt.Errorf("JSESSIONID not found")
	}

	valueStart := startIdx + len(target)
	valueEnd := len(cookieHeader)

	if semicolonIdx := strings.Index(cookieHeader[valueStart:], ";"); semicolonIdx != -1 {
		valueEnd = valueStart + semicolonIdx
	}

	value := strings.TrimSpace(cookieHeader[valueStart:valueEnd])
	value = strings.Trim(value, "\"'")

	return value, nil
}
