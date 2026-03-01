package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "root:123456@tcp(localhost:3307)/emp_db")
	if err != nil {
		log.Fatalf("internal error %q", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("error connecting: %q", err)
	}

	fmt.Println("success!")

	rows, err := db.Query(`SELECT * FROM employee`)
	if err != nil {
		log.Fatalf("查询失败: %q", err)
	}
	defer rows.Close()

	for rows.Next() {
		var emp_id, emp_name, gender,
			age, salary, hire_date, dept_id, is_valid, create_time string
		err = rows.Scan(&emp_id, &emp_name, &gender, &age, &salary, &hire_date, &dept_id, &is_valid, &create_time)
		if err != nil {
			log.Fatalf("录入失败: %q", err)
		}

		fmt.Printf("emp_id: %s, emp_name: %s, gender: %s, age: %s, salary: %s, hire_date: %s, dept_id: %s, is_valid: %s, create_time: %s\n",
			emp_id, emp_name, gender, age, salary, hire_date, dept_id, is_valid, create_time)
	}
}
