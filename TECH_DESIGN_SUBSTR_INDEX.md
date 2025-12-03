# substr 和 index 算术函数技术设计方案

## 1. 需求分析

### 1.1 bash 中的用法

在 bash 中，`substr` 和 `index` 函数在算术展开 `$((...))` 中使用：

```bash
# substr 函数示例
VAR="hello"
echo $((substr($VAR, 0, 3)))  # 注意：bash 中 substr 返回字符串，但算术展开只能返回数字
# 实际上，bash 的 substr 函数在算术展开中返回的是子字符串的数值表示（通常是字符的 ASCII 码）

# index 函数示例
VAR="hello"
echo $((index($VAR, "ll")))   # 输出 3（从 1 开始，找到 "ll" 的位置）
echo $((index($VAR, "xyz")))  # 输出 0（未找到）
```

### 1.2 函数签名

- `substr(s, start, length)` - 提取子字符串
  - `s`: 字符串（通过变量展开获取）
  - `start`: 起始位置（从 1 开始，bash 的行为）
  - `length`: 子字符串长度
  - 返回：子字符串的数值表示（简化实现中，可以返回子字符串的长度或第一个字符的 ASCII 码）

- `index(s, t)` - 查找子字符串位置
  - `s`: 源字符串
  - `t`: 要查找的子字符串
  - 返回：子字符串的位置（从 1 开始），未找到返回 0

### 1.3 当前架构限制

1. **参数解析器限制**：
   - `parseArithmeticFunctionArgs` 只解析算术表达式，返回 `[]int64`
   - 无法区分数字参数和字符串参数

2. **函数调用限制**：
   - `evaluateArithmeticFunction` 只接受 `[]int64` 参数
   - 无法传递字符串参数

3. **字符串支持缺失**：
   - 算术表达式中无法直接传递字符串参数
   - 变量展开在解析前完成，展开后的字符串无法被识别为字符串参数

## 2. 技术方案设计

### 2.1 方案概述

**核心思路**：在变量展开阶段识别字符串参数，并在解析时保留字符串信息。

### 2.2 数据结构设计

#### 2.2.1 参数类型枚举

```go
// ArithmeticArgType 算术函数参数类型
type ArithmeticArgType int

const (
    ArithmeticArgTypeNumber ArithmeticArgType = iota // 数字参数
    ArithmeticArgTypeString                          // 字符串参数
)
```

#### 2.2.2 参数结构体

```go
// ArithmeticFunctionArg 算术函数参数
type ArithmeticFunctionArg struct {
    Type    ArithmeticArgType // 参数类型
    Number  int64             // 数字值（当 Type == ArithmeticArgTypeNumber 时使用）
    String  string            // 字符串值（当 Type == ArithmeticArgTypeString 时使用）
}
```

### 2.3 实现步骤

#### 步骤 1：修改参数解析器

**文件**：`internal/executor/executor.go`

**函数**：`parseArithmeticFunctionArgs`

**修改内容**：
1. 修改函数签名：`func parseArithmeticFunctionArgs(expr string, pos *int) ([]ArithmeticFunctionArg, error)`
2. 在解析参数时，识别字符串参数：
   - 如果参数是变量展开（如 `$VAR`），在展开后检查是否为字符串
   - 如果参数是字符串字面量（如 `"string"` 或 `'string'`），解析为字符串参数
   - 否则，解析为数字参数

**关键问题**：
- 变量展开在 `evaluateArithmetic` 中完成，展开后的表达式已经是纯数字和运算符
- 需要在展开前识别哪些变量会被用作字符串参数

**解决方案**：
- 方案 A：在解析函数调用时，根据函数名判断参数类型
  - 对于 `substr` 和 `index`，第一个参数是字符串
  - 对于 `index`，第二个参数也是字符串
- 方案 B：在变量展开阶段标记字符串参数
  - 在 `evaluateArithmetic` 中，识别函数调用
  - 对于需要字符串参数的函数，标记相应的变量展开为字符串参数
- 方案 C：修改算术表达式解析流程
  - 在解析前，先识别函数调用
  - 对于需要字符串参数的函数，在展开变量时保留字符串信息

**推荐方案**：方案 A + 方案 C 的混合方案
- 在 `parseArithmeticFactor` 中识别函数调用
- 对于 `substr` 和 `index`，在解析参数时特殊处理
- 在解析参数时，如果遇到变量展开（如 `$VAR`），先展开变量，然后根据函数签名判断是否为字符串参数

#### 步骤 2：修改函数调用逻辑

**文件**：`internal/executor/executor.go`

**函数**：`parseArithmeticFactor`, `evaluateArithmeticFunction`

**修改内容**：
1. 在 `parseArithmeticFactor` 中：
   - 识别需要字符串参数的函数（`substr`, `index`）
   - 调用新的参数解析函数，获取混合参数列表
2. 在 `evaluateArithmeticFunction` 中：
   - 修改函数签名：`func evaluateArithmeticFunction(name string, args []ArithmeticFunctionArg) (int64, error)`
   - 对于普通函数，提取数字参数
   - 对于需要字符串参数的函数，提取字符串参数和数字参数，调用 `evaluateArithmeticFunctionWithStrings`

#### 步骤 3：实现字符串参数解析

**新函数**：`parseArithmeticStringArg`

**功能**：
- 解析字符串参数（变量展开或字符串字面量）
- 返回字符串值

**实现细节**：
- 识别变量展开：`$VAR` 或 `${VAR}`
- 识别字符串字面量：`"string"` 或 `'string'`
- 展开变量并返回字符串值

#### 步骤 4：完善函数实现

**文件**：`internal/executor/executor.go`

**函数**：`evaluateArithmeticFunctionWithStrings`

**修改内容**：
- 完善 `substr` 和 `index` 的实现
- 处理边界情况
- 确保与 bash 行为一致

### 2.4 实现难点和解决方案

#### 难点 1：变量展开时机

**问题**：变量展开在算术表达式解析前完成，展开后的字符串无法被识别为字符串参数。

**解决方案**：
- 在解析函数调用时，先识别函数名
- 对于需要字符串参数的函数，在解析参数时特殊处理
- 在解析参数时，如果遇到变量引用（如 `$VAR`），先不展开，而是标记为字符串参数
- 在调用函数时，再展开变量获取字符串值

#### 难点 2：向后兼容性

**问题**：修改参数解析器可能影响现有函数。

**解决方案**：
- 保持现有函数的参数解析逻辑不变
- 只在识别到需要字符串参数的函数时，使用新的解析逻辑
- 充分测试现有函数，确保向后兼容

#### 难点 3：字符串字面量解析

**问题**：算术表达式中可能包含字符串字面量（如 `"string"`），需要正确解析。

**解决方案**：
- 在解析参数时，识别引号（`"` 或 `'`）
- 解析引号内的字符串
- 去除引号，返回字符串值

### 2.5 实现流程

```
1. 在 parseArithmeticFactor 中识别函数调用
   ↓
2. 检查函数名是否为 substr 或 index
   ↓
3. 如果是，调用 parseArithmeticFunctionArgsWithStrings 解析参数
   ↓
4. parseArithmeticFunctionArgsWithStrings 解析混合参数：
   - 对于字符串参数位置，解析变量展开或字符串字面量
   - 对于数字参数位置，解析算术表达式
   ↓
5. 调用 evaluateArithmeticFunction，传递混合参数
   ↓
6. evaluateArithmeticFunction 根据函数名：
   - 对于普通函数，提取数字参数，调用 evaluateArithmeticFunction
   - 对于 substr/index，提取字符串和数字参数，调用 evaluateArithmeticFunctionWithStrings
   ↓
7. 返回结果
```

### 2.6 代码结构

```
internal/executor/executor.go
├── ArithmeticArgType (新类型)
├── ArithmeticFunctionArg (新结构体)
├── parseArithmeticFunctionArgs (修改)
│   └── 返回 []ArithmeticFunctionArg
├── parseArithmeticFunctionArgsWithStrings (新函数)
│   └── 解析混合参数（字符串 + 数字）
├── parseArithmeticStringArg (新函数)
│   └── 解析字符串参数
├── parseArithmeticFactor (修改)
│   └── 识别需要字符串参数的函数
├── evaluateArithmeticFunction (修改)
│   └── 接受 []ArithmeticFunctionArg，分发到相应函数
└── evaluateArithmeticFunctionWithStrings (完善)
    ├── substr 实现
    └── index 实现
```

## 3. 测试方案

### 3.1 单元测试

1. **参数解析测试**：
   - 测试数字参数解析
   - 测试字符串参数解析（变量展开）
   - 测试字符串参数解析（字符串字面量）
   - 测试混合参数解析

2. **函数调用测试**：
   - 测试 substr 函数
   - 测试 index 函数
   - 测试向后兼容性（现有函数仍正常工作）

3. **边界情况测试**：
   - 负数索引
   - 超出范围
   - 空字符串
   - 未找到子字符串

### 3.2 集成测试

1. 测试完整的算术表达式
2. 测试嵌套函数调用
3. 测试与其他算术函数的组合使用

## 4. 风险评估

### 4.1 技术风险

1. **向后兼容性风险**：中等
   - **缓解措施**：充分测试现有函数，确保向后兼容

2. **字符串参数识别风险**：中等
   - **缓解措施**：在解析阶段明确标记字符串参数位置

3. **性能影响**：低
   - **缓解措施**：只在需要时解析字符串参数

### 4.2 实现风险

1. **复杂度增加**：中等
   - **缓解措施**：分步骤实现，每个步骤完成后测试

2. **错误处理**：中等
   - **缓解措施**：添加详细的错误处理和测试

## 5. 实施计划

1. **子任务 2**：扩展参数解析器（3-4 小时）
2. **子任务 3**：修改函数调用逻辑（2-3 小时）
3. **子任务 4**：实现 substr 函数（2-3 小时）
4. **子任务 5**：实现 index 函数（2-3 小时）
5. **子任务 6**：集成测试和文档（2-3 小时）

## 6. 总结

本方案通过扩展参数解析器以支持混合参数类型（数字 + 字符串），实现了 `substr` 和 `index` 函数。关键是在解析阶段识别需要字符串参数的函数，并在解析参数时特殊处理。方案保持了向后兼容性，并提供了清晰的实现路径。

