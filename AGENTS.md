# AGENTS.md

本文件是 AI 协作的项目级硬约束入口。所有 CodeStable 子工作流默认遵守本文件的所有规则。

## 代码规范

- Go 标准项目布局
- 遵循 go-zero / jzero 框架约定

## 禁止事项

- 不修改 go.mod 中未声明的外部依赖
- 不提交未经验证的架构改动

## 已知坑

- `go get` 某些外部库（如 `golang.org/x/crypto`、`pquerna/otp`）时可能触发 Go 版本自动升级。执行 `go get` 后需检查 `go.mod` 的 `go` 行是否被意外提升，若用户未说明允许升级则手动回退

## UI 验证要求

- 本仓库为 Go 库项目，不涉及前端 UI
