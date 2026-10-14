package minecraft

import (
	"Aether/pkg/models"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

func writeVarInt(w io.Writer, val int) error {
	for {
		b := byte(val & 0x7F)
		val >>= 7
		if val != 0 {
			b |= 0x80
		}
		if _, err := w.Write([]byte{b}); err != nil {
			return err
		}
		if val == 0 {
			break
		}
	}
	return nil
}

func readVarInt(r io.Reader) (int, error) {
	var result int
	var shift uint
	buf := make([]byte, 1)
	for {
		if _, err := r.Read(buf); err != nil {
			return 0, err
		}
		b := buf[0]
		result |= int(b&0x7F) << shift
		shift += 7
		if shift > 35 {
			return 0, fmt.Errorf("varint too long")
		}
		if (b & 0x80) == 0 {
			break
		}
	}
	return result, nil
}

func PingServerAddress(addr string, timeout time.Duration) (*models.ServerLiveStatus, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
		portStr = "25565"
	}
	port, _ := strconv.Atoi(portStr)
	if port <= 0 {
		port = 25565
	}

	t0 := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	// 1. Build Handshake Packet
	var hsPayload bytes.Buffer
	_ = writeVarInt(&hsPayload, 0x00) // Packet ID 0x00
	_ = writeVarInt(&hsPayload, 763)  // Protocol version 1.20.1
	// Host string
	_ = writeVarInt(&hsPayload, len(host))
	hsPayload.WriteString(host)
	// Port
	_ = binary.Write(&hsPayload, binary.BigEndian, uint16(port))
	// Next state 1 (status)
	_ = writeVarInt(&hsPayload, 1)

	// Send Handshake with prepended length
	var hsPacket bytes.Buffer
	_ = writeVarInt(&hsPacket, hsPayload.Len())
	hsPacket.Write(hsPayload.Bytes())
	if _, err := conn.Write(hsPacket.Bytes()); err != nil {
		return nil, err
	}

	// 2. Status Request Packet: length 1, packet ID 0x00
	if _, err := conn.Write([]byte{0x01, 0x00}); err != nil {
		return nil, err
	}

	// 3. Read Status Response
	_, err = readVarInt(conn) // total length
	if err != nil {
		return nil, err
	}

	pktID, err := readVarInt(conn)
	if err != nil || pktID != 0x00 {
		return nil, fmt.Errorf("invalid packet id: %d", pktID)
	}

	jsonLen, err := readVarInt(conn)
	if err != nil || jsonLen <= 0 || jsonLen > 1000000 {
		return nil, fmt.Errorf("invalid json length")
	}

	jsonBuf := make([]byte, jsonLen)
	if _, err := io.ReadFull(conn, jsonBuf); err != nil {
		return nil, err
	}

	pingMs := time.Since(t0).Milliseconds()
	if pingMs < 1 {
		pingMs = 1
	}

	var parsed struct {
		Version struct {
			Name string `json:"name"`
		} `json:"version"`
		Players struct {
			Max    int `json:"max"`
			Online int `json:"online"`
			Sample []struct {
				Name string `json:"name"`
			} `json:"sample"`
		} `json:"players"`
		Description interface{} `json:"description"`
		Favicon     string      `json:"favicon"`
	}

	if err := json.Unmarshal(jsonBuf, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	motd := "Chill Town Minecraft Server"
	if parsed.Description != nil {
		switch d := parsed.Description.(type) {
		case string:
			motd = d
		case map[string]interface{}:
			if t, ok := d["text"].(string); ok && strings.TrimSpace(t) != "" {
				motd = strings.TrimSpace(t)
			}
		}
	}

	var playerList []string
	for _, p := range parsed.Players.Sample {
		if p.Name != "" {
			playerList = append(playerList, p.Name)
		}
	}

	verName := parsed.Version.Name
	if verName == "" {
		verName = "Forge 1.20.1"
	}

	return &models.ServerLiveStatus{
		Online:     true,
		PingMs:     pingMs,
		Players:    parsed.Players.Online,
		MaxPlayers: parsed.Players.Max,
		Version:    verName,
		MOTD:       motd,
		Favicon:    parsed.Favicon,
		PlayerList: playerList,
	}, nil
}

// GetServerLiveStatus attempts to ping the official Minecraft server across local tunnel, LAN, and standard ports
func GetServerLiveStatus() *models.ServerLiveStatus {
	// Priority 1: Local tunnel listener (127.0.0.1:25565, 127.0.0.1:25566)
	// Priority 2: Local home LAN (192.168.1.7:25565)
	candidates := []string{
		"127.0.0.1:25565",
		"127.0.0.1:25566",
		"192.168.1.7:25565",
	}

	for _, cand := range candidates {
		status, err := PingServerAddress(cand, 1200*time.Millisecond)
		if err == nil && status != nil && status.Online {
			return status
		}
	}

	// Offline fallback
	return &models.ServerLiveStatus{
		Online:     false,
		PingMs:     -1,
		Players:    0,
		MaxPlayers: 20,
		Version:    "Forge 1.20.1",
		MOTD:       "Máy chủ đang tạm dừng hoặc ngoại tuyến",
	}
}
