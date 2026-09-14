// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package logstore

import (
	"os/exec"
	"strings"
	"testing"
)

// serverPkg 是下游 server 插件的包路径前缀。
const serverPkg = "Wavelet/openflare/plugins/server"

// forbiddenImports 业务域禁止直接触碰的底层日志实现。
var forbiddenImports = []string{
	serverPkg + "/kernel/repository/analytics",
}

// allowedAnalyticsDelegation 允许直接依赖 analytics 仓储的委托层：
//   - repository：持久化门面，ListOpenFlareLatestMetricSnapshotsSince 的
//     CH 快速路径仍直连 analytics（LIMIT 1 BY node_id）；小时级聚合读已改走 logstore；
//   - repository/logstore：CH 后端实现按设计委托 analytics。
//
// 除此之外，依赖闭包内任何包都禁止引入 analytics 仓储。
var allowedAnalyticsDelegation = map[string]bool{
	serverPkg + "/kernel/repository":          true,
	serverPkg + "/kernel/repository/logstore": true,
}

// allowedInfraPersistence 允许业务域包引入的 infra/persistence 子包。
var allowedInfraPersistence = []string{
	serverPkg + "/infra/persistence/batchwriter", // batchwriter 统计类型
	serverPkg + "/infra/persistence/idgen",       // 雪花 ID 生成（无日志依赖）
}

// domainScopes 是 server 插件内的业务域包（等价于改造前的 internal/apps/...）。
// 持久化与基础设施层（repository/infra/model/…）不受本门禁约束。
var domainScopes = []string{
	"domain/site", "domain/fleet", "domain/pages", "domain/waf", "domain/tls",
	"domain/cloudflare", "domain/observability", "domain/dashboard", "domain/option",
	"updater",
}

func TestDomainsMustNotImportLogBackendDirectly(t *testing.T) {
	t.Chdir(moduleRoot(t))

	wanted := make([]string, 0, len(domainScopes))
	patterns := make([]string, 0, len(domainScopes))
	for _, d := range domainScopes {
		wanted = append(wanted, serverPkg+"/"+d)
		patterns = append(patterns, "./openflare/plugins/server/"+d+"/...")
	}

	args := append([]string{"list", "-test", "-f", `{{.ImportPath}} {{join .Imports " "}}`}, patterns...)
	//nolint:gosec // 固定参数，无外部输入
	out, err := exec.Command("go", args...).Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}

	scanned := 0
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkg := fields[0]
		if !hasAnyPrefix(pkg, wanted) {
			continue
		}
		scanned++
		for _, imp := range fields[1:] {
			for _, forbidden := range forbiddenImports {
				if imp == forbidden && !allowedAnalyticsDelegation[pkg] {
					t.Errorf("%s must not import forbidden log backend %s", pkg, forbidden)
				}
			}
			if strings.HasPrefix(imp, serverPkg+"/infra/persistence/") {
				allowed := false
				for _, a := range allowedInfraPersistence {
					if imp == a || strings.HasPrefix(imp, a+"/") {
						allowed = true
						break
					}
				}
				if !allowed {
					t.Errorf("%s must not import infra/persistence subpackage directly: %s", pkg, imp)
				}
			}
		}
	}
	// 扫描到 0 个包说明包路径已漂移，门禁会静默失效——必须报错而非给绿灯。
	if scanned == 0 {
		t.Fatalf("no domain package scanned; domainScopes is stale: %v", wanted)
	}
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if s == p || strings.HasPrefix(s, p+"/") {
			return true
		}
	}
	return false
}

// moduleRoot 向 go 查询模块根目录，避免依赖测试文件所在深度的相对路径。
func moduleRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	return strings.TrimSpace(string(out))
}
