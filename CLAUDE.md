# CLAUDE.md

本文件是 `vdb-guardian` 项目的强制开发规范。所有 AI Agent、开发者和自动化工具在本仓库内进行设计、编码、测试、提交和文档更新时，都必须遵守本文件。

若本文件与临时口头要求冲突，应优先向项目负责人确认；未经确认，不得降低测试、注释、提交和安全要求。

---

## 1. 项目定位

`vdb-guardian` 是一个面向企业级部署的异构向量数据库迁移一致性验证平台。

项目核心目标：

1. 验证源向量数据库与目标向量数据库迁移后的检索行为一致性。
2. 通过“检索行为指纹”量化迁移前后的检索语义差异。
3. 支持从 Milvus 到 pgvector 的第一阶段验证，并为 Qdrant、Weaviate、Elastic/OpenSearch、Pinecone 等连接器预留扩展能力。
4. 从第一天开始按企业级工程要求设计：可测试、可扩展、可观测、可部署、可维护。

项目采用 Go + Python 的 monorepo 架构：

- Go 负责控制面、CLI、API、任务编排、连接器、Artifact Store、部署入口。
- Python 负责检索行为指纹算法、边界候选集合、指纹距离、一致性评分和实验分析。

第一阶段采用：

- Go 单体控制面；
- Python 子进程算法引擎；
- CLI-first；
- API-ready；
- 模块化 monorepo；
- 后续可平滑演进为 Go API / Go Worker / Python Fingerprint Service 的企业级部署形态。

---

## 2. 总体开发原则

### 2.1 企业级优先

所有代码必须按长期维护项目编写，而不是临时脚本。

禁止：

- 临时硬编码；
- 无测试的生产代码；
- 无注释的公共方法；
- 随意散落的脚本；
- 将真实密钥、token、数据库密码、连接串写入代码或文档；
- 为了快速通过而绕开架构边界。

必须：

- 模块边界清晰；
- 接口可扩展；
- 代码可测试；
- 错误可诊断；
- 配置可声明；
- 日志和指标可预留；
- 文档随代码同步更新。

### 2.2 先计划，后编码

复杂开发任务开始前，必须先输出工作计划并等待项目负责人审核。

计划至少包括：

1. 本次目标；
2. 涉及文件；
3. 模块边界；
4. 测试策略；
5. 提交计划；
6. 风险点；
7. 是否会影响现有行为。

未经审核，不得直接开始大规模编码。

### 2.3 TDD 强制执行

本项目采用严格 TDD。

规则：

1. 先写失败测试；
2. 运行测试，确认因目标功能缺失而失败；
3. 编写最小实现；
4. 运行测试，确认通过；
5. 必要时重构；
6. 运行完整测试；
7. 通过后才允许提交。

禁止：

- 先写生产代码再补测试；
- 没有看到测试失败就实现；
- 测试立即通过但未验证失败路径；
- 只做人工测试；
- 用无意义测试覆盖率欺骗质量要求。

每个新增方法、函数、结构体关键行为都必须有对应单元测试。

---

## 3. 代码注释规范

### 3.1 Go 注释规范

所有导出类型、导出函数、导出方法、导出常量、导出变量都必须有 Go doc 注释。

注释必须说明：

1. 该类型/方法的作用；
2. 输入参数含义；
3. 返回值含义；
4. 错误条件；
5. 与项目领域模型的关系。

示例：

```go
// IsTerminal reports whether the state represents a completed job lifecycle state.
//
// It returns true for SUCCEEDED, FAILED, and CANCELLED because these states do not
// allow normal forward progress. It returns false for all intermediate states so
// the job runner can continue checkpoint-based execution.
func (s State) IsTerminal() bool {
    ...
}
```

禁止只有形式没有内容的注释，例如：

```go
// IsTerminal checks terminal.
```

### 3.2 Python 注释规范

所有 public class、public function、public method 必须写 docstring。

docstring 至少说明：

1. 函数做什么；
2. 参数含义；
3. 返回值含义；
4. 可能抛出的异常；
5. 与检索行为指纹、边界候选或指纹距离的关系。

示例：

```python
def jaccard_distance(left: set[str], right: set[str]) -> float:
    """Compute the Jaccard distance between two identifier sets.

    The distance is used by the fingerprint engine to quantify how much two
    neighbor or boundary candidate sets differ. A distance of 0.0 means the sets
    are equivalent, while a distance of 1.0 means they have no overlap.

    Args:
        left: The first set of vector identifiers.
        right: The second set of vector identifiers.

    Returns:
        A normalized distance in the inclusive range [0.0, 1.0].
    """
```

### 3.3 注释不能替代清晰命名

变量、函数、文件名必须语义清晰。不要用注释解释糟糕命名。

禁止：

```text
x, y, tmp, data2, doThing, handleStuff
```

除非是极短局部变量或数学表达式中常用符号。

---

## 4. 测试与格式化规范

### 4.1 统一格式化要求

所有代码提交前必须进行格式化。格式化是强制质量门禁，不是可选优化。

Go 代码必须使用：

```bash
gofmt -w <go files>
```

或通过统一入口：

```bash
make fmt-go
```

Python 代码必须使用 `ruff format`，并通过 `ruff check` 做基础静态检查：

```bash
cd python
uv run ruff format .
uv run ruff check .
```

或通过统一入口：

```bash
make fmt-python
make lint-python
```

仓库根目录必须提供统一格式化入口：

```bash
make fmt
```

提交前必须至少运行：

```bash
make fmt
make test
```

如果后续引入 Markdown、YAML、JSON 格式化工具，应统一接入 `make fmt`，不得各自为政。

禁止提交未格式化代码。

### 4.2 Go 测试

Go 代码必须使用标准测试框架 `testing`。

测试命令：

```bash
go test ./...
```

每个 Go package 必须至少包含对应测试文件。

测试要求：

1. 正常路径；
2. 错误路径；
3. 边界值；
4. 空输入；
5. 非法输入；
6. 幂等行为；
7. 状态转换行为。

测试命名必须描述行为：

```go
func TestStateIsTerminalReturnsTrueForCompletedStates(t *testing.T) { ... }
```

禁止模糊命名：

```go
func TestState(t *testing.T) { ... }
```

### 4.3 Python 测试

Python 测试使用 `pytest`，由 `uv` 管理依赖。

测试命令：

```bash
cd python
uv run pytest
```

Python 算法必须重点测试：

1. 空集合；
2. 完全相同集合；
3. 完全不同集合；
4. 部分重叠集合；
5. topK 边界；
6. topK 大于结果数量；
7. 阈值变化；
8. 权重非法；
9. 输出范围是否在 [0, 1]；
10. JSON 输入输出协议是否稳定。

### 4.4 总测试命令

仓库根目录必须提供统一测试入口：

```bash
make test
```

该命令至少包含：

```bash
go test ./...
cd python && uv run pytest
```

提交前必须运行并通过：

```bash
make test
```

如果某次提交只改文档，可说明原因，但不得破坏现有测试。

---

## 5. 目录和模块规范

### 5.1 推荐目录结构

```text
vdb-guardian/
├── cmd/
│   ├── vdbg/
│   └── vdb-guardian-server/
├── internal/
│   ├── version/
│   ├── jobs/
│   ├── connectors/
│   ├── engine/
│   ├── artifacts/
│   ├── config/
│   └── observability/
├── python/
│   └── vdb_fingerprint_engine/
├── configs/
├── docs/
├── deployments/
└── tests/
```

### 5.2 Go 模块边界

- `cmd/vdbg`：CLI 入口。
- `cmd/vdb-guardian-server`：服务端入口。
- `internal/jobs`：任务状态机和任务运行模型。
- `internal/connectors`：向量数据库连接器接口和实现。
- `internal/engine`：Go 控制面与 Python 指纹引擎的协议边界。
- `internal/artifacts`：指纹、报告和中间产物存储抽象。
- `internal/config`：配置加载和校验。
- `internal/observability`：日志、指标、追踪预留。

禁止跨层随意调用。核心业务逻辑不得写在 `cmd/` 中。

### 5.3 Python 模块边界

- `schemas.py`：输入输出协议和数据结构。
- `boundary.py`：边界候选集合选择。
- `distance.py`：集合距离、边界翻转率、指纹距离。
- `scoring.py`：一致性评分。
- `fingerprint.py`：检索行为指纹构造。
- `cli.py`：Python 引擎命令行入口。

Python 算法引擎不得直接依赖 Go 内部实现。

---

## 6. 架构规范

### 6.1 Go 和 Python 的职责边界

Go 负责企业级系统控制：

- CLI；
- API server；
- Job runner；
- Connector；
- Artifact Store；
- 配置校验；
- 日志指标；
- 部署入口。

Python 负责算法：

- 检索行为单元；
- 稳定近邻集合；
- 边界候选集合；
- 相似度衰减特征；
- 指纹距离；
- 一致性评分；
- 指纹差异报告中的算法指标。

Go 与 Python 通过稳定 JSON 协议通信。协议变化必须同时更新：

1. Go schema；
2. Python schema；
3. 单元测试；
4. 文档。

### 6.2 连接器规范

所有向量数据库连接器必须实现统一接口。

连接器必须支持：

1. 连接；
2. 健康检查；
3. 计数；
4. 搜索；
5. 关闭；
6. 超时控制；
7. 错误返回。

连接器不得将数据库特有字段泄露到核心算法层。数据库差异应在 connector 层归一化。

### 6.3 Artifact Store 规范

指纹文件、报告文件、中间产物不得随意散落。

必须通过 Artifact Store 抽象读写。

第一阶段允许实现：

- local store；
- memory store。

后续必须能扩展到：

- S3；
- MinIO；
- OSS；
- GCS。

### 6.4 Job 状态机规范

任务必须通过明确状态推进。

推荐状态：

```text
CREATED
VALIDATING_CONFIG
CONNECTING_SOURCE
CONNECTING_TARGET
SAMPLING_QUERIES
COLLECTING_SOURCE_RESULTS
COLLECTING_TARGET_RESULTS
RUNNING_FINGERPRINT_ENGINE
GENERATING_REPORT
SUCCEEDED
FAILED
CANCELLED
```

状态转换必须可测试。终态必须明确。

---

## 7. 配置规范

配置必须声明式，优先使用 YAML。

示例配置必须使用占位值，不得包含真实密钥。

禁止提交：

- `.env`；
- 真实数据库连接串；
- GitHub token；
- 云服务密钥；
- 客户数据路径；
- 私有证书。

示例连接串必须写成：

```text
postgresql://postgres:[REDACTED]@localhost:5433/postgres
```

或：

```text
postgresql://postgres:postgres@localhost:5433/postgres
```

仅限本地示例环境。

---


### 7.1 Python 依赖管理规范

Python 子项目必须优先使用 `uv` 管理依赖、虚拟环境和命令执行。`pip` 仅作为系统兼容、故障排查或安装底层工具时的备用手段，不作为项目依赖的标准安装入口。

标准命令：

```bash
cd python
uv sync
uv run pytest
uv run ruff format .
uv run ruff check .
```

禁止在未说明原因的情况下使用全局 `pip install` 安装项目依赖，避免污染系统 Python 环境或造成不可复现的构建结果。

---

## 8. 安全规范

### 8.1 凭据安全

任何 token、密码、私钥、云服务 AK/SK、数据库真实连接串都不得提交。

如果工具输出中出现敏感信息，必须在报告或文档中写为：

```text
[REDACTED]
```

### 8.2 数据安全

默认使用合成数据进行测试。

不得擅自读取、复制、上传用户真实业务数据。

如果需要使用真实数据，必须先获得项目负责人明确批准，并说明：

1. 数据来源；
2. 数据字段；
3. 是否脱敏；
4. 存储位置；
5. 删除方式。

### 8.3 Docker 安全

不得擅自删除已有容器、镜像、volume、network。

涉及 Docker 操作前必须说明：

1. 会创建什么容器；
2. 使用什么端口；
3. 使用什么 volume；
4. 是否会影响已有服务；
5. 如何清理。

---

## 9. Git 工作流规范

### 9.1 分支规范

不得直接在 `main` 分支开发。

分支命名：

- `feat/<description>`：新功能；
- `fix/<description>`：问题修复；
- `docs/<description>`：文档；
- `test/<description>`：测试；
- `chore/<description>`：工程配置；
- `refactor/<description>`：重构。

当前企业级骨架开发分支：

```text
feat/enterprise-scaffold
```

### 9.2 提交规范

采用 Conventional Commits。

格式：

```text
type(scope): short summary

- Detail 1
- Detail 2
- Detail 3
```

示例：

```text
feat(engine): add boundary candidate distance metrics

- Add boundary candidate selection based on topK expanded search results
- Add boundary flip rate calculation for fingerprint comparison
- Add unit tests for empty, identical, and partially overlapping sets
```

每次提交前必须：

```bash
make test
```

测试未通过不得提交。

### 9.3 推送规范

推送前必须确认：

```bash
git status --short
git branch --show-current
make test
```

不得强推 `main`。

如需 force push feature branch，必须先说明原因并获得确认。

---

## 10. 文档规范

任何新增能力都必须同步更新文档。

至少包括：

1. README；
2. `docs/architecture.md`；
3. 相关模块 spec；
4. 配置示例；
5. 测试命令。

README 必须包含：

1. 项目定位；
2. 架构概览；
3. 快速开始；
4. 本地开发；
5. 测试命令；
6. 配置示例；
7. Roadmap。

文档不得夸大能力。未实现的能力必须标注为 planned 或 roadmap。

---

## 11. 错误处理规范

### 11.1 Go 错误处理

Go 中不得忽略 error。

禁止：

```go
result, _ := doSomething()
```

除非有明确注释说明为什么可以忽略。

错误必须包含上下文：

```go
return fmt.Errorf("load job config %q: %w", path, err)
```

### 11.2 Python 异常处理

Python 中不得裸 `except`。

禁止：

```python
try:
    ...
except:
    pass
```

异常必须明确类型，并包含可诊断信息。

---

## 12. 可观测性规范

企业级模块必须预留可观测性。

后续服务化时至少支持：

1. structured logging；
2. Prometheus metrics；
3. job id trace；
4. connector latency；
5. engine duration；
6. job status count；
7. failure reason。

第一阶段不强制完成全部实现，但接口设计不得阻碍后续接入。

---

## 13. 性能和可靠性规范

1. 大数据量处理必须考虑流式处理或分页。
2. Connector 查询必须支持 context timeout。
3. 长任务必须可 checkpoint。
4. 指纹和报告应作为 artifact 存储。
5. 任务失败必须有明确错误原因。
6. 任务取消必须可控。
7. 后续引入并发时必须控制并发度和资源占用。

---

## 14. 专利一致性规范

本项目代码和文档应服务于以下专利核心概念：

1. 检索行为指纹；
2. 稳定近邻集合；
3. 边界候选集合；
4. 相似度衰减特征；
5. 指纹距离；
6. 边界候选翻转差异；
7. 迁移一致性评分；
8. 指纹差异报告。

命名应尽量与专利交底书保持一致。

不得将项目实现表述为简单的“迁移 + benchmark + 调参”组合。

应强调：

```text
本项目验证的是异构向量数据库迁移后的检索行为层一致性，而不仅是数据完整性或性能指标。
```

---

## 15. 禁止事项

未经项目负责人批准，禁止：

1. 删除远程分支；
2. 修改 main 分支历史；
3. 删除 Docker volume；
4. 删除用户已有容器；
5. 上传真实业务数据；
6. 提交密钥或 token；
7. 跳过测试提交代码；
8. 绕过 TDD；
9. 引入重型依赖而不说明原因；
10. 将未实现能力写成已实现；
11. 在没有计划的情况下大规模重构；
12. 擅自更改许可证。

---

## 16. 开发前检查清单

每次开始开发前必须确认：

- [ ] 当前不在 main 分支；
- [ ] 工作区干净或已说明已有改动；
- [ ] 明确本次目标；
- [ ] 明确涉及文件；
- [ ] 已有测试计划；
- [ ] 不会提交敏感信息；
- [ ] 不会破坏现有接口；
- [ ] 已获得必要审核。

---

## 17. 提交前检查清单

每次提交前必须确认：

- [ ] 所有新增 public 方法都有注释；
- [ ] 所有新增方法都有单元测试；
- [ ] 已运行 `make fmt`；
- [ ] 已运行 `make lint`；
- [ ] 已运行 `make test`；
- [ ] 测试全部通过；
- [ ] README 或 docs 已同步更新；
- [ ] `git diff` 已自查；
- [ ] 没有密钥、token、真实连接串；
- [ ] commit message 详细说明提交内容。

---

## 18. Agent 执行要求

AI Agent 在本项目中工作时必须：

1. 先读取本文件；
2. 遵守 TDD；
3. 使用工具实际检查文件和环境，不凭空假设；
4. 写代码前先说明计划；
5. 执行过程中持续汇报进度，尤其在完成阶段性任务、测试失败、发现阻塞、修改计划或准备提交前必须同步当前状态；
6. 进度汇报必须包含：已完成内容、正在执行内容、下一步动作、风险或阻塞、测试状态；
7. 遇到权限、凭据、Docker、远程推送、系统安装等问题及时反馈；
8. 不把计划当完成结果；
9. 不在测试失败时提交；
10. 不将敏感信息输出到最终报告。

---

## 19. 当前阶段约束

当前阶段为企业级项目骨架建设阶段。

允许：

- 创建 Go module；
- 创建 Python uv 子项目；
- 创建 CLI/server 最小入口；
- 创建 connector/interface/job/artifact/engine 抽象；
- 创建单元测试；
- 更新 README 和 docs；
- 推送 feature branch。

暂不允许，除非另行批准：

- 启动真实 Milvus/pgvector 容器；
- 连接真实数据库；
- 引入 Kubernetes/Helm 完整部署；
- 引入 Web UI；
- 接入真实业务数据；
- 合并到 main。

---

## 20. 规范变更

本文件可以随着项目演进更新，但更新必须：

1. 单独说明变更原因；
2. 更新相关文档；
3. 通过测试；
4. 使用独立 commit 或在 commit message 中明确说明。

任何降低质量门槛的变更必须经过项目负责人明确批准。
