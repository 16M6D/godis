package main

import (
	"golang.org/x/sys/unix"
	"log"
)

const BACKLOG int = 64 // 全连接队列数量

func Accept(fd int) (int, error) {
	nfd, _, err := unix.Accept(fd) // 暂时不关心客户端地址
	return nfd, err
}

func Connect(host [4]byte, port int) (int, error) {
	s, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0) // ipv4, tcp
	if err != nil {
		log.Printf("init socket err: %v\n", err)
		return -1, err
	}
	var addr unix.SockaddrInet4
	addr.Addr = host
	addr.Port = port
	err = unix.Connect(s, &addr)
	if err != nil {
		log.Printf("connect err: %v\n", err)
		return -1, err
	}
	return s, nil
}

func Read(fd int, buf []byte) (int, error) {
	return unix.Read(fd, buf)
}

func Write(fd int, buf []byte) (int, error) {
	return unix.Write(fd, buf)
}

func Close(fd int) {
	unix.Close(fd)
}

func TcpServer(port int) (int, error) {
	s, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		log.Printf("init socket err: %v\n", err)
		return -1, err
	}
	err = unix.SetsockoptInt(s, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1) // 多个socket绑定同一个端口, 由内核分发连接
	if err != nil {
		log.Printf("set SO_REUSEPORT err: %v\n", err)
		unix.Close(s)
		return -1, err
	}
	var addr unix.SockaddrInet4 // 0.0.0.0:port = 监听本机上所有网卡地址
	// golang.syscall will handle htons
	addr.Port = port
	// golang will set addr.Addr = any(0)
	err = unix.Bind(s, &addr) // fd与地址绑定
	if err != nil {
		log.Printf("bind addr err: %v\n", err)
		unix.Close(s)
		return -1, err
	}
	err = unix.Listen(s, BACKLOG)
	if err != nil {
		log.Printf("listen socket err: %v\n", err)
		unix.Close(s)
		return -1, err
	}
	return s, nil // 调用者得到这个fd后需要自己写cfd accept的循环
}
