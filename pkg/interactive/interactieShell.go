// pkg/interactive/handler.go
package interactive

import (
	"BackendTemplate/pkg/logger"
	"BackendTemplate/pkg/utils"
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

// 会话接口，不依赖sendcommand
type SessionHandler interface {
	HandleBrowserConnection()
	SendToBrowser(data map[string]interface{})
	Close()
	ProcessInput(input []byte)
	ReceiveAgentOutput(output []byte)
}

// 简化版的会话管理器
type SessionManager struct {
	mu       sync.RWMutex
	Sessions map[string]*InteractiveSession
}

var (
	DefaultManager = &SessionManager{
		Sessions: make(map[string]*InteractiveSession),
	}
)

// InteractiveSession 不依赖sendcommand
type InteractiveSession struct {
	BrowserConn *websocket.Conn
	Uid         string
	SessionID   string

	OutputQueue chan []byte
	InputQueue  chan []byte
	CloseChan   chan struct{}

	sendChan chan []byte // ⭐ 新增：所有写操作统一走它

	mu        sync.RWMutex
	Connected bool

	CommandHandler func(uid, sessionID string, input []byte)
	CloseHandler   func(uid, sessionID string)
}

func (s *InteractiveSession) WritePump() {
	for {
		select {
		case msg := <-s.sendChan:
			s.BrowserConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.BrowserConn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-s.CloseChan:
			return
		}
	}
}

// GetOrCreateSession 创建或获取会话
func (sm *SessionManager) GetOrCreateSession(sessionID, uid string, conn *websocket.Conn) *InteractiveSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.Sessions[sessionID]; exists {
		session.mu.Lock()
		session.BrowserConn = conn
		session.Connected = true
		session.mu.Unlock()
		return session
	}

	session := &InteractiveSession{
		BrowserConn: conn,
		Uid:         uid,
		SessionID:   sessionID,
		OutputQueue: make(chan []byte, 100),
		InputQueue:  make(chan []byte, 100),
		CloseChan:   make(chan struct{}),
		Connected:   true,
	}

	sm.Sessions[sessionID] = session
	return session
}

// HandleAgentOutput 处理Agent输出
func HandleAgentOutput(data []byte) (sessionID string, output []byte, ok bool) {
	buf := bytes.NewBuffer(data)
	sessionIdLenByte := make([]byte, 4)
	buf.Read(sessionIdLenByte)
	sessionIdLen := utils.ReadInt(sessionIdLenByte)
	sessionIdBytes := make([]byte, sessionIdLen)
	buf.Read(sessionIdBytes)
	command := buf.Bytes()

	return string(sessionIdBytes), command, true
}

// SendOutputToSession 发送输出到指定会话
func SendOutputToSession(uid, sessionID string, output []byte) bool {
	session := DefaultManager.GetSession(sessionID)
	if session == nil {
		//log.Printf("Session %s not found", sessionID)
		return false
	}

	if session.Uid != uid {
		//log.Printf("Session %s does not belong to uid %s", sessionID, uid)
		return false
	}

	select {
	case session.OutputQueue <- output:
		return true
	case <-time.After(1 * time.Second):
		//log.Printf("Output queue timeout for session %s", sessionID)
		return false
	}
}

// GetSession 获取会话
func (sm *SessionManager) GetSession(sessionID string) *InteractiveSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.Sessions[sessionID]
}

// RemoveSession 移除会话
func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.Sessions, sessionID)
}

// 会话方法
func (s *InteractiveSession) HandleBrowserConnection() {
	defer s.Close()

	// 设置读取超时
	s.BrowserConn.SetReadDeadline(time.Now().Add(30 * time.Minute))

	// 启动输出转发
	go s.forwardOutputToBrowser()

	for {
		messageType, data, err := s.BrowserConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				logger.Error("读取WebSocket消息错误: %v", err)
			}
			break
		}

		// 重置读取超时
		s.BrowserConn.SetReadDeadline(time.Now().Add(30 * time.Minute))

		if messageType == websocket.PingMessage {
			// 响应Ping消息
			s.BrowserConn.WriteMessage(websocket.PongMessage, nil)
			continue
		}

		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(data, &msg); err != nil {
			logger.Error("解析JSON失败: %v", err)
			continue
		}

		switch msg["type"] {
		case "heartbeat":
			// 响应心跳
			s.SendToBrowser(map[string]interface{}{
				"type":      "heartbeat_response",
				"timestamp": time.Now().Unix(),
			})
			s.BrowserConn.SetReadDeadline(time.Now().Add(30 * time.Minute))
		case "init":
			// 处理初始化
			if uid, ok := msg["uid"].(string); ok && uid == s.Uid {
				s.SendToBrowser(map[string]interface{}{
					"type":    "session_info",
					"message": "",
				})
			}
		case "input":
			if input, ok := msg["data"].(string); ok && input != "" {
				s.ProcessInput([]byte(input))
			}
		case "close":
			return
		}
	}
}

func (s *InteractiveSession) ProcessInput(input []byte) {
	if s.CommandHandler != nil {
		s.CommandHandler(s.Uid, s.SessionID, input)
	}
}

func (s *InteractiveSession) forwardOutputToBrowser() {
	for {
		select {
		case output := <-s.OutputQueue:
			s.SendToBrowser(map[string]interface{}{
				"type": "output",
				"data": string(output),
			})
		case <-s.CloseChan:
			return
		}
	}
}

func (s *InteractiveSession) SendToBrowser(data map[string]interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.BrowserConn == nil || !s.Connected {
		return
	}

	// 关键：设置写超时，避免WriteMessage阻塞
	if err := s.BrowserConn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		logger.Error("设置写超时失败: %v", err)
		return
	}

	jsonData, _ := json.Marshal(data)
	if err := s.BrowserConn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		logger.Error("发送消息到浏览器失败: %v", err)
	}
}

func (s *InteractiveSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.Connected {
		return
	}

	s.Connected = false
	if s.BrowserConn != nil {
		s.BrowserConn.Close()
	}

	close(s.CloseChan)

	// 通知管理器移除
	DefaultManager.RemoveSession(s.SessionID)

	// 调用关闭处理器
	if s.CloseHandler != nil {
		s.CloseHandler(s.Uid, s.SessionID)
	}
}
