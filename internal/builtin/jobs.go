package builtin

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// JobManager 作业管理器接口（避免循环导入）
type JobManager interface {
	GetJob(jobID int) (Job, bool)
	GetAllJobs() []Job
	SetCurrentJob(jobID int)
	GetCurrentJob() Job
}

// Job 作业接口
type Job interface {
	GetID() int
	GetPID() int
	GetCmd() string
	GetStatus() JobStatus
	GetProcess() *os.Process
	SetStatus(status JobStatus)
}

// JobStatus 作业状态
type JobStatus int

const (
	JobRunning JobStatus = iota
	JobStopped
	JobDone
)

func (js JobStatus) String() string {
	switch js {
	case JobRunning:
		return "Running"
	case JobStopped:
		return "Stopped"
	case JobDone:
		return "Done"
	default:
		return "Unknown"
	}
}

// 全局变量存储JobManager引用（用于jobs/fg/bg命令）
var globalJobManager JobManager

// SetJobManager 设置JobManager引用
func SetJobManager(jm JobManager) {
	globalJobManager = jm
}

// jobs 显示作业列表
func jobs(args []string, env map[string]string) error {
	if globalJobManager == nil {
		return fmt.Errorf("jobs: job manager未初始化")
	}

	allJobs := globalJobManager.GetAllJobs()

	if len(allJobs) == 0 {
		return nil // 没有作业，不输出任何内容
	}

	for _, job := range allJobs {
		status := job.GetStatus().String()
		fmt.Printf("[%d] %s %s\n", job.GetID(), status, job.GetCmd())
	}

	return nil
}

// fg 将后台任务转到前台
func fg(args []string, env map[string]string) error {
	if globalJobManager == nil {
		return fmt.Errorf("fg: job manager未初始化")
	}

	var job Job
	var ok bool

	if len(args) == 0 {
		// 没有参数，使用当前作业或最后一个作业
		job = globalJobManager.GetCurrentJob()
		if job == nil {
			allJobs := globalJobManager.GetAllJobs()
			if len(allJobs) == 0 {
				return fmt.Errorf("fg: 当前没有作业")
			}
			job = allJobs[len(allJobs)-1]
		}
	} else {
		// 解析作业ID（支持 %1 或 1 格式）
		jobIDStr := args[0]
		if strings.HasPrefix(jobIDStr, "%") {
			jobIDStr = jobIDStr[1:]
		}
		jobID, err := strconv.Atoi(jobIDStr)
		if err != nil {
			return fmt.Errorf("fg: 无效的作业ID: %s", args[0])
		}
		job, ok = globalJobManager.GetJob(jobID)
		if !ok {
			return fmt.Errorf("fg: 作业 %d 不存在", jobID)
		}
	}

	if job.GetStatus() == JobDone {
		return fmt.Errorf("fg: 作业 %d 已完成", job.GetID())
	}

	// 设置当前作业
	globalJobManager.SetCurrentJob(job.GetID())

	// 等待进程完成
	if job.GetProcess() != nil {
		_, err := job.GetProcess().Wait()
		if err != nil {
			return err
		}
		job.SetStatus(JobDone)
	}

	return nil
}

// bg 继续后台任务
func bg(args []string, env map[string]string) error {
	if globalJobManager == nil {
		return fmt.Errorf("bg: job manager未初始化")
	}

	var job Job
	var ok bool

	if len(args) == 0 {
		// 没有参数，使用当前作业或最后一个作业
		job = globalJobManager.GetCurrentJob()
		if job == nil {
			allJobs := globalJobManager.GetAllJobs()
			if len(allJobs) == 0 {
				return fmt.Errorf("bg: 当前没有作业")
			}
			job = allJobs[len(allJobs)-1]
		}
	} else {
		// 解析作业ID（支持 %1 或 1 格式）
		jobIDStr := args[0]
		if strings.HasPrefix(jobIDStr, "%") {
			jobIDStr = jobIDStr[1:]
		}
		jobID, err := strconv.Atoi(jobIDStr)
		if err != nil {
			return fmt.Errorf("bg: 无效的作业ID: %s", args[0])
		}
		job, ok = globalJobManager.GetJob(jobID)
		if !ok {
			return fmt.Errorf("bg: 作业 %d 不存在", jobID)
		}
	}

	if job.GetStatus() == JobDone {
		return fmt.Errorf("bg: 作业 %d 已完成", job.GetID())
	}

	if job.GetStatus() == JobRunning {
		return fmt.Errorf("bg: 作业 %d 已在运行", job.GetID())
	}

	// 如果作业被停止，发送SIGCONT信号继续执行
	if job.GetProcess() != nil {
		// Windows不支持SIGCONT，这里简化处理
		// 在Unix系统上，可以使用 job.GetProcess().Signal(syscall.SIGCONT)
		// Windows上无法真正恢复进程，这里只是标记为运行
		job.SetStatus(JobRunning)
		fmt.Printf("[%d] %d\n", job.GetID(), job.GetPID())
	}

	return nil
}
