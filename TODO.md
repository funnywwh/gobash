# gobash 未完成任务清单

## 🔴 代码中的TODO项（需要修复/完善）

### 1. ✅ exit命令的退出码解析 - **已完成**
**位置**: `internal/builtin/builtin.go:127`
**状态**: 已修复，现在可以正确解析退出码
**修复内容**: 使用 `strconv.Atoi` 解析退出码参数

```go
// 修复后的代码
func exit(args []string, env map[string]string) error {
    code := 0
    if len(args) > 0 {
        if parsed, err := strconv.Atoi(args[0]); err == nil {
            code = parsed
        }
    }
    os.Exit(code)
    return nil
}
```

### 2. ✅ for循环的位置参数支持 - **已完成**
**位置**: `internal/executor/executor.go:357`
**状态**: 已实现，支持 `for i; do ... done` 语法
**修复内容**: 从环境变量中读取位置参数（$1, $2, ...）并遍历

```go
// 修复后的代码
if len(stmt.In) == 0 {
    // 从环境变量中获取位置参数
    argCount := 0
    if countStr, ok := e.env["#"]; ok {
        fmt.Sscanf(countStr, "%d", &argCount)
    }
    // 遍历位置参数并执行循环体
    ...
}
```

### 3. ✅ set命令的选项实现 - **已完成**
**位置**: `internal/builtin/builtin.go:186` 和 `internal/shell/shell.go`
**状态**: 已实现，支持 `set -x`, `set -e`, `set +x`, `set +e` 等选项
**修复内容**: 
- 在Shell结构体中添加options字段存储选项状态
- 在Executor中添加选项支持，实现 `-x`（显示执行的命令）和 `-e`（遇到错误立即退出）
- 在Shell中直接处理set命令，支持设置/取消设置选项

**支持的选项**:
- `set -x` / `set +x`: 显示/隐藏执行的命令（xtrace）
- `set -e` / `set +e`: 遇到错误立即退出/继续执行（errexit）
- `set -xe`: 可以组合多个选项
- `set`: 显示当前选项状态

## 🟡 可选增强功能（README中列出）

### 1. 箭头键浏览历史
**状态**: 未实现
**需求**: 需要readline库支持（如 `github.com/chzyer/readline`）
**优先级**: 中
**难度**: ⭐⭐⭐ 需要外部库集成

### 2. 命令自动补全功能
**状态**: 未实现
**需求**: Tab键补全命令、文件名、变量名等
**优先级**: 中
**难度**: ⭐⭐⭐⭐ 复杂

### 3. 更多Bash特性
**状态**: 未实现
**包括**:
- 数组支持（`arr=(1 2 3)`）
- 关联数组（`declare -A arr`）
- 进程替换（`<(command)`, `>(command)`）
- 命令替换（`` `command` `` 或 `$(command)`）
- 算术展开（`$((expr))`）

**优先级**: 低
**难度**: ⭐⭐⭐⭐⭐ 非常复杂

### 4. 作业控制
**状态**: 未实现
**包括**:
- 后台任务（`command &`）
- `fg` 命令（前台任务）
- `bg` 命令（后台任务）
- `jobs` 命令（显示作业列表）
- `Ctrl+Z` 信号处理

**优先级**: 中
**难度**: ⭐⭐⭐⭐ 复杂（需要信号处理）

### 5. 更多测试用例和文档
**状态**: 部分完成
**需求**:
- 单元测试
- 集成测试
- API文档
- 使用示例

**优先级**: 中
**难度**: ⭐⭐ 中等

### 6. ✅ 更多内置命令 - **部分完成**
**状态**: 已实现head, tail, wc
**已实现**:
- `head` - 显示文件的前几行
- `tail` - 显示文件的后几行
- `wc` - 统计行数、字数、字符数

**待实现**:
- 无（所有计划的内置命令已完成）

**已实现**:
- `grep` - 文本搜索 ✅
- `sort` - 排序 ✅
- `uniq` - 去重 ✅
- `cut` - 剪切字段 ✅

**已实现**:
- `grep` - 文本搜索 ✅
- `sort` - 排序 ✅
- `uniq` - 去重 ✅

**优先级**: 中
**难度**: ⭐⭐ 中等

## 📊 优先级建议

### 高优先级（建议先完成）
1. ✅ **exit命令的退出码解析** - 简单且常用
2. ✅ **for循环的位置参数支持** - 常用Bash语法

### 中优先级（可选）
3. ⚠️ **set命令的选项实现** - 部分常用
4. ⚠️ **箭头键浏览历史** - 提升用户体验
5. ⚠️ **作业控制** - 实用功能

### 低优先级（未来考虑）
6. ⚠️ **命令自动补全** - 复杂但有用
7. ⚠️ **更多Bash特性** - 逐步实现
8. ⚠️ **测试和文档** - 持续改进

## 🎯 快速修复建议

### 修复exit命令（5分钟）
```go
func exit(args []string, env map[string]string) error {
    code := 0
    if len(args) > 0 {
        if parsed, err := strconv.Atoi(args[0]); err == nil {
            code = parsed
        }
    }
    os.Exit(code)
    return nil
}
```

### 修复for循环位置参数（15分钟）
需要从环境变量中获取位置参数（`$1`, `$2`, ...）并传递给for循环。

