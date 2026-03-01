// use transaction
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	ID       int    `json:"-"` // 不在JSON中序列化
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name"`
}

var (
	userStore = make(map[string]*User)
	mu        sync.RWMutex
	db        *sql.DB
)

func main() {
	initusers()

	initsql()

	fmt.Println("success!")

	// for serial
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL
		)`
	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("创建用户表单失败: %q", err)
	} else {
		fmt.Println("成功创建用户表单")
	}

	// for transaction
	createTableSQL = `
		CREATE TABLE IF NOT EXISTS users1 (
			id INT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL
		)`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("创建用户表单失败: %q", err)
	} else {
		fmt.Println("成功创建用户表单")
	}

	// serial
	if err = serial(); err != nil {
		log.Fatalf("error occurs: %q", err)
	}

	// transaction
	if err = trx(); err != nil {
		log.Fatalf("error occurs: %q", err)
	}

	fmt.Println("Done")
}

func initsql() {
	var err error
	db, err = sql.Open("mysql", "root:123456@tcp(localhost:3307)/test_users")
	if err != nil {
		log.Fatalf("internal error %q", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("error connecting: %q", err)
	}

	_, err = db.Exec(`DROP TABLE IF EXISTS users`)
	if err != nil {
		log.Fatalf("%q", err)
	}

	_, err = db.Exec(`DROP TABLE IF EXISTS users1`)
	if err != nil {
		log.Fatalf("%q", err)
	}
}

func initusers() {
	for i := 1; i <= 50; i++ {
		user := &User{
			ID:       i,
			Username: fmt.Sprintf("user%d", i),
			Password: fmt.Sprintf("password%d", i),
			Name:     fmt.Sprintf("用户%d", i),
		}
		userStore[user.Username] = user
	}
}

func serial() error {
	start := time.Now()

	stmt, err := db.Prepare(`INSERT INTO users (username, password, name) VALUES(?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("声明生成失败: %q", err)
	}
	defer stmt.Close()

	mu.RLock()
	for _, user := range userStore {
		_, err := stmt.Exec(user.Username, user.Password, user.Name)
		if err != nil {
			return fmt.Errorf("插入信息失败: %q", err)
		}
	}
	mu.RUnlock()
	fmt.Printf("%v\n", time.Since(start))
	return nil
}

func trx() error {
	start := time.Now()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("error transaction: %q", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO users1 (username, password, name) VALUES(?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("声明生成失败: %q", err)
	}
	defer stmt.Close()

	mu.RLock()
	for _, user := range userStore {
		_, err := stmt.Exec(user.Username, user.Password, user.Name)
		if err != nil {
			return fmt.Errorf("插入信息失败: %q", err)
		}
	}
	mu.RUnlock()

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("事务提交失败: %q", err)
	}

	fmt.Printf("%v\n", time.Since(start))
	return nil
}
