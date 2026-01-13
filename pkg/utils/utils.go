package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

func BytesCombine(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, []byte(""))
}

// Paginate takes a slice of any type, and returns a paginated slice.
// It returns an empty slice if the input is invalid or out of bounds.
func Paginate(slice interface{}, page, pageSize int) interface{} {
	// 获取输入切片的反射值
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice || rv.IsNil() {
		return nil
	}

	// 如果 pageSize <= 0，则直接返回空切片
	if pageSize <= 0 {
		return reflect.MakeSlice(rv.Type(), 0, 0).Interface()
	}

	// 计算总项数
	totalItems := rv.Len()

	// 边界处理：如果 totalItems <= 0，返回空切片
	if totalItems == 0 {
		return reflect.MakeSlice(rv.Type(), 0, 0).Interface()
	}

	// 计算总页数
	totalPages := (totalItems + pageSize - 1) / pageSize

	// 边界处理：如果 page <= 0，返回第一页
	if page <= 0 {
		page = 1
	}

	// 边界处理：如果 page 超出范围，返回最后一页
	if page > totalPages {
		page = totalPages
	}

	// 计算当前页的起始和结束索引
	start := (page - 1) * pageSize

	// 边界处理：确保 start 不小于 0
	if start < 0 {
		start = 0
	}

	end := start + pageSize

	// 确保 end 不超过切片的长度
	if end > totalItems {
		end = totalItems
	}

	// 使用反射创建一个新的切片，并复制数据
	paginatedSlice := reflect.MakeSlice(rv.Type(), end-start, end-start)
	reflect.Copy(paginatedSlice, rv.Slice(start, end))

	// 返回分页后的切片
	return paginatedSlice.Interface()
}

// ProcessInfo 结构体代表一行进程信息
type ProcessInfo struct {
	Pid  string `json:"PID"`
	PPid string `json:"PPID"`
	Name string `json:"Name"`
	Arch string `json:"Arch"`
	User string `json:"User"`
}

func ParsePid(input string) []ProcessInfo {
	var processes []ProcessInfo
	lines := strings.Split(strings.TrimSpace(input), "\n")

	lines = lines[1:]

	for _, line := range lines {
		fields := strings.Split(line, "\t")

		processes = append(processes, ProcessInfo{
			Name: fields[0],
			PPid: fields[1],
			Pid:  fields[2],
			Arch: fields[3],
			User: fields[4],
		})
	}

	return processes
}

// bytesToSize 将字节数转换为合适的单位，并保留一位小数。
func BytesToSize(bytesStr string) string {
	if bytesStr == "0" {
		return ""
	}
	// 将字符串转换为整数
	bytes, err := strconv.ParseInt(bytesStr, 10, 64)
	if err != nil {
		return ""
	}

	var units = []string{"B", "KB", "MB", "GB", "TB"}
	var unitIndex int
	var size float64
	size = float64(bytes)

	// 如果字节小于 1024 KB，则直接返回字节数
	if size < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}

	// 找到合适的单位并计算大小
	for size >= 1024 && unitIndex < len(units)-1 {
		unitIndex++
		size /= 1024
	}

	// 格式化输出，保留一位小数
	return fmt.Sprintf("%.1f%s", size, units[unitIndex])
}
func SplitByteArray(data []byte, chunkSize int) [][]byte {
	var result [][]byte
	for i := 0; i < len(data); i += chunkSize {
		// 如果剩余的字节长度小于 chunkSize，直接取剩余部分
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		result = append(result, data[i:end])
	}
	return result
}
func InitFunction() {
	_, err := os.Stat("./Downloads")
	if os.IsNotExist(err) {
		// 文件夹不存在，创建文件夹
		err = os.MkdirAll("./Downloads", os.ModePerm)
	}
}

// getExistingDrives 从给定的掩码中提取存在的逻辑驱动器
func GetExistingDrives(drivesMask []byte) []string {
	var existingDrives []string
	//
	//// 遍历 A 到 Z 的驱动器
	//for drive := 'A'; drive <= 'Z'; drive++ {
	//	// 计算当前驱动器的位掩码
	//	bit := 7 - (drive-'A')%8
	//	byteIndex := (drive - 'A') / 8
	//	mask := 1 << uint(bit)
	//
	//	// 检查当前驱动器是否存在
	//	if int(byteIndex) < len(drivesMask) && (drivesMask[byteIndex]&byte(mask) != 0) {
	//		existingDrives = append(existingDrives, string(drive)+":")
	//	}
	//}
	drivers := string(drivesMask)
	for _, driver := range drivers {
		existingDrives = append(existingDrives, string(driver)+":")
	}
	return existingDrives
}
func Uint32ToIP(ip uint32) net.IP {
	// 将uint32类型的数据转换为字节切片
	bytes := make([]byte, 4)
	bytes[0] = byte((ip >> 24) & 0xFF)
	bytes[1] = byte((ip >> 16) & 0xFF)
	bytes[2] = byte((ip >> 8) & 0xFF)
	bytes[3] = byte(ip & 0xFF)
	return net.IPv4(bytes[0], bytes[1], bytes[2], bytes[3])
}
func WriteInt(nInt int) []byte {
	bBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bBytes, uint32(nInt))
	return bBytes
}
func ReadInt(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}
func GetSafeFilePath(uid, filePath string) (string, error) {
	// 验证 UID 格式
	if uid == "" || strings.Contains(uid, "..") || strings.Contains(uid, "/") || strings.Contains(uid, "\\") {
		return "", fmt.Errorf("invalid UID")
	}

	// 获取安全文件名
	safeFileName := filepath.Base(filePath)
	if safeFileName == "" || safeFileName == "." || safeFileName == ".." {
		return "", fmt.Errorf("invalid filename")
	}

	// 清理文件名
	safeFileName = strings.ReplaceAll(safeFileName, "/", "")
	safeFileName = strings.ReplaceAll(safeFileName, "\\", "")

	// 构建安全路径
	downloadDir := filepath.Join("./Downloads", uid)
	fullPath := filepath.Join(downloadDir, safeFileName)

	// 验证路径安全性
	cleanFullPath := filepath.Clean(fullPath)
	cleanDownloadDir := filepath.Clean(downloadDir)

	if !strings.HasPrefix(cleanFullPath, cleanDownloadDir+string(os.PathSeparator)) && cleanFullPath != cleanDownloadDir {
		return "", fmt.Errorf("path traversal attempt")
	}

	return fullPath, nil
}
