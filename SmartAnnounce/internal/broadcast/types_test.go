package broadcast

import "testing"

func TestGenerateRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     GenerateRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: GenerateRequest{
				Text:    "今日特价鸡蛋",
				VoiceID: "female-shaonv",
				Speed:   1,
				Volume:  1,
			},
		},
		{
			name: "empty text",
			req: GenerateRequest{
				Text:    "   ",
				VoiceID: "female-shaonv",
				Speed:   1,
				Volume:  1,
			},
			wantErr: true,
		},
		{
			name: "empty voice id",
			req: GenerateRequest{
				Text:    "今日特价鸡蛋",
				VoiceID: " ",
				Speed:   1,
				Volume:  1,
			},
			wantErr: true,
		},
		{
			name: "invalid speed",
			req: GenerateRequest{
				Text:    "今日特价鸡蛋",
				VoiceID: "female-shaonv",
				Speed:   0.1,
				Volume:  1,
			},
			wantErr: true,
		},
		{
			name: "invalid volume",
			req: GenerateRequest{
				Text:    "今日特价鸡蛋",
				VoiceID: "female-shaonv",
				Speed:   1,
				Volume:  1.2,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExportRequest(t *testing.T) {
	t.Run("validate source path", func(t *testing.T) {
		req := ExportRequest{}
		if err := req.Validate(); err == nil {
			t.Fatal("Validate() expected error for empty source path")
		}
	})

	t.Run("default file name", func(t *testing.T) {
		req := ExportRequest{FileName: "今日特价播报"}
		if got := req.DefaultFileName(); got != "今日特价播报.mp3" {
			t.Fatalf("DefaultFileName() = %q, want %q", got, "今日特价播报.mp3")
		}
	})
}
