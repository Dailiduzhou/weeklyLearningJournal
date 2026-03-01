package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	Seat        string `json:"seat"`
	Is_Occupied bool   `json:"is_occupied"`
	Owner       string `json:"owner"`
	Start       string `json:"start"`
	End         string `json:"end"`
}

type SeatInfoResponse struct {
	SeatInfos []SeatInfo `json:"seat_infos"`
}

func Login(UserMessage UserMessage) (string, error) {
	const loginUrl = "https://demo.muxixyz.com/library/login"
	jsonData, err := json.Marshal(UserMessage)
	if err != nil {
		return "", fmt.Errorf("序列化JSON失败:%v", err)
	}

	request, err := http.NewRequest("POST", loginUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("构建请求时出错:%v", err)
	}
	request.Header.Set("content-type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("发送请求时失败：%v", err)
	}
	defer response.Body.Close() // resp.Body 是一个流stream，需要关闭来保证安全

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应时失败:%v", err)
	}
	var result Response
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("解析JSON时出错:%v", err)
	}

	if result.Code != 200 {
		return "", fmt.Errorf("登录失败: %s", result.Message)
	}

	var loginResp LoginResponse
	data, _ := json.Marshal(result.Data)
	err = json.Unmarshal(data, &loginResp)
	if err != nil {
		return "", fmt.Errorf("解析JSON时出错:%v", err)
	}
	//fmt.Printf("解析到的cookie:%v", loginResp.Cookie)
	return loginResp.Cookie[0], nil
}

func getRoomIDs(Cookie string) (GetRoomIDsInResponse, error) {

	const getRoomIDsURL = "https://demo.muxixyz.com/library/roomids"

	request, err := http.NewRequest("GET", getRoomIDsURL, nil)
	if err != nil {
		err = fmt.Errorf("构建获取房间ID请求时出错:%v", err)
		return GetRoomIDsInResponse{}, err
	}

	request.Header.Set("Cookie", Cookie)

	response, err := client.Do(request)
	if err != nil {
		err = fmt.Errorf("发送请求时失败:%v", err)
		return GetRoomIDsInResponse{}, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("构建响应体时失败:%v", err)
		return GetRoomIDsInResponse{}, err
	}
	defer response.Body.Close()

	var result Response
	err = json.Unmarshal(body, &result)
	if err != nil {
		err = fmt.Errorf("解析JSON时出错:%v", err)
		return GetRoomIDsInResponse{}, err
	}

	var getIDsResp GetRoomIDsInResponse
	data, _ := json.Marshal(result.Data)
	err = json.Unmarshal(data, &getIDsResp)
	if err != nil {
		err = fmt.Errorf("解析JSON时出错:%v", err)
		return GetRoomIDsInResponse{}, err
	}

	return getIDsResp, nil
}
func searchStudent(cookie string, searchRequest SearchRequest) (SearchResponse, error) {

	const getRoomIDsURL = "https://demo.muxixyz.com/library/inlibrary"

	jsonData, err := json.Marshal(searchRequest)
	if err != nil {
		err = fmt.Errorf("序列化 JSON 失败:%v", err)
		return SearchResponse{}, err
	}

	request, err := http.NewRequest("POST", getRoomIDsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		err = fmt.Errorf("构建请求时失败:%v", err)
		return SearchResponse{}, err
	}

	request.Header.Set("Cookie", cookie)
	request.Header.Set("content-type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		err = fmt.Errorf("发送请求时失败:%v", err)
		return SearchResponse{}, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("读取响应体时失败:%v", err)
		return SearchResponse{}, err
	}
	defer response.Body.Close()

	//fmt.Print(string(body))
	var result Response
	err = json.Unmarshal(body, &result)
	if err != nil {
		err = fmt.Errorf("解析json时出错:%v", err)
		return SearchResponse{}, err
	}

	var searchResp SearchResponse
	data, _ := json.Marshal(result.Data)
	err = json.Unmarshal(data, &searchResp)
	if err != nil {
		err = fmt.Errorf("解析json时出错:%v", err)
		return SearchResponse{}, err
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

	request, err := http.NewRequest("POST", seatinfoURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return SeatInfoResponse{}, fmt.Errorf("创建请求失败: %v", err)
	}
	request.Header.Set("Cookie", cookie)
	request.Header.Set("content-type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return SeatInfoResponse{}, fmt.Errorf("发送请求失败: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return SeatInfoResponse{}, fmt.Errorf("读取响应失败: %v", err)
	}

	var apiResponse struct {
		Code    int              `json:"code"`
		Message string           `json:"message"`
		Data    SeatInfoResponse `json:"data"`
	}
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return SeatInfoResponse{}, fmt.Errorf("解析JSON失败: %v", err)
	}

	if apiResponse.Code != 200 {
		return SeatInfoResponse{}, fmt.Errorf("API返回错误: %s", apiResponse.Message)
	}
	return apiResponse.Data, nil
}

func main() {
	client = &http.Client{}
	var a, b string

	fmt.Scanln(&a)
	fmt.Scanln(&b)

	loginMessage := UserMessage{
		UserName: a,
		PassWord: b,
	}

	cookie, err := Login(loginMessage)
	if err != nil {
		fmt.Printf("登陆出错:%v", err)
		return
	}

	// if cookie, err := Login(loginMessage); err != nil {
	// 	fmt.Printf("登陆出错:%v", err)
	// 	return
	// }

	roomIDsResp, err := getRoomIDs(cookie)
	if err != nil {
		fmt.Printf("获取房间号出错:%v", err)
		return
	}

	SearchRequest := SearchRequest{
		Name:    "鲁雨博",
		RoomIDs: roomIDsResp.RoomIDs,
	}

	searchResp, err := searchStudent(cookie, SearchRequest)
	if err != nil {
		fmt.Printf("查询出错:%v", err)
		return
	}

	seatInfoResp, err := getSeatInfos(cookie, roomIDsResp.RoomIDs)
	if err != nil {
		fmt.Printf("获取座位信息失败:%v", err)
		return
	}

	if searchResp.InLibrary {
		fmt.Println("有人背着你当卷王")
		fmt.Printf("姓名:%s\n", SearchRequest.Name)
		fmt.Printf("区域:%s\n", searchResp.Area)
		fmt.Printf("开始于:%s\n结束于:%s\n", searchResp.Start, searchResp.End)
	} else {
		fmt.Println("你查找的用户现在不在图书馆哦")
	}

	fmt.Println("座位信息:")
	for _, seat := range seatInfoResp.SeatInfos {
		status := "空闲"
		if seat.Is_Occupied {
			status = fmt.Sprintf("占用 (用户: %s, 时间: %s - %s)", seat.Owner, seat.Start, seat.End)
		}
		fmt.Printf("- 座位 %s: %s\n", seat.Seat, status)
	}
}
