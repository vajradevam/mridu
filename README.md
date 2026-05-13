# mridu

This is my first go at building a programming language — a bytecode VM interpreter written in Go. It's not complete by any stretch; consider it a work-in-progress proof of concept. I'll keep adding stuff over time.

## What it can do (so far)

- Dynamic typing (bool, nil, number, string)
- Functions, closures, and upvalues
- Classes with single inheritance and `super` calls
- Control flow: `if`, `while`, `for`
- Native functions (`clock`)
- Direct-to-bytecode Pratt parser → stack-based VM
- REPL and script execution

## What's missing (for now)

- Arrays, lists, maps
- Standard library beyond `clock`
- `break` / `continue`
- Garbage collection

## Project Structure

```
mridu-go/
├── cmd/mridu/main.go    # Binary entry point (REPL + file runner)
├── lang/                # Library package — scanner, compiler, VM, runtime
│   ├── scanner.go       # Lexer
│   ├── compiler.go      # Pratt parser → bytecode
│   ├── vm.go            # Stack-based interpreter
│   ├── chunk.go         # Bytecode chunk
│   ├── value.go         # Value and object types
│   ├── mridu_test.go          # Integration and AB tests
│   └── mridu_exhaustive_test.go  # Exhaustive tests (~500 cases)
├── programs/            # Sample .mridu programs
├── docs/                # Grammar and manual
├── Makefile             # Build, test, run targets
└── go.mod
```

## Usage

### Build

```sh
make build
```

### Run

```sh
./mridu programs/hello.mridu
# or REPL:
./mridu
```

### Test

```sh
make test          # all tests
make test-race     # with race detector
make test-short    # smoke tests only
```

## How the compiler works

It's a **single-pass Pratt parser** that spits out bytecode as it goes — no AST sitting around in memory.

### Pipeline

```
Source → Scanner (tokens) → Compiler (bytecode) → VM (execution)
```

### Scanner (`scanner.go`)

Hand-written lexer. Walks the source byte by byte, produces tokens. Handles line comments (`//`), nested block comments (`/* /* */ */`), and recognizes keywords from a lookup table.

### Compiler (`compiler.go`)

Parsing is driven by a **Precedence Climbing** (Pratt) algorithm with prefix/infix parse functions registered in a rule table:

| Token | Prefix | Infix | Precedence |
|-------|--------|-------|------------|
| Number | literal constant | — | — |
| String | string constant | — | — |
| Identifier | variable ref | — | — |
| `(` | grouping | call | Call |
| `.` | — | property access | Call |
| `-` | negate | subtract | Term |
| `+` | — | add | Term |
| `*``/` | — | multiply/divide | Factor |
| `==``!=` | — | equality | Equality |
| `<``<=``>``>=` | — | comparison | Comparison |
| `and` | — | short-circuit and | And |
| `or` | — | short-circuit or | Or |
| `!` | not | — | — |

Key details:
- **No AST**: each parse function calls `emitOp()` to write bytecode immediately.
- **Recursive descent** for declarations and statements.
- **Functions**: compiled into separate `ObjFunction` objects, then stitched together with `OP_CLOSURE` + upvalue metadata.
- **Upvalues**: the compiler walks the enclosing compiler chain to resolve captured variables. Each upvalue knows whether it wraps a local or another upvalue.
- **Error recovery**: panic-mode — skips to the next `;` or statement keyword.
- **Constants** live in a per-function constant pool (16-bit index). Jump offsets are 16-bit too.

### Bytecode

42 opcodes packed into flat `[]byte` chunks:

| Opcode | Operands | Stack Effect | Purpose |
|--------|----------|-------------|---------|
| `OP_CONSTANT` | u16 index | push constant | Load literal |
| `OP_NIL`/`OP_TRUE`/`OP_FALSE` | — | push value | Load built-in |
| `OP_POP` | — | pop | Discard value |
| `OP_DUP` | — | duplicate top | Duplicate stack top |
| `OP_GET_LOCAL`/`OP_SET_LOCAL` | u8 index | push/pop | Local variable access |
| `OP_GET_GLOBAL`/`OP_SET_GLOBAL`/`OP_DEFINE_GLOBAL` | u16 name index | push/pop | Global variable access |
| `OP_GET_UPVALUE`/`OP_SET_UPVALUE` | u8 index | push/pop | Closure capture access |
| `OP_CLOSE_UPVALUE` | — | pop | Close captured local |
| `OP_EQUAL`/`OP_GREATER`/`OP_LESS` | — | push bool | Comparison |
| `OP_ADD`/`OP_SUBTRACT`/`OP_MULTIPLY`/`OP_DIVIDE` | — | push result | Arithmetic |
| `OP_NOT`/`OP_NEGATE` | — | push result | Unary |
| `OP_PRINT` | — | pop | Output |
| `OP_JUMP` | u16 offset | — | Unconditional jump |
| `OP_JUMP_IF_FALSE` | u16 offset | — | Conditional jump |
| `OP_LOOP` | u16 offset | — | Backward jump |
| `OP_CALL` | u8 arg count | args→result | Function call |
| `OP_CLOSURE` | u16 fn index + upvalue data | push closure | Create closure |
| `OP_RETURN` | — | pop | Return from call |
| `OP_CLASS` | u16 name index | push class | Create class |
| `OP_GET_PROPERTY`/`OP_SET_PROPERTY` | u16 name index | push/pop | Instance field access |
| `OP_METHOD` | u16 name index | — | Bind method to class |
| `OP_INVOKE` | u16 name + u8 args | args→result | Method call (fast path) |
| `OP_INHERIT` | — | pop subclass | Set up inheritance |
| `OP_GET_SUPER` | u16 name index | push method | Superclass method access |
| `OP_SUPER_INVOKE` | u16 name + u8 args | args→result | Superclass method call |

### VM (`vm.go`)

Stack-based interpreter, max 64 call frames, max 256 stack slots. The main loop reads opcodes, dispatches through a `switch`, and pushes/pops values. Open upvalues are tracked as a linked list and closed when their scope exits.

## Example

```
fun fib(n) {
  if (n <= 1) return n;
  return fib(n - 1) + fib(n - 2);
}

print fib(10);   // 55
```

## Also check out

- `docs/grammar.md` — formal BNF grammar
- `docs/manual.md` — language reference manual

## License

MIT
