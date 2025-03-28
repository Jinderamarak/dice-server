package web

import (
	"dice-server/common/auth/token"
	"dice-server/game/common/client"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type EntryPoint struct {
	manager *client.WebSocketManager
}

func NewEntryPoint(manager *client.WebSocketManager) *EntryPoint {
	return &EntryPoint{manager: manager}
}

func (entry *EntryPoint) Run(fail chan<- error) {
	server := gin.Default()
	server.Use(corsMiddleware())
	server.GET("/game/farkle/:auth", entry.entryHandler)

	//	TODO: get actual host
	res := server.Run("localhost:9091")
	fail <- res
}

func corsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(204)
			return
		}

		ctx.Next()
	}
}

func (entry *EntryPoint) entryHandler(ctx *gin.Context) {
	auth := ctx.Param("auth")
	gameToken, err := token.ValidateGameToken(auth, []byte(token.SuperSecret))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	entry.manager.UpgradeClient(gameToken.GameID, gameToken.UserID, conn)
}
