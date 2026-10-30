// Package hash 提供 MD5 计算工具。
package hash

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// bufSize 计算 MD5 时的缓冲区大小。
const bufSize = 512 * 1024

// MD5Reader 计算 reader 的 MD5 与字节数。
func MD5Reader(r io.Reader) (string, int64, error) {
	h := md5.New()
	buf := make([]byte, bufSize)
	n, err := io.CopyBuffer(h, r, buf)
	if err != nil {
		return "", n, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// MD5File 计算磁盘文件的 MD5 与字节数。
func MD5File(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	return MD5Reader(f)
}

// MD5Bytes 计算字节切片 MD5。
func MD5Bytes(b []byte) string {
	sum := md5.Sum(b)
	return hex.EncodeToString(sum[:])
}

// ValidMD5 校验字符串是否为合法的小写 32 位 MD5。
func ValidMD5(s string) bool {
	if len(s) != 32 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
