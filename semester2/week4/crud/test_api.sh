#!/bin/bash

BASE_URL="http://localhost:8888/api/v1"

echo "===== 测试用户注册 ====="
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/user/register" \
	-H "Content-Type: application/json" \
	-d '{"username":"testuser","password":"testpass123"}')
echo "注册响应: $REGISTER_RESPONSE"
echo

echo "===== 测试用户登录 ====="
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/user/login" \
	-H "Content-Type: application/json" \
	-d '{"username":"testuser","password":"testpass123"}')
echo "登录响应: $LOGIN_RESPONSE"
echo

ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')
REFRESH_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.refresh_token')
USER_ID=$(echo $LOGIN_RESPONSE | jq -r '.id // 1')

if [ "$ACCESS_TOKEN" != "null" ] && [ "$ACCESS_TOKEN" != "" ]; then
	echo "===== 测试获取用户信息 ====="
	curl -s -X GET "$BASE_URL/user/$USER_ID" \
		-H "Authorization: Bearer $ACCESS_TOKEN" | jq
	echo

	echo "===== 测试更新用户资料 ====="
	curl -s -X POST "$BASE_URL/user/update" \
		-H "Authorization: Bearer $ACCESS_TOKEN" \
		-H "Content-Type: application/json" \
		-d "{\"id\":$USER_ID,\"username\":\"testuser_updated\"}" | jq
	echo

	echo "===== 测试刷新令牌 ====="
	REFRESH_RESPONSE=$(curl -s -X POST "$BASE_URL/user/refresh" \
		-H "Content-Type: application/json" \
		-d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
	echo "刷新响应: $REFRESH_RESPONSE"
	echo

	NEW_ACCESS_TOKEN=$(echo $REFRESH_RESPONSE | jq -r '.access_token')

	if [ "$NEW_ACCESS_TOKEN" != "null" ] && [ "$NEW_ACCESS_TOKEN" != "" ]; then
		echo "===== 使用新令牌访问 ====="
		curl -s -X GET "$BASE_URL/user/$USER_ID" \
			-H "Authorization: Bearer $NEW_ACCESS_TOKEN" | jq
		echo
	fi

	echo "===== 测试旧令牌是否失效 ====="
	curl -s -X GET "$BASE_URL/user/$USER_ID" \
		-H "Authorization: Bearer $ACCESS_TOKEN" | jq
	echo
else
	echo "登录失败，跳过后续测试"
fi

echo "===== 测试完成 ====="
