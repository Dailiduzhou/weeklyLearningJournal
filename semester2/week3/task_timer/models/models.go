package models

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Task struct {
	ID       string
	Interval time.Duration
	Timeout  time.Duration
}

type TaskManager struct {
	ctx         context.Context
	cancel      context.CancelFunc
	submitChan  chan Task
	taskCancels map[string]context.CancelFunc
	mu          sync.Mutex
	wg          sync.WaitGroup
}

func NewTaskManager() *TaskManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &TaskManager{
		ctx:         ctx,
		cancel:      cancel,
		submitChan:  make(chan Task),
		taskCancels: make(map[string]context.CancelFunc, 10),
	}
}

func (m *TaskManager) Start() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		slog.Info("任务管理器开启")
		for {
			select {
			case <-m.ctx.Done():
				slog.Info("接收到结束信号")
				return
			case task := <-m.submitChan:
				m.runTask(task)
			}
		}
	}()
}

func (m *TaskManager) Submit(t Task) {
	select {
	case m.submitChan <- t:
		slog.Info("任务提交成功", "ID", t.ID)
	case <-m.ctx.Done():
		slog.Warn("任务管理器已关闭")
	}
}

func (m *TaskManager) runTask(t Task) {
	taskCtx, taskcancel := context.WithCancel(m.ctx)

	taskCtx = context.WithValue(taskCtx, "taskID", t.ID)

	m.mu.Lock()
	m.taskCancels[t.ID] = taskcancel
	m.mu.Unlock()

	m.wg.Add(1)

	go func(ctx context.Context, task Task) {
		defer m.wg.Done()
		defer func() {
			m.mu.Lock()
			delete(m.taskCancels, task.ID)
			m.mu.Unlock()
		}()

		ticker := time.NewTicker(task.Interval)
		defer ticker.Stop()

		taskID := ctx.Value("taskID").(string)

		for {
			select {
			case <-ctx.Done():
				slog.Info("定时任务已结束", "taskID", taskID, "reason", ctx.Err())
				return
			case <-ticker.C:
				slog.Info("定时任务开始新一轮调度", "taskID", taskID)
				m.executeWork(ctx, t)
			}
		}
	}(taskCtx, t)
}

func (m *TaskManager) executeWork(ctx context.Context, t Task) {
	workCtx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel() // 避免上下文泄漏

	done := make(chan struct{})
	go func() {
		time.Sleep(600 * time.Millisecond)
		close(done)
	}()

	select {
	case <-workCtx.Done():
		slog.Info("任务本次执行被中断/超时", "taskID", t.ID, "detail", workCtx.Err())
	case <-done:
		slog.Info("任务本次执行顺利完成", "taskID", t.ID)
	}
}

// CancelTask 停止指定的某一个任务
func (m *TaskManager) CancelTask(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cancel, exists := m.taskCancels[taskID]; exists {
		slog.Info("正在请求取消任务", "taskID", taskID)
		cancel() // 触发 Level 2 Context 的取消
	} else {
		slog.Info("任务不存在或已停止", "taskID", taskID)
	}
}

// Stop 关闭整个管理器及其所有任务
func (m *TaskManager) Stop() {
	slog.Info("[操作] 正在关闭整个任务管理器...")
	m.cancel() // 触发 Level 1 Context 的取消，所有派生的 Context 都会收到信号
	m.wg.Wait()
	slog.Info("[系统] 所有任务已清理完毕，系统安全退出。")
}
