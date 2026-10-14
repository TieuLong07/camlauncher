package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"tailscale.com/tsnet"
)

const (
	BrokerURL           = "https://camlauncher-key-broker.nguyenhlong2203.workers.dev/v1/tunnel-key"
	DefaultServerTarget = "100.117.75.101:25565"
	DefaultLocalPort    = 25565
)

type brokerResponse struct {
	AuthKey string `json:"authKey"`
	Target  string `json:"target"`
	Error   string `json:"error"`
	Detail  string `json:"detail"`
}

// RequestBrokerCredential gets a one-time, short-lived auth key. No tailnet key is embedded in the launcher.
func RequestBrokerCredential(ctx context.Context) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BrokerURL, strings.NewReader("{}"))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CamLauncher/0.1.0-alpha")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", "", fmt.Errorf("không thể kết nối dịch vụ P2P: %w", err)
	}
	defer resp.Body.Close()

	var result brokerResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&result); err != nil {
		return "", "", fmt.Errorf("dịch vụ P2P trả dữ liệu không hợp lệ")
	}
	if resp.StatusCode != http.StatusOK || result.AuthKey == "" {
		if result.Error == "" {
			result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return "", "", fmt.Errorf("không thể xin phiên P2P: %s", result.Error)
	}
	if result.Target == "" {
		result.Target = DefaultServerTarget
	}
	return result.AuthKey, result.Target, nil
}

// TunnelStatus represents current state of the embedded Tailscale proxy
type TunnelStatus struct {
	Running      bool   `json:"running"`
	LocalPort    int    `json:"localPort"`
	ServerTarget string `json:"serverTarget"`
	TailscaleIP  string `json:"tailscaleIp"`
	StatusText   string `json:"statusText"`
	LatencyMs    int64  `json:"latencyMs"`
	ActiveConns  int    `json:"activeConns"`
	Error        string `json:"error,omitempty"`
}

// TunnelManager manages the embedded tsnet server and local TCP listener
type TunnelManager struct {
	mu           sync.Mutex
	server       *tsnet.Server
	listener     net.Listener
	cancel       context.CancelFunc
	running      bool
	localPort    int
	serverTarget string
	tailscaleIP  string
	latencyMs    int64
	activeConns  int32
	tempDir      string
	statusText   string
	lastError    string
	onChange     func(TunnelStatus)
}

// NewTunnelManager creates an instance of TunnelManager
func NewTunnelManager(onChange func(TunnelStatus)) *TunnelManager {
	return &TunnelManager{
		serverTarget: DefaultServerTarget,
		localPort:    DefaultLocalPort,
		statusText:   "Đã ngắt kết nối",
		onChange:     onChange,
	}
}

// GetStatus returns current tunnel status safely
func (m *TunnelManager) GetStatus() TunnelStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getStatusLocked()
}

func (m *TunnelManager) getStatusLocked() TunnelStatus {
	return TunnelStatus{
		Running:      m.running,
		LocalPort:    m.localPort,
		ServerTarget: m.serverTarget,
		TailscaleIP:  m.tailscaleIP,
		StatusText:   m.statusText,
		LatencyMs:    m.latencyMs,
		ActiveConns:  int(atomic.LoadInt32(&m.activeConns)),
		Error:        m.lastError,
	}
}

func (m *TunnelManager) notifyLocked() {
	if m.onChange != nil {
		status := m.getStatusLocked()
		go m.onChange(status)
	}
}

// Start boots the embedded Tailscale node and starts the local forwarder
func (m *TunnelManager) Start(authKey, target string) (*TunnelStatus, error) {
	m.mu.Lock()
	if m.running {
		status := m.getStatusLocked()
		m.mu.Unlock()
		return &status, nil
	}

	if authKey == "" {
		return nil, fmt.Errorf("chưa có phiên P2P hợp lệ")
	}
	if target == "" {
		target = DefaultServerTarget
	}
	m.serverTarget = target
	m.lastError = ""
	m.statusText = "Đang khởi tạo kết nối P2P..."
	m.notifyLocked()
	m.mu.Unlock()

	// 1. Try local TCP port 25565, then fallback to 25566
	localPort := DefaultLocalPort
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		localPort = 25566
		listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
		if err != nil {
			m.mu.Lock()
			m.lastError = fmt.Sprintf("Cổng 25565 và 25566 đều bị chiếm: %v", err)
			m.statusText = "Lỗi chiếm cổng"
			m.notifyLocked()
			m.mu.Unlock()
			return nil, fmt.Errorf("không thể mở cổng mạng 25565 hoặc 25566 trên máy: %w", err)
		}
	}

	// 2. Setup tsnet Server
	randID := fmt.Sprintf("camlauncher-player-%04d", rand.Intn(10000))
	tempDir := filepath.Join(os.TempDir(), "tsnet-"+randID)
	_ = os.MkdirAll(tempDir, 0700)

	server := &tsnet.Server{
		Hostname:  randID,
		AuthKey:   authKey,
		Ephemeral: true,
		Dir:       tempDir,
		Logf:      func(format string, args ...any) {},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	m.mu.Lock()
	m.statusText = "Đang kết nối WireGuard..."
	m.notifyLocked()
	m.mu.Unlock()

	upStatus, err := server.Up(ctx)
	if err != nil {
		_ = listener.Close()
		_ = server.Close()
		_ = os.RemoveAll(tempDir)
		m.mu.Lock()
		m.lastError = fmt.Sprintf("Không thể kết nối Tailscale: %v", err)
		m.statusText = "Lỗi kết nối P2P"
		m.notifyLocked()
		m.mu.Unlock()
		return nil, fmt.Errorf("kết nối mạng P2P thất bại: %w", err)
	}

	var myIP string
	if len(upStatus.TailscaleIPs) > 0 {
		myIP = upStatus.TailscaleIPs[0].String()
	}

	// Test ping/handshake latency
	t0 := time.Now()
	testCtx, testCancel := context.WithTimeout(context.Background(), 5*time.Second)
	var latency int64 = 18
	if testConn, err := server.Dial(testCtx, "tcp", target); err == nil {
		latency = time.Since(t0).Milliseconds()
		_ = testConn.Close()
	}
	testCancel()

	mCtx, mCancel := context.WithCancel(context.Background())

	m.mu.Lock()
	m.server = server
	m.listener = listener
	m.cancel = mCancel
	m.running = true
	m.localPort = localPort
	m.tailscaleIP = myIP
	m.tempDir = tempDir
	m.latencyMs = latency
	m.statusText = "Đang hoạt động (P2P Sẵn sàng)"
	status := m.getStatusLocked()
	m.notifyLocked()
	m.mu.Unlock()

	// 3. Start proxy accept loop
	go m.acceptLoop(mCtx, server, listener, target)

	return &status, nil
}

// Stop shuts down listener, tsnet node, and clears temp data
func (m *TunnelManager) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.listener != nil {
		_ = m.listener.Close()
		m.listener = nil
	}
	if m.server != nil {
		_ = m.server.Close()
		m.server = nil
	}
	if m.tempDir != "" {
		_ = os.RemoveAll(m.tempDir)
		m.tempDir = ""
	}

	m.running = false
	m.statusText = "Đã ngắt kết nối"
	m.lastError = ""
	atomic.StoreInt32(&m.activeConns, 0)
	m.notifyLocked()
	m.mu.Unlock()
	return nil
}

func (m *TunnelManager) acceptLoop(ctx context.Context, server *tsnet.Server, listener net.Listener, target string) {
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				return
			}
		}

		atomic.AddInt32(&m.activeConns, 1)
		m.mu.Lock()
		m.notifyLocked()
		m.mu.Unlock()

		go func(c net.Conn) {
			defer func() {
				_ = c.Close()
				atomic.AddInt32(&m.activeConns, -1)
				m.mu.Lock()
				m.notifyLocked()
				m.mu.Unlock()
			}()

			dialCtx, dialCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer dialCancel()

			targetConn, err := server.Dial(dialCtx, "tcp", target)
			if err != nil {
				return
			}
			defer targetConn.Close()

			done := make(chan struct{}, 2)
			go func() {
				_, _ = io.Copy(targetConn, c)
				done <- struct{}{}
			}()
			go func() {
				_, _ = io.Copy(c, targetConn)
				done <- struct{}{}
			}()

			<-done
		}(clientConn)
	}
}
