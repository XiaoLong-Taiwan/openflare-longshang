# OpenFlare Cordis 对齐 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 OpenFlare 控制面与 Wavelet Cordis 同构装配，删除全部平台副本，用 git merge 接入 Wavelet 上游，并保证金标准 v3.5.4 库可升级且测试全绿。

**Architecture:** 先在 Wavelet 补齐通用扩展点 W1–W9；再把 `core/pkg/plugins` 同步进 OpenFlare；将 `cmd` 改为 `newWaveletApp` + `server.New()` + `WithMigrationBaseline`；`server` 只留 `of_*` 业务；历史库经 Baseline stamp，不再执行 76 条混合链。对齐完成后再做一次 unrelated merge。

**Tech Stack:** Go 1.25、Gin、GORM、goose v3、Cobra、PostgreSQL/SQLite、可选 ClickHouse。

**规格来源:** `docs/superpowers/specs/2026-08-30-openflare-cordis-alignment-design.md`

## Global Constraints

- 禁止修改 `/Users/ryan/Code/Go/OpenFlare`（金标准，`9f79fb99` / v3.5.4）。
- 禁止修改 `frontend/` 源码。
- 禁止在 OpenFlare `server` 注册 `/api/health`、`/api/v1/user/self`、`/api/cap/*`。
- 禁止把 OpenFlare / `of_` / 边缘节点写进 Wavelet。
- 禁止 rebase / filter-repo / force-push 改写 OpenFlare 历史。
- 禁止把 pow 抽到 `pkg/pow`；禁止第二套关系库 migrator。
- Wavelet 工作在 `/Users/ryan/Code/Go/Wavelet`；OpenFlare 工作在 `/Users/ryan/Code/Go/OpenFlare-cordis`。
- 执行期隔离工作区按 `using-git-worktrees` 在动手时创建，本计划不预先建。

---

## 文件结构

| 仓库 | 路径 | 职责 |
| :--- | :--- | :--- |
| Wavelet | `backend/core/extpoints/router.go`、`scoped_extpoints.go` | W1 HandleRaw / BasePath |
| Wavelet | `backend/core/contracts/captcha.go` | `CaptchaService` |
| Wavelet | `backend/core/contracts/config_public.go` | `PublicConfigProvider` |
| Wavelet | `backend/core/contracts/push.go` | `PushRegistry` + `PushEventMeta` |
| Wavelet | `backend/core/app.go`、`context.go` | `WithMigrationBaseline` |
| Wavelet | `backend/plugins/domain/cap/plugin.go` | Provide CaptchaService；挂 `/api/cap/*` |
| Wavelet | `backend/plugins/domain/user/plugin.go` | 消费 CaptchaService；`GET /self` |
| Wavelet | `backend/plugins/domain/system/plugin.go` | `GET /api/health`；PublicConfigProvider |
| Wavelet | `backend/plugins/domain/upload/plugin.go` | 补挂 `/my` 等用户路由 |
| Wavelet | `backend/plugins/domain/message_gateway/` | 实现 PushRegistry |
| Wavelet | `backend/plugins/drivers/driver_http/` | `RedirectTrailingSlash` |
| Wavelet | `backend/cmd/app.go` | gooseEngine：Baseline + advisory lock |
| OpenFlare | `backend/cmd/{app,api,all,worker,scheduler,root}.go` | 与 Wavelet 同构 + `server.New()` |
| OpenFlare | `backend/OpenFlare/plugins/server/plugin.go` | 只注册 `of_*` 业务 |
| OpenFlare | `backend/OpenFlare/plugins/server/stamp/stamp.go` | Baseline stamp 实现 |
| OpenFlare | `backend/OpenFlare/plugins/server/migrations/{postgres,sqlite}/00001_initial.sql` | 仅 `of_*` |
| OpenFlare | `.gitattributes` | `merge=ours` 规则 |

---

### Task 1: 金标准 L1（只读，失败即停）

**Files:** 无（只读 `/Users/ryan/Code/Go/OpenFlare`）

**Interfaces:**
- Consumes: 金标准树 `9f79fb99`
- Produces: 确认 `go test ./...` 在金标准上为绿，作为后续门禁前提

- [ ] **Step 1: 确认 HEAD**

```bash
git -C /Users/ryan/Code/Go/OpenFlare rev-parse HEAD
git -C /Users/ryan/Code/Go/OpenFlare status --short
```

Expected: `9f79fb99...`，工作区空。若不是，停下来问人，不要改金标准。

- [ ] **Step 2: 跑金标准测试**

```bash
cd /Users/ryan/Code/Go/OpenFlare && go test ./...
```

Expected: exit 0。失败则本计划停止。

- [ ] **Step 3: 不提交**

本任务无代码变更。

---

### Task 2: Wavelet W1 HandleRaw

**Files:**
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/core/extpoints/router.go`（接口 + HandleRaw）
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/core/scoped_extpoints.go`
- Create: `/Users/ryan/Code/Go/Wavelet/backend/core/extpoints/router_raw_test.go`
- Test: 同上

**Interfaces:**
- Consumes: 无
- Produces: `RouterExtension.HandleRaw(method, path string, handlers ...any) RouteDefinition`；`BasePath() string`

参考实现已在 OpenFlare-cordis（不要发明第二套语义）：
`/Users/ryan/Code/Go/OpenFlare-cordis/backend/core/extpoints/router.go` 的 `addRoute` / `HandleRaw` / `joinPathPreservingTrailing` / `ensureLeadingSlash`，以及 `scoped_extpoints.go` 的 `HandleRaw`。

- [ ] **Step 1: 在 Wavelet 写入失败测试**

把下面文件写到 `/Users/ryan/Code/Go/Wavelet/backend/core/extpoints/router_raw_test.go`（与 OpenFlare 仓同文）：

```go
package extpoints

import "testing"

func TestHandleRawPreservesTrailingSlash(t *testing.T) {
	r := &RouterRegistry{}
	g := r.Group("/api/v1/nodes")

	if got := g.BasePath(); got != "/api/v1/nodes" {
		t.Fatalf("BasePath() = %q, want %q", got, "/api/v1/nodes")
	}
	slashless := g.Handle("GET", "")
	slashed := g.HandleRaw("GET", "/")

	if slashless.Path != "/api/v1/nodes" {
		t.Errorf("Handle(\"\") path = %q, want %q", slashless.Path, "/api/v1/nodes")
	}
	if slashed.Path != "/api/v1/nodes/" {
		t.Errorf("HandleRaw(\"/\") path = %q, want %q", slashed.Path, "/api/v1/nodes/")
	}
	if slashed.ID == slashless.ID {
		t.Error("HandleRaw must allocate its own route ID")
	}
	if got := len(r.Routes()); got != 2 {
		t.Errorf("registry routes = %d, want 2", got)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./core/extpoints/ -run TestHandleRaw -count=1
```

Expected: FAIL（`HandleRaw` undefined 或 `BasePath` undefined）。

- [ ] **Step 3: 实现**

在 `RouterExtension` 增加 `HandleRaw` 与 `BasePath`。`Handle` 继续 `cleanPath`。`HandleRaw` 走 `ensureLeadingSlash` / `joinPathPreservingTrailing`，**不**剥尾斜杠。`RouterRegistry` 抽出 `addRoute`。`scopedRouterExtension.HandleRaw` 与 `Handle` 一样登记 `OnDispose` → `UnregisterByID`。从 OpenFlare-cordis 对应文件拷语义，不要改行为。

- [ ] **Step 4: 测试通过**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./core/extpoints/ ./core/ -count=1
```

Expected: PASS。

- [ ] **Step 5: 提交（Wavelet 仓）**

```bash
cd /Users/ryan/Code/Go/Wavelet
git add backend/core/extpoints/router.go backend/core/extpoints/router_raw_test.go backend/core/scoped_extpoints.go
git commit -m "feat(core): add HandleRaw and BasePath for trailing-slash routes"
```

---

### Task 3: Wavelet W2 CaptchaService + `/api/cap/*` + user 门禁

**Files:**
- Create: `/Users/ryan/Code/Go/Wavelet/backend/core/contracts/captcha.go`
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/cap/plugin.go`
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/user/plugin.go`
- Create: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/cap/plugin_captcha_contract_test.go`
- Create: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/user/plugin_captcha_test.go`

**Interfaces:**
- Consumes: 现有 `cap.VerifyMiddleware`、`Challenge`、`Redeem`
- Produces:

```go
package contracts

type CaptchaService interface {
    VerifyMiddleware(scope string) any
    ChallengeHandler() any
    RedeemHandler() any
}
```

- [ ] **Step 1: 写契约测试（cap 插件 Apply 后能 Inject CaptchaService）**

```go
package cap

import (
	"context"
	"testing"

	"Wavelet/core"
	"Wavelet/core/contracts"
)

func TestApplyProvidesCaptchaService(t *testing.T) {
	ctx := core.NewContext(context.Background())
	if err := New().Apply(ctx); err != nil {
		t.Fatal(err)
	}
	svc, err := core.Inject[contracts.CaptchaService](ctx)
	if err != nil || svc == nil {
		t.Fatalf("Inject CaptchaService: svc=%v err=%v", svc, err)
	}
	if svc.ChallengeHandler() == nil || svc.RedeemHandler() == nil {
		t.Fatal("handlers must be non-nil")
	}
	if svc.VerifyMiddleware("login") == nil {
		t.Fatal("VerifyMiddleware(login) must be non-nil")
	}
}

func TestApplyRegistersUnversionedCapRoutes(t *testing.T) {
	ctx := core.NewContext(context.Background())
	if err := New().Apply(ctx); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"POST /api/v1/cap/challenge": false,
		"POST /api/cap/challenge":    false,
		"POST /api/cap/redeem":       false,
	}
	for _, rd := range ctx.Router().Routes() {
		key := rd.Method + " " + rd.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for key, ok := range want {
		if !ok {
			t.Errorf("missing route %s", key)
		}
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/cap/ -run TestApplyProvidesCaptchaService -count=1
```

Expected: FAIL（CaptchaService 未定义或未 Provide）。

- [ ] **Step 3: 实现契约与 cap 插件**

`captcha.go` 只含接口，无 OpenFlare 字样。

cap 内未导出适配：

```go
type captchaService struct{}

func (captchaService) VerifyMiddleware(scope string) any {
	return VerifyMiddleware(GetDefaultManager(), scope)
}
func (captchaService) ChallengeHandler() any { return Challenge }
func (captchaService) RedeemHandler() any    { return Redeem }
```

`Apply` 中：`core.Provide[contracts.CaptchaService](ctx, captchaService{})`。在现有 `/api/v1/cap` 组之外再：

```go
legacy := ctx.Router().Group("/api/cap")
legacy.POST("/challenge", Challenge)
legacy.POST("/redeem", Redeem)
ctx.Router().RegisterWhitelist("/api/cap/challenge", "/api/cap/redeem")
```

原 `/api/v1/cap` 组保留。

- [ ] **Step 4: user 插件消费 CaptchaService**

在 `user/plugin.go` 注册 login/register/send-email-code 时：

```go
passThrough := gin.HandlerFunc(func(c *gin.Context) { c.Next() })
loginCap, registerCap, emailCap := passThrough, passThrough, passThrough
if capSvc, err := core.Inject[contracts.CaptchaService](ctx); err == nil && capSvc != nil {
	if mw, ok := capSvc.VerifyMiddleware("login").(gin.HandlerFunc); ok {
		loginCap = mw
	}
	if mw, ok := capSvc.VerifyMiddleware("register").(gin.HandlerFunc); ok {
		registerCap = mw
	}
	if mw, ok := capSvc.VerifyMiddleware("send_email_code").(gin.HandlerFunc); ok {
		emailCap = mw
	}
}
userGroup.POST("/login", loginCap, Login)
userGroup.POST("/register", registerCap, Register)
userGroup.POST("/send-email-code", emailCap, SendEmailCode)
```

无 CaptchaService 时行为与现在相同（裸路由）。user 的 `Inject()` 不要把 CaptchaService 变成硬依赖。

user 测试：Apply 且未 Provide captcha 时三条路由仍在；Provide 假 CaptchaService 后 login 的 handler 链长度大于 1。

- [ ] **Step 5: 跑测试**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/cap/ ./plugins/domain/user/ ./core/contracts/ -count=1
```

Expected: PASS。

- [ ] **Step 6: 提交**

```bash
cd /Users/ryan/Code/Go/Wavelet
git add backend/core/contracts/captcha.go backend/plugins/domain/cap backend/plugins/domain/user
git commit -m "feat(cap): expose CaptchaService and unversioned /api/cap routes"
```

---

### Task 4: Wavelet W3 PublicConfigProvider

**Files:**
- Create: `/Users/ryan/Code/Go/Wavelet/backend/core/contracts/config_public.go`
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/system/plugin.go`
- Create: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/system/public_config_test.go`

**Interfaces:**
- Produces:

```go
package contracts

import "context"

type PublicConfigProvider interface {
    PublicConfig(ctx context.Context) (any, error)
}
```

- [ ] **Step 1: 失败测试**

```go
package system

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"Wavelet/core"
	"Wavelet/core/contracts"

	"github.com/gin-gonic/gin"
)

type stubPublic struct{ payload any }

func (s stubPublic) PublicConfig(context.Context) (any, error) { return s.payload, nil }

func TestPublicConfigUsesProviderWhenPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := core.NewContext(context.Background())
	core.Provide[contracts.PublicConfigProvider](ctx, stubPublic{payload: map[string]string{"k": "v"}})
	if err := New().Apply(ctx); err != nil {
		t.Fatal(err)
	}
	var handler gin.HandlerFunc
	for _, rd := range ctx.Router().Routes() {
		if rd.Method == "GET" && rd.Path == "/api/v1/config/public" {
			handler = rd.Handlers[0].(gin.HandlerFunc)
		}
	}
	if handler == nil {
		t.Fatal("public config route missing")
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/config/public", nil)
	handler(c)
	var body struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if string(body.Data) != `{"k":"v"}` && string(body.Data) != `{"k": "v"}` {
		t.Fatalf("data = %s, want flat map", body.Data)
	}
}
```

再加一个无 provider 的测试：响应 JSON 含 `"configs"` 与 `"app"`（保持 Wavelet 默认）。

- [ ] **Step 2: 跑测试确认失败**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/system/ -run TestPublicConfig -count=1
```

Expected: FAIL。

- [ ] **Step 3: 实现**

`GET /api/v1/config/public` handler 内：

```go
if p, err := core.Inject[contracts.PublicConfigProvider](ctx); err == nil && p != nil {
    data, err := p.PublicConfig(c.Request.Context())
    if err != nil {
        logger.ErrorF(c.Request.Context(), "[System] public config provider failed: %v", err)
        response.AbortInternal(c, "public config unavailable")
        return
    }
    c.JSON(http.StatusOK, response.OK(data))
    return
}
// 原 {configs, app} 默认分支
```

`Inject` 在请求时解析，这样 server 后挂 Provide 仍生效（若 Apply 顺序导致 Provide 晚于 system.Apply，改用 `core.When` 存到局部变量，或在 handler 里每次 `Inject`）。**必须在 handler 里 Inject**，不要在 `Apply` 时缓存成 nil。

- [ ] **Step 4: 测试通过并提交**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/system/ -count=1
git -C /Users/ryan/Code/Go/Wavelet add backend/core/contracts/config_public.go backend/plugins/domain/system
git -C /Users/ryan/Code/Go/Wavelet commit -m "feat(system): allow PublicConfigProvider to replace public config payload"
```

---

### Task 5: Wavelet W4 RedirectTrailingSlash

**Files:**
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/drivers/driver_http/config.go`
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/drivers/driver_http/engine.go`
- Create: `/Users/ryan/Code/Go/Wavelet/backend/plugins/drivers/driver_http/engine_slash_test.go`

**Interfaces:**
- Produces: `httpAppConfig.RedirectTrailingSlash bool`，tag `config:"redirect_trailing_slash" env:"APP_REDIRECT_TRAILING_SLASH" default:"true"`

- [ ] **Step 1: 失败测试**

```go
package driver_http

import "testing"

func TestBuildEngineDefaultRedirectsTrailingSlash(t *testing.T) {
	eng, err := BuildEngineWithConfig(httpAppConfig{}, httpRedisConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if !eng.RedirectTrailingSlash {
		t.Fatal("default RedirectTrailingSlash must be true")
	}
}

func TestBuildEngineCanDisableRedirectTrailingSlash(t *testing.T) {
	eng, err := BuildEngineWithConfig(httpAppConfig{RedirectTrailingSlash: false}, httpRedisConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if eng.RedirectTrailingSlash {
		t.Fatal("RedirectTrailingSlash must honor false")
	}
}
```

注意：`httpAppConfig` 里 bool 零值是 false，与 default true 冲突。测试里「默认」必须走 `BuildEngineWithConfig` 在 **未显式设 false 且 Bind 之后** 的行为。实现上用 `*bool` 或单独 `redirectTrailingSlashSet`。推荐：

```go
RedirectTrailingSlash *bool `config:"redirect_trailing_slash" env:"APP_REDIRECT_TRAILING_SLASH"`
```

`BuildEngineWithConfig`：`redirect := true`；若 `cfg.RedirectTrailingSlash != nil { redirect = *cfg.RedirectTrailingSlash }`。测试「默认」传 `httpAppConfig{}`（nil → true）；「关闭」传 `ptr(false)`。

- [ ] **Step 2: 跑测试确认失败**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/drivers/driver_http/ -run TestBuildEngine -count=1
```

Expected: FAIL。

- [ ] **Step 3: 在 `BuildEngineWithConfig` 里 `r.RedirectTrailingSlash = redirect`，默认 true。**

- [ ] **Step 4: 测试通过并提交**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/drivers/driver_http/ -count=1
git -C /Users/ryan/Code/Go/Wavelet add backend/plugins/drivers/driver_http
git -C /Users/ryan/Code/Go/Wavelet commit -m "feat(http): make trailing-slash redirect configurable"
```

---

### Task 6: Wavelet W5 upload 用户路由补挂

**Files:**
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/upload/plugin.go`（约 123–129 行）
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/upload/plugin_test.go`（若无路由测试则创建）

**Interfaces:**
- Consumes: 已有 `handler.ListMyFiles`、`handler.UpdateMyFile`、`handler.DownloadFile`、`handler.BatchDownloadFiles`
- Produces: 用户组额外路径 `GET /api/v1/upload/my`、`PUT /api/v1/upload/:id`、`GET /api/v1/upload/download/:id`、`POST /api/v1/upload/download/batch`

- [ ] **Step 1: 写失败测试**（Apply 后检查 Routes 包含上述四条，且仍含 `GET /api/v1/upload` 与 `POST /api/v1/upload/batch-download`）

Apply 需要 AuthService / DB 等。若现有 `plugin_test.go` 已有假依赖，跟它走。没有则用 `core.Provide` 三个 stub：`contracts.DBService`、`contracts.StorageService`、`contracts.AuthService`（`RequireAuthMiddleware` 返回 `gin.HandlerFunc` pass-through）。

- [ ] **Step 2: 跑测试确认缺路由**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/upload/ -run TestUserUploadRoutes -count=1
```

Expected: FAIL missing `/my`。

- [ ] **Step 3: 在 `uploadGroup` 增加**

```go
uploadGroup.GET("/my", handler.ListMyFiles)
uploadGroup.PUT("/:id", handler.UpdateMyFile)
uploadGroup.GET("/download/:id", handler.DownloadFile)
uploadGroup.POST("/download/batch", handler.BatchDownloadFiles)
```

原 `GET ""`、`POST /batch-download` 不动。

- [ ] **Step 4: 测试通过并提交**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/upload/ -count=1
git -C /Users/ryan/Code/Go/Wavelet add backend/plugins/domain/upload
git -C /Users/ryan/Code/Go/Wavelet commit -m "feat(upload): mount existing my/update/download routes on user API"
```

---

### Task 7: Wavelet W6 PushRegistry

**Files:**
- Create: `/Users/ryan/Code/Go/Wavelet/backend/core/contracts/push.go`
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/message_gateway/plugin.go`
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/message_gateway/service/push.go`（适配器，不改注册语义）
- Create: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/message_gateway/plugin_registry_test.go`

**Interfaces:**
- Produces:

```go
package contracts

import "context"

type PushNotificationTemplate struct {
    Title   string
    Content string
    Level   string
    Ext     map[string]any
}

type PushEventMeta struct {
    Key             string
    Name            string
    Description     string
    DefaultTemplate PushNotificationTemplate
}

type PushRegistry interface {
    RegisterBuiltInEvent(meta PushEventMeta)
    SyncEvents(ctx context.Context) error
}
```

- [ ] **Step 1: 失败测试** — Apply 后 `Inject[PushRegistry]` 成功；`RegisterBuiltInEvent` 一个 key 后再 `GetBuiltInEvents`（或 registry 再读）能看到该 key。

- [ ] **Step 2: 跑测试确认失败**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/message_gateway/ -run TestPushRegistry -count=1
```

Expected: FAIL。

- [ ] **Step 3: 实现** — message_gateway 内部把 `PushEventMeta` 转成现有 `model.EventMetadata` 再调 `RegisterBuiltInEvent` / `SyncEvents`。`Apply` 里 `core.Provide[contracts.PushRegistry](ctx, ...)`。禁止在契约里出现 `openflare`。

- [ ] **Step 4: 测试通过并提交**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/message_gateway/ -count=1
git -C /Users/ryan/Code/Go/Wavelet add backend/core/contracts/push.go backend/plugins/domain/message_gateway
git -C /Users/ryan/Code/Go/Wavelet commit -m "feat(message_gateway): expose PushRegistry contract"
```

---

### Task 8: Wavelet W7 + W8 health 与 `/user/self`

**Files:**
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/system/plugin.go`
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/user/plugin.go`
- Create: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/system/health_route_test.go`
- Create: `/Users/ryan/Code/Go/Wavelet/backend/plugins/domain/user/self_route_test.go`

**Interfaces:**
- Produces: `GET /api/health` → `response.OKNil()`；`GET /api/v1/user/self` 使用 `loginMW` + 返回当前用户（与 profile 同源，handler 调 `AuthService.GetCurrentUser` 或现有 UserInfo 等价实现）

- [ ] **Step 1: 失败测试**

system：Apply 后 Routes 含 `GET /api/health`，用 httptest 调 handler，body 的 `error_msg` 为空且 `data` 为 `null`。`GET /healthz` 仍在。

user：Apply 后 Routes 含 `GET /api/v1/user/self`，且该路由的 middleware/handler 数量 ≥ login 保护的 `/profile`（必须带鉴权）。

- [ ] **Step 2: 跑测试确认失败**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/system/ ./plugins/domain/user/ -run 'TestHealth|TestSelf' -count=1
```

Expected: FAIL missing routes。

- [ ] **Step 3: 实现**

```go
ctx.Router().GET("/api/health", func(c *gin.Context) {
    c.JSON(http.StatusOK, response.OKNil())
})
ctx.Router().RegisterWhitelist("/api/health")
```

`/healthz` 保持 `{status: ok}`，不要改。

user：

```go
userGroup.GET("/self", loginMW, Self)
```

`Self` 用 `contracts.AuthService.GetCurrentUser`（已有 SetAuthService）写成与 `UpdateProfile` 同文件的小 handler，JSON `response.OK(user)`。

- [ ] **Step 4: 测试通过并提交**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./plugins/domain/system/ ./plugins/domain/user/ -count=1
git -C /Users/ryan/Code/Go/Wavelet add backend/plugins/domain/system backend/plugins/domain/user
git -C /Users/ryan/Code/Go/Wavelet commit -m "feat(platform): add GET /api/health and GET /api/v1/user/self"
```

---

### Task 9: Wavelet W9 MigrationBaseline + 锁

**Files:**
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/core/app.go`
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/core/context.go`
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/core/app_test.go`
- Modify: `/Users/ryan/Code/Go/Wavelet/backend/cmd/app.go`（`gooseEngine.Migrate`）

**Interfaces:**
- Produces:

```go
func WithMigrationBaseline(fn func(*Context) error) AppOption
func (c *Context) MigrationBaseline() func(*Context) error
```

语义：`Prepare` 把 App 上的 fn 拷到 root Context；`gooseEngine.Migrate` 在第一次成功 `CreateVersionTable` 之后、任何 `provider.Up` 之前调用 `ctx.MigrationBaseline()`（nil 则跳过）。引擎与选项名禁止产品名。

- [ ] **Step 1: 失败测试（core）**

```go
func TestWithMigrationBaselineRunsBeforeEngineMigrate(t *testing.T) {
    var order []string
    engine := MigrationRunner(func(ctx *Context, _ []MigrationEntry) error {
        order = append(order, "engine")
        if ctx.MigrationBaseline() == nil {
            t.Fatal("baseline must be visible on context inside Migrate")
        }
        return ctx.MigrationBaseline()(ctx)
    })
    app := NewApp(
        WithMigrationEngine(engine),
        WithMigrationBaseline(func(*Context) error {
            order = append(order, "baseline")
            return nil
        }),
    )
    // 注册一个空迁移条目的测试插件，使 RunMigrations 不会因 entries==0 直接返回
    // ...
}
```

若现有测试插件难写：最小插件 `Name() string` + `Apply` 里 `ctx.Migrations().Register("t", someFS)`。也可用不走完整 Prepare、只测 `WithMigrationBaseline` 把 fn 放到 `app.ctx`：调用内部 `Prepare` 后 `app.Context().MigrationBaseline() != nil`。

另测：fn 返回 error 时 `RunMigrations` 返回该 error，且自定义 engine 的 `Migrate` **不被调用**——注意 spec 是表创建之后、Up 之前。core 层只负责把 fn 放到 Context；**调用点在 gooseEngine**。core 测试断言 Prepare 后 Context 上能取到 fn。cmd 测试断言顺序 `create-table → baseline → up`。

- [ ] **Step 2: 跑测试确认失败**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./core/ -run TestWithMigrationBaseline -count=1
```

Expected: FAIL。

- [ ] **Step 3: 实现**

`App` 加字段 `migrationBaseline func(*Context) error`。`WithMigrationBaseline` 写入。`Prepare` 成功装好 ctx 后：`a.ctx.setMigrationBaseline(a.migrationBaseline)`（未导出 setter）。

`gooseEngine.Migrate`：解析 DB 后，用与 `CreateVersionTable` 相同的 DDL **先建一次** `w_schema_versions`（已有 IF NOT EXISTS），然后：

```go
if fn := ctx.MigrationBaseline(); fn != nil {
    if err := fn(ctx); err != nil {
        return fmt.Errorf("migration baseline: %w", err)
    }
}
```

再进入现有 per-plugin `Up` 循环。

锁：Postgres 在 baseline 与 Up 外包 `pg_advisory_lock(0x77617665)`（任意稳定 int64，注释写明用途），`defer pg_advisory_unlock`。SQLite 不额外加锁（单写者）。

- [ ] **Step 4: 测试通过并提交**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./core/ ./cmd/ -count=1
git -C /Users/ryan/Code/Go/Wavelet add backend/core backend/cmd/app.go
git -C /Users/ryan/Code/Go/Wavelet commit -m "feat(core): add WithMigrationBaseline hook before goose Up"
```

- [ ] **Step 5: Wavelet 全量测试**

```bash
cd /Users/ryan/Code/Go/Wavelet/backend && go test ./...
```

Expected: PASS。失败先修，再进入 OpenFlare 任务。

---

### Task 10: 同步框架进 OpenFlare-cordis

**Files:**
- Modify: `backend/{core,pkg,plugins}`（rsync）
- Modify: `backend/OpenFlare/upstream-patches.md`（清空条目）

**Interfaces:**
- Consumes: Task 2–9 已进 Wavelet 工作树
- Produces: OpenFlare `backend/{core,pkg,plugins}` 与 Wavelet 对应目录零差（排除 uploads/dist/data/*.db）

- [ ] **Step 1: 同步**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis
scripts/sync-upstream.sh /Users/ryan/Code/Go/Wavelet/backend
scripts/sync-upstream.sh --check /Users/ryan/Code/Go/Wavelet/backend
```

Expected: `--check` 无 `>f`/`<f` 内容差异。

- [ ] **Step 2: 清空补丁清单**

`backend/OpenFlare/upstream-patches.md` 改为只保留标题与一句「当前无本地内核补丁」。删除过时的 HandleRaw 说明。

- [ ] **Step 3: 编译**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis/backend && go test ./core/... ./pkg/... ./plugins/...
```

Expected: PASS。OpenFlare 业务测试此步允许仍绿或因接口变化失败；失败留到 Task 12 修，但 **上游包必须绿**。

- [ ] **Step 4: 提交**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis
git add backend/core backend/pkg backend/plugins backend/OpenFlare/upstream-patches.md
git commit -m "chore(cordis): sync Wavelet core/pkg/plugins after W1-W9"
```

---

### Task 11: 重导金标准夹具

**Files:**
- Modify: `docs/superpowers/specs/baseline/{schema-A.sql,versions-A.txt,routes.txt,routes-engine.txt}`（仅当与金标准现跑不一致）

**Interfaces:**
- Consumes: `/Users/ryan/Code/Go/OpenFlare` 二进制
- Produces: 与金标准现跑一致的夹具

- [ ] **Step 1: 用金标准导出 schema 与版本**

在 **临时目录** 跑金标准（不要写金标准树）：

```bash
GOLD=/Users/ryan/Code/Go/OpenFlare
TMP=$(mktemp -d)
(cd "$GOLD" && go build -o "$TMP/of-baseline" .)
OF_DB_PATH="$TMP/a.db" timeout 25 "$TMP/of-baseline" api >"$TMP/a.log" 2>&1 || true
sqlite3 "$TMP/a.db" "SELECT name,sql FROM sqlite_master WHERE type='table' ORDER BY name;" > /tmp/schema-A.sql
sqlite3 "$TMP/a.db" "SELECT version_id FROM goose_db_version ORDER BY version_id;" > /tmp/versions-A.txt || \
sqlite3 "$TMP/a.db" "SELECT version FROM goose_db_version ORDER BY version;" > /tmp/versions-A.txt
```

与 `docs/superpowers/specs/baseline/` 对比。不一致则以金标准为准更新夹具。版本最大应为 `202608090003`。

- [ ] **Step 2: 提交夹具（仅有 diff 时）**

```bash
git add docs/superpowers/specs/baseline
git commit -m "docs(cordis): refresh baseline fixtures from OpenFlare v3.5.4"
```

无 diff 则不提交。

---

### Task 12: OpenFlare cmd 同构 + server 只挂业务

**Files:**
- Create: `backend/cmd/app.go`（从 Wavelet `backend/cmd/app.go` 拷贝后改）
- Modify: `backend/cmd/{api.go,all.go,worker.go,scheduler.go,root.go,banner.go}`
- Delete: `backend/cmd/bootstrap.go`、`backend/cmd/http_server.go`
- Modify: `backend/OpenFlare/plugins/server/plugin.go`
- Modify: `backend/OpenFlare/plugins/server/openflare/apiutil/middleware.go`
- Modify: `backend/OpenFlare/plugins/server/router/v1/v1.go`、`openflare/*.go`（不再调 RegisterUserRoutes/RegisterAdminRoutes）
- Modify: `config.example.yaml`（`app.redirect_trailing_slash: false` 或与 W4 字段对齐）

**Interfaces:**
- Consumes: Wavelet `newWaveletApp` 插件列表；`core.WithMigrationBaseline`（stamp 函数可先用 `func(*core.Context) error { return nil }` 占位，Task 14 替换成真 stamp）
- Produces: `func newOpenFlareApp(profile core.Profile, opts ...core.AppOption) *core.App`

- [ ] **Step 1: 写失败测试 `backend/cmd/app_test.go`**

```go
func TestNewOpenFlareAppRegistersServerAndWaveletUser(t *testing.T) {
    app := newOpenFlareApp(core.ProfileAPI, core.WithConfigSource(testSource(t)))
    if err := app.Prepare(); err != nil {
        t.Fatal(err)
    }
    names := map[string]bool{}
    for _, p := range app.Plugins() { // 若 Plugins() 未导出，改为检查 ctx.Router 路径
        names[p.Name()] = true
    }
    for _, n := range []string{"user", "auth", "cap", "admin", "server"} {
        if !names[n] {
            t.Errorf("missing plugin %s", n)
        }
    }
}
```

若 `App` 没有导出插件列表，改为 `Prepare` 后从 `app.Context().Router().Routes()` 断言同时存在 `GET /api/v1/user/self`（Wavelet user）与一条 OpenFlare 业务路径（例如 `GET /api/v1/d/nodes` 或现网 zone 路径——以 `docs/superpowers/specs/baseline/routes-engine.txt` 里 `of_` 控制台路径为准）。**不得**再由 server 注册 `POST /api/cap/challenge` 作为「server 自己的」——这条应来自 cap 插件。

另测：`GET /api/health` 存在。

- [ ] **Step 2: 跑测试确认失败**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis/backend && go test ./cmd/ -run TestNewOpenFlareApp -count=1
```

Expected: FAIL（没有 `newOpenFlareApp`）。

- [ ] **Step 3: 实现装配根**

1. 复制 Wavelet `cmd/app.go` 为 OpenFlare `cmd/app.go`（保留 `gooseEngine`）。
2. `newOpenFlareApp` = `newWaveletApp` 的 Use 列表，在 domain 之后、`driver_http` 之前插入 `ofserver.New()`。
3. `core.NewApp` 增加 `core.WithMigrationBaseline(stamp.Legacy)`——此时 stamp 可返回 nil。
4. `driver_http.New()` **不要** `WithEngine`。
5. `api.go`/`all.go`/`worker.go`/`scheduler.go` 改为 `runProfileApp(...)`，与 Wavelet 逐字同一调用。
6. `root.go` 改为 Wavelet 的 hostConfig + idgen.Init；去掉 `PreRun` migrator。
7. 删除 `bootstrap.go`、`http_server.go`。

`server.Apply`：

```go
func (p *Plugin) Apply(ctx *core.Context) error {
    var auth contracts.AuthService
    if err := core.Using[contracts.AuthService](ctx, func(s contracts.AuthService) { auth = s }); err != nil {
        return err
    }
    ofrouter.RegisterV1Routes(ctx.Router().Group("/api/v1"), auth)
    ofrouter.RegisterRoutes(ctx.Router().Group("/api/v1"), auth)
    return nil
}
```

删除 `RegisterUserRoutes` / `RegisterAdminRoutes` / `RegisterRootRoutes` 对平台路径的调用。`apiutil.AdminMiddlewares` 改为接收 `contracts.AuthService`：

```go
func AdminMiddlewares(auth contracts.AuthService) []any {
    return []any{auth.RequireAuthMiddleware(), auth.RequireAdminMiddleware()}
}
```

updater 路由改挂在 OpenFlare 自己的 admin 组上（`/api/v1/admin/update*`），中间件用 `AdminMiddlewares(auth)`，不要 import 已删的 admin handler 包。

`config.example.yaml` 增加 `app.redirect_trailing_slash: false`（若 W4 把 key 放在 app 下；若放在独立 http 前缀则以实际 tag 为准）。DeclareConfig 在 driver_http，OpenFlare 只要 yaml 有该键。

- [ ] **Step 4: 测试通过**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis/backend && go test ./cmd/ -count=1
```

Expected: PASS。此时 `go test ./OpenFlare/...` 会因删除平台路由引用而红，下一任务修。

- [ ] **Step 5: 提交**

```bash
git add backend/cmd backend/OpenFlare/plugins/server/plugin.go backend/OpenFlare/plugins/server/openflare/apiutil backend/OpenFlare/plugins/server/router config.example.yaml
git commit -m "feat(cmd): assemble control plane via Wavelet plugins plus server"
```

---

### Task 13: 删除平台副本并改 import

**Files:**
- Delete: `backend/OpenFlare/plugins/server/{oauth,cap,user,upload,config,health,listener}`
- Delete: `backend/OpenFlare/plugins/server/pkg/{cap,push}`
- Delete: `backend/OpenFlare/plugins/server/admin/` 下除 `updater/` 外全部
- Delete: `backend/OpenFlare/plugins/server/infra/`（config/persistence/task/objectstore/diskcache；CH 小函数先挪走再删）
- Delete: `backend/OpenFlare/plugins/server/model` 与 `repository` 中 `w_*` 实体（users、uploads、auth_source、system_configs、push_*、task_execution、templates、schedule、access_token、analytics/user_access_log）
- Modify: 所有仍 import 上述包的 `openflare/**`、`admin/updater`、测试
- Modify: `platform/bootstrap` — 删除或缩成空；任务注册改到 `server.Apply` 的 `ctx.Task()` / `ctx.Schedule()`
- Create: `backend/OpenFlare/plugins/server/publicconfig/provider.go`

**Interfaces:**
- Consumes: `contracts.AuthService`、`CaptchaService`（不要在 server 再挂 cap）、`PushRegistry`、`PublicConfigProvider`、`contracts.DBService`、`StorageService`、`TaskService`
- Produces: `publicconfig.Provider` 实现 `PublicConfigProvider`，`PublicConfig` 返回 `map[string]string`（visibility=1 的 key/value，与金标准 `GetPublicConfig` 相同）

- [ ] **Step 1: 写失败测试 `plugin_parity_test.go` 改为子集断言**

当前测试要求精确相等且只 `server.New().Apply`。改成：

```go
func TestPluginRoutesContainGoldenBaseline(t *testing.T) {
    app := newOpenFlareAppForTest(t) // 与 cmd 测试同一装配，可用 sqlite + WithConfigValues
    if err := app.Prepare(); err != nil {
        t.Fatal(err)
    }
    got := routeSet(app.Context())
    want := loadBaseline(t)
    for k := range want {
        if !got[k] {
            t.Errorf("missing golden route %s", k)
        }
    }
}
```

`newOpenFlareAppForTest` 必须在 `cmd` 或 `server` 测试里能引用装配函数。若 `cmd` 测试不能 import 业务，把装配函数放 `cmd`，parity 测试移到 `backend/cmd/parity_test.go`。

先提交测试改动并确认：**在平台副本仍在、Apply 已不挂平台路由时**，缺 `POST /api/v1/user/login` 等会失败——等 Wavelet 插件挂上后应变绿。

- [ ] **Step 2: 跑 parity 看缺哪些金标准路径**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis/backend && go test ./cmd/ -run TestPluginRoutesContainGoldenBaseline -count=1
```

记下 missing 列表。缺 login/cap/health/self/upload/my 说明 Wavelet 插件没挂上；缺 `/api/v1/d/...` 说明 OpenFlare 业务没挂上。

- [ ] **Step 3: 按 missing 修 Apply / 契约中间件，然后删副本**

替换规则（全文 grep）：

| 旧 | 新 |
| :--- | :--- |
| `oauth.LoginRequired()` | `auth.RequireAuthMiddleware().(gin.HandlerFunc)` |
| `admin.LoginAdminRequired()` | `auth.RequireAdminMiddleware().(gin.HandlerFunc)` |
| `db.DB(ctx)` | `contracts.DBService.GORM().WithContext(ctx)`（经 Inject/字段） |
| `config.Config.*` | `ctx.Config().Bind` / `String`/`Bool` |
| `push.RegisterBuiltInEvent` | `PushRegistry.RegisterBuiltInEvent` |
| `repository` 的 user/upload/w_* | 禁止；改走契约或删调用 |

`server.Apply` 增加：

```go
core.Provide[contracts.PublicConfigProvider](ctx, publicconfig.New(ctx))
if pr, err := core.Inject[contracts.PushRegistry](ctx); err == nil {
    for _, meta := range ofevents.All() {
        pr.RegisterBuiltInEvent(meta)
    }
    _ = pr.SyncEvents(ctx.GoContext())
}
ctx.Migrations().Register("server", ofMigrations)
```

openflare 异步任务：现 `infra/task/handlers` 里属于 OF 的 handler 迁到 `openflare/tasks`，在 `Apply` 里 `ctx.Task().Register`。平台任务（user:send_email_code 等）不要再注册。

删目录后 `go test ./OpenFlare/...` 以编译器为准修到绿。

- [ ] **Step 4: 测试**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis/backend && go test ./cmd/ ./OpenFlare/...
```

Expected: PASS。`rg 'OpenFlare/plugins/server/(oauth|cap|user|upload|config|health)\\b' backend/OpenFlare` 无匹配。

- [ ] **Step 5: 提交**

```bash
git add -A backend/OpenFlare backend/cmd
git commit -m "refactor(server): drop Wavelet-duplicated domains and use contracts"
```

---

### Task 14: stamp + `of_*` 压缩迁移 + 删除关系库 migrator

**Files:**
- Create: `backend/OpenFlare/plugins/server/stamp/stamp.go`
- Create: `backend/OpenFlare/plugins/server/stamp/stamp_test.go`
- Create: `backend/OpenFlare/plugins/server/migrations/postgres/00001_initial.sql`
- Create: `backend/OpenFlare/plugins/server/migrations/sqlite/00001_initial.sql`
- Modify: `backend/cmd/app.go`（`WithMigrationBaseline(stamp.Legacy)`）
- Create: `backend/OpenFlare/plugins/server/chmigrate/chmigrate.go`（从旧 `migrator.MigrateClickHouse` 抽出，只处理 `of_node_*` 文件）
- Delete: `backend/OpenFlare/plugins/server/infra/persistence/migrator/`（关系库部分；CH sql 挪到 `chmigrate/goose/clickhouse`，去掉 `create_user_access_logs`）

**Interfaces:**
- Consumes: `core.Context` + `contracts.DBService`
- Produces: `func Legacy(ctx *core.Context) error`

`Legacy` 行为：

1. Inject `DBService`；拿 `sql.DB`。
2. 探测 `goose_db_version` 是否存在。不存在则 return nil（新装）。
3. `SELECT COALESCE(MAX(version_id),0)` 或 goose 默认列名 `version`（金标准用 goose 默认表：列是 `version_id` 还是 `version` 以 Task 11 导出为准，代码按实表扫描 `information_schema`/`pragma`）。
4. 插入 `w_schema_versions (plugin_id, version_id)`：`('openflare/legacy', 0)`、`('openflare/legacy', maxVer)`、`('server', 1)`，全部 `ON CONFLICT DO NOTHING`。
5. 若 `maxVer >= 202607120002 && maxVer < 202607130001`，调用 `zone.ImportLegacyTx`。金标准 `202608090003` 跳过。
6. 幂等：再调一次零 INSERT。

- [ ] **Step 1: 写失败测试（用 `t.TempDir()` sqlite）**

建一张假 `goose_db_version`（版本 `202608090003`）和空的 `w_schema_versions`（用与 Wavelet 相同的 DDL）。调用 `Legacy`。断言三行 stamp 存在；再调一次行数不变。另测：无 `goose_db_version` 时不写 `openflare/legacy`。

- [ ] **Step 2: 跑测试确认失败**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis/backend && go test ./OpenFlare/plugins/server/stamp/ -count=1
```

Expected: FAIL。

- [ ] **Step 3: 实现 stamp；cmd 传入 `WithMigrationBaseline(stamp.Legacy)`。**

生成 `of_*` initial：从 Task 11 的 `$TMP/a.db`（或金标准跑出来的库）抽取：

```bash
sqlite3 "$TMP/a.db" "SELECT sql FROM sqlite_master WHERE type='table' AND name LIKE 'of_%' ORDER BY name;"
```

写成 sqlite `00001_initial.sql`（每条 `CREATE TABLE IF NOT EXISTS`，加 `-- +goose Up`）。postgres 目录用同一批表，类型按现有 OF postgres 历史链的最终形态（对照金标准若只有 sqlite 夹具，postgres 从 `migrator/goose/postgres` 里 of_* 的最终 CREATE/ALTER 手工收敛成一张 IF NOT EXISTS，**禁止 DROP**）。双方言版本号都是 `00001`。

CH：复制 `goose/clickhouse` 到 `chmigrate`，删除用户访问日志那个文件。`server.Apply` 在 API profile 下调用 `chmigrate.Up`（保留旧错误日志行为）。

删除 `migrator.Migrate` 与 Cobra PreRun。76 条 sql **不再 embed**。可留在 git 里但不要 `go:embed`。

- [ ] **Step 4: 测试通过**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis/backend && go test ./OpenFlare/plugins/server/stamp/ ./cmd/ -count=1
```

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/OpenFlare/plugins/server/stamp backend/OpenFlare/plugins/server/migrations backend/OpenFlare/plugins/server/chmigrate backend/cmd
git commit -m "feat(server): stamp legacy goose versions and own of_* migrations only"
```

---

### Task 15: L3 升级测试（金标准库 → Cordis）

**Files:**
- Create: `backend/cmd/upgrade_from_golden_test.go`

**Interfaces:**
- Consumes: 金标准二进制（只读调用）、`newOpenFlareApp`
- Produces: 自动化 A→C 升级断言

- [ ] **Step 1: 写失败测试**

测试内：

1. `t.TempDir()`。
2. `exec.Command` 构建金标准：`go build -o $tmp/gold /Users/ryan/Code/Go/OpenFlare`（module 路径按金标准 `main.go`）。
3. 设 `OF_DB_PATH`/`CONFIG` 指向 temp sqlite，跑金标准 `api` 数秒直到 `goose` 成功（看日志或轮询表 `of_nodes` 存在）。超时失败。
4. 抽样 `INSERT` 一行可识别数据到 `of_zones` 或读 seed 行数。
5. 对 **同一 db 文件** `newOpenFlareApp(ProfileAPI)` + `Prepare()`（不要 `Run` 以免占端口；若 Prepare 含迁移即足够）。
6. 断言：`w_schema_versions` 有 `openflare/legacy` 与 `server`；`goose_db_version` max 仍为 `202608090003`；抽样行仍在；`of_*` 表列不丢。
7. 第二次 `Prepare()` 不增加 `openflare/legacy` 行数。

Postgres：若 CI/本机有 `DATABASE`，用 `t.Skip` 以外的短测试连本机 PG；无 PG 则 sqlite 必跑，PG 用 `go test -tags pgupgrade` 可选。规格要求 Postgres 必做——测试读 `TEST_PG_DSN`，未设则 `t.Fatal` 在 `TestUpgradePostgresFromGolden` **除非** 文档写明开发机需提供。计划要求：sqlite 子测试无条件跑；Postgres 子测试在 `TEST_PG_DSN` 为空时 Skip，并在 Task 16 的人工清单里用 docker postgres 跑一遍（下面 Step 4）。

- [ ] **Step 2: 跑测试确认失败（stamp 未接好会红）**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis/backend && go test ./cmd/ -run TestUpgradeFromGolden -count=1 -timeout 120s
```

- [ ] **Step 3: 修到 PASS**

- [ ] **Step 4: 本地 Postgres**

```bash
docker run -d --name of-pg-test -e POSTGRES_PASSWORD=test -p 5432:5432 postgres:16
TEST_PG_DSN='postgres://postgres:test@127.0.0.1:5432/postgres?sslmode=disable' \
  go test ./cmd/ -run TestUpgradePostgresFromGolden -count=1 -timeout 180s
```

Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/cmd/upgrade_from_golden_test.go
git commit -m "test(cmd): upgrade sqlite/postgres from OpenFlare v3.5.4 golden"
```

---

### Task 16: L2 全量门禁

**Files:** 无新功能文件；修测试/swagger 直到绿

- [ ] **Step 1: Cordis 全测**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis/backend && go test ./...
```

Expected: PASS。修到绿。

- [ ] **Step 2: swagger 超集**

```bash
cd /Users/ryan/Code/Go/OpenFlare-cordis && make swagger
python3 - <<'PY'
import json
gold=json.load(open("/Users/ryan/Code/Go/OpenFlare/docs/swagger.json"))
new=json.load(open("docs/swagger.json"))
def ops(d):
    s=set()
    for p,v in d.get("paths",{}).items():
        for m in v:
            if m in ("get","post","put","delete","patch","head","options"):
                s.add(f"{m.upper()} {p}")
    return s
missing=sorted(ops(gold)-ops(new))
if missing:
    raise SystemExit("missing: "+"\n".join(missing))
print("ok", len(ops(gold)), "golden ops present; extra", len(ops(new)-ops(gold)))
PY
```

Expected: 无 missing。`GET /api/v1/config/public` 的成功响应仍是扁平 map（查 swagger schema / 手写测试打 handler）。

- [ ] **Step 3: 前端零 diff**

```bash
git -C /Users/ryan/Code/Go/OpenFlare-cordis diff --name-only -- frontend/
```

Expected: 空。

- [ ] **Step 4: 金标准树未被修改**

```bash
git -C /Users/ryan/Code/Go/OpenFlare status --short
```

Expected: 空。

- [ ] **Step 5: 提交本任务产生的测试/swagger 修复**（若有）

---

### Task 17: git merge 接入 Wavelet 上游

**Files:**
- Modify: `.gitattributes`
- Delete: `scripts/sync-upstream.sh`（接线成功后）
- Modify: `backend/OpenFlare/README.md`（同步方式改为 `git merge wavelet/main`）

**Interfaces:**
- Consumes: Task 16 全绿；Wavelet 含 W1–W9 的 main/工作树
- Produces: `git merge-base HEAD wavelet/main` 非空

- [ ] **Step 1: 写 `.gitattributes` 追加**

```
backend/OpenFlare/**    merge=ours
frontend/**             merge=ours
docs/changelog/**       merge=ours
docs/superpowers/**     merge=ours
```

先提交这一文件。

```bash
git add .gitattributes
git commit -m "chore(git): keep OpenFlare-owned paths on merge"
```

- [ ] **Step 2: 添加 remote 并第一次 merge**

```bash
git remote add wavelet /Users/ryan/Code/Go/Wavelet 2>/dev/null || git remote set-url wavelet /Users/ryan/Code/Go/Wavelet
git fetch wavelet
git merge --allow-unrelated-histories wavelet/main
```

冲突：`backend/{core,pkg,plugins}` 以 Wavelet 为准；`frontend/`、`backend/OpenFlare/`、`docs/superpowers/`、`docs/changelog/` 保持 OpenFlare（`git checkout --ours`）。`backend/cmd` 三路解决：保留 `server.New()`、`WithMigrationBaseline`、agent/relay/flared。不要 rebase。

- [ ] **Step 3: 验证历史**

```bash
git merge-base HEAD wavelet/main   # 非空
git log --first-parent -5          # 仍是 OpenFlare 提交叙事
git diff --name-only -- frontend/  # 相对 merge 前应无意外
cd backend && go test ./cmd/ ./OpenFlare/plugins/server/stamp/ -count=1
```

- [ ] **Step 4: 删除 rsync 脚本并提交 merge**

```bash
git rm scripts/sync-upstream.sh
git add backend
git commit --no-edit  # 若 merge 已产生 commit 则再 chore 删脚本
```

若 Step 2 的 merge 已提交，则：

```bash
git rm scripts/sync-upstream.sh
git commit -m "chore(cordis): drop rsync sync after git upstream is connected"
```

---

## 自检（对照 spec）

| Spec | Task |
| :--- | :--- |
| D1 契约冻结 / 不改前端 | 16 Step 2–3 |
| D2 删全部平台副本 | 13 |
| D3 stamp | 14、15 |
| D4 health/self/cap 在 Wavelet | 3、8；12 禁止 server 再挂 |
| D5 git merge | 17 |
| D6 不承担同类职责 | 12–14 |
| D7 金标准只读 | 1、16 Step 4 |
| D8 先对齐再 merge | 16 在 17 前 |
| W1–W9 | 2–9 |
| cmd 同构 | 12 |
| L1/L2/L3 | 1、16、15 |
| 不做 pow / 第二套 migrator / 改 agent CLI | 全局约束 |

无 TBD。`CaptchaService` / `PublicConfigProvider` / `PushRegistry` / `WithMigrationBaseline` / `stamp.Legacy` 名称前后一致。
