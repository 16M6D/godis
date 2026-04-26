package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	defaultBindAddr = "0.0.0.0"
	defaultPort     = 6379
	defaultDBNum    = 16
)

type serverConfig struct {
	bind  string
	port  int
	dbNum int
}

type redisServer struct {
	cfg      serverConfig
	listener net.Listener
	closing  chan struct{}
	wg       sync.WaitGroup
}

func initServerConfig(cfg *serverConfig) {
	cfg.bind = defaultBindAddr
	cfg.port = defaultPort
	cfg.dbNum = defaultDBNum
}

func loadServerConfig(cfg *serverConfig) {
	flag.StringVar(&cfg.bind, "bind", cfg.bind, "bind address")
	flag.IntVar(&cfg.port, "port", cfg.port, "tcp port")
	flag.IntVar(&cfg.dbNum, "databases", cfg.dbNum, "number of logical databases")
	flag.Parse()
}

func printBanner(cfg serverConfig) {
	log.Printf("Godis is starting")
	log.Printf("Configuration loaded: bind=%s port=%d databases=%d", cfg.bind, cfg.port, cfg.dbNum)
}

func initServer(server *redisServer) error {
	addr := fmt.Sprintf("%s:%d", server.cfg.bind, server.cfg.port)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	server.listener = ln
	server.closing = make(chan struct{})
	return nil
}

func setupSignalHandler(server *redisServer) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		log.Printf("Received signal %s, scheduling shutdown", sig)
		server.close()
	}()
}

func aeMain(server *redisServer) {
	for {
		conn, err := server.listener.Accept()
		if err != nil {
			select {
			case <-server.closing:
				server.wg.Wait()
				log.Printf("Godis is now ready to exit, bye bye...")
				return
			default:
			}

			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				log.Printf("Temporary accept error: %v", err)
				time.Sleep(50 * time.Millisecond)
				continue
			}

			log.Printf("Unrecoverable accept error: %v", err)
			return
		}

		server.wg.Add(1)
		go server.handleClient(conn)
	}
}

func (s *redisServer) handleClient(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	remote := conn.RemoteAddr().String()
	log.Printf("Accepted connection from %s", remote)

	_, _ = io.WriteString(conn, "+OK godis is ready\r\n")

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error from %s: %v", remote, err)
			}
			return
		}

		log.Printf("Received from %s: %q", remote, line)
		_, _ = io.WriteString(conn, "-ERR command handling is not implemented yet\r\n")
	}
}

func (s *redisServer) close() {
	select {
	case <-s.closing:
		return
	default:
		close(s.closing)
	}

	if s.listener != nil {
		_ = s.listener.Close()
	}
}

func main() {
	server := &redisServer{}

	initServerConfig(&server.cfg)
	loadServerConfig(&server.cfg)

	printBanner(server.cfg)

	if err := initServer(server); err != nil {
		log.Fatalf("Failed initializing the server: %v", err)
	}
	defer server.close()

	setupSignalHandler(server)

	log.Printf("Server initialized, ready to accept connections tcp://%s:%d", server.cfg.bind, server.cfg.port)
	aeMain(server)
}
