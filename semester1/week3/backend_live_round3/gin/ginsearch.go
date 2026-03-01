package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

var client *http.Client

type UserMessage struct {
	UserName string `json:"username"`
	PassWord string `json:"password"`
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type LoginResponse struct {
	Cookie []string `json:"cookie"`
}

type GetRoomIDsInResponse struct {
	RoomIDs []string `json:"room_ids"`
}

type SearchRequest struct {
	Name    string   `json:"name"`
	RoomIDs []string `json:"room_ids"`
}

type SearchResponse struct {
	Seat      string `json:"seat"`
	InLibrary bool   `json:"is_in_library"`
	Area      string `json:"area"`
	Start     string `json:"start"`
	End       string `json:"end"`
}

type SeatInfo struct {
	Seat       string `json:"seat"`
	IsOccupied bool   `json:"is_occupied"`
	Owner      string `json:"owner"`
	Start      string `json:"start"`
	End        string `json:"end"`
}

type SeatInfoResponse struct {
	SeatInfos []SeatInfo `json:"seat_infos"`
}

// ---- 封装原先的 HTTP 调用逻辑 ----

func Login(userMessage UserMessage) (string, error) {
	const loginURL = "https://demo.muxixyz.com/library/login"
	jsonData, err := json.Marshal(userMessage)
	if err != nil {
		return "", fmt.Errorf("序列化 JSON 失败: %v", err)
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("构建请求时出错: %v", err)
	}
	req.Header.Set("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求时失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应时失败: %v", err)
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析 JSON 时出错: %v", err)
	}

	if result.Code != 200 {
		return "", fmt.Errorf("登录失败: %s", result.Message)
	}

	var loginResp LoginResponse
	data, _ := json.Marshal(result.Data)
	if err := json.Unmarshal(data, &loginResp); err != nil {
		return "", fmt.Errorf("解析 JSON 时出错: %v", err)
	}

	if len(loginResp.Cookie) == 0 {
		return "", fmt.Errorf("未返回 cookie")
	}

	return loginResp.Cookie[0], nil
}

func getRoomIDs(cookie string) (GetRoomIDsInResponse, error) {
	const getRoomIDsURL = "https://demo.muxixyz.com/library/roomids"

	req, err := http.NewRequest("GET", getRoomIDsURL, nil)
	if err != nil {
		return GetRoomIDsInResponse{}, fmt.Errorf("构建获取房间 ID 请求时出错: %v", err)
	}

	req.Header.Set("Cookie", cookie)

	resp, err := client.Do(req)
	if err != nil {
		return GetRoomIDsInResponse{}, fmt.Errorf("发送请求时失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GetRoomIDsInResponse{}, fmt.Errorf("读取响应体时失败: %v", err)
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return GetRoomIDsInResponse{}, fmt.Errorf("解析 JSON 时出错: %v", err)
	}

	if result.Code != 200 {
		return GetRoomIDsInResponse{}, fmt.Errorf("获取房间 ID 失败: %s", result.Message)
	}

	var getIDsResp GetRoomIDsInResponse
	data, _ := json.Marshal(result.Data)
	if err := json.Unmarshal(data, &getIDsResp); err != nil {
		return GetRoomIDsInResponse{}, fmt.Errorf("解析 JSON 时出错: %v", err)
	}

	return getIDsResp, nil
}

func searchStudent(cookie string, searchRequest SearchRequest) (SearchResponse, error) {
	const searchURL = "https://demo.muxixyz.com/library/inlibrary"

	jsonData, err := json.Marshal(searchRequest)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("序列化 JSON 失败: %v", err)
	}

	req, err := http.NewRequest("POST", searchURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return SearchResponse{}, fmt.Errorf("构建请求时失败: %v", err)
	}

	req.Header.Set("Cookie", cookie)
	req.Header.Set("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("发送请求时失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("读取响应体时失败: %v", err)
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return SearchResponse{}, fmt.Errorf("解析 JSON 时出错: %v", err)
	}

	if result.Code != 200 {
		return SearchResponse{}, fmt.Errorf("查询失败: %s", result.Message)
	}

	var searchResp SearchResponse
	data, _ := json.Marshal(result.Data)
	if err := json.Unmarshal(data, &searchResp); err != nil {
		return SearchResponse{}, fmt.Errorf("解析 JSON 时出错: %v", err)
	}

	return searchResp, nil
}

func getSeatInfos(cookie string, roomIDs []string) (SeatInfoResponse, error) {
	const seatinfoURL = "https://demo.muxixyz.com/library/seatinfo"

	requestData := struct {
		RoomIDs []string `json:"room_ids"`
	}{
		RoomIDs: roomIDs,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return SeatInfoResponse{}, fmt.Errorf("序列化请求参数失败: %v", err)
	}

	req, err := http.NewRequest("POST", seatinfoURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return SeatInfoResponse{}, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Cookie", cookie)
	req.Header.Set("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return SeatInfoResponse{}, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SeatInfoResponse{}, fmt.Errorf("读取响应失败: %v", err)
	}

	var apiResponse struct {
		Code    int              `json:"code"`
		Message string           `json:"message"`
		Data    SeatInfoResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return SeatInfoResponse{}, fmt.Errorf("解析 JSON 失败: %v", err)
	}

	if apiResponse.Code != 200 {
		return SeatInfoResponse{}, fmt.Errorf("API 返回错误: %s", apiResponse.Message)
	}

	return apiResponse.Data, nil
}

// ---- Gin 路由与处理函数 ----

// POST /login
// body: { "username": "...", "password": "..." }
// resp: { "cookie": "..." }
func loginHandler(c *gin.Context) {
	var msg UserMessage
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "detail": err.Error()})
		return
	}

	cookie, err := Login(msg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cookie": cookie,
	})
}

// GET /rooms?cookie=xxx
func getRoomsHandler(c *gin.Context) {
	cookie := c.Query("cookie")
	if cookie == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 cookie 参数"})
		return
	}

	rooms, err := getRoomIDs(cookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rooms)
}

// POST /search
// body: { "cookie": "...", "name": "...", "room_ids": ["..."] }
// 如果 room_ids 为空，会自动先调用 getRoomIDs
func searchHandler(c *gin.Context) {
	var req struct {
		Cookie  string   `json:"cookie"`
		Name    string   `json:"name"`
		RoomIDs []string `json:"room_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "detail": err.Error()})
		return
	}
	if req.Cookie == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cookie 和 name 为必填"})
		return
	}

	// 如果没传 roomIDs，就自动获取
	if len(req.RoomIDs) == 0 {
		roomResp, err := getRoomIDs(req.Cookie)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "获取房间号失败", "detail": err.Error()})
			return
		}
		req.RoomIDs = roomResp.RoomIDs
	}

	searchReq := SearchRequest{
		Name:    req.Name,
		RoomIDs: req.RoomIDs,
	}

	searchResp, err := searchStudent(req.Cookie, searchReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "查询出错", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, searchResp)
}

// POST /seats
// body: { "cookie": "...", "room_ids": ["..."] }
// 如果 room_ids 为空，会自动先调用 getRoomIDs
func seatsHandler(c *gin.Context) {
	var req struct {
		Cookie  string   `json:"cookie"`
		RoomIDs []string `json:"room_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "detail": err.Error()})
		return
	}
	if req.Cookie == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cookie 为必填"})
		return
	}

	if len(req.RoomIDs) == 0 {
		roomResp, err := getRoomIDs(req.Cookie)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "获取房间号失败", "detail": err.Error()})
			return
		}
		req.RoomIDs = roomResp.RoomIDs
	}

	seatInfoResp, err := getSeatInfos(req.Cookie, req.RoomIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取座位信息失败", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, seatInfoResp)
}

// POST /check
// body: { "username": "...", "password": "...", "name": "要查询的名字" }
// 整合了你原来 main 的流程并返回综合 JSON
func checkAllHandler(c *gin.Context) {
	var req struct {
		UserName string `json:"username"`
		PassWord string `json:"password"`
		Name     string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "detail": err.Error()})
		return
	}
	if req.UserName == "" || req.PassWord == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username、password、name 均为必填"})
		return
	}

	// 1. 登录
	cookie, err := Login(UserMessage{
		UserName: req.UserName,
		PassWord: req.PassWord,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "登录失败", "detail": err.Error()})
		return
	}

	// 2. 获取房间
	roomIDsResp, err := getRoomIDs(cookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取房间号失败", "detail": err.Error()})
		return
	}

	// 3. 查询学生是否在馆
	searchReq := SearchRequest{
		Name:    req.Name,
		RoomIDs: roomIDsResp.RoomIDs,
	}
	searchResp, err := searchStudent(cookie, searchReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "查询出错", "detail": err.Error()})
		return
	}

	// 4. 获取座位信息
	seatInfoResp, err := getSeatInfos(cookie, roomIDsResp.RoomIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取座位信息失败", "detail": err.Error()})
		return
	}

	// 返回综合 JSON（包括提示文案）
	var tip string
	if searchResp.InLibrary {
		tip = fmt.Sprintf("有人背着你当卷王\n姓名:%s\n区域:%s\n开始于:%s\n结束于:%s",
			req.Name, searchResp.Area, searchResp.Start, searchResp.End)
	} else {
		tip = "你查找的用户现在不在图书馆哦"
	}

	c.JSON(http.StatusOK, gin.H{
		"cookie":         cookie,
		"room_ids":       roomIDsResp.RoomIDs,
		"search_result":  searchResp,
		"seat_info":      seatInfoResp,
		"human_readable": tip,
	})
}

func main() {
	client = &http.Client{}

	r := gin.Default()

	// 基本路由
	r.POST("/login", loginHandler)
	r.GET("/rooms", getRoomsHandler)
	r.POST("/search", searchHandler)
	r.POST("/seats", seatsHandler)
	r.POST("/check", checkAllHandler)

	// 启动服务
	// 默认监听 8080 端口: http://localhost:8080
	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}
