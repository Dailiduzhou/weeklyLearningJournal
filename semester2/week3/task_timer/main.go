package main

import (
	"fmt"
	"time"

	"github.com/Dailiduzhou/weeklyLearningJournal/semester2/week3/task_timer/models"
)

func main() {
	manager := models.NewTaskManager()
	manager.Start()

	// 提交任务 A：间隔 1 秒执行，单次执行最大允许超时时间 1 秒 (能顺利完成)
	manager.Submit(models.Task{ID: "Task-A", Interval: 1 * time.Second, Timeout: 1 * time.Second})

	// 提交任务 B：间隔 1 秒执行，单次执行最大允许超时时间 0.3 秒 (必定触发执行超时)
	manager.Submit(models.Task{ID: "Task-B", Interval: 1 * time.Second, Timeout: 300 * time.Millisecond})

	// 让系统运行一会，观察输出
	time.Sleep(3 * time.Second)

	fmt.Println("\n=================================")
	// 演示 1：独立取消 Task-A（Task-B 将继续运行）
	manager.CancelTask("Task-A")

	// 继续运行一会，观察只有 Task-B 在跑
	time.Sleep(2 * time.Second)

	fmt.Println("\n=================================")
	// 演示 2：全局取消。关闭管理器（触发嵌套取消：剩下的 Task-B 也会被强制停止）
	manager.Stop()
}
