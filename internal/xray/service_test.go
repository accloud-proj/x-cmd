package xray

import "testing"

func TestRuntimeConfigUsesMixedInbound(t *testing.T) {
	config := RuntimeConfig(map[string]any{"protocol": "freedom"}, 1091)
	inbound := config["inbounds"].([]any)[0].(map[string]any)
	if inbound["protocol"] != "mixed" || inbound["port"] != 1091 {
		t.Fatalf("unexpected inbound: %#v", inbound)
	}
}
