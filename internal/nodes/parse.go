package nodes

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Parsed struct {
	Name     string
	Protocol string
	Outbound map[string]any
}

func Parse(uri string) (Parsed, error) {
	uri = strings.TrimSpace(uri)
	switch {
	case strings.HasPrefix(uri, "xray://"):
		return parseXrayOutbound(uri)
	case strings.HasPrefix(uri, "vmess://"):
		return parseVMess(uri)
	case strings.HasPrefix(uri, "vless://"), strings.HasPrefix(uri, "trojan://"):
		return parseURLNode(uri)
	case strings.HasPrefix(uri, "ss://"):
		return parseShadowsocks(uri)
	case strings.HasPrefix(uri, "http://"), strings.HasPrefix(uri, "https://"):
		return parseHTTP(uri)
	case strings.HasPrefix(uri, "socks://"), strings.HasPrefix(uri, "socks5://"):
		return parseSOCKS(uri)
	default:
		return Parsed{}, fmt.Errorf("不支持的节点协议")
	}
}

// DecodeV2RayNSubscription decodes a v2rayN Base64 or plain URI subscription.
func DecodeV2RayNSubscription(body []byte) []string {
	text := strings.TrimSpace(string(body))
	if decoded, err := decodeBase64(text); err == nil && strings.Contains(string(decoded), "://") {
		text = string(decoded)
	}
	var result []string
	for _, line := range strings.Fields(text) {
		if _, err := Parse(line); err == nil {
			result = append(result, line)
		}
	}
	return result
}

func parseXrayOutbound(uri string) (Parsed, error) {
	rawValue := strings.TrimPrefix(uri, "xray://")
	fragment := ""
	if index := strings.LastIndex(rawValue, "#"); index >= 0 {
		fragment, _ = url.QueryUnescape(rawValue[index+1:])
		rawValue = rawValue[:index]
	}
	raw, err := decodeBase64(rawValue)
	if err != nil {
		return Parsed{}, fmt.Errorf("无效 xray 出站编码: %w", err)
	}
	var outbound map[string]any
	if err := json.Unmarshal(raw, &outbound); err != nil {
		return Parsed{}, fmt.Errorf("无效 xray 出站 JSON: %w", err)
	}
	protocol, ok := outbound["protocol"].(string)
	if !ok || strings.TrimSpace(protocol) == "" {
		return Parsed{}, fmt.Errorf("xray 出站 JSON 缺少 protocol")
	}
	if _, ok := outbound["settings"]; !ok {
		return Parsed{}, fmt.Errorf("xray 出站 JSON 缺少 settings")
	}
	name := fragment
	if name == "" {
		name, _ = outbound["tag"].(string)
	}
	return Parsed{Name: fallbackName(name, protocol), Protocol: protocol, Outbound: outbound}, nil
}

func parseVMess(uri string) (Parsed, error) {
	raw, err := decodeBase64(strings.TrimPrefix(uri, "vmess://"))
	if err != nil {
		return Parsed{}, fmt.Errorf("无效 VMess 编码: %w", err)
	}
	var value struct {
		Name, Address, Port, ID, AlterID, Security string
		Network, HeaderType, Host, Path, TLS, SNI  string
	}
	var source map[string]any
	if err := json.Unmarshal(raw, &source); err != nil {
		return Parsed{}, fmt.Errorf("无效 VMess JSON: %w", err)
	}
	value.Name = stringValue(source["ps"])
	value.Address = stringValue(source["add"])
	value.Port = stringValue(source["port"])
	value.ID = stringValue(source["id"])
	value.AlterID = stringValue(source["aid"])
	value.Security = stringValue(source["scy"])
	value.Network = stringValue(source["net"])
	value.HeaderType = stringValue(source["type"])
	value.Host = stringValue(source["host"])
	value.Path = stringValue(source["path"])
	value.TLS = stringValue(source["tls"])
	value.SNI = stringValue(source["sni"])
	port, err := strconv.Atoi(value.Port)
	if err != nil || value.Address == "" || value.ID == "" {
		return Parsed{}, fmt.Errorf("VMess 地址、端口或用户 ID 无效")
	}
	alterID, _ := strconv.Atoi(value.AlterID)
	security := value.Security
	if security == "" {
		security = "auto"
	}
	outbound := map[string]any{"protocol": "vmess", "settings": map[string]any{"vnext": []any{map[string]any{"address": value.Address, "port": port, "users": []any{map[string]any{"id": value.ID, "alterId": alterID, "security": security}}}}}}
	outbound["streamSettings"] = streamSettings(value.Network, value.TLS, value.SNI, value.Host, value.Path, value.HeaderType, nil)
	return Parsed{Name: fallbackName(value.Name, value.Address), Protocol: "vmess", Outbound: outbound}, nil
}

func parseURLNode(uri string) (Parsed, error) {
	u, err := url.Parse(uri)
	if err != nil || u.User == nil || u.Hostname() == "" {
		return Parsed{}, fmt.Errorf("无效节点链接")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return Parsed{}, fmt.Errorf("无效端口")
	}
	protocol := u.Scheme
	credential := u.User.Username()
	query := u.Query()
	user := map[string]any{"id": credential, "encryption": defaultString(query.Get("encryption"), "none")}
	if flow := query.Get("flow"); flow != "" {
		user["flow"] = flow
	}
	server := map[string]any{"address": u.Hostname(), "port": port, "password": credential}
	if flow := query.Get("flow"); flow != "" {
		server["flow"] = flow
	}
	outbound := map[string]any{"protocol": protocol, "settings": map[string]any{"servers": []any{server}}}
	if protocol == "vless" {
		outbound["settings"] = map[string]any{"vnext": []any{map[string]any{"address": u.Hostname(), "port": port, "users": []any{user}}}}
	}
	outbound["streamSettings"] = streamSettings(query.Get("type"), query.Get("security"), query.Get("sni"), query.Get("host"), query.Get("path"), query.Get("headerType"), query)
	name, _ := url.QueryUnescape(u.Fragment)
	return Parsed{Name: fallbackName(name, u.Hostname()), Protocol: protocol, Outbound: outbound}, nil
}

func parseShadowsocks(uri string) (Parsed, error) {
	raw := strings.TrimPrefix(uri, "ss://")
	name := ""
	if index := strings.Index(raw, "#"); index >= 0 {
		name, _ = url.QueryUnescape(raw[index+1:])
		raw = raw[:index]
	}
	if index := strings.Index(raw, "?"); index >= 0 {
		raw = raw[:index]
	}
	var userInfo, hostPort string
	if index := strings.LastIndex(raw, "@"); index >= 0 {
		userInfo, hostPort = raw[:index], raw[index+1:]
		if decoded, err := decodeBase64(userInfo); err == nil {
			userInfo = string(decoded)
		}
	} else {
		decoded, err := decodeBase64(raw)
		if err != nil {
			return Parsed{}, fmt.Errorf("无效 Shadowsocks 编码")
		}
		combined := string(decoded)
		index := strings.LastIndex(combined, "@")
		if index < 0 {
			return Parsed{}, fmt.Errorf("无效 Shadowsocks 链接")
		}
		userInfo, hostPort = combined[:index], combined[index+1:]
	}
	credentials := strings.SplitN(userInfo, ":", 2)
	hostURL, err := url.Parse("ss://" + hostPort)
	port, portErr := strconv.Atoi(hostURL.Port())
	if err != nil || portErr != nil || len(credentials) != 2 {
		return Parsed{}, fmt.Errorf("无效 Shadowsocks 参数")
	}
	server := map[string]any{"address": hostURL.Hostname(), "port": port, "method": credentials[0], "password": credentials[1]}
	return Parsed{Name: fallbackName(name, hostURL.Hostname()), Protocol: "shadowsocks", Outbound: map[string]any{"protocol": "shadowsocks", "settings": map[string]any{"servers": []any{server}}}}, nil
}

func parseHTTP(uri string) (Parsed, error) {
	u, port, err := parseProxyURL(uri)
	if err != nil {
		return Parsed{}, err
	}
	settings := map[string]any{"address": u.Hostname(), "port": port}
	if u.User != nil {
		settings["user"] = u.User.Username()
		if password, ok := u.User.Password(); ok {
			settings["pass"] = password
		}
	}
	outbound := map[string]any{"protocol": "http", "settings": settings}
	if u.Scheme == "https" {
		outbound["streamSettings"] = map[string]any{"security": "tls", "tlsSettings": map[string]any{"serverName": defaultString(u.Query().Get("sni"), u.Hostname())}}
	}
	name, _ := url.QueryUnescape(u.Fragment)
	return Parsed{Name: fallbackName(name, u.Hostname()), Protocol: "http", Outbound: outbound}, nil
}

func parseSOCKS(uri string) (Parsed, error) {
	u, port, err := parseProxyURL(uri)
	if err != nil {
		return Parsed{}, err
	}
	settings := map[string]any{"address": u.Hostname(), "port": port}
	if u.User != nil {
		settings["user"] = u.User.Username()
		if password, ok := u.User.Password(); ok {
			settings["pass"] = password
		}
	}
	name, _ := url.QueryUnescape(u.Fragment)
	return Parsed{Name: fallbackName(name, u.Hostname()), Protocol: "socks", Outbound: map[string]any{"protocol": "socks", "settings": settings}}, nil
}

func parseProxyURL(uri string) (*url.URL, int, error) {
	u, err := url.Parse(uri)
	if err != nil || u.Hostname() == "" {
		return nil, 0, fmt.Errorf("无效代理链接")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return nil, 0, fmt.Errorf("无效代理端口")
	}
	return u, port, nil
}

func streamSettings(network, security, sni, host, path, headerType string, query url.Values) map[string]any {
	settings := map[string]any{"network": defaultString(network, "tcp"), "security": security}
	if security == "tls" {
		settings["tlsSettings"] = map[string]any{"serverName": sni}
	}
	if security == "reality" && query != nil {
		settings["realitySettings"] = map[string]any{"serverName": sni, "fingerprint": query.Get("fp"), "publicKey": query.Get("pbk"), "shortId": query.Get("sid"), "spiderX": query.Get("spx")}
	}
	switch network {
	case "ws":
		settings["wsSettings"] = map[string]any{"path": path, "headers": map[string]any{"Host": host}}
	case "grpc":
		settings["grpcSettings"] = map[string]any{"serviceName": defaultString(queryValue(query, "serviceName"), path)}
	case "http", "h2":
		settings["httpSettings"] = map[string]any{"path": path, "host": splitNonEmpty(host)}
	case "tcp":
		if headerType == "http" {
			settings["tcpSettings"] = map[string]any{"header": map[string]any{"type": "http", "request": map[string]any{"path": splitNonEmpty(path), "headers": map[string]any{"Host": splitNonEmpty(host)}}}}
		}
	}
	return settings
}

func decodeBase64(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	for _, encoding := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
		if decoded, err := encoding.DecodeString(value); err == nil {
			return decoded, nil
		}
	}
	return nil, fmt.Errorf("base64 解码失败")
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

func fallbackName(name, host string) string {
	if name != "" {
		return name
	}
	return host
}
func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
func queryValue(values url.Values, key string) string {
	if values == nil {
		return ""
	}
	return values.Get(key)
}
func splitNonEmpty(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}
