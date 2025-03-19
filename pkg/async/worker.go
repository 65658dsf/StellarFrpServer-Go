package async

import (
	"context"
	"fmt"
	"sync"
	"time"

	"stellarfrp/pkg/logger"
)

// Task 表示一个异步任务
type Task struct {
	ID       string
	Handler  func(ctx context.Context) error
	Timeout  time.Duration
	RetryMax int
}

// Result 表示任务执行结果
type Result struct {
	TaskID    string
	Completed bool
	Error     error
	StartTime time.Time
	EndTime   time.Time
}

// Worker 异步任务处理器
type Worker struct {
	taskQueue chan Task
	results   map[string]Result
	mu        sync.RWMutex
	logger    *logger.Logger
	wg        sync.WaitGroup
}

// NewWorker 创建一个新的工作器
func NewWorker(queueSize int, logger *logger.Logger) *Worker {
	return &Worker{
		taskQueue: make(chan Task, queueSize),
		results:   make(map[string]Result),
		logger:    logger,
	}
}

// Start 启动工作器
func (w *Worker) Start(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		w.wg.Add(1)
		go w.processTask()
	}
}

// Stop 停止工作器
func (w *Worker) Stop() {
	close(w.taskQueue)
	w.wg.Wait()
}

// AddTask 将任务加入队列
func (w *Worker) AddTask(handler func()) {
	task := Task{
		ID: fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Handler: func(ctx context.Context) error {
			handler()
			return nil
		},
	}
	w.taskQueue <- task
}

// GetResult 获取任务结果
func (w *Worker) GetResult(taskID string) (Result, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	result, exists := w.results[taskID]
	return result, exists
}

// processTask 处理任务的工作循环
func (w *Worker) processTask() {
	defer w.wg.Done()

	for task := range w.taskQueue {
		w.executeTask(task)
	}
}

// executeTask 执行单个任务
func (w *Worker) executeTask(task Task) {
	result := Result{
		TaskID:    task.ID,
		StartTime: time.Now(),
	}

	w.logger.Info("Starting async task", "task_id", task.ID)

	// 创建带超时的上下文
	ctx := context.Background()
	if task.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, task.Timeout)
		defer cancel()
	}

	// 执行任务，支持重试
	var err error
	for attempt := 0; attempt <= task.RetryMax; attempt++ {
		if attempt > 0 {
			w.logger.Info("Retrying task", "task_id", task.ID, "attempt", attempt)
			time.Sleep(time.Second * time.Duration(attempt)) // 简单的退避策略
		}

		err = task.Handler(ctx)
		if err == nil {
			break
		}

		w.logger.Error("Task execution failed", "task_id", task.ID, "attempt", attempt, "error", err)
	}

	result.EndTime = time.Now()
	result.Error = err
	result.Completed = (err == nil)

	// 存储结果
	w.mu.Lock()
	w.results[task.ID] = result
	w.mu.Unlock()

	if err != nil {
		w.logger.Error("Async task failed", "task_id", task.ID, "error", err)
	} else {
		w.logger.Info("Async task completed successfully", "task_id", task.ID, "duration", result.EndTime.Sub(result.StartTime))
	}
}
