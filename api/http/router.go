package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"hanafi_fiqh_qa/internal/auth"
	"hanafi_fiqh_qa/internal/base/errors"
	"hanafi_fiqh_qa/internal/base/request"
	"hanafi_fiqh_qa/internal/user"
)

func initRouter(server *Server) {
	router := &router{
		Server: server,
	}

	router.init()
}

type router struct {
	*Server
}

func (r *router) init() {
	r.engine.Use(r.trace())
	r.engine.Use(r.recover())
	r.engine.Use(r.logger())

	r.engine.POST("/login", r.login)

	r.engine.POST("/users", r.addUser)
	r.engine.GET("/users/me", r.authenticate, r.getMe)
	r.engine.PUT("/users/me", r.authenticate, r.updateMe)
	r.engine.PATCH("/users/me/password", r.authenticate, r.changeMyPassword)

	r.engine.NoRoute(r.methodNotFound)
}

func (r *router) login(c *gin.Context) {
	var loginUserDto auth.LoginUserDto

	if err := bindBody(&loginUserDto, c); err != nil {
		errorResponse(err, nil, r.config.DetailedError()).reply(c)
		return
	}

	user, err := r.authService.Login(c, loginUserDto)
	if err != nil {
		errorResponse(err, nil, r.config.DetailedError()).reply(c)
		return
	}

	okResponse(user).reply(c)
}

func (r *router) authenticate(c *gin.Context) {
	token := c.Request.Header.Get("Authorization")

	userId, err := r.authService.VerifyAccessToken(token)
	if err != nil {
		response := errorResponse(err, nil, r.config.DetailedError())
		c.AbortWithStatusJSON(response.Status, response)
	}

	setUserId(c, userId)
}

func (r *router) addUser(c *gin.Context) {
	var addUserDto user.AddUserDto

	if err := bindBody(&addUserDto, c); err != nil {
		errorResponse(err, nil, r.config.DetailedError()).reply(c)
		return
	}

	user, err := r.userUsecases.Add(contextWithReqInfo(c), addUserDto)
	if err != nil {
		errorResponse(err, nil, r.config.DetailedError()).reply(c)
		return
	}

	okResponse(user).reply(c)
}

func (r *router) updateMe(c *gin.Context) {
	var updateUserDto user.UpdateUserDto

	reqInfo := getReqInfo(c)
	updateUserDto.Id = reqInfo.UserId

	if err := bindBody(&updateUserDto, c); err != nil {
		errorResponse(err, nil, r.config.DetailedError()).reply(c)
		return
	}

	err := r.userUsecases.Update(contextWithReqInfo(c), updateUserDto)
	if err != nil {
		errorResponse(err, nil, r.config.DetailedError()).reply(c)
		return
	}

	okResponse(nil).reply(c)
}

func (r *router) changeMyPassword(c *gin.Context) {
	var changeUserPasswordDto user.ChangeUserPasswordDto

	reqInfo := getReqInfo(c)
	changeUserPasswordDto.Id = reqInfo.UserId

	if err := bindBody(&changeUserPasswordDto, c); err != nil {
		errorResponse(err, nil, r.config.DetailedError()).reply(c)
		return
	}

	err := r.userUsecases.ChangePassword(contextWithReqInfo(c), changeUserPasswordDto)
	if err != nil {
		errorResponse(err, nil, r.config.DetailedError()).reply(c)
		return
	}

	okResponse(nil).reply(c)
}

func (r *router) getMe(c *gin.Context) {
	reqInfo := getReqInfo(c)

	user, err := r.userUsecases.GetById(contextWithReqInfo(c), reqInfo.UserId)
	if err != nil {
		errorResponse(err, nil, r.config.DetailedError()).reply(c)
		return
	}

	okResponse(user).reply(c)
}

func (r *router) methodNotFound(c *gin.Context) {
	err := errors.New(errors.NotFoundError, "method not found")
	errorResponse(err, nil, r.config.DetailedError()).reply(c)
}

func (r *router) recover() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		response := internalErrorResponse(nil)
		c.AbortWithStatusJSON(response.Status, response)
	})
}

func (r *router) trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceId := c.Request.Header.Get("Trace-Id")
		if traceId == "" {
			traceId, _ = r.crypto.GenerateUUID()
		}

		setTraceId(c, traceId)
	}
}

func (r *router) logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		var parsedReqInfo request.RequestInfo

		reqInfo, exists := param.Keys[reqInfoKey]
		if exists {
			parsedReqInfo = reqInfo.(request.RequestInfo)
		}

		return fmt.Sprintf("%s - [HTTP] TraceId: %s; UserId: %d; Method: %s; Path: %s; Status: %d, Latency: %s;\n\n",
			param.TimeStamp.Format(time.RFC1123),
			parsedReqInfo.TraceId,
			parsedReqInfo.UserId,
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
		)
	})
}

func bindBody(payload interface{}, c *gin.Context) error {
	err := c.BindJSON(payload)

	if err != nil {
		return errors.New(errors.BadRequestError, err.Error())
	}

	return nil
}

type response struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func okResponse(data interface{}) *response {
	return &response{
		Status:  http.StatusOK,
		Message: "ok",
		Data:    data,
	}
}

func internalErrorResponse(data interface{}) *response {
	status, message := http.StatusInternalServerError, "internal error"

	return &response{
		Status:  status,
		Message: message,
		Data:    data,
	}
}

func errorResponse(err error, data interface{}, withDetails bool) *response {
	status, message, details := parseError(err)

	if withDetails && details != "" {
		message = details
	}
	return &response{
		Status:  status,
		Message: message,
		Data:    data,
	}
}

func (r *response) reply(c *gin.Context) {
	c.JSON(r.Status, r)
}
