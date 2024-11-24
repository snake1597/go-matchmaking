package matchmaking

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type MatchmakingServer struct {
	upgrader websocket.Upgrader
}

func NewMatchmakingServer(gin *gin.Engine) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	m := &MatchmakingServer{
		upgrader: upgrader,
	}

	r := gin.Group("/api/v1/matchmaking")
	{
		r.POST("/queueing", m.Queueing)
	}
}
