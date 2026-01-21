package ipc

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	PipeName       = `\\.\pipe\tsf_ime_service`
	BufferSize     = 4096
	MaxMessageSize = 1024 * 1024 // 1MB
)

// MessageHandler 处理消息的接口
type MessageHandler interface {
	HandleConvert(data ConvertData) ([]Candidate, error)
}

// Server 命名管道服务器
type Server struct {
	handler MessageHandler
	logger  *slog.Logger
}

// NewServer 创建新的服务器
func NewServer(handler MessageHandler, logger *slog.Logger) *Server {
	return &Server{
		handler: handler,
		logger:  logger,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	s.logger.Info("Starting IPC server", "pipe", PipeName)

	// 创建允许所有用户访问的安全描述符
	// SDDL 字符串: D:P(A;;GA;;;WD)(A;;GA;;;SY)(A;;GA;;;BA)
	// WD = Everyone, SY = SYSTEM, BA = Built-in Administrators
	sddl := "D:P(A;;GA;;;WD)(A;;GA;;;SY)(A;;GA;;;BA)"
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		s.logger.Error("Failed to create security descriptor", "error", err)
		// 继续使用默认安全设置
		sd = nil
	} else {
		s.logger.Info("Security descriptor created", "sddl", sddl)
	}

	var sa *windows.SecurityAttributes
	if sd != nil {
		sa = &windows.SecurityAttributes{
			Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
			SecurityDescriptor: sd,
		}
	}

	for {
		// 创建命名管道
		pipePath, err := windows.UTF16PtrFromString(PipeName)
		if err != nil {
			return fmt.Errorf("failed to convert pipe name: %w", err)
		}

		handle, err := windows.CreateNamedPipe(
			pipePath,
			windows.PIPE_ACCESS_DUPLEX,
			windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT,
			windows.PIPE_UNLIMITED_INSTANCES,
			BufferSize,
			BufferSize,
			0,
			sa, // 使用安全描述符
		)

		if err != nil {
			return fmt.Errorf("failed to create named pipe: %w", err)
		}

		s.logger.Info("Waiting for client connection...")

		// 等待客户端连接
		err = windows.ConnectNamedPipe(handle, nil)
		if err != nil && err != windows.ERROR_PIPE_CONNECTED {
			windows.CloseHandle(handle)
			s.logger.Error("Failed to connect named pipe", "error", err)
			continue
		}

		s.logger.Info("Client connected")

		// 处理客户端连接（在新的 goroutine 中）
		go s.handleClient(handle)
	}
}

// handleClient 处理客户端连接
func (s *Server) handleClient(handle windows.Handle) {
	defer windows.CloseHandle(handle)

	s.logger.Info("Starting to handle client messages")

	for {
		s.logger.Info("Waiting to read next message...")

		// 读取消息
		request, err := s.readMessage(handle)
		if err != nil {
			if err != io.EOF {
				s.logger.Error("Failed to read message", "error", err)
			}
			break
		}

		// 处理请求
		response := s.processRequest(request)

		// 发送响应
		if err := s.writeMessage(handle, response); err != nil {
			s.logger.Error("Failed to write response", "error", err)
			break
		}
	}

	s.logger.Info("Client disconnected")
}

// readMessage 读取消息
func (s *Server) readMessage(handle windows.Handle) (*Request, error) {
	// 读取消息长度
	var messageLen uint32
	var bytesRead uint32

	s.logger.Info("Attempting to read message length from pipe...")

	err := windows.ReadFile(handle, (*[4]byte)(unsafe.Pointer(&messageLen))[:], &bytesRead, nil)
	if err != nil {
		s.logger.Error("ReadFile (length) failed", "error", err)
		return nil, err
	}

	s.logger.Info("Read message length", "length", messageLen, "bytesRead", bytesRead)

	if messageLen == 0 || messageLen > MaxMessageSize {
		s.logger.Error("Invalid message length", "length", messageLen, "max", MaxMessageSize)
		return nil, fmt.Errorf("invalid message length: %d", messageLen)
	}

	// 读取消息内容
	buffer := make([]byte, messageLen)
	err = windows.ReadFile(handle, buffer, &bytesRead, nil)
	if err != nil {
		s.logger.Error("ReadFile (content) failed", "error", err)
		return nil, err
	}

	s.logger.Info("Read message content", "bytesRead", bytesRead, "expected", messageLen)

	// 显示原始 JSON（前200字节）
	jsonPreview := string(buffer)
	if len(jsonPreview) > 200 {
		jsonPreview = jsonPreview[:200] + "..."
	}
	s.logger.Info("Received JSON", "content", jsonPreview)

	// 解析 JSON
	var request Request
	if err := json.Unmarshal(buffer, &request); err != nil {
		s.logger.Error("Failed to parse JSON", "error", err, "raw", string(buffer))
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	s.logger.Info("Parsed request", "type", request.Type, "input", request.Data.Input)

	return &request, nil
}

// writeMessage 写入消息
func (s *Server) writeMessage(handle windows.Handle, response *Response) error {
	// 序列化为 JSON
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// 写入消息长度
	messageLen := uint32(len(data))
	var bytesWritten uint32

	err = windows.WriteFile(handle, (*[4]byte)(unsafe.Pointer(&messageLen))[:], &bytesWritten, nil)
	if err != nil {
		return err
	}

	// 写入消息内容
	err = windows.WriteFile(handle, data, &bytesWritten, nil)
	if err != nil {
		return err
	}

	return nil
}

// processRequest 处理请求
func (s *Server) processRequest(request *Request) *Response {
	s.logger.Info("Processing request", "type", request.Type)

	switch request.Type {
	case RequestTypeConvert:
		s.logger.Info("Matched RequestTypeConvert case")
		candidates, err := s.handler.HandleConvert(request.Data)
		if err != nil {
			s.logger.Error("HandleConvert failed", "error", err)
			return &Response{
				Status: "error",
				Error:  err.Error(),
			}
		}

		s.logger.Info("HandleConvert succeeded", "candidatesCount", len(candidates))
		return &Response{
			Status:     "success",
			Candidates: candidates,
		}

	default:
		s.logger.Error("Unknown request type", "type", request.Type, "expected", RequestTypeConvert)
		return &Response{
			Status: "error",
			Error:  fmt.Sprintf("unknown request type: %s", request.Type),
		}
	}
}
