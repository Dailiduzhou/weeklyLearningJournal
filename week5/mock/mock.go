package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

const loginURL = "https://account.ccnu.edu.cn/cas/login"

func main() {
	client := &http.Client{}

	req, err := http.NewRequest("GET", loginURL, nil)
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

	var jsessionid string
	for key, items := range resp.Header {
		if key == "Set-Cookie" {
			jsessionid = items[0]
		}
	}
	if len(jsessionid) == 0 {
		fmt.Printf("failed at finding jsessionid")
		return
	}

	// jsessionidString := resp.Header.Get("Set-Cookie")
	jsessionid, err = extractJSessionID(jsessionid)
	if err != nil {
		fmt.Printf("failed at extracting: %q", err)
	}

	req1, err := http.NewRequest("POST", loginURL+";jsessionid="+jsessionid, nil)
	if err != nil {
		fmt.Printf("failed at newing req: %q", err)
		return
	}

	req1.Header.Set("Cookie", "JSESSIONID="+jsessionid)

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
			fmt.Printf("\nitem: %s\n", item)
		}
	}

	fmt.Println(string(body))
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
