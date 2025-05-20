package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"msps/internal/app/model/common"
	"msps/internal/app/model/domain"
	"net/http"
	"sync"
)

type MailVerifyMap struct {
	Map map[string]EmailVerifyInfo
	mu  sync.Mutex
}

type EmailVerifyInfo struct {
	Success bool `json:"success"`
}

type MailProbeMap struct {
	Map map[string]domain.EmailProbeReq
	mu  sync.Mutex
}

func NewMailVerifyMap() *MailVerifyMap {
	return &MailVerifyMap{
		Map: make(map[string]EmailVerifyInfo),
	}
}

func NewMailProbeMap() *MailProbeMap {
	return &MailProbeMap{
		Map: make(map[string]domain.EmailProbeReq),
	}
}

func (m *MailVerifyMap) SetEmailVerifyInfo(id string, info EmailVerifyInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Map[id] = info
}

func (m *MailProbeMap) SetEmailProbeReq(id string, req domain.EmailProbeReq) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Map[id] = req
}

func (q *MailQueue) Dequeue() (domain.EmailReq, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.Queue) == 0 {
		return domain.EmailReq{}, errQueueEmpty
	}

	req := <-q.Queue
	q.remainingCount++
	return req, nil
}

var (
	errQueueEmpty = errors.New("queue is empty")
	VerifyMap     = NewMailVerifyMap()
	ProbeMap      = NewMailProbeMap()
)

type Agent struct {
}

// HandleSentEmail
// @Summary 邮件获取
// @Description 处理来自Agent的邮件获取请求
// @tags Agent
// @Accept json
// @Produce json
// @Success 200 {object} common.Response "{"success":true,"msg":"","data":null}"
// @Failure 400 {object} common.Response "{"success":false,"msg":"请求参数错误","data":null}"
// @Failure 401 {object} common.Response "{"success":false,"msg":"用户未登录","data":null}"
// @Failure 403 {object} common.Response "{"success":false,"msg":"访问受限","data":null}"
// @Failure 404 {object} common.Response "{"success":false,"msg":"路径不存在","data":null}"
// @Failure 500 {object} common.Response "{"success":false,"msg":"Internal Server Error","data":null}"
// @Router /a/m [post]
func (a *Agent) HandleSentEmail(c *gin.Context) {
	req, err := EmailQueue.Dequeue()
	if err != nil {
		if errors.Is(err, errQueueEmpty) {
			c.JSON(http.StatusServiceUnavailable, common.NewResponse(common.WithMsg("队列为空")))
			return
		} else {
			c.JSON(http.StatusInternalServerError, common.NewResponse(common.WithMsg(common.MsgInternalServerError)))
			return
		}
	}

	c.JSON(http.StatusOK, common.NewResponse(common.WithSuccess(true), common.WithPayload(req)))
}

// HealthCheck
// @Summary 健康检查
// @Description 处理来自Agent的心跳请求
// @tags Agent
// @Accept json
// @Produce json
// @Success 200 {object} common.Response "{"success":true,"msg":"","data":null}"
// @Failure 400 {object} common.Response "{"success":false,"msg":"请求参数错误","data":null}"
// @Failure 401 {object} common.Response "{"success":false,"msg":"用户未登录","data":null}"
// @Failure 403 {object} common.Response "{"success":false,"msg":"访问受限","data":null}"
// @Failure 404 {object} common.Response "{"success":false,"msg":"路径不存在","data":null}"
// @Failure 500 {object} common.Response "{"success":false,"msg":"Internal Server Error","data":null}"
// @Router /a/h [post]
func (a *Agent) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, common.NewResponse(common.WithSuccess(true)))
}

// HandleVerifyEmail
// @Summary 邮件发送结果接收
// @Description 处理来自Agent的邮件结果请求
// @tags Agent
// @Accept json
// @Produce json
// @Param body body domain.EmailVerifyReq true "请求参数"
// @Success 200 {object} common.Response "{"success":true,"msg":"","data":null}"
// @Failure 400 {object} common.Response "{"success":false,"msg":"请求参数错误","data":null}"
// @Failure 401 {object} common.Response "{"success":false,"msg":"用户未登录","data":null}"
// @Failure 403 {object} common.Response "{"success":false,"msg":"访问受限","data":null}"
// @Failure 404 {object} common.Response "{"success":false,"msg":"路径不存在","data":null}"
// @Failure 500 {object} common.Response "{"success":false,"msg":"Internal Server Error","data":null}"
// @Router /a/v [post]
func (a *Agent) HandleVerifyEmail(c *gin.Context) {
	var req domain.EmailVerifyReq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewResponse(common.WithMsg(common.MsgInvalidParam)))
		return
	}

	// TODO 处理邮件确认请求
	verifyInfo := EmailVerifyInfo{
		Success: req.Success,
	}
	VerifyMap.SetEmailVerifyInfo(req.ID, verifyInfo)

	c.JSON(http.StatusOK, common.NewResponse(common.WithSuccess(true)))
}
