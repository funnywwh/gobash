package executor

import (
	"os"
	"os/exec"
	"sync"
	"time"
	"gobash/internal/builtin"
)

// Job 作业结构
type Job struct {
	ID        int           // 作业ID
	PID       int           // 进程ID
	Cmd       string        // 命令字符串
	Status    JobStatus     // 状态
	StartTime time.Time     // 开始时间
	Process   *os.Process   // 进程对象
	mu        sync.Mutex    // 互斥锁
}

// GetID 获取作业ID
func (j *Job) GetID() int {
	return j.ID
}

// GetPID 获取进程ID
func (j *Job) GetPID() int {
	return j.PID
}

// GetCmd 获取命令字符串
func (j *Job) GetCmd() string {
	return j.Cmd
}

// GetStatus 获取状态
func (j *Job) GetStatus() builtin.JobStatus {
	j.mu.Lock()
	defer j.mu.Unlock()
	return builtin.JobStatus(j.Status)
}

// SetStatus 设置状态
func (j *Job) SetStatus(status builtin.JobStatus) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = JobStatus(status)
}

// GetProcess 获取进程对象
func (j *Job) GetProcess() *os.Process {
	return j.Process
}

// 使用builtin包中定义的JobStatus类型
type JobStatus = builtin.JobStatus

const (
	JobRunning = builtin.JobRunning
	JobStopped = builtin.JobStopped
	JobDone    = builtin.JobDone
)

// JobManager 作业管理器
type JobManager struct {
	jobs    map[int]*Job
	nextID  int
	current int // 当前作业ID（+表示前台，-表示后台）
	mu      sync.Mutex
}

// NewJobManager 创建新的作业管理器
func NewJobManager() *JobManager {
	return &JobManager{
		jobs:   make(map[int]*Job),
		nextID: 1,
	}
}

// AddJob 添加作业
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
	}

	jm.jobs[jm.nextID] = job
	id := jm.nextID
	jm.nextID++

	// 在goroutine中等待进程完成
	go func() {
		cmd.Wait()
		jm.mu.Lock()
		if job, ok := jm.jobs[id]; ok {
			job.Status = JobDone
		}
		jm.mu.Unlock()
	}()

	return id
}

// GetJob 获取作业（返回接口类型以匹配builtin包的接口）
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
func (jm *JobManager) RemoveJob(id int) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	delete(jm.jobs, id)
}

// GetCurrentJob 获取当前作业（返回接口类型以匹配builtin包的接口）
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
func (jm *JobManager) SetCurrentJob(id int) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	jm.current = id
}

