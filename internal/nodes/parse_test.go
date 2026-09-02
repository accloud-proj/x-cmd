package nodes

import (
	"encoding/base64"
	"testing"
)

func TestParseSupportedLinks(t *testing.T) {
	vmess := base64.RawStdEncoding.EncodeToString([]byte(`{"ps":"vm","add":"example.com","port":"443","id":"00000000-0000-0000-0000-000000000001","net":"ws","path":"/x","tls":"tls"}`))
	tests := []string{
		"vmess://" + vmess,
		"vless://00000000-0000-0000-0000-000000000001@example.com:443?security=tls&type=ws#vl",
		"trojan://secret@example.com:443?security=tls#tr",
		"ss://" + base64.RawURLEncoding.EncodeToString([]byte("aes-256-gcm:secret")) + "@example.com:8388#ss",
	}
	for _, link := range tests {
		parsed, err := Parse(link)
		if err != nil {
			t.Errorf("Parse(%q): %v", link, err)
			continue
		}
		if parsed.Name == "" || parsed.Outbound["protocol"] == nil {
			t.Errorf("incomplete result for %q: %#v", link, parsed)
		}
	}
}

func TestDecodeV2RayNSubscription(t *testing.T) {
	body := base64.StdEncoding.EncodeToString([]byte("trojan://secret@example.com:443#one\ninvalid://value"))
	links := DecodeV2RayNSubscription([]byte(body))
	if len(links) != 1 {
		t.Fatalf("got %d links", len(links))
	}
}

func TestV2RayNSubscriptionRejectsClashYAML(t *testing.T) {
	links := DecodeV2RayNSubscription([]byte("proxies:\n  - name: example\n    type: vmess\n    server: example.com"))
	if len(links) != 0 {
		t.Fatalf("Clash YAML must not be parsed: %#v", links)
	}
}

func TestTrojanPasswordIsOnServer(t *testing.T) {
	parsed, err := Parse("trojan://secret@example.com:443?security=tls#tr")
	if err != nil {
		t.Fatal(err)
	}
	settings := parsed.Outbound["settings"].(map[string]any)
	server := settings["servers"].([]any)[0].(map[string]any)
	if server["password"] != "secret" {
		t.Fatalf("unexpected Trojan server: %#v", server)
	}
	if _, exists := server["users"]; exists {
		t.Fatalf("Trojan server must not contain users: %#v", server)
	}
}

func TestParseHTTPAndSOCKSLinks(t *testing.T) {
	tests := []struct {
		link     string
		protocol string
	}{
		{"http://user:pass@example.com:8080#http-node", "http"},
		{"https://user:pass@example.com:8443?sni=proxy.example.com#https-node", "http"},
		{"socks5://user:pass@example.com:1080#socks-node", "socks"},
	}
	for _, test := range tests {
		parsed, err := Parse(test.link)
		if err != nil {
			t.Errorf("Parse(%q): %v", test.link, err)
			continue
		}
		if parsed.Protocol != test.protocol {
			t.Errorf("Parse(%q) protocol = %q", test.link, parsed.Protocol)
		}
	}
}

func TestParseNativeXrayOutbound(t *testing.T) {
	settings := `{"protocol":"wireguard","settings":{"secretKey":"private","address":["10.0.0.2/32"],"peers":[{"endpoint":"example.com:51820","publicKey":"public"}]}}`
	link := "xray://" + base64.RawURLEncoding.EncodeToString([]byte(settings)) + "#wireguard-node"
	parsed, err := Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Protocol != "wireguard" || parsed.Name != "wireguard-node" {
		t.Fatalf("unexpected native outbound: %#v", parsed)
	}
}

func TestNativeXrayOutboundRequiresSettings(t *testing.T) {
	link := "xray://" + base64.RawURLEncoding.EncodeToString([]byte(`{"protocol":"hysteria"}`))
	if _, err := Parse(link); err == nil {
		t.Fatal("expected missing settings error")
	}
}
