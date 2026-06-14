package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// WriteMihomoConfig generates or repairs config/config.yaml.
// On first run it creates the config; on subsequent runs it repairs known issues
// (e.g. wrong proxy-provider path from older versions) without destroying user edits.
func WriteMihomoConfig() error {
	dir := getExecutableDir()
	path := filepath.Join(dir, "config", "config.yaml")

	if _, err := os.Stat(path); err == nil {
		return repairMihomoConfig(path)
	}
	return writeDefaultMihomoConfig(path)
}

func repairMihomoConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read mihomo config for repair: %w", err)
	}

	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse mihomo config for repair: %w", err)
	}

	changed := false

	// Fix proxy-provider path: old versions used ./mihomo/nodes/* which is wrong
	if pp, ok := cfg["proxy-providers"].(map[string]any); ok {
		for _, v := range pp {
			if p, ok := v.(map[string]any); ok {
				if ppath, ok := p["path"].(string); ok {
					if strings.Contains(ppath, "mihomo/nodes") {
						p["path"] = "./nodes/all.yaml"
						changed = true
					}
				}
			}
		}
	}

	// Fix rules: ensure MATCH rule exists and is set to PROXY (not DIRECT unless user changed)
	if rules, ok := cfg["rules"].([]any); ok {
		foundMatch := false
		for i, r := range rules {
			if s, ok := r.(string); ok && strings.HasPrefix(s, "MATCH,") {
				foundMatch = true
				if strings.HasSuffix(s, ",DIRECT") {
					rules[i] = "MATCH,PROXY"
					changed = true
				}
				break
			}
		}
		if !foundMatch {
			rules = append(rules, "MATCH,PROXY")
			cfg["rules"] = rules
			changed = true
		}
	}

	if !changed {
		return nil
	}

	newData, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal repaired mihomo config: %w", err)
	}
	if err := os.WriteFile(path, newData, 0644); err != nil {
		return fmt.Errorf("write repaired mihomo config: %w", err)
	}
	return nil
}

func writeDefaultMihomoConfig(path string) error {
	cfg := map[string]any{
		"mixed-port": 20171,
		"bind-address": "*",
		"allow-lan": true,
		"mode":       "rule",
		"log-level":  "info",
		"ipv6":          false,
		"geo-auto-update": false,
		"geo-update-interval": 99999,
		"profile": map[string]any{
			"store-selected": true,
			"store-fake-ip":  true,
		},
		"dns": map[string]any{
			"enable":        true,
			"ipv6":          false,
			"enhanced-mode": "fake-ip",
			"fake-ip-range": "198.18.0.1/16",
			"nameserver":    []string{"223.5.5.5", "119.29.29.29"},
		},
		"proxy-providers": map[string]any{
			"submill": map[string]any{
				"type": "file",
				"path": "./nodes/all.yaml",
				"health-check": map[string]any{
					"enable":   true,
					"url":      "http://www.gstatic.com/generate_204",
					"interval": 300,
				},
			},
		},
		"proxy-groups": []map[string]any{
			{
				"name":    "PROXY",
				"type":    "select",
				"proxies": []string{"auto", "balance", "DIRECT"},
			},
			{
				"name":      "auto",
				"type":      "url-test",
				"use":       []string{"submill"},
				"url":       "http://www.gstatic.com/generate_204",
				"interval":  300,
				"tolerance": 20,
			},
			{
				"name":     "balance",
				"type":     "load-balance",
				"use":      []string{"submill"},
				"url":      "http://www.gstatic.com/generate_204",
				"interval": 300,
				"strategy": "consistent-hashing",
			},
			{
				"name":    "FALLBACK",
				"type":    "fallback",
				"proxies": []string{"auto", "balance", "DIRECT"},
			},
		},
                "rules": []string{
                        // LAN/private IPs
                        "IP-CIDR,192.168.0.0/16,DIRECT,no-resolve",
                        "IP-CIDR,10.0.0.0/8,DIRECT,no-resolve",
                        "IP-CIDR,172.16.0.0/12,DIRECT,no-resolve",
                        "IP-CIDR,127.0.0.0/8,DIRECT,no-resolve",
                        "IP-CIDR,100.64.0.0/10,DIRECT,no-resolve",
                        // Chinese mainland domains -> DIRECT
                        "DOMAIN-SUFFIX,cn,DIRECT",
                        "DOMAIN-SUFFIX,baidu.com,DIRECT",
                        "DOMAIN-SUFFIX,bdstatic.com,DIRECT",
                        "DOMAIN-SUFFIX,bdimg.com,DIRECT",
                        "DOMAIN-SUFFIX,taobao.com,DIRECT",
                        "DOMAIN-SUFFIX,tmall.com,DIRECT",
                        "DOMAIN-SUFFIX,alicdn.com,DIRECT",
                        "DOMAIN-SUFFIX,alipay.com,DIRECT",
                        "DOMAIN-SUFFIX,aliyun.com,DIRECT",
                        "DOMAIN-SUFFIX,jd.com,DIRECT",
                        "DOMAIN-SUFFIX,360buyimg.com,DIRECT",
                        "DOMAIN-SUFFIX,qq.com,DIRECT",
                        "DOMAIN-SUFFIX,tencent.com,DIRECT",
                        "DOMAIN-SUFFIX,weixin.com,DIRECT",
                        "DOMAIN-SUFFIX,wechat.com,DIRECT",
                        "DOMAIN-SUFFIX,gtimg.com,DIRECT",
                        "DOMAIN-SUFFIX,qpic.cn,DIRECT",
                        "DOMAIN-SUFFIX,sina.com.cn,DIRECT",
                        "DOMAIN-SUFFIX,sina.cn,DIRECT",
                        "DOMAIN-SUFFIX,weibo.com,DIRECT",
                        "DOMAIN-SUFFIX,bilibili.com,DIRECT",
                        "DOMAIN-SUFFIX,biliapi.com,DIRECT",
                        "DOMAIN-SUFFIX,bilivideo.com,DIRECT",
                        "DOMAIN-SUFFIX,hdslb.com,DIRECT",
                        "DOMAIN-SUFFIX,zhihu.com,DIRECT",
                        "DOMAIN-SUFFIX,zhimg.com,DIRECT",
                        "DOMAIN-SUFFIX,csdn.net,DIRECT",
                        "DOMAIN-SUFFIX,163.com,DIRECT",
                        "DOMAIN-SUFFIX,126.com,DIRECT",
                        "DOMAIN-SUFFIX,126.net,DIRECT",
                        "DOMAIN-SUFFIX,netease.com,DIRECT",
                        "DOMAIN-SUFFIX,sohu.com,DIRECT",
                        "DOMAIN-SUFFIX,sogou.com,DIRECT",
                        "DOMAIN-SUFFIX,mi.com,DIRECT",
                        "DOMAIN-SUFFIX,xiaomi.com,DIRECT",
                        "DOMAIN-SUFFIX,meituan.com,DIRECT",
                        "DOMAIN-SUFFIX,dianping.com,DIRECT",
                        "DOMAIN-SUFFIX,pinduoduo.com,DIRECT",
                        "DOMAIN-SUFFIX,bytedance.com,DIRECT",
                        "DOMAIN-SUFFIX,toutiao.com,DIRECT",
                        "DOMAIN-SUFFIX,douyin.com,DIRECT",
                        "DOMAIN-SUFFIX,feishu.cn,DIRECT",
                        "DOMAIN-SUFFIX,dingtalk.com,DIRECT",
                        "DOMAIN-SUFFIX,huawei.com,DIRECT",
                        "DOMAIN-SUFFIX,huaweicloud.com,DIRECT",
                        "DOMAIN-SUFFIX,ctrip.com,DIRECT",
                        "DOMAIN-SUFFIX,qunar.com,DIRECT",
                        "DOMAIN-SUFFIX,iqiyi.com,DIRECT",
                        "DOMAIN-SUFFIX,youku.com,DIRECT",
                        "DOMAIN-SUFFIX,mgtv.com,DIRECT",
                        "DOMAIN-SUFFIX,ifeng.com,DIRECT",
                        "DOMAIN-SUFFIX,360.cn,DIRECT",
                        // Apple
                        "DOMAIN-SUFFIX,apple.com,DIRECT",
                        "DOMAIN-SUFFIX,icloud.com,DIRECT",
                        "DOMAIN-SUFFIX,mzstatic.com,DIRECT",
                        // Microsoft
                        "DOMAIN-SUFFIX,microsoft.com,DIRECT",
                        "DOMAIN-SUFFIX,live.com,DIRECT",
                        "DOMAIN-SUFFIX,office.com,DIRECT",
                        "DOMAIN-SUFFIX,office.net,DIRECT",
                        // Common CDN / services
                        "DOMAIN-SUFFIX,gstatic.com,DIRECT",
                        "DOMAIN-SUFFIX,github.com,DIRECT",
                        "DOMAIN-SUFFIX,github.io,DIRECT",
                        "DOMAIN-SUFFIX,githubassets.com,DIRECT",
                        "DOMAIN-SUFFIX,githubusercontent.com,DIRECT",
                        "DOMAIN-SUFFIX,gravatar.com,DIRECT",
                        "DOMAIN-KEYWORD,alicdn,DIRECT",
                        "DOMAIN-KEYWORD,alipay,DIRECT",
                        "DOMAIN-KEYWORD,taobao,DIRECT",
                        "DOMAIN-KEYWORD,baidu,DIRECT",
                        // Default: proxy everything else (blocked sites)
                        "MATCH,PROXY",
                },
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal mihomo config: %w", err)
	}

	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write mihomo config: %w", err)
	}
	return nil
}

// getExecutableDir returns the directory of the running executable.
func getExecutableDir() string {
	ex, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(ex)
}



