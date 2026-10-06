package executor

import (
	"context"
	"net"
	"os"
	"time"
	"uptime/internal/domain"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type ICMPExecutor struct{}

func NewICMPExecutor() *ICMPExecutor {
	return &ICMPExecutor{}
}

func (e *ICMPExecutor) Execute(ctx context.Context, m *domain.Monitor) (*domain.MonitorResult, error) {
	start := time.Now()
	
	// Try unprivileged UDP first (works in Docker without root)
	network := "udp4"
	listenAddr := "0.0.0.0"
	
	c, err := icmp.ListenPacket(network, listenAddr)
	if err != nil {
		// Fallback to raw socket (requires root)
		network = "ip4:icmp"
		c, err = icmp.ListenPacket(network, listenAddr)
		if err != nil {
			return nil, err
		}
	}
	defer c.Close()

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte("UPTIME-CHECK"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return nil, err
	}

	dst, err := net.ResolveIPAddr("ip4", m.Target)
	if err != nil {
		return &domain.MonitorResult{
			Time: time.Now(), MonitorID: m.ID, Status: domain.StatusDown, ErrorMessage: err.Error(),
		}, nil
	}

	var addr net.Addr
	if network == "udp4" {
		addr = &net.UDPAddr{IP: dst.IP}
	} else {
		addr = dst
	}

	if _, err := c.WriteTo(wb, addr); err != nil {
		return &domain.MonitorResult{
			Time: time.Now(), MonitorID: m.ID, Status: domain.StatusDown, ErrorMessage: err.Error(),
		}, err
	}

	reply := make([]byte, 1500)
	deadline := time.Now().Add(time.Duration(m.TimeoutSeconds) * time.Second)
	if deadline.After(time.Now().Add(2 * time.Second)) {
		deadline = time.Now().Add(2 * time.Second) // Cap at 2s for ICMP
	}
	c.SetReadDeadline(deadline)
	
	n, peer, err := c.ReadFrom(reply)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &domain.MonitorResult{
			Time: time.Now(), MonitorID: m.ID, Status: domain.StatusDown, LatencyMS: int(latency), ErrorMessage: "request timeout",
		}, nil
	}

	// For ip4:icmp, we need to skip the IP header (usually 20 bytes)
	var body []byte
	if network == "ip4:icmp" {
		header, err := ipv4.ParseHeader(reply[:n])
		if err == nil {
			body = reply[header.Len:n]
		} else {
			body = reply[:n]
		}
	} else {
		body = reply[:n]
	}

	rm, err := icmp.ParseMessage(1, body)
	if err != nil {
		return &domain.MonitorResult{
			Time: time.Now(), MonitorID: m.ID, Status: domain.StatusDown, LatencyMS: int(latency), ErrorMessage: "failed to parse reply",
		}, nil
	}

	if rm.Type != ipv4.ICMPTypeEchoReply {
		return &domain.MonitorResult{
			Time: time.Now(), MonitorID: m.ID, Status: domain.StatusDown, LatencyMS: int(latency), ErrorMessage: "not an echo reply",
		}, nil
	}

	_ = peer // We could verify peer IP here

	return &domain.MonitorResult{
		Time: time.Now(), MonitorID: m.ID, Status: domain.StatusUp, LatencyMS: int(latency),
	}, nil
}
