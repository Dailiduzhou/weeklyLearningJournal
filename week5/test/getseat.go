package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	username = "xxx"
	password = "xxx"
)

var Lt, Execution, Cookies string

// var start, end string

func main() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{
		Jar: jar,
	}

	requesturl := "https://account.ccnu.edu.cn/cas/login?service=http://kjyy.ccnu.edu.cn/loginall.aspx?page="
	res, err := client.Get(requesturl)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Println(res.Header)

	it := regexp.MustCompile(`name="lt"\s+value="([^"]+)"`)
	execution := regexp.MustCompile(`name="execution"\s+value="([^"]+)"`)
	cookies := regexp.MustCompile(`JSESSIONID=([^;]+)`)

	match := it.FindStringSubmatch(string(body))
	if len(match) > 1 {
		fmt.Println("lt value:", match[1])
		Lt = match[1]
	} else {
		fmt.Println("No match found")
	}

	matchs := execution.FindStringSubmatch(string(body))
	if len(matchs) > 1 {
		fmt.Println("execution value:", matchs[1])
		Execution = matchs[1]
	} else {
		fmt.Println("No match found")
	}

	matches := cookies.FindStringSubmatch(fmt.Sprintf("%v", res.Header))
	if len(matches) > 1 {
		fmt.Println("cookies value:", matches[1])
		Cookies = "JSESSIONID=" + matches[1]
	} else {
		fmt.Println("No match found")
	}

	//fmt.Println(Lt, Execution, Cookies)

	data := url.Values{}
	data.Set("username", username)
	data.Set("password", password)
	data.Set("lt", Lt)
	data.Set("execution", Execution)
	data.Set("_eventId", "submit")
	data.Set("submit", "登录")

	playload := strings.NewReader(data.Encode())

	targeturl := "https://account.ccnu.edu.cn/cas/login;jsessionid=" + matches[1] + "?service=http://kjyy.ccnu.edu.cn/loginall.aspx?page="
	req, err := http.NewRequest("POST", targeturl, playload)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Cookie", Cookies)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer resp.Body.Close()

	//fmt.Println(resp.Body)
	//fmt.Println(resp.Header)

	now := time.Now()
	date := now.Format("2006-01-02")

	minute := now.Minute()
	nextMinute := (minute/5 + 1) * 5
	if nextMinute == 60 {
		nextMinute = 0
		now = now.Add(time.Hour) // 进位到下一个小时
	}
	nextTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, now.Location())

	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())

	fullURL := fmt.Sprintf("http://kjyy.ccnu.edu.cn/ClientWeb/pro/ajax/device.aspx?"+
		"byType=devcls&classkind=8&display=fp&md=d&room_id=101699179"+
		"&purpose=&selectOpenAty=&cld_name=default&date=%s"+
		"&fr_start=%s&fr_end=22:00&act=get_rsv_sta&_=%s", date, nextTime.Format("15:04"), timestamp)
	//fmt.Println(fullURL)

	requ, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	response, err := client.Do(requ)
	if err != nil {
		log.Fatal(err)
		return
	}

	defer response.Body.Close()

	Body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(string(Body))

	//getseaturl := "http://kjyy.ccnu.edu.cn/ClientWeb/pro/ajax/reserve.aspx?" + "dialogid=&dev_id=101699806" + "&lab_id=&kind_id=&room_id=&type=dev&prop=&test_id=&term=&Vnumber=&classkind=&test_name=&start=2025-3-28+13:00&end=2025-3-28+14:00&start_time=&end_time=&up_file=&memo=&act=set_resv"
	//request, err := http.NewRequest("GET", getseaturl, nil)
	//if err != nil {
	//	log.Fatal(err)
	//	return
	//}
	////fmt.Println(getseaturl)
	//
	//responses, err := client.Do(request)
	//if err != nil {
	//	log.Fatal(err)
	//	return
	//}
	//defer responses.Body.Close()
	//
	//bodys, err := io.ReadAll(responses.Body)
	//if err != nil {
	//	log.Fatal(err)
	//	return
	//}
	//fmt.Println(string(bodys))

}
