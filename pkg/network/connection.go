package network

import (
	"net"
	"strconv"
	"time"
)

// CheckPort 检查指定主机和端口是否可连接
// 返回值：true表示可连接，false表示不可连接
func CheckPort(host string, port int) bool {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	timeout := 5 * time.Second

	// 尝试建立TCP连接
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}
