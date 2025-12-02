# gobash 词法、语法分析和变量展开完全重构计划

## 项目概述

参考 bash 源码（parse.y, subst.c, input.c），完全重构 gobash 的词法分析、语法分析和变量展开系统，提高与 bash 的兼容性。

## 当前进度

### ✅ 已完成的工作

#### 1. 词法分析器改进（阶段 1 部分完成）

- ✅ **Token 类型扩展** (`internal/lexer/token.go`)
  - 添加了新的重定向 token：`REDIRECT_HEREDOC`, `REDIRECT_HEREDOC_STRIP`, `REDIRECT_HEREDOC_TABS`, `REDIRECT_DUP_IN`, `REDIRECT_DUP_OUT`, `REDIRECT_CLOBBER`, `REDIRECT_RW`
  - 添加了新的操作符 token：`SEMI_SEMI`, `SEMI_AND`, `SEMI_SEMI_AND`, `BAR_AND`, `AND_GREATER`, `AND_GREATER_GREATER`
  - 添加了参数展开 token：`PARAM_EXPAND`
  - 添加了字符串 token：`STRING_DOLLAR_SINGLE` ($'...'), `STRING_DOLLAR_DOUBLE` ($"...")
  - 添加了 Here-document token：`HEREDOC_MARKER`, `HEREDOC_CONTENT`
  - 添加了赋值 token：`ASSIGNMENT_WORD`
  - 添加了复合命令 token：`SUBSHELL_START`, `SUBSHELL_END`, `GROUP_START`, `GROUP_END`

- ✅ **Lexer 改进** (`internal/lexer/lexer.go`)
  - 改进了重定向识别：支持 `<<`, `<<-`, `<<<`, `<&`, `>&`, `>|`, `<>`
  - 改进了操作符识别：支持 `;;`, `;&`, `;;&`, `|&`, `&>`, `&>>`
  - 添加了 `$'...'` ANSI-C 字符串支持（`readDollarSingleQuote()`）
  - 添加了 `$"..."` 国际化字符串支持（`readDollarDoubleQuote()`）
  - 改进了参数展开识别：能够识别 `${VAR...}` 的所有形式（包括嵌套的引号、命令替换等）
  - 所有现有测试通过

### 📋 待完成的工作

## 详细 TODO 列表

### 阶段 1: 词法分析器重构（部分完成）

#### ✅ 已完成
- [x] 研究 bash 的词法分析实现（read_token, read_token_word, shell_getc）
- [x] 扩展 token.go，添加新的 token 类型（参数展开、Here-document 等）
- [x] 重构 lexer.go，实现类似 bash 的 readToken() 和 readTokenWord() 函数
- [x] 改进引号处理（单引号、双引号、反引号、$'...', $"..."）
- [x] 改进变量识别（支持所有参数展开形式）

#### 🔄 进行中
- [ ] 添加 Here-document 支持（<<EOF ... EOF）
  - [ ] 实现 Here-document 标记识别
  - [ ] 实现 Here-document 内容读取
  - [ ] 处理 Here-document 的引号（带引号的标记不展开变量）
  - [ ] 处理 Here-document 的制表符剥离（<<-）
  - [ ] 测试 Here-document 功能

#### ⏳ 待开始
- [ ] 改进空白字符和换行符处理
  - [ ] 正确处理引号内的空白字符
  - [ ] 正确处理转义的换行符
  - [ ] 正确处理多行命令

- [ ] 支持多字节字符（UTF-8）
  - [ ] 正确处理 UTF-8 编码的字符
  - [ ] 正确处理多字节字符的引号
  - [ ] 正确处理多字节字符的变量名

- [ ] 改进命令替换嵌套处理
  - [ ] 正确处理 `command` 和 $(command) 的嵌套
  - [ ] 正确处理嵌套中的引号
  - [ ] 正确处理嵌套中的转义

- [ ] 改进算术展开嵌套处理
  - [ ] 正确处理 $((expr)) 的嵌套括号
  - [ ] 正确处理嵌套中的变量展开

- [ ] 词法分析器测试
  - [ ] 添加新功能的单元测试
  - [ ] 添加边界情况测试
  - [ ] 添加错误处理测试

### 阶段 2: 语法分析器重构（未开始）

#### 研究阶段
- [ ] 研究 bash 的语法分析实现（parse.y 语法规则）
  - [ ] 分析 bash 的语法规则结构
  - [ ] 理解 bash 的 AST 节点类型
  - [ ] 理解 bash 的解析优先级
  - [ ] 理解 bash 的错误处理机制

#### AST 结构重构
- [ ] 重构 ast.go，改进 AST 结构
  - [ ] 添加参数展开节点（`ParamExpandExpression`）
  - [ ] 添加复合命令节点（`SubshellCommand`, `GroupCommand`）
  - [ ] 添加更详细的重定向节点（支持所有重定向类型）
  - [ ] 添加 Here-document 节点
  - [ ] 添加命令链节点（`CommandChain`）
  - [ ] 添加条件命令节点（`ConditionalCommand`）

#### 解析器重构
- [ ] 重构 parser.go，实现新的解析逻辑
  - [ ] 实现命令解析改进
    - [ ] 支持简单命令
    - [ ] 支持管道命令
    - [ ] 支持命令链（`;`, `&&`, `||`）
    - [ ] 支持后台命令（`&`）
  - [ ] 实现重定向解析改进
    - [ ] 支持所有重定向类型（>, <, >>, <<, <&, >&, >|, <>, etc.）
    - [ ] 支持文件描述符重定向（2>, 1>, etc.）
    - [ ] 支持 Here-document
    - [ ] 支持 Here-string（<<<）
  - [ ] 实现控制流解析改进
    - [ ] if/else/elif/fi
    - [ ] for/in/do/done
    - [ ] while/do/done
    - [ ] case/in/esac
    - [ ] function 定义
  - [ ] 实现复合命令解析
    - [ ] 子shell `(command)`
    - [ ] 命令组 `{ command; }`
    - [ ] 条件命令 `[[ condition ]]`
  - [ ] 改进数组和关联数组语法解析
  - [ ] 改进错误处理和错误报告
  - [ ] 支持多行语句的正确解析

#### 语法分析器测试
- [ ] 添加新功能的单元测试
- [ ] 添加边界情况测试
- [ ] 添加错误处理测试
- [ ] 运行现有测试，确保兼容性

### 阶段 3: 变量展开系统重构（未开始）

#### 研究阶段
- [ ] 研究 bash 的变量展开实现（subst.c 中的 expand_string_internal, param_expand）
  - [ ] 分析 expand_string_internal 的实现
  - [ ] 分析 param_expand 的实现
  - [ ] 理解变量展开的优先级
  - [ ] 理解单词分割（IFS）的实现
  - [ ] 理解路径名展开（通配符）的实现

#### 创建变量展开模块
- [ ] 创建新的变量展开模块（`internal/executor/subst.go`）
  - [ ] 定义展开上下文结构
  - [ ] 定义展开标志
  - [ ] 定义展开结果类型

#### 实现参数展开函数
- [ ] 实现完整的参数展开：
  - [ ] `${VAR:-word}` - 使用默认值
  - [ ] `${VAR:=word}` - 赋值默认值
  - [ ] `${VAR:?word}` - 显示错误
  - [ ] `${VAR:+word}` - 使用替代值
  - [ ] `${VAR#pattern}` - 删除最短匹配前缀
  - [ ] `${VAR##pattern}` - 删除最长匹配前缀
  - [ ] `${VAR%pattern}` - 删除最短匹配后缀
  - [ ] `${VAR%%pattern}` - 删除最长匹配后缀
  - [ ] `${VAR:offset}` - 子字符串
  - [ ] `${VAR:offset:length}` - 子字符串
  - [ ] `${#VAR}` - 字符串长度
  - [ ] `${VAR[@]}` - 数组展开（带引号时每个元素单独展开）
  - [ ] `${VAR[*]}` - 数组展开（所有元素作为一个单词）
  - [ ] `${!VAR}` - 间接引用
  - [ ] `${VAR[expr]}` - 数组/关联数组访问

#### 实现其他展开功能
- [ ] 改进算术展开
  - [ ] 支持完整的算术表达式
  - [ ] 支持所有算术运算符
  - [ ] 支持算术函数
  - [ ] 正确处理变量展开
- [ ] 改进命令替换
  - [ ] 正确处理嵌套
  - [ ] 正确处理转义
  - [ ] 正确处理退出码
- [ ] 改进数组访问
  - [ ] `${arr[0]}` - 普通数组
  - [ ] `${arr[key]}` - 关联数组
  - [ ] `${arr[@]}` - 数组展开
  - [ ] `${arr[*]}` - 数组展开
- [ ] 实现单词分割（IFS）
  - [ ] 根据 IFS 分割单词
  - [ ] 正确处理 IFS 为空的情况
  - [ ] 正确处理 IFS 为默认值的情况
- [ ] 实现路径名展开（通配符）
  - [ ] 支持 `*` 通配符
  - [ ] 支持 `?` 通配符
  - [ ] 支持 `[...]` 字符类
  - [ ] 支持 `**` 递归匹配（如果启用）
  - [ ] 正确处理隐藏文件
- [ ] 实现波浪号展开（~）
  - [ ] `~` - 当前用户主目录
  - [ ] `~user` - 指定用户主目录
  - [ ] `~+` - 当前工作目录
  - [ ] `~-` - 上一个工作目录

#### 变量展开系统测试
- [ ] 添加新功能的单元测试
- [ ] 添加边界情况测试
- [ ] 添加错误处理测试
- [ ] 运行现有测试，确保兼容性

### 阶段 4: 集成和测试（未开始）

- [ ] 集成所有重构的模块
  - [ ] 确保词法分析器与语法分析器兼容
  - [ ] 确保语法分析器与执行器兼容
  - [ ] 确保变量展开系统与执行器兼容
- [ ] 运行现有测试，确保兼容性
  - [ ] 运行所有单元测试
  - [ ] 运行集成测试
  - [ ] 运行脚本测试
- [ ] 修复回归问题
  - [ ] 修复破坏的测试
  - [ ] 修复功能回归
  - [ ] 修复性能回归
- [ ] 添加新功能测试
  - [ ] 为新功能添加测试用例
  - [ ] 添加兼容性测试
  - [ ] 添加性能测试
- [ ] 性能优化
  - [ ] 分析性能瓶颈
  - [ ] 优化关键路径
  - [ ] 优化内存使用

## 技术细节

### Token 类型扩展（已完成）

已添加的 token 类型：
- 重定向：`REDIRECT_HEREDOC`, `REDIRECT_HEREDOC_STRIP`, `REDIRECT_HEREDOC_TABS`, `REDIRECT_DUP_IN`, `REDIRECT_DUP_OUT`, `REDIRECT_CLOBBER`, `REDIRECT_RW`
- 操作符：`SEMI_SEMI`, `SEMI_AND`, `SEMI_SEMI_AND`, `BAR_AND`, `AND_GREATER`, `AND_GREATER_GREATER`
- 变量：`PARAM_EXPAND`
- 字符串：`STRING_DOLLAR_SINGLE`, `STRING_DOLLAR_DOUBLE`
- Here-document：`HEREDOC_MARKER`, `HEREDOC_CONTENT`
- 赋值：`ASSIGNMENT_WORD`
- 复合命令：`SUBSHELL_START`, `SUBSHELL_END`, `GROUP_START`, `GROUP_END`

### AST 结构扩展（待实现）

需要添加的 AST 节点：
- `ParamExpandExpression` - 参数展开表达式
- `SubshellCommand` - 子shell 命令 `(command)`
- `GroupCommand` - 命令组 `{ command; }`
- `ConditionalCommand` - 条件命令 `[[ condition ]]`
- `CommandChain` - 命令链（`;`, `&&`, `||`）
- `HereDocument` - Here-document
- `HereString` - Here-string `<<<`
- 更详细的重定向节点（支持所有类型）

### 变量展开优先级

按照 bash 的规范，变量展开的优先级如下：

1. **波浪号展开** (`~`)
2. **参数展开**（变量展开）
   - `${VAR}`, `${VAR:-word}`, `${VAR:=word}`, etc.
3. **命令替换**
   - `` `command` ``, `$(command)`
4. **算术展开**
   - `$((expr))`
5. **路径名展开**（通配符）
   - `*`, `?`, `[...]`
6. **单词分割**（IFS）
7. **引号移除**

### 关键函数设计

#### 词法分析器（部分完成）

- ✅ `NextToken()` - 读取下一个 token（已实现）
- ✅ `readVariable()` - 读取变量（已改进）
- ✅ `readDollarSingleQuote()` - 读取 $'...' 字符串（已实现）
- ✅ `readDollarDoubleQuote()` - 读取 $"..." 字符串（已实现）
- ⏳ `readHereDocument()` - 读取 Here-document（待实现）
- ⏳ `readTokenWord()` - 读取单词 token（待实现）

#### 语法分析器（待实现）

- ⏳ `parseCommand()` - 解析命令
- ⏳ `parseRedirect()` - 解析重定向（需要支持所有类型）
- ⏳ `parseControlFlow()` - 解析控制流
- ⏳ `parseCompoundCommand()` - 解析复合命令
- ⏳ `parseParamExpand()` - 解析参数展开

#### 变量展开系统（待实现）

- ⏳ `expandStringInternal()` - 类似 bash 的 expand_string_internal
- ⏳ `paramExpand()` - 类似 bash 的 param_expand
- ⏳ `expandWord()` - 类似 bash 的 expand_word
- ⏳ `wordSplit()` - 单词分割（IFS）
- ⏳ `pathnameExpand()` - 路径名展开（通配符）
- ⏳ `tildeExpand()` - 波浪号展开

## 实施步骤

### 阶段 1: 词法分析器重构（部分完成）

**已完成：**
1. ✅ 研究 bash 的 read_token 和 read_token_word 实现
2. ✅ 重构 token.go，添加新的 token 类型
3. ✅ 重构 lexer.go，实现新的 token 读取机制
4. ✅ 添加引号处理改进
5. ✅ 添加变量识别改进

**待完成：**
6. ⏳ 添加 Here-document 支持
7. ⏳ 改进空白字符和换行符处理
8. ⏳ 支持多字节字符（UTF-8）
9. ⏳ 改进命令替换嵌套处理
10. ⏳ 改进算术展开嵌套处理
11. ⏳ 测试词法分析器

### 阶段 2: 语法分析器重构（未开始）

1. ⏳ 研究 bash 的 parse.y 语法规则
2. ⏳ 重构 ast.go，改进 AST 结构
3. ⏳ 重构 parser.go，实现新的解析逻辑
4. ⏳ 改进命令解析
5. ⏳ 改进重定向解析
6. ⏳ 改进控制流解析
7. ⏳ 测试语法分析器

### 阶段 3: 变量展开系统重构（未开始）

1. ⏳ 研究 bash 的 subst.c 实现
2. ⏳ 创建新的变量展开模块（subst.go）
3. ⏳ 实现参数展开函数
4. ⏳ 实现算术展开改进
5. ⏳ 实现命令替换改进
6. ⏳ 实现数组访问改进
7. ⏳ 实现单词分割和路径名展开
8. ⏳ 测试变量展开系统

### 阶段 4: 集成和测试（未开始）

1. ⏳ 集成所有重构的模块
2. ⏳ 运行现有测试，确保兼容性
3. ⏳ 修复回归问题
4. ⏳ 添加新功能测试
5. ⏳ 性能优化

## 注意事项

- **保持向后兼容性**：确保现有脚本仍能运行
- **逐步重构**：每个阶段完成后进行测试
- **理解设计意图**：参考 bash 源码时注意理解其设计意图，而不是简单复制
- **平台兼容性**：考虑 Windows 平台的兼容性
- **代码质量**：保持代码可读性和可维护性
- **测试驱动**：每个功能都要有对应的测试

## 预期成果

- ✅ 更符合 bash 行为的词法分析（部分完成）
- ⏳ 更准确的语法解析
- ⏳ 完整的变量展开功能
- ⏳ 更好的错误处理
- ⏳ 更高的 bash 兼容性

## 参考资源

- bash 源码：`bash/parse.y` (语法分析器)
- bash 源码：`bash/subst.c` (变量展开)
- bash 源码：`bash/input.c` (输入处理)
- bash 文档：Bash Reference Manual

## 更新日志

### 2024-12-XX
- ✅ 完成 token 类型扩展
- ✅ 完成重定向和操作符识别改进
- ✅ 完成 $'...' 和 $"..." 支持
- ✅ 完成参数展开识别改进
- ✅ 所有现有测试通过

---

**文档版本**: 1.0  
**最后更新**: 2024-12-XX  
**维护者**: gobash 开发团队

