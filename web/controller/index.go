package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
	"x-ui/logger"
	"x-ui/web/job"
	"x-ui/web/service"
	"x-ui/web/session"

	"github.com/gin-gonic/gin"
)

type LoginForm struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

type IndexController struct {
	BaseController

	userService service.UserService
}

type Location struct {
	Country string `json:"country"`
	City    string `json:"city"`
}

func NewIndexController(g *gin.RouterGroup) *IndexController {
	a := &IndexController{}
	a.initRouter(g)
	return a
}

func (a *IndexController) initRouter(g *gin.RouterGroup) {
	g.GET("/", a.index)
	g.POST("/login", a.login)
	g.GET("/logout", a.logout)
}

func (a *IndexController) index(c *gin.Context) {
	if session.IsLogin(c) {
		c.Redirect(http.StatusTemporaryRedirect, "xui/")
		return
	}
	html(c, "login.html", "登录", nil)
}

func (a *IndexController) login(c *gin.Context) {
	ip := c.ClientIP()
	url := "http://ip-api.com/json/" + ip
	response, err := http.Get(url)
	if err != nil {
		logger.Error(err)
	}
	location := Location{}
	body, _ := io.ReadAll(response.Body)
	err = json.Unmarshal(body, &location)
	if err != nil {
		logger.Error(err)
	}
	if location.Country != "China" && location.Country != "china" {
		logger.Infof("国外IP: %s, 国家: %s", ip, location.Country)
		pureJsonMsg(c, false, "非国内IP")
		return
	}
	var form LoginForm
	err = c.ShouldBind(&form)
	if err != nil {
		pureJsonMsg(c, false, "数据格式错误")
		return
	}
	if form.Username == "" {
		pureJsonMsg(c, false, "请输入用户名")
		return
	}
	if form.Password == "" {
		pureJsonMsg(c, false, "请输入密码")
		return
	}
	user := a.userService.CheckUser(form.Username, form.Password)
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	if user == nil {
		job.NewStatsNotifyJob().UserLoginNotify(form.Username, getRemoteIp(c), timeStr, 0)
		logger.Infof("wrong username or password: \"%s\" \"%s\" ip: \"%s\"", form.Username, form.Password, c.ClientIP())
		pureJsonMsg(c, false, "用户名或密码错误")
		return
	} else {
		logger.Infof("%s login success,Ip Address:%s\n", form.Username, getRemoteIp(c))
		job.NewStatsNotifyJob().UserLoginNotify(form.Username, getRemoteIp(c), timeStr, 1)
	}

	err = session.SetLoginUser(c, user)
	logger.Info("user", user.Id, "login success")
	jsonMsg(c, "登录", err)
}

func (a *IndexController) logout(c *gin.Context) {
	user := session.GetLoginUser(c)
	if user != nil {
		logger.Info("user", user.Id, "logout")
	}
	session.ClearSession(c)
	c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path"))
}
