package method

import (
	"net"
	"testing"
	"time"

	"github.com/rebecaachambers/submill/config"
)

func TestUploadToS3(t *testing.T) {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:9000", 2*time.Second)
	if err != nil {
		t.Skip("MinIO not available, skipping integration test")
	}
	conn.Close()

	config.GlobalConfig.S3Endpoint = "127.0.0.1:9000"
	config.GlobalConfig.S3AccessID = "123"
	config.GlobalConfig.S3SecretKey = "123"
	config.GlobalConfig.S3Bucket = "public"
	config.GlobalConfig.S3UseSSL = false
	type args struct {
		data     []byte
		filename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TEST MINIO",
			args: args{
				data:     []byte("test"),
				filename: "test.yaml",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UploadToS3(tt.args.data, tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("UploadToS3() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}