package utils

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rebecaachambers/submill/config"
)

// ExecuteCallback 鎵ц鍥炶皟鑴氭湰
func ExecuteCallback(successCount int) {
	callbackScript := config.GlobalConfig.CallbackScript
	if callbackScript == "" {
		return
	}

	slog.Info(fmt.Sprintf("鎵ц鍥炶皟鑴氭湰: %s", callbackScript))

	// 妫€鏌ヨ剼鏈枃浠舵槸鍚﹀瓨锟?
	if _, err := os.Stat(callbackScript); os.IsNotExist(err) {
		slog.Error(fmt.Sprintf("鍥炶皟鑴氭湰涓嶅瓨锟? %s", callbackScript))
		return
	}

	// 鍦ㄩ潪Windows绯荤粺涓婃鏌ュ苟璁剧疆鎵ц鏉冮檺
	if runtime.GOOS != "windows" {
		err := os.Chmod(callbackScript, 0755) // rwxr-xr-x 鏉冮檺
		if err != nil {
			slog.Warn(fmt.Sprintf("璁剧疆鑴氭湰鎵ц鏉冮檺澶辫触: %v", err))
		}

		// 妫€鏌ヨ剼鏈槸鍚︽湁shebang
		content, err := os.ReadFile(callbackScript)
		if err == nil && len(content) > 0 {
			hasShebang := false
			if len(content) >= 2 && content[0] == '#' && content[1] == '!' {
				hasShebang = true
			}

			if !hasShebang {
				slog.Warn("script missing shebang, please add #!/bin/bash or similar at beginning")
			}
		}
	}

	// 鏍规嵁鎿嶄綔绯荤粺绫诲瀷閫夋嫨涓嶅悓鐨勬墽琛屾柟锟?
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Windows 绯荤粺
		if strings.HasSuffix(strings.ToLower(callbackScript), ".bat") ||
			strings.HasSuffix(strings.ToLower(callbackScript), ".cmd") {
			// 浣跨敤瀹屾暣璺緞锛屽苟姝ｇ‘澶勭悊甯︾┖鏍肩殑璺緞
			absPath, err := filepath.Abs(callbackScript)
			if err != nil {
				slog.Error(fmt.Sprintf("鑾峰彇鑴氭湰缁濆璺緞澶辫触: %v", err))
				return
			}
			cmd = exec.Command("cmd", "/C", absPath)
		} else if strings.HasSuffix(strings.ToLower(callbackScript), ".ps1") {
			// PowerShell 鑴氭湰
			absPath, err := filepath.Abs(callbackScript)
			if err != nil {
				slog.Error(fmt.Sprintf("鑾峰彇鑴氭湰缁濆璺緞澶辫触: %v", err))
				return
			}
			// 浣跨敤 -ExecutionPolicy Bypass 缁曡繃鎵ц绛栫暐闄愬埗
			cmd = exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", absPath)
		} else {
			cmd = exec.Command(callbackScript)
		}
		// 璁剧疆宸ヤ綔鐩綍涓鸿剼鏈墍鍦ㄧ洰锟?
		cmd.Dir = filepath.Dir(callbackScript)
		cmd.SysProcAttr = hideWindowSysProcAttr()
	} else {
		// Unix/Linux/MacOS 绯荤粺
		cmd = exec.Command(callbackScript)
	}

	// 璁剧疆鐜鍙橀噺锛屼紶閫掓垚鍔熻妭鐐规暟锟?
	cmd.Env = append(os.Environ(), fmt.Sprintf("SUCCESS_COUNT=%d", successCount))

	// 鎵ц鍛戒护
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error(fmt.Sprintf("鎵ц鍥炶皟鑴氭湰澶辫触: %v, 杈撳嚭: %s", err, string(output)))
		return
	}
	slog.Info("鍥炶皟鑴氭湰鎵ц鎴愬姛")
}

