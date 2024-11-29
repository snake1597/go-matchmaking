package matchmaking

import (
	"encoding/json"
	"go-matchmaking/enum"
	"go-matchmaking/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 問題
// 發生錯誤要通知排隊失敗
// 在多pod的架構下 要如何把排隊中的人 放到同一局的遊戲裡
// 排到隊 但還是可以取消要怎麼實作
// 只能單一連線 處理併發連接

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

	m.hubSrv.Register(userRank.UserID, conn)

	queueingInfo := &model.UserQueueingInfo{
		UserID: userRank.UserID,
		Rank:   userRank.Rank,
	}

	infoByte, err := json.Marshal(queueingInfo)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.NewServerError(enum.ErrorException, err))
		return
	}

	err = m.queueSrv.Publish(infoByte)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.NewServerError(enum.ErrorException, err))
		return
	}
}
