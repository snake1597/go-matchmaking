package matchmaking

import (
	"go-matchmaking/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type MatchmakingServer struct {
	upgrader websocket.Upgrader
	hubSrv   service.HubService
	queueSrv service.QueueService
}

func NewMatchmakingServer(
	gin *gin.Engine,
	hubSrv service.HubService,
	queueSrv service.QueueService,
) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	m := &MatchmakingServer{
		upgrader: upgrader,
		hubSrv:   hubSrv,
		queueSrv: queueSrv,
	}

	r := gin.Group("/api/v1/matchmaking")
	{
		r.POST("/queueing", m.Queueing)
	}
}
