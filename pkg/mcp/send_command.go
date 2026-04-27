package mcp

import (
"encoding/json"
"fmt"
"sync"
"time"
)

var clientMutexes sync.Map

func getClientMutex(uid string) *sync.Mutex {
	m, _ := clientMutexes.LoadOrStore(uid, &sync.Mutex{})
	return m.(*sync.Mutex)
}

func getShellContentLength(uid string) int {
	respStr, err := localApiCall("GET", "/api/client/shell/getshellcontent", map[string]string{"uid": uid}, nil)
	if err != nil {
		return 0
	}
	var res map[string]interface{}
	if err := json.Unmarshal([]byte(respStr), &res); err == nil {
		if data, ok := res["data"].(string); ok {
			return len(data)
		}
	}
	return 0
}

func getShellContentString(uid string) string {
	respStr, err := localApiCall("GET", "/api/client/shell/getshellcontent", map[string]string{"uid": uid}, nil)
	if err != nil {
		return ""
	}
	var res map[string]interface{}
	if err := json.Unmarshal([]byte(respStr), &res); err == nil {
		if data, ok := res["data"].(string); ok {
			return data
		}
	}
	return ""
}

func executeCommandSync(uid, cmd string) (string, error) {
	mu := getClientMutex(uid)
	mu.Lock()
	defer mu.Unlock()

	// initial length
	startLen := getShellContentLength(uid)

	body := map[string]interface{}{"uid": uid, "command": cmd}
	_, err := localApiCall("POST", "/api/client/shell/sendcommand", nil, body)
	if err != nil {
		return "", err
	}

	// Wait for `[+] command is executing`
	time.Sleep(200 * time.Millisecond)
	lenAfterAck := getShellContentLength(uid)
	
	// We will poll up to 60 times (6 seconds)
	for i := 0; i < 60; i++ {
		time.Sleep(100 * time.Millisecond)
		currentContent := getShellContentString(uid)
		currentLen := len(currentContent)

		if currentLen > lenAfterAck {
			// Find the diff
			diff := currentLen - startLen
			if diff > 0 {
				newOutput := currentContent[startLen:]
				return fmt.Sprintf("%s", newOutput), nil
			}
			return "Command executed.", nil
		}
	}

	// Maybe it finished really fast and the lenAfterAck is actually the final length!
	// Let's just return what we got
currentContent := getShellContentString(uid)
if len(currentContent) > startLen {
return currentContent[startLen:], nil
}

return "Command timed out waiting for output (might still be running or produced no output).", nil
}
