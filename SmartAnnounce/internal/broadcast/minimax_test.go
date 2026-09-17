package broadcast

import (
	"encoding/json"
	"testing"
)

func TestMinimaxBuildPayloadUsesRequestVolume(t *testing.T) {
	client := &MinimaxClient{model: defaultMinimaxModel}

	payload, err := client.buildPayload(GenerateRequest{
		Text:    "今日特价鸡蛋",
		VoiceID: "female-shaonv",
		Speed:   1.15,
		Volume:  0.65,
	})
	if err != nil {
		t.Fatalf("buildPayload() error = %v", err)
	}

	var request minimaxRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if request.VoiceSetting.Vol != 0.65 {
		t.Fatalf("voice_setting.vol = %v, want %v", request.VoiceSetting.Vol, 0.65)
	}
}
