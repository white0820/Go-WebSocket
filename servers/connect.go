package servers

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/woodylan/go-websocket/api"
	"github.com/woodylan/go-websocket/define/retcode"
	"github.com/woodylan/go-websocket/tools/util"
	"net/http"
)

const (
	// 最大的消息大小
	maxMessageSize = 8192
)

type Controller struct {
}

type renderData struct {
	ClientId string `json:"clientId"`
}

// 使用http请求将连接升级为websocket
func (c *Controller) Run(w http.ResponseWriter, r *http.Request) {
	conn, err := (&websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// 允许所有CORS跨域请求
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}).Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("upgrade error: %v", err)
		http.NotFound(w, r)
		return
	}

	//设置读取消息大小上限
	conn.SetReadLimit(maxMessageSize)

	//解析参数
	systemId := r.FormValue("systemId")
	if len(systemId) == 0 {
		_ = Render(conn, "", "", retcode.SYSTEM_ID_ERROR, "系统ID不能为空", []string{})
		_ = conn.Close()
		return
	}

	// 根据服务器的ip和端口号通过配置里面的密钥进行对称加密，生成的密文作为id
	clientId := util.GenClientId()

	// 一个clientsocket中包含很多信息，其中也包括建立的websocket连接
	clientSocket := NewClient(clientId, systemId, conn)

	// 将刚才生成的clientsocket添加到该systemid服务器下的clientsocket列表中
	Manager.AddClient2SystemClient(systemId, clientSocket)

	//一直循环去读取客户端消息
	clientSocket.Read()

	if err = api.ConnRender(conn, renderData{ClientId: clientId}); err != nil {
		_ = conn.Close()
		return
	}

	// 用户连接事件
	Manager.Connect <- clientSocket
}
