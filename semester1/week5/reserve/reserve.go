package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/term"
)

const (
	loginURL   = "https://account.ccnu.edu.cn/cas/login"
	libURL     = "http://kjyy.ccnu.edu.cn/clientweb/xcus/ic2/Default.aspx"
	requestURL = loginURL + "?service=" + libURL
)

type ReserveResp struct {
	Msg string `json:"msg"`
}

func main() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal("创建cookie jar失败:", err)
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
		log.Fatal("解析登录页面失败:", err)
	}

	lt, exists := doc.Find("input[name='lt']").Attr("value")
	if !exists {
		log.Fatal("未找到lt参数")
	}

	execution, exists := doc.Find("input[name='execution']").Attr("value")
	if !exists {
		log.Fatal("未找到execution参数")
	}

	// fmt.Println(lt)

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

	loginDataString := loginData.Encode()
	req1, err := http.NewRequest("POST", loginURL+";jsessionid="+jsessionid+"?service="+libURL, strings.NewReader(loginDataString))
	if err != nil {
		log.Fatal("创建登录请求失败:", err)
	}

	req1.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req1.Header.Add("Cookie", "JSESSIONID="+jsessionid)

	resp1, err := client.Do(req1)
	if err != nil {
		log.Fatalf("failed at sending req: %q", err)
	}

	defer resp1.Body.Close()

	var devID, start, end string
	fmt.Print("Device ID: ")
	fmt.Scanln(&devID)
	fmt.Print("Start time(1970-1-1,00:00): ")
	fmt.Scanln(&start)
	start = strings.ReplaceAll(start, ":", "%3A")
	start = strings.ReplaceAll(start, ",", "+")
	fmt.Print("End time(1970-1-1,00:00): ")
	fmt.Scanln(&end)
	end = strings.ReplaceAll(end, ":", "%3A")
	end = strings.ReplaceAll(end, ",", "+")

	base := fmt.Sprintf("http://kjyy.ccnu.edu.cn/ClientWeb/pro/ajax/reserve.aspx?dialogid=&lab_id=&kind_id=&room_id=&type=dev&prop=&test_id=&term=&Vnumber=&classkind=&test_name=&up_file=&memo=&act=set_resv&_=1764240101435&dev_id=%s&start=%s&end=%s&start_time=%s&end_time=%s",
		devID, start, end, ParseTime(start), ParseTime(end))

	req2, err := http.NewRequest("GET", base, nil)
	if err != nil {
		log.Fatal(err)
	}

	resp2, err := client.Do(req2)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp2.Body)
	if err != nil {
		log.Fatal(err)
	}

	var reserveResponse ReserveResp
	err = json.Unmarshal(body, &reserveResponse)
	if err != nil {
		log.Printf("err unmarshaling: %q", err)
	}
	fmt.Println(reserveResponse.Msg)
}

func ParseTime(str string) string {
	tail := str[len(str)-7:]
	hhmm := strings.Split(tail, "%3A")
	hh, _ := strconv.Atoi(hhmm[0])
	mm, _ := strconv.Atoi(hhmm[1])

	return strconv.Itoa(hh*100 + mm)
}
