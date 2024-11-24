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
