package job

import (
	"errors"

	"mall/admin_core/pkg/response"

	"github.com/gin-gonic/gin"
)

type API struct{ service *Service }

func NewAPI(service *Service) *API { return &API{service: service} }

// List godoc
// @Summary 获取内置后台任务状态
// @Description 返回四个内置定时任务的最近运行状态、执行计数和错误摘要。
// @ID listSystemJobs
// @Tags system-job
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response[[]job.JobStatusResponse]
// @Router /api/system/jobs [get]
func (a *API) List(c *gin.Context) {
	rows, err := a.service.List(c.Request.Context())
	if err != nil {
		if errors.Is(err, ErrManagerClosed) {
			response.Error(c, "本实例任务管理器未运行，正在等待恢复")
			return
		}
		response.Error(c, "获取任务状态失败")
		return
	}
	response.Data(c, rows, len(rows))
}

// Retry godoc
// @Summary 人工重试最终失败的后台任务
// @Description 仅接受最终失败且租约已释放的内置任务，返回后由后台异步执行。
// @ID retrySystemJob
// @Tags system-job
// @Produce json
// @Security BearerAuth
// @Param job_name path string true "内置任务名称"
// @Success 200 {object} response.Response[job.RetryAccepted]
// @Router /api/system/jobs/{job_name}/retry [post]
func (a *API) Retry(c *gin.Context) {
	jobName := c.Param("job_name")
	err := a.service.Retry(jobName)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnknownJob):
			response.Error(c, "未知任务")
		case errors.Is(err, ErrNotFailed):
			response.Error(c, "仅最终失败的任务可以人工重试")
		case errors.Is(err, ErrJobLocked):
			response.Error(c, "任务正在运行或租约尚未过期")
		default:
			response.Error(c, "任务重试未能启动")
		}
		return
	}
	response.Data(c, RetryAccepted{JobName: jobName, Status: StatusRunning}, 0)
}
