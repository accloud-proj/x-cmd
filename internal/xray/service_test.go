package xray

import "testing"

func TestRuntimeConfigUsesMixedInbound(t *testing.T) {
	config := RuntimeConfig(map[string]any{"protocol": "freedom"}, 1091, false)
	inbound := config["inbounds"].([]any)[0].(map[string]any)
	if inbound["protocol"] != "mixed" || inbound["port"] != 1091 || inbound["listen"] != "127.0.0.1" {
		t.Fatalf("unexpected inbound: %#v", inbound)
	}
}

func TestRuntimeConfigAllowsLANConnections(t *testing.T) {
	config := RuntimeConfig(map[string]any{"protocol": "freedom"}, 1091, true)
	inbound := config["inbounds"].([]any)[0].(map[string]any)
	if inbound["listen"] != lanListenAddress() {
		t.Fatalf("listen = %q, want %q", inbound["listen"], lanListenAddress())
	}
}

func TestLANListenAddressUsesIPv6WhenDualStackIsAvailable(t *testing.T) {
	if got := chooseLANListenAddress(func() bool { return true }); got != "::" {
		t.Fatalf("dual-stack listen address = %q, want ::", got)
	}
	if got := chooseLANListenAddress(func() bool { return false }); got != "0.0.0.0" {
		t.Fatalf("IPv4 fallback address = %q, want 0.0.0.0", got)
	}
}

func TestFirstAvailablePortSkipsUnavailablePorts(t *testing.T) {
	var checked []int
	port, err := firstAvailablePort(1091, func(port int) bool {
		checked = append(checked, port)
		return port == 1093
	})
	if err != nil {
		t.Fatal(err)
	}
	if port != 1093 {
		t.Fatalf("port = %d, want 1093", port)
	}
	if len(checked) != 3 {
		t.Fatalf("checked ports = %v", checked)
	}
}
