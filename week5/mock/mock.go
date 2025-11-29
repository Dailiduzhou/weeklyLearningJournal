package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/term"
)

const (
	loginURL   = "https://account.ccnu.edu.cn/cas/login"
	libURL     = "http://kjyy.ccnu.edu.cn/clientweb/xcus/ic2/Default.aspx"
	requestURL = loginURL + "?service=" + libURL
)

type queryResponse struct {
	ID          string `json:"id"`
	Pid         string `json:"Pid"`
	Name        string `json:"name"`
	Label       string `json:"label"`
	SzLogonName string `json:"szLogonName"`
	SzHandphone string `json:"szHandPhone"`
	SzTel       string `json:"SzTel"`
	SzEmail     string `json:"szEmail"`
}

type QueryError struct {
	Term      int       `json:"term"`
	ErrorType string    `json:"error_type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	mu          sync.Mutex
	errMu       sync.Mutex
	UsersStore  = make(map[int]string)
	ErrorsStore = make([]QueryError, 0)
	userMax     int
	userMin     int
)

func main() {
	// params := url.Values{}
	// params.Add("service", libURL)
	// fullURL := loginURL + "?" + libURL
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

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
	// fmt.Println(loginDataString)
	req1, err := http.NewRequest("POST", loginURL+";jsessionid="+jsessionid+"?service="+libURL, strings.NewReader(loginDataString))
	if err != nil {
		log.Fatal("创建登录请求失败:", err)
	}

	// 也没用
	// req1.Header.Set("Cookie", "JSESSIONID="+jsessionid)
	req1.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req1.Header.Add("Cookie", "JSESSIONID="+jsessionid)

	resp1, err := client.Do(req1)
	if err != nil {
		log.Fatalf("failed at sending req: %q", err)
	}

	// body, err := io.ReadAll(resp1.Body)
	// if err != nil {
	// 	fmt.Printf("failed at reading body: %q", err)
	// 	return
	// }
	defer resp1.Body.Close()

	// for key, items := range resp1.Header {
	// 	fmt.Println(key)
	// 	for _, item := range items {
	// 		fmt.Printf("item: %s\n", item)
	// 	}
	// }

	// fmt.Println(string(body))

	fmt.Println("查询学号范围")
	fmt.Println("userMax:")
	fmt.Scanln(&userMax)
	fmt.Println("userMin:")
	fmt.Scanln(&userMin)

	maxConcurrency := 10
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	results := make(chan struct {
		term int
		name string
	}, 100)

	numConsumers := 4
	var consumerWG sync.WaitGroup
	consumerWG.Add(numConsumers)

	for range numConsumers {
		go func() {
			defer consumerWG.Done()
			for result := range results {
				mu.Lock()
				UsersStore[result.term] = result.name
				mu.Unlock()
			}
		}()
	}

	batchSize := 50
	numBatches := (userMax - userMin + 1 + batchSize - 1) / batchSize

	for i := range numBatches {
		start := userMin + i*batchSize
		end := start + batchSize - 1
		if end > userMax {
			end = userMax
		}

		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			select {
			case <-ctx.Done():
				return
			default:
			}

			for term := s; term <= e; term++ {
				err := queryUser(ctx, client, term, results)
				if err != nil {
					recordError(term, err)
				}
			}
		}(start, end)
	}

	wg.Wait()
	close(results)

	consumerWG.Wait()

	saveToFiles(UsersStore)
	saveErrorsToFile(ErrorsStore)
}

func queryUser(ctx context.Context, client *http.Client, term int, results chan<- struct {
	term int
	name string
}) error {
	searchURL := fmt.Sprintf("http://kjyy.ccnu.edu.cn/ClientWeb/pro/ajax/data/searchAccount.aspx?type=logonname&ReservaApply=ReservaApply&term=%d&_=1764144604145", term)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	if len(body) == 0 || strings.TrimSpace(string(body)) == "[]" {
		return nil
	}

	bodyStr := strings.TrimSpace(string(body))
	if bodyStr == "[]" {
		return nil
	}

	bodyStr = strings.TrimPrefix(bodyStr, "[")
	bodyStr = strings.TrimSuffix(bodyStr, "]")
	body = []byte(bodyStr)

	var qresp queryResponse
	if err := json.Unmarshal(body, &qresp); err != nil {
		return fmt.Errorf("解析JSON失败: %w, 响应: %s", err, bodyStr)
	}

	if qresp.Name == "" {
		return nil
	}

	select {
	case results <- struct {
		term int
		name string
	}{term: term, name: qresp.Name}:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func recordError(term int, err error) {
	errMu.Lock()
	defer errMu.Unlock()

	errType := "unknown"
	errMsg := err.Error()

	switch {
	case strings.Contains(errMsg, "创建请求失败"):
		errType = "create_request_error"
	case strings.Contains(errMsg, "请求失败"):
		errType = "request_error"
	case strings.Contains(errMsg, "状态码"):
		errType = "status_code_error"
	case strings.Contains(errMsg, "读取响应失败"):
		errType = "read_response_error"
	case strings.Contains(errMsg, "解析JSON失败"):
		errType = "json_parse_error"
	case strings.Contains(errMsg, "context"):
		errType = "context_error"
	}

	ErrorsStore = append(ErrorsStore, QueryError{
		Term:      term,
		ErrorType: errType,
		Message:   errMsg,
		Timestamp: time.Now(),
	})
}

func saveErrorsToFile(errors []QueryError) {
	if len(errors) == 0 {
		fmt.Println("没有错误需要保存")
		return
	}

	file, err := os.Create("errors.json")
	if err != nil {
		log.Printf("创建错误文件失败: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(errors); err != nil {
		log.Printf("保存错误JSON失败: %v", err)
		return
	}

	fmt.Printf("错误信息已保存到 errors.json，共 %d 条记录\n", len(errors))
}

func saveToFiles(data map[int]string) {
	file, err := os.Create("users.json")
	if err != nil {
		log.Printf("创建文件失败: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		log.Printf("保存JSON失败: %v", err)
	}

	fmt.Println("结果已保存到 users.json")
}

// func extractJSessionID(cookieHeader string) (string, error) {

// 	lowerHeader := strings.ToLower(cookieHeader)
// 	target := "jsessionid="

// 	startIdx := strings.Index(lowerHeader, target)
// 	if startIdx == -1 {
// 		return "", fmt.Errorf("JSESSIONID not found")
// 	}

// 	valueStart := startIdx + len(target)
// 	valueEnd := len(cookieHeader)

// 	if semicolonIdx := strings.Index(cookieHeader[valueStart:], ";"); semicolonIdx != -1 {
// 		valueEnd = valueStart + semicolonIdx
// 	}

// 	value := strings.TrimSpace(cookieHeader[valueStart:valueEnd])
// 	value = strings.Trim(value, "\"'")

// 	return value, nil
// }
