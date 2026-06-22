# 脚本编辑器语法提示和代码高亮 Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** 为 MockTrue 脚本编辑器补上真实的语法高亮和更可用的 Monaco 代码提示，让脚本节点输入 `input.` / `output.` / `state.` 等内容时能触发上下文补全，并用 MockTrue Script 的受限 API 进行高亮、Hover 和诊断。

**Architecture:** 现有 `ScriptEditor.vue` 已经使用 Monaco 并注册了自定义语言 `mocktrue-script`，但 `scriptLanguage.ts` 只注册 completion/hover，没有 tokenizer 或 language configuration，因此不会有自定义代码高亮；补全也没有 `triggerCharacters` 和上下文过滤，用户输入点号时体验像“没有提示”。实现应集中在 `frontend/src/serial/views/scriptLanguage.ts`，用 Monaco Monarch tokenizer + context-aware completion provider 增强当前语言服务，保持后端 goja runtime 不变。

**Tech Stack:** Vue 3、TypeScript、Vite、Vitest、Monaco Editor `^0.55.1`、Naive UI；验证命令使用 `pnpm test -- --run src/serial/views/scriptLanguage.test.ts src/serial/views/ScriptEditor.test.ts`，必要时跑 `pnpm exec vue-tsc --noEmit` 和 `pnpm run build:dev`。

---

## Current Context / Assumptions

- 当前编辑器组件：`frontend/src/serial/views/ScriptEditor.vue:1-127`
  - 使用 `monaco.editor.create(...)`。
  - `language: scriptLanguageId`，即 `mocktrue-script`。
  - 已开启 `automaticLayout`、关闭 minimap、暗色主题 `vs-dark`。
- 当前语言服务：`frontend/src/serial/views/scriptLanguage.ts:1-388`
  - 已有脚本 API symbol 列表。
  - 已有按节点类型过滤的 completion。
  - 已有 hover 和基础 diagnostics。
  - `registerScriptLanguage()` 目前只调用：
    - `monaco.languages.register({ id: scriptLanguageId })`
    - `registerCompletionItemProvider(...)`
    - `registerHoverProvider(...)`
  - 缺少：
    - `monaco.languages.setMonarchTokensProvider(...)`
    - `monaco.languages.setLanguageConfiguration(...)`
    - completion 的 `triggerCharacters`
    - completion 按当前输入上下文收窄，例如 `input.` 后只提示 `bytes()/hex()/text()`。
- 当前测试：
  - `frontend/src/serial/views/scriptLanguage.test.ts:1-111`
  - `frontend/src/serial/views/ScriptEditor.test.ts:1-143`
- 当前 Vite 配置：`frontend/vite.config.ts:1-19`，暂未看到显式 Monaco worker 配置；本任务先不改 worker，除非构建或浏览器验证发现 worker 报错阻塞 Monaco 语言能力。
- README 已说明技术栈包含 Monaco Editor：`README.md:20-25`。

## Proposed Approach

1. 先用 TDD 增加语言服务单元测试，证明当前缺少语法高亮注册和触发式补全。
2. 扩展 `MonacoLike` 类型，覆盖计划调用的 Monaco API：
   - `languages.setMonarchTokensProvider`
   - `languages.setLanguageConfiguration`
   - completion provider 的 `triggerCharacters`
   - `CompletionItemInsertTextRule.InsertAsSnippet`
3. 在 `registerScriptLanguage()` 中注册 Monarch tokenizer 和语言配置。
4. 让 completion 支持上下文：
   - 空白/普通位置：仍返回全部当前节点可用 API。
   - 输入 `input.`：只返回 `bytes()`、`hex()`、`text("utf-8")`。
   - 输入 `output.`：只返回 `bytes(...)`、`hex(...)`、`text(...)`。
   - 输入 `state.`：只返回 `get(...)`、`set(...)`、`delete(...)`。
   - 对 generator 节点，不返回 `input.*`。
   - 对 analyzer 节点，不返回 `output.*`。
5. 在 `ScriptEditor.vue` 打开 Monaco 辅助提示设置，确保用户输入时能自动触发：
   - `quickSuggestions: { other: true, comments: false, strings: false }`
   - `suggestOnTriggerCharacters: true`
   - `tabCompletion: 'on'`
6. 只触碰前端语言服务和对应测试；不更改后端脚本 runtime。

## Files Likely to Change

- Modify: `frontend/src/serial/views/scriptLanguage.ts`
- Modify: `frontend/src/serial/views/scriptLanguage.test.ts`
- Modify: `frontend/src/serial/views/ScriptEditor.vue`
- Modify: `frontend/src/serial/views/ScriptEditor.test.ts`

## Files Likely NOT to Change

- `internal/modules/serial/script_runtime.go` — 运行时已有 goja API，本任务只增强编辑器体验。
- `frontend/vite.config.ts` — 仅在构建/浏览器验证发现 Monaco worker 报错时再考虑。
- `frontend/package.json` — 已有 `monaco-editor` 依赖，不新增依赖。

---

## Step-by-Step Plan

### Task 1: Add tests proving language registration includes highlighter configuration

**Objective:** 用测试锁定 `mocktrue-script` 必须注册 tokenizer 和 language configuration。

**Files:**
- Modify: `frontend/src/serial/views/scriptLanguage.test.ts:90-110`

**Step 1: Write failing test**

在 `registers Monaco completion and hover providers once` 测试附近添加 Monaco mock 字段，并断言高亮配置被注册一次。

```ts
it('registers Monaco syntax highlighting and language configuration once', () => {
  const monaco = {
    languages: {
      register: vi.fn(),
      registerCompletionItemProvider: vi.fn(),
      registerHoverProvider: vi.fn(),
      setMonarchTokensProvider: vi.fn(),
      setLanguageConfiguration: vi.fn(),
      CompletionItemKind: {
        Function: 1,
        Variable: 2,
        Field: 3,
      },
    },
  }

  registerScriptLanguage(monaco as any)
  registerScriptLanguage(monaco as any)

  expect(monaco.languages.setMonarchTokensProvider).toHaveBeenCalledTimes(1)
  expect(monaco.languages.setMonarchTokensProvider).toHaveBeenCalledWith(
    'mocktrue-script',
    expect.objectContaining({
      tokenizer: expect.objectContaining({
        root: expect.any(Array),
      }),
    })
  )
  expect(monaco.languages.setLanguageConfiguration).toHaveBeenCalledTimes(1)
  expect(monaco.languages.setLanguageConfiguration).toHaveBeenCalledWith(
    'mocktrue-script',
    expect.objectContaining({
      comments: expect.objectContaining({ lineComment: '//' }),
      brackets: expect.arrayContaining([['{', '}'], ['[', ']'], ['(', ')']]),
      autoClosingPairs: expect.arrayContaining([
        expect.objectContaining({ open: '"', close: '"' }),
        expect.objectContaining({ open: '(', close: ')' }),
      ]),
    })
  )
})
```

**Step 2: Run test to verify failure**

Run from repo root:

```bash
cd frontend
pnpm test -- --run src/serial/views/scriptLanguage.test.ts
```

Expected: FAIL because `setMonarchTokensProvider` and `setLanguageConfiguration` are not called.

**Step 3: Commit after task if implementing incrementally**

```bash
git add frontend/src/serial/views/scriptLanguage.test.ts
git commit -m "test: cover script language highlighting registration"
```

---

### Task 2: Implement Monarch tokenizer and language configuration

**Objective:** 注册 MockTrue Script 的基本语法高亮、括号/注释/自动闭合配置。

**Files:**
- Modify: `frontend/src/serial/views/scriptLanguage.ts:33-44`
- Modify: `frontend/src/serial/views/scriptLanguage.ts:260-273`

**Step 1: Extend `MonacoLike` type**

在 `MonacoLike.languages` 中增加可选 API：

```ts
type MonacoLike = {
  MarkerSeverity?: {
    Error?: number
    Warning?: number
  }
  languages: {
    CompletionItemKind?: Record<string, unknown>
    CompletionItemInsertTextRule?: Record<string, unknown>
    register?: (language: { id: string }) => unknown
    registerCompletionItemProvider?: (...args: any[]) => unknown
    registerHoverProvider?: (...args: any[]) => unknown
    setMonarchTokensProvider?: (...args: any[]) => unknown
    setLanguageConfiguration?: (...args: any[]) => unknown
  }
}
```

**Step 2: Add tokenizer constants near `symbols` or below it**

Add a dedicated tokenizer that highlights:
- MockTrue globals / helpers: `input`, `output`, `state`, `field`, `error`, `drop`, `crc16`, `sum8`, `now`
- JS-like keywords: `const`, `let`, `var`, `if`, `else`, `for`, `while`, `return`, `function`, etc.
- numbers, strings, comments, delimiters.

```ts
const scriptKeywords = [
  'break', 'case', 'catch', 'const', 'continue', 'default', 'do', 'else', 'false',
  'finally', 'for', 'function', 'if', 'let', 'new', 'null', 'return', 'switch',
  'throw', 'true', 'try', 'typeof', 'undefined', 'var', 'while',
]

const scriptBuiltins = [
  'input', 'output', 'state', 'field', 'error', 'drop', 'crc16', 'sum8', 'now',
]

const scriptMonarchLanguage = {
  defaultToken: '',
  tokenPostfix: '.mocktrue-script',
  keywords: scriptKeywords,
  builtins: scriptBuiltins,
  tokenizer: {
    root: [
      [/\/\/.*$/, 'comment'],
      [/\/\*/, 'comment', '@comment'],
      [/"([^"\\]|\\.)*$/, 'string.invalid'],
      [/"/, 'string', '@string_double'],
      [/'([^'\\]|\\.)*$/, 'string.invalid'],
      [/'/, 'string', '@string_single'],
      [/`/, 'string', '@string_backtick'],
      [/\b(?:0[xX][0-9a-fA-F]+|\d+(?:\.\d+)?)\b/, 'number'],
      [/[{}()[\]]/, '@brackets'],
      [/[;,.]/, 'delimiter'],
      [/[+\-*\/%=<>!&|?:]+/, 'operator'],
      [/[a-zA-Z_$][\w$]*/, {
        cases: {
          '@keywords': 'keyword',
          '@builtins': 'predefined',
          '@default': 'identifier',
        },
      }],
      [/\s+/, 'white'],
    ],
    comment: [
      [/[^/*]+/, 'comment'],
      [/\*\//, 'comment', '@pop'],
      [/[/*]/, 'comment'],
    ],
    string_double: [
      [/[^\\"]+/, 'string'],
      [/\\./, 'string.escape'],
      [/"/, 'string', '@pop'],
    ],
    string_single: [
      [/[^\\']+/, 'string'],
      [/\\./, 'string.escape'],
      [/'/, 'string', '@pop'],
    ],
    string_backtick: [
      [/[^\\`]+/, 'string'],
      [/\\./, 'string.escape'],
      [/`/, 'string', '@pop'],
    ],
  },
}

const scriptLanguageConfiguration = {
  comments: {
    lineComment: '//',
    blockComment: ['/*', '*/'],
  },
  brackets: [
    ['{', '}'],
    ['[', ']'],
    ['(', ')'],
  ],
  autoClosingPairs: [
    { open: '{', close: '}' },
    { open: '[', close: ']' },
    { open: '(', close: ')' },
    { open: '"', close: '"' },
    { open: "'", close: "'" },
    { open: '`', close: '`' },
  ],
  surroundingPairs: [
    { open: '{', close: '}' },
    { open: '[', close: ']' },
    { open: '(', close: ')' },
    { open: '"', close: '"' },
    { open: "'", close: "'" },
    { open: '`', close: '`' },
  ],
}
```

**Step 3: Register tokenizer/config in `registerScriptLanguage()`**

Insert after `monaco.languages.register?.({ id: scriptLanguageId })`:

```ts
monaco.languages.setMonarchTokensProvider?.(scriptLanguageId, scriptMonarchLanguage)
monaco.languages.setLanguageConfiguration?.(scriptLanguageId, scriptLanguageConfiguration)
```

**Step 4: Run test to verify pass**

```bash
cd frontend
pnpm test -- --run src/serial/views/scriptLanguage.test.ts
```

Expected: PASS for the new highlighter registration test and existing language tests.

**Step 5: Commit**

```bash
git add frontend/src/serial/views/scriptLanguage.ts frontend/src/serial/views/scriptLanguage.test.ts
git commit -m "feat: add mocktrue script syntax highlighting"
```

---

### Task 3: Add tests for dot-triggered, context-aware completion

**Objective:** 补全不只是返回全部 API，而是在 `input.` / `output.` / `state.` 后提示对应成员，并受节点类型限制。

**Files:**
- Modify: `frontend/src/serial/views/scriptLanguage.test.ts`

**Step 1: Write failing test for provider trigger characters**

Update existing `registers Monaco completion and hover providers once` test or add a new one:

```ts
it('registers completion trigger characters for dot member access', () => {
  const monaco = {
    languages: {
      register: vi.fn(),
      registerCompletionItemProvider: vi.fn(),
      registerHoverProvider: vi.fn(),
      setMonarchTokensProvider: vi.fn(),
      setLanguageConfiguration: vi.fn(),
      CompletionItemKind: {
        Function: 1,
        Variable: 2,
        Field: 3,
      },
    },
  }

  registerScriptLanguage(monaco as any)

  expect(monaco.languages.registerCompletionItemProvider).toHaveBeenCalledWith(
    'mocktrue-script',
    expect.objectContaining({
      triggerCharacters: ['.', '('],
      provideCompletionItems: expect.any(Function),
    })
  )
})
```

Expected initially: FAIL because current provider has no `triggerCharacters`.

**Step 2: Write failing test for contextual member suggestions**

Add a test using `completionItemsForModel()`:

```ts
it('narrows completion items for dotted MockTrue API objects', () => {
  const monaco = {
    languages: {
      CompletionItemKind: {
        Function: 1,
        Variable: 2,
        Field: 3,
      },
    },
  }
  const model = {
    uri: { toString: () => 'inmemory://script-transform' },
    getLineContent: () => 'input.',
    getWordAtPosition: () => null,
  }
  registerScriptModel(model.uri.toString(), 'serial.script.transform')

  const labels = completionItemsForModel(monaco as any, model as any, { lineNumber: 1, column: 7 }).map(item => item.label)

  expect(labels).toEqual(['bytes()', 'hex()', 'text(encoding)'])
})
```

Add generator/analyzer restrictions:

```ts
it('does not suggest unavailable dotted APIs for node types', () => {
  const monaco = {
    languages: {
      CompletionItemKind: {
        Function: 1,
        Variable: 2,
        Field: 3,
      },
    },
  }
  const generatorModel = {
    uri: { toString: () => 'inmemory://script-generator' },
    getLineContent: () => 'input.',
    getWordAtPosition: () => null,
  }
  registerScriptModel(generatorModel.uri.toString(), 'serial.script.generator')

  expect(completionItemsForModel(monaco as any, generatorModel as any, { lineNumber: 1, column: 7 })).toEqual([])

  const analyzerModel = {
    uri: { toString: () => 'inmemory://script-analyzer' },
    getLineContent: () => 'output.',
    getWordAtPosition: () => null,
  }
  registerScriptModel(analyzerModel.uri.toString(), 'serial.script.analyzer')

  expect(completionItemsForModel(monaco as any, analyzerModel as any, { lineNumber: 1, column: 8 })).toEqual([])
})
```

**Step 3: Run test to verify failure**

```bash
cd frontend
pnpm test -- --run src/serial/views/scriptLanguage.test.ts
```

Expected: FAIL because `completionItemsForModel()` currently accepts only `(monaco, model)` and always returns full labels for the node type.

**Step 4: Commit tests if implementing incrementally**

```bash
git add frontend/src/serial/views/scriptLanguage.test.ts
git commit -m "test: cover script member completions"
```

---

### Task 4: Implement context-aware completion items

**Objective:** 用户输入点号时出现可用方法提示，且插入文本正确补全方法调用。

**Files:**
- Modify: `frontend/src/serial/views/scriptLanguage.ts:217-284`

**Step 1: Update completion provider signature**

Change the provider to pass Monaco position:

```ts
monaco.languages.registerCompletionItemProvider?.(scriptLanguageId, {
  triggerCharacters: ['.', '('],
  provideCompletionItems: (model: ScriptModelLike, position: { lineNumber: number; column: number }) => ({
    suggestions: completionItemsForModel(monaco, model, position),
  }),
})
```

**Step 2: Update exported function signature**

```ts
export function completionItemsForModel(
  monaco: MonacoLike,
  model: ScriptModelLike,
  position?: { lineNumber: number; column: number }
) {
  return completionItemsForNode(monaco, scriptNodeTypeForModel(model), completionContextForModel(model, position))
}
```

**Step 3: Add completion context types and helpers**

```ts
type CompletionContext =
  | { kind: 'all' }
  | { kind: 'member'; objectName: 'input' | 'output' | 'state' }

function completionContextForModel(
  model: ScriptModelLike,
  position?: { lineNumber: number; column: number }
): CompletionContext {
  if (!position) return { kind: 'all' }
  const line = model.getLineContent(position.lineNumber) ?? ''
  const beforeCursor = line.slice(0, Math.max(0, position.column - 1))
  const memberMatch = /\b(input|output|state)\.$/.exec(beforeCursor)
  if (memberMatch) {
    return { kind: 'member', objectName: memberMatch[1] as 'input' | 'output' | 'state' }
  }
  return { kind: 'all' }
}
```

**Step 4: Add symbol member metadata**

Extend `ScriptSymbol`:

```ts
interface ScriptSymbol {
  label: string
  hoverWord: string
  insertText: string
  detail: string
  documentation: string
  kind: 'function' | 'variable' | 'field'
  memberOf?: 'input' | 'output' | 'state'
  memberLabel?: string
  memberInsertText?: string
  unavailableFor?: ScriptNodeType[]
}
```

Update existing symbol definitions:

```ts
{
  label: 'input.bytes()',
  hoverWord: 'input.bytes',
  insertText: 'input.bytes()',
  memberOf: 'input',
  memberLabel: 'bytes()',
  memberInsertText: 'bytes()',
  // ...keep existing detail/documentation/kind/unavailableFor
}
```

Apply similarly:
- `input.hex()` → `memberLabel: 'hex()'`, `memberInsertText: 'hex()'`
- `input.text(encoding)` → `memberLabel: 'text(encoding)'`, `memberInsertText: 'text("${1:utf-8}")'`
- `output.bytes(bytes)` → `memberLabel: 'bytes(bytes)'`, `memberInsertText: 'bytes($1)'`
- `output.hex(hex)` → `memberLabel: 'hex(hex)'`, `memberInsertText: 'hex("$1")'`
- `output.text(text, encoding)` → `memberLabel: 'text(text, encoding)'`, `memberInsertText: 'text("$1", "${2:utf-8}")'`
- `state.get(key)` → `memberLabel: 'get(key)'`, `memberInsertText: 'get("$1")'`
- `state.set(key, value)` → `memberLabel: 'set(key, value)'`, `memberInsertText: 'set("$1", $2)'`
- `state.delete(key)` → `memberLabel: 'delete(key)'`, `memberInsertText: 'delete("$1")'`

**Step 5: Filter and format completion items**

Change `completionItemsForNode`:

```ts
export function completionItemsForNode(
  monaco: MonacoLike,
  nodeType: string,
  context: CompletionContext = { kind: 'all' }
) {
  const kinds = monaco.languages.CompletionItemKind ?? {}
  const insertTextRules = monaco.languages.CompletionItemInsertTextRule ?? {}
  const symbols = symbolsForNode(nodeType).filter(symbol => {
    if (context.kind === 'all') return true
    return symbol.memberOf === context.objectName
  })

  return symbols.map(symbol => ({
    label: context.kind === 'member' ? (symbol.memberLabel ?? symbol.label) : symbol.label,
    kind: kindForSymbol(symbol, kinds),
    detail: symbol.detail,
    documentation: symbol.documentation,
    insertText: context.kind === 'member' ? (symbol.memberInsertText ?? symbol.insertText) : symbol.insertText,
    insertTextRules: numberKind(insertTextRules.InsertAsSnippet, 4),
  }))
}
```

Note: Keep `numberKind(..., 4)` fallback because Monaco's `InsertAsSnippet` enum value is `4` in common versions; tests should not depend on exact Monaco runtime enum unless mock supplies it.

**Step 6: Run tests**

```bash
cd frontend
pnpm test -- --run src/serial/views/scriptLanguage.test.ts
```

Expected: PASS.

**Step 7: Commit**

```bash
git add frontend/src/serial/views/scriptLanguage.ts frontend/src/serial/views/scriptLanguage.test.ts
git commit -m "feat: add contextual script completions"
```

---

### Task 5: Enable automatic suggestions in ScriptEditor options

**Objective:** Monaco 编辑器在用户输入时主动弹出补全，而不是只能手动 Ctrl+Space。

**Files:**
- Modify: `frontend/src/serial/views/ScriptEditor.vue:33-42`
- Modify: `frontend/src/serial/views/ScriptEditor.test.ts:57-66`

**Step 1: Write failing test**

Update `creates a fixed-layout Monaco editor and syncs script changes` to assert these options:

```ts
expect(monacoApi.api.editor.create).toHaveBeenCalledWith(
  expect.any(HTMLElement),
  expect.objectContaining({
    value: 'output.bytes(input.bytes())',
    language: 'mocktrue-script',
    automaticLayout: true,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    quickSuggestions: { other: true, comments: false, strings: false },
    suggestOnTriggerCharacters: true,
    tabCompletion: 'on',
  })
)
```

**Step 2: Run test to verify failure**

```bash
cd frontend
pnpm test -- --run src/serial/views/ScriptEditor.test.ts
```

Expected: FAIL because these options are not configured yet.

**Step 3: Implement editor options**

In `ScriptEditor.vue`, add options inside `monaco.editor.create(...)`:

```ts
quickSuggestions: { other: true, comments: false, strings: false },
suggestOnTriggerCharacters: true,
tabCompletion: 'on',
```

Final relevant block should look like:

```ts
editor = monaco.editor.create(editorEl.value, {
  value: props.modelValue,
  language: scriptLanguageId,
  theme: 'vs-dark',
  automaticLayout: true,
  minimap: { enabled: false },
  scrollBeyondLastLine: false,
  wordWrap: 'on',
  fontSize: 12,
  quickSuggestions: { other: true, comments: false, strings: false },
  suggestOnTriggerCharacters: true,
  tabCompletion: 'on',
})
```

**Step 4: Run test to verify pass**

```bash
cd frontend
pnpm test -- --run src/serial/views/ScriptEditor.test.ts
```

Expected: PASS.

**Step 5: Commit**

```bash
git add frontend/src/serial/views/ScriptEditor.vue frontend/src/serial/views/ScriptEditor.test.ts
git commit -m "feat: enable script editor suggestions"
```

---

### Task 6: Verify integration with existing graph panel tests

**Objective:** 确认增强 ScriptEditor 后没有破坏脚本节点内容区和脚本配置保存。

**Files:**
- Test only: `frontend/src/serial/views/SerialGraphPanel.test.ts`

**Step 1: Run targeted integration tests**

```bash
cd frontend
pnpm test -- --run src/serial/views/SerialGraphPanel.test.ts
```

Expected: PASS.

**Step 2: If failures mention mocked ScriptEditor**

`SerialGraphPanel.test.ts` currently mocks `ScriptEditor.vue` around lines `20-23` and emits `script-change` in tests around `1108` and `1121`. If failures arise, verify the mock still declares:

```ts
emits: ['script-change', 'update:modelValue']
```

No production code should be changed for this unless the test mock is incompatible with current component emits.

**Step 3: Commit only if test adjustments are required**

```bash
git add frontend/src/serial/views/SerialGraphPanel.test.ts
git commit -m "test: keep graph panel script editor mock current"
```

---

### Task 7: Run frontend validation suite

**Objective:** 验证相关单测、类型检查和 dev build 都通过。

**Files:**
- No code changes unless validation finds a real issue.

**Step 1: Run targeted tests**

```bash
cd frontend
pnpm test -- --run src/serial/views/scriptLanguage.test.ts src/serial/views/ScriptEditor.test.ts src/serial/views/SerialGraphPanel.test.ts
```

Expected: all targeted tests PASS.

**Step 2: Run frontend test suite**

```bash
cd frontend
pnpm test -- --run
```

Expected: all Vitest tests PASS.

**Step 3: Run type check**

```bash
cd frontend
pnpm exec vue-tsc --noEmit
```

Expected: no TypeScript errors.

**Step 4: Run dev build**

```bash
cd frontend
pnpm run build:dev
```

Expected: build completes successfully.

**Step 5: Optional manual verification**

Start the app if needed:

```bash
wails3 dev -config ./build/config.yml -port 9245
```

Manual checks:
- Open a graph script node.
- Confirm comments/strings/numbers/API names are highlighted in the script editor.
- Type `input.` in transform/analyzer node: should suggest `bytes()`, `hex()`, `text(encoding)`.
- Type `input.` in generator node: should not suggest input members.
- Type `output.` in transform/generator node: should suggest `bytes(bytes)`, `hex(hex)`, `text(text, encoding)`.
- Type `output.` in analyzer node: should not suggest output members.
- Type `state.`: should suggest `get(key)`, `set(key, value)`, `delete(key)`.
- Hover `crc16` or `output.bytes`: should show existing docs.
- Existing diagnostics still appear for unsupported APIs.

---

## Risks, Tradeoffs, and Open Questions

### Risks

- **Monaco worker issue:** Syntax highlighting via Monarch normally does not require a separate language worker, but build/runtime may still warn if Monaco worker config is incomplete. Only modify `frontend/vite.config.ts` if actual validation shows a blocking issue.
- **TypeScript strictness:** Current `MonacoLike` uses loose `any[]` signatures. Keep it pragmatic unless `vue-tsc` requires stronger typing.
- **Completion labels:** Member completion labels like `bytes()` are better after `input.`, but full global labels like `input.bytes()` should remain when not in member context.
- **Snippet insertion enum:** If `CompletionItemInsertTextRule.InsertAsSnippet` is unavailable in tests or runtime, fallback value `4` should be used carefully. If this feels brittle, omit `insertTextRules` and rely on Monaco’s default plain insertion; snippet placeholders would then be less useful.

### Tradeoffs

- This plan implements JavaScript-like highlighting tailored to MockTrue Script rather than importing full TypeScript/JavaScript language service. That is simpler and aligned with the app’s restricted API model.
- Diagnostics remain regex-based and shallow. That is acceptable for this feature; full AST parsing is YAGNI unless users report false positives/negatives.
- Context-aware completion handles only simple dotted access immediately before the cursor. It will not fully parse expressions, which is enough for `input.` / `output.` / `state.` prompts.

### Open Questions

- Should MockTrue Script support endpoint-only `serial.*` API in a future phase? Current diagnostics explicitly say “第一阶段不提供 serial API”; this feature should not change that.
- Should editor theme follow the app/system theme? `docs/script-plan/workflow-frontend-monaco.html` mentions theme switching as acceptance, but current `ScriptEditor.vue` hardcodes `vs-dark`. If the current user only asks for syntax提示/高亮, keep theme sync out of scope unless requested.
- Should keywords include all JavaScript built-ins (`Array`, `Math`, etc.)? Only add if runtime exposes/permits them or if users need docs; avoid implying unsupported APIs.

---

## Acceptance Criteria

- `mocktrue-script` registers a Monarch tokenizer and language configuration exactly once.
- Script editor displays code highlighting for comments, strings, numbers, operators, keywords, and MockTrue API names.
- Completion provider triggers on `.` and `(`.
- `input.` suggests only input methods where input is available.
- `output.` suggests only output methods where output is available.
- `state.` suggests state methods for script nodes.
- Existing node-type filtering remains intact:
  - generator has no `input.*` suggestions and keeps diagnostic on `input.*`.
  - analyzer has no `output.*` suggestions and keeps diagnostic on `output.*`.
- Existing script changes still emit `update:modelValue` and `script-change`.
- Targeted tests, full frontend tests, `vue-tsc`, and `build:dev` pass.

## Final Verification Commands

Run from repository root:

```bash
cd frontend
pnpm test -- --run src/serial/views/scriptLanguage.test.ts src/serial/views/ScriptEditor.test.ts src/serial/views/SerialGraphPanel.test.ts
pnpm test -- --run
pnpm exec vue-tsc --noEmit
pnpm run build:dev
```

Expected final state: all commands exit 0.
