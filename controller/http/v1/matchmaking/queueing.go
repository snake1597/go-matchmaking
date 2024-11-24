package matchmaking

import (
	"go-matchmaking/enum"
	"go-matchmaking/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (m *MatchmakingServer) Queueing(ctx *gin.Context) {
	conn, err := m.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.NewServerError(enum.ErrorException, err))
		return
	}

	userRank := &model.UserRank{}
	err = ctx.ShouldBindJSON(&userRank)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model.NewServerError(enum.ErrorException, err))
		return
	}

}

// 問題
// 如果nats斷線 要怎麼通知在排隊的client
// 要如何把排隊中的人 放到同一局的遊戲裡 再多pod的架構下
