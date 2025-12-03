# gobash 词法、语法分析和变量展开完全重构计划

## 项目概述

参考 bash 源码（parse.y, subst.c, input.c），完全重构 gobash 的词法分析、语法分析和变量展开系统，提高与 bash 的兼容性。

## 当前进度

### ✅ 已完成的工作

#### 1. 词法分析器改进（阶段 1 基本完成）

- ✅ **Here-document 支持**
  - 在 lexer 中识别 `<<` 和 `<<-` token
  - 在 AST 中添加 `HereDocument` 结构
  - 在 parser 中解析 Here-document 分隔符（支持带引号和不带引号）
  - 在执行器中实现 `readHereDocument()` 函数
  - 支持带引号的分隔符（不展开变量）
  - 支持制表符剥离（`<<-`）
  - 添加其他重定向类型支持（`<&`, `>&`, `>|`, `<>`, `<<<`）

#### 2. 语法分析器和执行器改进（阶段 1 扩展）

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

#### ✅ 已完成
- [x] 添加 Here-document 支持（<<EOF ... EOF）
  - [x] 实现 Here-document 标记识别（在 lexer 中识别 << 和 <<-）
  - [x] 在 AST 中添加 HereDocument 结构
  - [x] 在 parser 中解析 Here-document 分隔符（支持带引号和不带引号）
  - [x] 在执行器中实现 Here-document 内容读取（readHereDocument）
  - [x] 处理 Here-document 的引号（带引号的标记不展开变量）
  - [x] 处理 Here-document 的制表符剥离（<<-）
  - [x] 添加其他重定向类型支持（<&, >&, >|, <>, <<<）
  - [x] 测试 Here-document 功能（待添加测试用例）

- [x] 改进命令替换和算术展开嵌套处理
  - [x] 改进 readCommandSubstitution 处理嵌套的 $(...)、引号、转义
  - [x] 改进 readCommandSubstitutionParen 处理嵌套的命令替换
  - [x] 改进 readArithmeticExpansion 处理嵌套的 $((...))
  - [x] 添加嵌套处理测试用例

- [x] 语法分析器 AST 扩展
  - [x] 添加 ParamExpandExpression（参数展开表达式）
  - [x] 添加 SubshellCommand（子shell命令）
  - [x] 添加 GroupCommand（命令组）
  - [x] 添加 CommandChain（命令链）

- [x] 语法分析器 Parser 改进
  - [x] 添加对复合命令的解析（subshell, group_command）
  - [x] 添加对命令链的解析（; && ||）
  - [x] 添加对参数展开的解析（${VAR...}）
  - [x] 改进重定向解析（支持所有重定向类型）

- [x] 改进命令替换嵌套处理
  - [x] 正确处理 `command` 和 $(command) 的嵌套
  - [x] 正确处理嵌套中的引号
  - [x] 正确处理嵌套中的转义
  - [x] 添加嵌套处理测试用例

- [x] 改进算术展开嵌套处理
  - [x] 正确处理 $((expr)) 的嵌套括号
  - [x] 正确处理嵌套中的变量展开
  - [x] 添加嵌套处理测试用例

#### ⏳ 待开始
- [x] 改进空白字符和换行符处理
  - [x] 正确处理引号内的空白字符（引号内的空白字符在 readString 中被保留）
  - [x] 正确处理转义的换行符（在 readString 中处理 \n 转义序列）
  - [x] 正确处理多行命令（行尾的反斜杠会忽略换行符）

- [x] 支持多字节字符（UTF-8）
  - [x] 步骤 1: 修改 Lexer 结构体，添加 UTF-8 支持字段
    - [x] 添加 `chRune rune` 字段存储当前字符的 rune 值
    - [x] 添加 `chWidth int` 字段存储当前字符的字节宽度
    - [x] 保留 `ch byte` 字段用于 ASCII 字符的快速比较
  - [x] 步骤 2: 重构 readChar() 函数支持 UTF-8
    - [x] 使用 `utf8.DecodeRuneInString` 读取 UTF-8 字符
    - [x] 正确处理多字节字符的字节位置更新
    - [x] 正确处理行号和列号的更新（多字节字符列号只增加 1）
    - [x] 保持 ASCII 字符的快速路径（ch < 128）
  - [x] 步骤 3: 添加 peekCharRune() 辅助函数
    - [x] 实现查看下一个 rune 但不移动位置的功能
    - [x] 用于多字节字符的预览
  - [x] 步骤 4: 修改 readIdentifier() 支持 UTF-8
    - [x] 使用 `unicode.IsLetter` 和 `unicode.IsDigit` 检查多字节字符
    - [x] 正确处理多字节字符的标识符读取（已修复字符截断问题）
  - [x] 步骤 5: 修改 readString() 支持 UTF-8
    - [x] 正确处理多字节字符的引号匹配
    - [x] 正确处理多字节字符的字符串内容读取
    - [x] 使用 `strings.Builder` 的 `WriteRune` 方法
  - [x] 步骤 6: 修改 NextToken() 支持 UTF-8
    - [x] 处理多字节字符进入 default 分支的逻辑
    - [x] 正确处理多字节字符的变量名识别（已修复字符截断问题）
    - [x] 正确处理多字节字符的标识符识别（已修复字符截断问题）
  - [x] 步骤 7: 修改 readIdentifierOrPath() 支持 UTF-8
    - [x] 正确处理多字节字符的路径读取（已修复字符截断问题）
    - [x] 正确处理多字节字符的分隔符检查
  - [x] 步骤 8: 添加 UTF-8 支持测试用例
    - [x] 测试中文变量名和字符串（所有测试通过）
    - [x] 测试包含中文的引号字符串（已通过）
    - [x] 测试包含日文的变量名（已通过）
    - [x] 测试多字节字符的路径（已通过）
    - [x] 确保所有现有测试仍然通过（所有测试通过）

- [x] 词法分析器测试
  - [x] 添加新功能的单元测试（重定向类型、操作符、$'...'、$"..."、参数展开等）
  - [x] 添加边界情况测试（空输入、空白字符、换行符等）
  - [x] 添加错误处理测试（未闭合引号等）

### 阶段 2: 语法分析器重构（进行中）

#### ✅ 研究阶段
- [x] 研究 bash 的语法分析实现（parse.y 语法规则）
  - [x] 分析 bash 的语法规则结构
  - [x] 理解 bash 的 AST 节点类型
  - [x] 理解 bash 的解析优先级
  - [x] 理解 bash 的错误处理机制

#### ✅ AST 结构重构
- [x] 重构 ast.go，改进 AST 结构
  - [x] 添加 ParamExpandExpression（参数展开表达式）
  - [x] 添加 SubshellCommand（子shell命令）
  - [x] 添加 GroupCommand（命令组）
  - [x] 添加 CommandChain（命令链）
  - [x] 添加更详细的重定向节点（支持所有重定向类型）
  - [x] 添加 Here-document 节点

#### 🔄 解析器重构（进行中）
- [x] 重构 parser.go，实现新的解析逻辑
  - [x] 实现命令解析改进
    - [x] 支持简单命令
    - [x] 支持管道命令
    - [x] 支持命令链（`;`, `&&`, `||`）
    - [x] 支持后台命令（`&`）
  - [x] 实现重定向解析改进
    - [x] 支持所有重定向类型（>, <, >>, <<, <&, >&, >|, <>, etc.）
    - [x] 支持文件描述符重定向（2>, 1>, etc.）
    - [x] 支持 Here-document
    - [x] 支持 Here-string（<<<）
  - [x] 实现控制流解析改进
    - [x] if/else/elif/fi
    - [x] for/in/do/done
    - [x] while/do/done
    - [x] case/in/esac
    - [x] function 定义
  - [x] 实现复合命令解析
    - [x] 子shell `(command)`
    - [x] 命令组 `{ command; }`
    - [x] 条件命令 `[[ condition ]]`
  - [x] 改进数组和关联数组语法解析
    - [x] 支持带索引的数组赋值语法 `arr=([0]=a [1]=b [2]=c)`
    - [x] 支持不连续索引的数组赋值 `arr=([0]=a [2]=c)`
    - [x] 支持字符串键的关联数组赋值 `arr=([key1]=val1 [key2]=val2)`
    - [x] 更新 AST 结构以支持索引数组赋值
    - [x] 更新执行器以正确处理索引数组赋值
  - [x] 改进错误处理和错误报告（部分完成）
    - [x] 改进语法分析器错误处理
      - [x] 创建结构化错误类型（ParseError, ErrorType）
      - [x] 添加详细的错误位置信息（token 位置，行号、列号）
      - [x] 添加错误类型分类（未闭合括号、未闭合大括号、未闭合控制流等）
      - [x] 在关键解析函数中添加错误检测（parseSubshell, parseGroupCommand, parseIfStatement, parseForStatement, parseWhileStatement, parseCaseStatement）
      - [ ] 添加错误恢复机制（跳过错误继续解析）
      - [ ] 改进错误消息格式（参考 bash 的错误格式）
    - [ ] 改进词法分析器错误处理
      - [ ] 添加详细的错误位置信息（行号、列号）
      - [ ] 添加错误类型分类（语法错误、字符错误等）
      - [ ] 改进错误消息的可读性
    - [ ] 改进执行器错误处理
      - [ ] 统一错误类型和错误消息格式
      - [ ] 添加错误上下文信息（命令、参数等）
      - [ ] 改进错误传播机制
      - [ ] 添加错误代码（退出码）
    - [ ] 改进 Shell 层的错误报告
      - [ ] 统一错误输出格式（参考 bash）
      - [ ] 添加文件名和行号信息
      - [ ] 改进交互式和非交互式模式的错误显示
  - [x] 支持多行语句的正确解析（基本完成）
    - [x] 改进 lexer 对行尾反斜杠的处理
      - [x] 确保行尾反斜杠正确忽略换行符
      - [x] 处理反斜杠后的空白字符
      - [x] 处理反斜杠后的注释
    - [x] 改进 parser 对多行语句的解析
      - [x] 正确处理换行符作为语句分隔符（在 ParseProgram 和 parseBlockStatement 中）
      - [x] 正确处理多行控制流语句（if/else/fi, for/do/done, while/do/done, case/esac）
      - [x] 正确处理多行命令（管道、重定向等）
    - [x] 改进 shell 层的多行语句处理
      - [x] isStatementComplete 函数已实现，能够检测未完成的控制流语句
      - [x] 正确处理脚本模式下的多行语句（ExecuteReader 函数）
      - [ ] 改进交互式模式下的多行输入提示
    - [ ] 添加多行语句的测试用例

#### 语法分析器测试
- [x] 添加新功能的单元测试（复合命令、命令链、case语句、while语句、参数展开、新重定向类型等）
- [x] 添加边界情况测试（空输入、空白字符、单个命令、嵌套结构等）
- [x] 添加错误处理测试（未闭合引号、未闭合括号、未闭合控制流等）
- [x] 运行现有测试，确保兼容性（所有现有测试通过）

### 阶段 3: 变量展开系统重构（进行中）

#### ✅ 研究阶段
- [x] 研究 bash 的变量展开实现（subst.c 中的 expand_string_internal, param_expand）
  - [x] 分析 expand_string_internal 的实现
  - [x] 分析 param_expand 的实现
  - [x] 理解变量展开的优先级
  - [x] 理解单词分割（IFS）的实现（已实现 wordSplit 函数）
  - [x] 理解路径名展开（通配符）的实现（已实现 pathnameExpand 函数）

#### ✅ 创建变量展开模块
- [x] 创建新的变量展开模块（`internal/executor/subst.go`）
  - [x] 定义展开上下文结构（ExpandContext）
  - [x] 定义展开标志（ExpandFlags）
  - [x] 定义展开结果类型

#### 🔄 实现参数展开函数（进行中）
- [x] 实现基本的参数展开：
  - [x] `${VAR:-word}` - 使用默认值
  - [x] `${VAR:=word}` - 赋值默认值
  - [x] `${VAR:?word}` - 显示错误
  - [x] `${VAR:+word}` - 使用替代值
  - [x] `${VAR#pattern}` - 删除最短匹配前缀
  - [x] `${VAR##pattern}` - 删除最长匹配前缀
  - [x] `${VAR%pattern}` - 删除最短匹配后缀
  - [x] `${VAR%%pattern}` - 删除最长匹配后缀
  - [x] `${VAR:offset}` - 子字符串
  - [x] `${VAR:offset:length}` - 子字符串
  - [x] `${#VAR}` - 字符串长度
  - [x] `${!VAR}` - 间接引用
  - [x] `${VAR[@]}` - 数组展开（带引号时每个元素单独展开，已通过 expandArray 实现）
  - [x] `${VAR[*]}` - 数组展开（所有元素作为一个单词，已通过 expandArray 实现）
  - [x] `${VAR[expr]}` - 数组/关联数组访问（已通过 getArrayElement 实现）

#### 实现其他展开功能
- [x] 改进算术展开
  - [x] 支持完整的算术表达式（已重构为完整的递归下降解析器）
  - [x] 支持所有算术运算符（+, -, *, /, %, **, <<, >>, &, |, ^, ~, <, <=, >, >=, ==, !=, &&, ||, !）
  - [x] 支持算术函数（基本完成）
    - [x] 实现基本算术函数
      - [x] `abs(x)` - 绝对值
      - [x] `min(x, y, ...)` - 最小值
      - [x] `max(x, y, ...)` - 最大值
      - [x] `length(x)` - 数字的字符串长度（简化实现）
      - [x] `int(x)` - 取整（对于整数，直接返回）
      - [x] `rand()` - 随机数（0-32767）
      - [x] `srand([seed])` - 设置随机数种子
      - [ ] `substr(s, start, length)` - 子字符串（需要字符串支持）
      - [ ] `index(s, t)` - 查找子字符串位置（需要字符串支持）
    - [x] 改进 parseArithmeticFactor 函数以支持函数调用
    - [x] 添加函数参数解析（支持多个参数）
    - [ ] 添加算术函数测试用例
  - [x] 正确处理变量展开（在 evaluateArithmetic 中已处理）
- [x] 改进命令替换
  - [x] 正确处理嵌套（在 expandCommandSubstitutionCommand 中展开嵌套的命令替换）
  - [x] 正确处理转义（在 expandVariablesInString 中已处理）
  - [x] 正确处理退出码（添加了 getExitCode 函数，命令替换在子shell中执行）
- [x] 改进数组访问
  - [x] `${arr[0]}` - 普通数组（通过 getArrayElement 实现）
  - [x] `${arr[key]}` - 关联数组（通过 getArrayElement 实现）
  - [x] `${arr[@]}` - 数组展开（通过 expandArray 实现，每个元素作为单独的词）
  - [x] `${arr[*]}` - 数组展开（通过 expandArray 实现，所有元素作为一个词，使用 IFS 分隔）
- [x] 实现单词分割（IFS）
  - [x] 根据 IFS 分割单词（实现 wordSplit 函数）
  - [x] 正确处理 IFS 为空的情况（每个字符都是独立的单词）
  - [x] 正确处理 IFS 为默认值的情况（压缩连续的空白字符）
- [x] 实现路径名展开（通配符）
  - [x] 支持 `*` 通配符（通过 filepath.Glob 实现）
  - [x] 支持 `?` 通配符（通过 filepath.Glob 实现）
  - [x] 支持 `[...]` 字符类（通过 filepath.Glob 实现，支持 `[!...]` 和 `[^...]` 否定字符类）
  - [x] 支持 `**` 递归匹配（如果启用）
    - [x] 实现 pathnameExpandRecursive 函数处理 ** 模式
    - [x] 实现 matchRecursive 函数递归匹配目录
    - [x] 支持 globstar 选项（通过环境变量 GLOBSTAR 或 options["globstar"]）
    - [x] 支持各种 ** 模式：
      - `**` - 匹配当前目录及其所有子目录
      - `**/pattern` - 匹配所有目录中的 pattern
      - `pattern/**` - 匹配 pattern 目录及其所有子目录
      - `prefix/**/suffix` - 匹配 prefix 目录下任意深度的 suffix
    - [ ] 添加 ** 递归匹配的测试用例
  - [x] 正确处理隐藏文件（如果模式不以 `.` 开头，不匹配隐藏文件）
- [x] 实现波浪号展开（~）
  - [x] `~` - 当前用户主目录（通过 HOME 或 USERPROFILE 环境变量）
  - [x] `~user` - 指定用户主目录（基本实现，支持当前用户）
  - [x] `~+` - 当前工作目录（通过 PWD 环境变量或 os.Getwd()）
  - [x] `~-` - 上一个工作目录（通过 OLDPWD 环境变量）

#### 变量展开系统测试
- [x] 添加新功能的单元测试（单词分割、路径名展开、波浪号展开、数组展开等）
- [x] 添加边界情况测试（空 IFS、默认 IFS、无匹配通配符等）
- [x] 添加错误处理测试（未设置环境变量、无效模式等）
- [x] 运行现有测试，确保兼容性（所有现有测试通过）

### 阶段 4: 集成和测试（未开始）

- [x] 集成所有重构的模块
  - [x] 确保词法分析器与语法分析器兼容
    - [x] 创建集成测试文件 internal/integration_test.go
    - [x] 运行 lexer 和 parser 的集成测试（TestLexerParserIntegration）
    - [x] 验证所有新 token 类型被正确解析
    - [x] 验证 UTF-8 支持在解析器中正常工作
    - [x] 验证 Here-document、条件命令、数组赋值等新功能
  - [x] 确保语法分析器与执行器兼容
    - [x] 运行 parser 和 executor 的集成测试（TestParserExecutorIntegration）
    - [x] 验证所有新 AST 节点被正确执行
    - [x] 验证变量赋值、数组赋值等基本功能
  - [x] 确保变量展开系统与执行器兼容
    - [x] 运行变量展开的集成测试（TestVariableExpansionIntegration）
    - [x] 验证所有展开类型（参数展开、算术展开、命令替换等）正常工作
  - [x] 运行端到端测试
    - [x] 测试完整的命令执行流程（TestEndToEndIntegration）
    - [x] 验证所有重构功能在集成后正常工作
    - [x] 验证简单命令、变量赋值、if 语句、for 循环、多行语句等
- [x] 运行现有测试，确保兼容性
  - [x] 运行所有单元测试（词法分析器、语法分析器、执行器测试通过）
  - [ ] 运行集成测试（待添加）
  - [ ] 运行脚本测试（待添加）
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

### 2024-12-03（最新）
- ✅ 完成 Here-document 功能测试用例添加
  - 添加了 lexer、parser 和 executor 的完整测试用例
  - 所有测试通过
- ✅ 完成 UTF-8 多字节字符支持
  - 完成了所有 8 个详细步骤
  - 修复了字符截断问题（在 readIdentifier、readVariable 和 readIdentifierOrPath 中）
  - 所有 UTF-8 测试通过（7/7）
  - 确保所有现有测试仍然通过
- ✅ 完成条件命令 `[[ condition ]]` 功能验证
  - 功能已完整实现并正常工作
  - 支持 &&、||、! 运算符和括号表达式
  - 添加了测试用例
  - 验证了在 if 语句中的使用

### 2024-12-XX
- ✅ 完成单词分割（IFS）功能实现
- ✅ 完成路径名展开（通配符）功能实现
- ✅ 完成波浪号展开（~）功能实现
- ✅ 完成变量展开系统测试
- ✅ 修复 parser_test_additional.go 中的编译错误
- 📋 分解"支持多字节字符（UTF-8）"任务为 8 个详细步骤

### 2024-12-XX
- ✅ 完成 token 类型扩展
- ✅ 完成重定向和操作符识别改进
- ✅ 完成 $'...' 和 $"..." 支持
- ✅ 完成参数展开识别改进
- ✅ 完成 Here-document 基本支持
  - 在 AST 中添加 HereDocument 结构
  - 在 parser 中解析 Here-document
  - 在执行器中实现内容读取
  - 支持带引号分隔符和制表符剥离
- ✅ 添加其他重定向类型支持（<&, >&, >|, <>, <<<）
- ✅ 改进命令替换嵌套处理（正确处理嵌套的 $(...)、引号、转义等）
- ✅ 改进算术展开嵌套处理（正确处理嵌套的 $((...))、引号等）
- ✅ 添加嵌套处理测试用例
- ✅ 完成语法分析器 AST 扩展
  - 添加 ParamExpandExpression（参数展开表达式）
  - 添加 SubshellCommand（子shell命令）
  - 添加 GroupCommand（命令组）
  - 添加 CommandChain（命令链）
- ✅ 完成语法分析器 Parser 改进
  - 添加对复合命令的解析（subshell, group_command）
  - 添加对命令链的解析（; && ||）
  - 添加对参数展开的解析（${VAR...}）
  - 改进重定向解析（支持所有重定向类型）
- ✅ 改进控制流解析
  - 改进 case 语句解析（支持 SEMI_SEMI, SEMI_AND, SEMI_SEMI_AND）
  - 所有控制流语句（if/else, for, while, case, function）已实现
- ✅ 开始变量展开系统重构
  - 研究 bash 的变量展开实现
  - 创建变量展开模块（subst.go）
  - 实现基本参数展开功能（${VAR:-word}, ${VAR:=word}, ${VAR#pattern} 等）
  - 集成到执行器
- ✅ 改进空白字符和换行符处理
  - 正确处理引号内的空白字符
  - 正确处理转义的换行符（\n 转义序列）
  - 正确处理多行命令（行尾的反斜杠会忽略换行符）
- ✅ 所有现有测试通过

---

**文档版本**: 1.0  
**最后更新**: 2024-12-XX  
**维护者**: gobash 开发团队

