package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/google/uuid"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Event struct {
	Description string `json:"description"`
	DDL         string `json:"ddl"`
	Finished    bool   `json:"finished"`
	Belong      string `json:"belong"`
}

// 扩展模型：为了方便API操作，我们需要ID
type TodoItem struct {
	ID int `json:"id"`
	Event
}

var (
	users     = make(map[string]string)
	todos     = make(map[int]*TodoItem)
	sessions  = make(map[string]string)
	todoIDSeq = 1
	dataLock  sync.RWMutex
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

type authHandler func(http.ResponseWriter, *http.Request, string)

func requireAuth(next authHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("token")
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
			return
		}

		dataLock.RLock()
		username, exists := sessions[c.Value]
		dataLock.RUnlock()

		if !exists {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Session无效"})
			return
		}

		// 执行业务逻辑，传入 username
		next(w, r, username)
	}
}

func main() {
	mux := http.NewServeMux()

	// 1. 注册
	mux.HandleFunc("POST /register", func(w http.ResponseWriter, r *http.Request) {
		var u User
		if err := readJSON(r, &u); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "格式错误"})
			return
		}
		dataLock.Lock()
		defer dataLock.Unlock()
		if _, ok := users[u.Username]; ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "用户已存在"})
			return
		}
		users[u.Username] = u.Password
		writeJSON(w, http.StatusOK, map[string]string{"message": "注册成功"})
	})

	// 2. 登录
	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		var u User
		if err := readJSON(r, &u); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "格式错误"})
			return
		}
		dataLock.RLock()
		pwd, ok := users[u.Username]
		dataLock.RUnlock()

		if !ok || pwd != u.Password {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "失败"})
			return
		}

		token := uuid.NewString()
		dataLock.Lock()
		sessions[token] = u.Username
		dataLock.Unlock()

		http.SetCookie(w, &http.Cookie{
			Name: "token", Value: token, Path: "/", HttpOnly: true, MaxAge: 3600,
		})
		writeJSON(w, http.StatusOK, map[string]string{"message": "登录成功", "token": token})
	})

	// 3. 查 (GET /todos) - 可读他人
	mux.HandleFunc("GET /todos", requireAuth(func(w http.ResponseWriter, r *http.Request, user string) {
		dataLock.RLock()
		defer dataLock.RUnlock()
		list := make([]*TodoItem, 0, len(todos))
		for _, v := range todos {
			list = append(list, v)
		}
		writeJSON(w, http.StatusOK, list)
	}))

	// 4. 增 (POST /todos) - 写自己
	mux.HandleFunc("POST /todos", requireAuth(func(w http.ResponseWriter, r *http.Request, user string) {
		var evt Event
		if err := readJSON(r, &evt); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "格式错误"})
			return
		}
		evt.Belong = user // 强制归属
		evt.Finished = false

		dataLock.Lock()
		defer dataLock.Unlock()

		item := &TodoItem{ID: todoIDSeq, Event: evt}
		todos[todoIDSeq] = item
		todoIDSeq++
		writeJSON(w, http.StatusOK, item)
	}))

	// 5. 删 (DELETE /todos/{id}) - 仅限自己
	mux.HandleFunc("DELETE /todos/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request, user string) {
		idStr := r.PathValue("id")
		id, _ := strconv.Atoi(idStr)

		dataLock.Lock()
		defer dataLock.Unlock()

		item, ok := todos[id]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "未找到"})
			return
		}
		if item.Belong != user {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作"})
			return
		}
		delete(todos, id)
		writeJSON(w, http.StatusOK, map[string]string{"message": "已删除"})
	}))

	// 6. 改 (PUT /todos/{id}) - 仅限自己
	mux.HandleFunc("PUT /todos/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request, user string) {
		idStr := r.PathValue("id")
		id, _ := strconv.Atoi(idStr)
		var input Event
		if err := readJSON(r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "格式错误"})
			return
		}

		dataLock.Lock()
		defer dataLock.Unlock()

		item, ok := todos[id]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "未找到"})
			return
		}
		if item.Belong != user {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作"})
			return
		}

		item.Description = input.Description
		item.DDL = input.DDL
		writeJSON(w, http.StatusOK, item)
	}))

	// 7. Check/Uncheck - 仅限自己
	handleStatus := func(w http.ResponseWriter, r *http.Request, user string, status bool) {
		idStr := r.PathValue("id")
		id, _ := strconv.Atoi(idStr)

		dataLock.Lock()
		defer dataLock.Unlock()

		item, ok := todos[id]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "未找到"})
			return
		}
		if item.Belong != user {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "无权操作"})
			return
		}
		item.Finished = status
		writeJSON(w, http.StatusOK, item)
	}

	mux.HandleFunc("PATCH /todos/{id}/check", requireAuth(func(w http.ResponseWriter, r *http.Request, user string) {
		handleStatus(w, r, user, true)
	}))
	mux.HandleFunc("PATCH /todos/{id}/uncheck", requireAuth(func(w http.ResponseWriter, r *http.Request, user string) {
		handleStatus(w, r, user, false)
	}))

	fmt.Println("Server running on :8081")
	http.ListenAndServe(":8081", mux)
}
