package xray

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func TestOutbound(ctx context.Context, binary string, outbound map[string]any, testURL string, timeout time.Duration) (time.Duration, error) {
	port, err := freePort()
	if err != nil {
		return 0, err
	}
	config := map[string]any{
		"log":       map[string]any{"loglevel": "warning"},
		"inbounds":  []any{map[string]any{"listen": "127.0.0.1", "port": port, "protocol": "socks", "settings": map[string]any{"udp": false}}},
		"outbounds": []any{outbound},
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return 0, err
	}
	file, err := os.CreateTemp("", "x-cmd-probe-*.json")
	if err != nil {
		return 0, err
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := file.Write(raw); err != nil {
		file.Close()
		return 0, err
	}
	file.Close()
	processCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	command := exec.CommandContext(processCtx, binary, "run", "-c", path)
	if err := command.Start(); err != nil {
		return 0, fmt.Errorf("启动 xray 失败: %w", err)
	}
	defer func() {
		cancel()
		_ = command.Wait()
	}()
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	if err := waitForPort(ctx, address, minDuration(timeout, 3*time.Second)); err != nil {
		return 0, fmt.Errorf("xray 未能启动: %w", err)
	}
	probeCtx, probeCancel := context.WithTimeout(ctx, timeout)
	defer probeCancel()
	transport := &http.Transport{DialContext: func(ctx context.Context, network, target string) (net.Conn, error) {
		return dialSOCKS5(ctx, address, target)
	}}
	client := &http.Client{Transport: transport}
	request, err := http.NewRequestWithContext(probeCtx, http.MethodGet, testURL, nil)
	if err != nil {
		return 0, err
	}
	started := time.Now()
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1024))
	if response.StatusCode >= http.StatusBadRequest {
		return 0, fmt.Errorf("探测地址返回 HTTP %s", response.Status)
	}
	return time.Since(started), nil
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func waitForPort(ctx context.Context, address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		connection, err := (&net.Dialer{Timeout: 100 * time.Millisecond}).DialContext(ctx, "tcp", address)
		if err == nil {
			connection.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return fmt.Errorf("等待端口 %s 超时", address)
}

func dialSOCKS5(ctx context.Context, proxyAddress, target string) (net.Conn, error) {
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", proxyAddress)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (net.Conn, error) { connection.Close(); return nil, err }
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	}
	if _, err := connection.Write([]byte{5, 1, 0}); err != nil {
		return fail(err)
	}
	response := make([]byte, 2)
	if _, err := io.ReadFull(connection, response); err != nil || response[1] != 0 {
		return fail(fmt.Errorf("SOCKS5 协商失败"))
	}
	host, portText, err := net.SplitHostPort(target)
	if err != nil {
		return fail(err)
	}
	port, _ := strconv.Atoi(portText)
	request := []byte{5, 1, 0}
	if ip := net.ParseIP(host); ip != nil && ip.To4() != nil {
		request = append(request, 1)
		request = append(request, ip.To4()...)
	} else if ip != nil {
		request = append(request, 4)
		request = append(request, ip.To16()...)
	} else {
		if len(host) > 255 {
			return fail(fmt.Errorf("目标主机名过长"))
		}
		request = append(request, 3, byte(len(host)))
		request = append(request, host...)
	}
	request = binary.BigEndian.AppendUint16(request, uint16(port))
	if _, err := connection.Write(request); err != nil {
		return fail(err)
	}
	reader := bufio.NewReader(connection)
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil || header[1] != 0 {
		return fail(fmt.Errorf("SOCKS5 连接失败"))
	}
	length := 0
	switch header[3] {
	case 1:
		length = 4
	case 4:
		length = 16
	case 3:
		value, err := reader.ReadByte()
		if err != nil {
			return fail(err)
		}
		length = int(value)
	default:
		return fail(fmt.Errorf("SOCKS5 返回未知地址类型"))
	}
	if _, err := io.CopyN(io.Discard, reader, int64(length+2)); err != nil {
		return fail(err)
	}
	_ = connection.SetDeadline(time.Time{})
	return connection, nil
}

func minDuration(left, right time.Duration) time.Duration {
	if left < right {
		return left
	}
	return right
}
