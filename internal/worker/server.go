package worker

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
)

type UDSServer struct {
	path     string
	registry *Registry
	listener net.Listener
	closed   chan struct{}
	once     sync.Once
}

func NewUDSServer(path string, registry *Registry) *UDSServer {
	return &UDSServer{path: path, registry: registry, closed: make(chan struct{})}
}

func (s *UDSServer) Start() error {
	if s.path == "" {
		return errors.New("uds socket path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	_ = os.Remove(s.path)
	listener, err := net.Listen("unix", s.path)
	if err != nil {
		return err
	}
	s.listener = listener
	go s.accept()
	return nil
}

func (s *UDSServer) Close() error {
	var err error
	s.once.Do(func() {
		close(s.closed)
		if s.listener != nil {
			err = s.listener.Close()
		}
		_ = os.Remove(s.path)
	})
	return err
}

func (s *UDSServer) accept() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.closed:
				return
			default:
				continue
			}
		}
		go s.handle(conn)
	}
}

func (s *UDSServer) handle(conn net.Conn) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	for {
		var message Message
		if err := decoder.Decode(&message); err != nil {
			return
		}
		s.registry.HandleMessage(message)
	}
}
