package main

import (
	"runtime"
	"time"
)

// goWorker is the actual executor who runs the tasks,
// it starts a goroutine that accepts tasks and
// performs function calls.
type goWorker struct {
	// pool who owns this worker.
	pool *Pool

	// task is a job should be done.
	task chan func()

	// recycleTime will be updated when putting a worker back into queue.
	recycleTime time.Time
}

// run starts a goroutine to repeat the process
// that performs the function calls.
func (w *goWorker) run() {
	w.pool.addRunning(1)
	go func() {
		// 这里用于退出worker的时候做扫尾处理
		defer func() {
			w.pool.addRunning(-1)
			w.pool.workerCache.Put(w)
			// 这里表示是发生异常退出
			if p := recover(); p != nil {
				if ph := w.pool.options.PanicHandler; ph != nil {
					ph(p)
				} else {
					w.pool.options.Logger.Printf("worker exits from a panic: %v\n", p)
					var buf [4096]byte
					n := runtime.Stack(buf[:], false)
					w.pool.options.Logger.Printf("worker exits from panic: %s\n", string(buf[:n]))
				}
			}
			// 通知等待的任务创建新的worker来执行
			w.pool.cond.Signal()
		}()

		for f := range w.task {
			// 说明收到了关闭worker的信号
			if f == nil {
				return
			}
			// 执行外部提交的任务
			f()
			// 将worker放回到pool当中
			if ok := w.pool.revertWorker(w); !ok {
				return
			}
		}
	}()
}
