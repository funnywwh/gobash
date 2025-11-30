package executor

import (
	"os"
	"os/exec"
	"sync"
	"time"
	"gobash/internal/builtin"
)

// Job 作业结构
// 表示一个后台任务，包含作业ID、进程ID、命令字符串、状态等信息
type Job struct {
	ID        int           // 作业ID
	PID       int           // 进程ID
	Cmd       string        // 命令字符串
	Status    JobStatus     // 状态
	StartTime time.Time     // 开始时间
	Process   *os.Process   // 进程对象
	cmd       *exec.Cmd     // 保存cmd引用以便Wait
	done      chan struct{}  // 进程完成通知channel
	mu        sync.Mutex    // 互斥锁
}

// GetID 获取作业ID
// 返回作业的唯一标识符
func (j *Job) GetID() int {
	return j.ID
}

// GetPID 获取进程ID
// 返回作业对应的操作系统进程ID
func (j *Job) GetPID() int {
	return j.PID
}

// GetCmd 获取命令字符串
// 返回作业执行的命令字符串
func (j *Job) GetCmd() string {
	return j.Cmd
}

// GetStatus 获取状态
// 返回作业的当前状态（Running、Stopped、Done）
func (j *Job) GetStatus() builtin.JobStatus {
	j.mu.Lock()
	defer j.mu.Unlock()
	return builtin.JobStatus(j.Status)
}

// SetStatus 设置状态
// 更新作业的状态，使用互斥锁保证线程安全
func (j *Job) SetStatus(status builtin.JobStatus) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = JobStatus(status)
}

// GetProcess 获取进程对象
// 返回作业对应的操作系统进程对象
func (j *Job) GetProcess() *os.Process {
	return j.Process
}

// Wait 等待作业完成
// 阻塞直到作业完成，通过done channel实现
func (j *Job) Wait() error {
	if j.done == nil {
		return nil // 如果done channel不存在，说明作业已经完成或不存在
	}
	<-j.done
	return nil
}

// 使用builtin包中定义的JobStatus类型
type JobStatus = builtin.JobStatus

const (
	JobRunning = builtin.JobRunning
	JobStopped = builtin.JobStopped
	JobDone    = builtin.JobDone
)

// JobManager 作业管理器
// 管理所有后台作业，提供添加、查询、删除作业的功能
type JobManager struct {
	jobs    map[int]*Job
	nextID  int
	current int // 当前作业ID（+表示前台，-表示后台）
	mu      sync.Mutex
}

// NewJobManager 创建新的作业管理器
// 初始化作业管理器，返回一个新的JobManager实例
func NewJobManager() *JobManager {
	return &JobManager{
		jobs:   make(map[int]*Job),
		nextID: 1,
	}
}

// AddJob 添加作业
// 将一个新的后台任务添加到管理器中，返回作业ID
// 在goroutine中等待进程完成并更新状态
func (jm *JobManager) AddJob(cmd *exec.Cmd, cmdStr string) int {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	job := &Job{
		ID:        jm.nextID,
		PID:       cmd.Process.Pid,
		Cmd:       cmdStr,
		Status:    JobRunning,
		StartTime: time.Now(),
		Process:   cmd.Process,
		cmd:       cmd,
		done:      make(chan struct{}),
	}

	jm.jobs[jm.nextID] = job
	id := jm.nextID
	jm.nextID++

	// 在goroutine中等待进程完成
	go func(jobID int, doneChan chan struct{}) {
		cmd.Wait()
		close(doneChan)
		jm.mu.Lock()
		if job, ok := jm.jobs[jobID]; ok {
			job.Status = JobDone
		}
		jm.mu.Unlock()
	}(id, job.done)

	return id
}

// GetJob 获取作业（返回接口类型以匹配builtin包的接口）
// 根据作业ID查找作业，返回Job接口和是否找到的布尔值
func (jm *JobManager) GetJob(id int) (builtin.Job, bool) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	job, ok := jm.jobs[id]
	if !ok {
		return nil, false
	}
	return job, true
}

// GetAllJobs 获取所有作业（返回接口类型以匹配builtin包的接口）
// 返回所有未完成的作业列表（不包括已完成的作业）
func (jm *JobManager) GetAllJobs() []builtin.Job {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	jobs := make([]builtin.Job, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		// 只返回未完成的作业
		if job.Status != JobDone {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// RemoveJob 移除作业（清理已完成的作业）
// 从管理器中删除指定ID的作业
func (jm *JobManager) RemoveJob(id int) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	delete(jm.jobs, id)
}

// GetCurrentJob 获取当前作业（返回接口类型以匹配builtin包的接口）
// 返回当前活动的作业，如果没有则返回nil
func (jm *JobManager) GetCurrentJob() builtin.Job {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	if jm.current > 0 {
		if job, ok := jm.jobs[jm.current]; ok {
			return job
		}
	}
	return nil
}

// SetCurrentJob 设置当前作业
// 将指定ID的作业设置为当前活动作业
func (jm *JobManager) SetCurrentJob(id int) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	jm.current = id
}

