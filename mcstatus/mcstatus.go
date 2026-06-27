package mcstatus

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

type Player struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type StatusResponse struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int      `json:"max"`
		Online int      `json:"online"`
		Sample []Player `json:"sample"`
	} `json:"players"`
	MOTD struct {
		Text  string `json:"text"`
		Clean string `json:"clean"`
		Extra []struct {
			Text string `json:"text"`
		} `json:"extra"`
	} `json:"motd"`
	Favicon        string `json:"favicon"`
	Description    struct {
		Text string `json:"text"`
	} `json:"description"`
	RoundTripLatency int64 `json:"-"`
}

func encodeVarInt(val int) []byte {
	var buf []byte
	for {
		b := byte(val & 0x7F)
		val >>= 7
		if val != 0 {
			b |= 0x80
		}
		buf = append(buf, b)
		if val == 0 {
			break
		}
	}
	return buf
}

func encodeString(s string) []byte {
	data := []byte(s)
	return append(encodeVarInt(len(data)), data...)
}

func readVarIntFromReader(reader *bufio.Reader) (int, error) {
	var result int
	var numRead int
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		val := int(b & 0x7F)
		result |= val << (7 * numRead)
		numRead++
		if b&0x80 == 0 {
			break
		}
		if numRead > 5 {
			return 0, fmt.Errorf("varint too big")
		}
	}
	return result, nil
}

func resolveSRV(host string) (string, int, error) {
	_, addrs, err := net.LookupSRV("minecraft", "tcp", host)
	if err != nil {
		return host, 0, err
	}
	if len(addrs) == 0 {
		return host, 0, fmt.Errorf("no SRV records found")
	}
	srv := addrs[0]
	target := strings.TrimSuffix(srv.Target, ".")
	return target, int(srv.Port), nil
}

func Status(host string, port int) (*StatusResponse, error) {
	origHost := host
	origPort := port

	srvHost, srvPort, err := resolveSRV(host)
	if err == nil && srvPort != 0 {
		host = srvHost
		port = srvPort
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Build handshake packet
	handshake := encodeVarInt(0x00)
	handshake = append(handshake, encodeVarInt(47)...) // protocol version 1.8 (compatible with all servers)
	handshake = append(handshake, encodeString(origHost)...)
	handshake = append(handshake, byte(origPort>>8), byte(origPort&0xFF)) // port as UInt16 BE
	handshake = append(handshake, encodeVarInt(1)...) // next state: status

	pkt := append(encodeVarInt(len(handshake)), handshake...)
	if _, err := conn.Write(pkt); err != nil {
		return nil, fmt.Errorf("failed to send handshake: %w", err)
	}

	// Send status request
	reqData := encodeVarInt(0x00)
	reqPkt := append(encodeVarInt(len(reqData)), reqData...)
	if _, err := conn.Write(reqPkt); err != nil {
		return nil, fmt.Errorf("failed to send status request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)

	if _, err := readVarIntFromReader(reader); err != nil {
		return nil, fmt.Errorf("failed to read packet length: %w", err)
	}

	if _, err := readVarIntFromReader(reader); err != nil {
		return nil, fmt.Errorf("failed to read packet id: %w", err)
	}

	jsonLen, err := readVarIntFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read json length: %w", err)
	}

	jsonData := make([]byte, jsonLen)
	total := 0
	for total < jsonLen {
		n, err := reader.Read(jsonData[total:])
		if err != nil {
			return nil, fmt.Errorf("failed to read json data: %w", err)
		}
		total += n
	}

	latency := time.Since(start).Milliseconds()

	var status StatusResponse
	if err := json.Unmarshal(jsonData, &status); err != nil {
		return nil, fmt.Errorf("failed to parse json: %w", err)
	}
	status.RoundTripLatency = latency

	if status.MOTD.Clean == "" {
		status.MOTD.Clean = status.Description.Text
		if status.MOTD.Clean == "" && len(status.MOTD.Extra) > 0 {
			var parts []string
			for _, e := range status.MOTD.Extra {
				parts = append(parts, e.Text)
			}
			status.MOTD.Clean = strings.Join(parts, "")
		}
	}

	return &status, nil
}

var _ = binary.BigEndian
