package encrypt

import (
	"BackendTemplate/pkg/database"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/curve25519"
	"io"
	r "math/rand"
	"strconv"
)

// 密钥生成
func generateKey() []byte {
	key := make([]byte, 32) // 256 bits key
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return key
}

// 加密函数
func encryptAES(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], data)

	return ciphertext, nil
}

// 解密函数
func decryptAES(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext is too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}

func EncryptNormal(data []byte) ([]byte, error) {
	key := generateKey()
	encryptedData, _ := encryptAES([]byte(hex.EncodeToString(data)), key)
	// key与加密结果放到一起
	keyAndData := append(key, encryptedData...)
	return keyAndData, nil
}

func DecryptNormal(data []byte) ([]byte, error) {
	if len(data) > 0 {
		key := data[:32]
		encryptedData := data[32:]
		decryptedData, _ := decryptAES(encryptedData, key)
		plainData, _ := hex.DecodeString(string(decryptedData))
		return plainData, nil
	}
	return nil, nil
}

func Encrypt(data []byte, uid string) ([]byte, error) {
	var client database.Clients
	database.Engine.Where("uid = ?", uid).Get(&client)
	pubkeyBytes, _ := base64.StdEncoding.DecodeString(client.PublicKey)

	var pubKey [32]byte
	copy(pubKey[:], pubkeyBytes[:32])
	key, _ := ComputeSharedSecret(Key.Private, pubKey)

	//key := generateKey()
	encryptedData, _ := encryptAES([]byte(hex.EncodeToString(data)), key[:])
	// key与加密结果放到一起
	//keyAndData := append(key, encryptedData...)
	return encryptedData, nil
}

func Decrypt(data []byte, uid string) ([]byte, error) {
	if len(data) > 0 {
		var client database.Clients
		database.Engine.Where("uid = ?", uid).Get(&client)
		pubkeyBytes, _ := base64.StdEncoding.DecodeString(client.PublicKey)

		var pubKey [32]byte
		copy(pubKey[:], pubkeyBytes[:32])
		key, _ := ComputeSharedSecret(Key.Private, pubKey)

		//key := data[:32]
		//encryptedData := data[32:]
		decryptedData, _ := decryptAES(data, key[:])
		plainData, _ := hex.DecodeString(string(decryptedData))
		return plainData, nil
	}
	return nil, nil
}

// EncodeBase64 将 []byte 编码为 Base64 并返回 []byte
func EncodeBase64(data []byte) ([]byte, error) {
	encodedString := base64.StdEncoding.EncodeToString(data)
	return []byte(encodedString), nil
}

// DecodeBase64 将 Base64 编码的 []byte 解码回原始的 []byte
func DecodeBase64(encodedData []byte) ([]byte, error) {
	decodedData, err := base64.StdEncoding.DecodeString(string(encodedData))
	if err != nil {
		return nil, err
	}
	return decodedData, nil
}

// StringToMD5 计算字符串的 MD5 哈希值
func BytesToMD5(s []byte) string {
	// 创建一个新的 MD5 哈希对象
	hasher := md5.New()

	// 将字符串写入哈希对象
	hasher.Write(s)

	// 获取哈希值的字节切片
	md5Bytes := hasher.Sum(nil)

	// 将字节切片转换为十六进制字符串
	return fmt.Sprintf("%x", md5Bytes)
}

func GenRandomLogID() string {
	// 设定随机数的范围
	min := 3000000000
	max := 4000000000 // 确保最大值大于 3796460677，以包含它

	// 生成随机数
	randomNumber := r.Intn(max-min) + min
	randomString := strconv.Itoa(randomNumber)
	return randomString
}

// 生成一个指定长度的随机字节数组
func GenRandomBytes() []byte {
	min := 3000
	max := 4000 // 确保最大值大于 3796460677，以包含它
	// 生成随机数
	n := r.Intn(max-min) + min
	// 创建一个长度为 n 的字节切片
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil
	}
	return b
}

type KeyPair struct {
	Public  [32]byte
	Private [32]byte
}

var Key KeyPair

func GenerateKeyPair() {
	database.Engine.Where("1=1")
	count, _ := database.Engine.Count(&database.Key{})
	if count != 0 {
		var key database.Key
		database.Engine.Where("1=1").Get(&key)
		prikeyBytes, _ := base64.StdEncoding.DecodeString(key.PrivateKey)
		pubkeyBytes, _ := base64.StdEncoding.DecodeString(key.PublicKey)
		var prikey, pubkey [32]byte
		copy(prikey[:], prikeyBytes[:32])
		copy(pubkey[:], pubkeyBytes[:32])
		Key = KeyPair{
			Public:  pubkey,
			Private: prikey,
		}
		return
	}
	var publicKey, privateKey [32]byte

	// 生成私钥（需要随机数）
	if _, err := rand.Read(privateKey[:]); err != nil {
		return
	}

	// 计算公钥
	curve25519.ScalarBaseMult(&publicKey, &privateKey)
	database.Engine.Where("1 = 1").Delete(&database.Key{})
	database.Engine.Insert(&database.Key{
		PublicKey:  base64.StdEncoding.EncodeToString(publicKey[:]),
		PrivateKey: base64.StdEncoding.EncodeToString(privateKey[:]),
	})
	Key = KeyPair{
		Public:  publicKey,
		Private: privateKey,
	}
	return
}

// 计算共享密钥
func ComputeSharedSecret(privateKey, peerPublicKey [32]byte) ([32]byte, error) {
	sharedSecret, err := curve25519.X25519(privateKey[:], peerPublicKey[:])
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to compute shared secret: %w", err)
	}

	var result [32]byte
	copy(result[:], sharedSecret)
	return result, nil
}
