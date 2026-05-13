package lang

import (
	"fmt"
	"os"
)

type Precedence int

const (
	PREC_NONE Precedence = iota
	PREC_ASSIGNMENT
	PREC_OR
	PREC_AND
	PREC_EQUALITY
	PREC_COMPARISON
	PREC_TERM
	PREC_FACTOR
	PREC_UNARY
	PREC_CALL
	PREC_PRIMARY
)

type ParseFn func(bool)
type ParseRule struct {
	prefix ParseFn
	infix  ParseFn
	prec   Precedence
}

type Local struct {
	name       string
	depth      int
	isCaptured bool
}

type Upvalue struct {
	index   int
	isLocal bool
}

type FuncType int

const (
	TYPE_FUNCTION FuncType = iota
	TYPE_INITIALIZER
	TYPE_METHOD
	TYPE_SCRIPT
)

type Compiler struct {
	enclosing  *Compiler
	function   *Object
	funcType   FuncType
	locals     []Local
	scopeDepth int
	upvalues   []Upvalue

	scanner   *Scanner
	current   Token
	previous  Token
	hadError  bool
	panicMode bool
}

type ClassCompiler struct {
	enclosing     *ClassCompiler
	hasSuperclass bool
}

var current *Compiler
var currentClass *ClassCompiler

func compile(source string) *Object {
	if rules == nil {
		initRules()
	}
	scanner := NewScanner(source)
	compiler := &Compiler{
		scanner:    scanner,
		funcType:   TYPE_SCRIPT,
		scopeDepth: 0,
	}
	compiler.function = NewObjFunction("")
	compiler.locals = make([]Local, 1)
	compiler.locals[0] = Local{name: "", depth: 0}

	current = compiler
	currentClass = nil

	advance()
	for !match(TOKEN_EOF) {
		declaration()
	}

	function := current.function
	if current.hadError {
		return nil
	}
	emitReturn()
	return function
}

// ---- parsing helpers ----

func advance() {
	current.previous = current.current
	for {
		current.current = current.scanner.ScanToken()
		if current.current.Type == TOKEN_ERROR {
			errorAtCurrent(current.current.Lexeme)
		}
		if current.current.Type != TOKEN_ERROR {
			break
		}
	}
}

func consume(typ TokenType, msg string) {
	if current.current.Type == typ {
		advance()
		return
	}
	errorAtCurrent(msg)
}

func match(typ TokenType) bool {
	if !check(typ) {
		return false
	}
	advance()
	return true
}

func check(typ TokenType) bool {
	return current.current.Type == typ
}

// ---- error reporting ----

func errorAtCurrent(msg string) {
	errorAtMsg(msg)
}

func errorAtMsg(msg string) {
	current.hadError = true
	current.panicMode = true
	fmt.Fprintf(os.Stderr, "[line %d] Error: %s\n", current.previous.Line, msg)
}

func synchronize() {
	current.panicMode = false
	for current.current.Type != TOKEN_EOF {
		if current.previous.Type == TOKEN_SEMICOLON {
			return
		}
		switch current.current.Type {
		case TOKEN_CLASS, TOKEN_FUN, TOKEN_VAR, TOKEN_FOR, TOKEN_IF, TOKEN_WHILE, TOKEN_PRINT, TOKEN_RETURN:
			return
		}
		advance()
	}
}

// ---- parsing ----

func declaration() {
	if match(TOKEN_CLASS) {
		classDecl()
	} else if match(TOKEN_FUN) {
		funDecl()
	} else if match(TOKEN_VAR) {
		varDecl()
	} else {
		statement()
	}
	if current.panicMode && current.hadError {
		synchronize()
	}
}

func classDecl() {
	consume(TOKEN_IDENTIFIER, "Expect class name.")
	className := current.previous.Lexeme
	nameConstant := identifierConstant(current.previous)
	declareVariable()
	emitOp(OP_CLASS)
	emitShort(nameConstant)
	defineVariable(nameConstant)

	classCompiler := &ClassCompiler{enclosing: currentClass, hasSuperclass: false}
	currentClass = classCompiler

	if match(TOKEN_LESS) {
		consume(TOKEN_IDENTIFIER, "Expect superclass name.")
		variable(false)
		if current.previous.Lexeme == className {
			errorAtMsg("A class cannot inherit from itself.")
		}
		beginScope()
		addLocal("super")
		defineVariable(0)
		namedVariable(className, false)
		emitOp(OP_INHERIT)
		classCompiler.hasSuperclass = true
	}

	namedVariable(className, false)
	consume(TOKEN_LEFT_BRACE, "Expect '{' before class body.")

	for !check(TOKEN_RIGHT_BRACE) && !check(TOKEN_EOF) {
		method()
	}
	consume(TOKEN_RIGHT_BRACE, "Expect '}' after class body.")
	emitOp(OP_POP)

	if classCompiler.hasSuperclass {
		endScope()
	}

	currentClass = currentClass.enclosing
}

func funDecl() {
	global := parseVariable("Expect function name.")
	markInitialized()
	function_(TYPE_FUNCTION)
	defineVariable(global)
}

func function_(fnType FuncType) {
	compiler := &Compiler{
		enclosing:  current,
		scanner:    current.scanner,
		funcType:   fnType,
		scopeDepth: 0,
	}
	compiler.function = NewObjFunction(current.previous.Lexeme)
	compiler.locals = make([]Local, 1)
	compiler.locals[0] = Local{name: "", depth: 0}

	if fnType != TYPE_FUNCTION {
		compiler.locals[0] = Local{name: "this", depth: 0}
	}

	prev := current
	current = compiler

	beginScope()
	consume(TOKEN_LEFT_PAREN, "Expect '(' after function name.")
	if !check(TOKEN_RIGHT_PAREN) {
		for {
			current.function.funcArity++
			if current.function.funcArity > 255 {
				errorAtMsg("Can't have more than 255 parameters.")
			}
			paramConstant := parseVariable("Expect parameter name.")
			defineVariable(paramConstant)
			if !match(TOKEN_COMMA) {
				break
			}
		}
	}
	consume(TOKEN_RIGHT_PAREN, "Expect ')' after parameters.")
	consume(TOKEN_LEFT_BRACE, "Expect '{' before function body.")
	block()
	emitReturn()

	compilerFn := current.function
	upvalueCount := len(current.upvalues)
	compilerFn.funcUpvalues = upvalueCount

	lastToken := current.current
	current = prev
	current.current = lastToken
	emitOp(OP_CLOSURE)
	emitShort(makeConstant(ObjVal(compilerFn)))

	for i := 0; i < upvalueCount; i++ {
		uv := compiler.upvalues[i]
		var isLocal byte = 0
		if uv.isLocal {
			isLocal = 1
		}
		emitBytes(isLocal, byte(uv.index))
	}
}

func varDecl() {
	global := parseVariable("Expect variable name.")
	if match(TOKEN_EQUAL) {
		expression()
	} else {
		emitOp(OP_NIL)
	}
	consume(TOKEN_SEMICOLON, "Expect ';' after variable declaration.")
	defineVariable(global)
}

func statement() {
	if match(TOKEN_PRINT) {
		printStatement()
	} else if match(TOKEN_FOR) {
		forStatement()
	} else if match(TOKEN_IF) {
		ifStatement()
	} else if match(TOKEN_RETURN) {
		returnStatement()
	} else if match(TOKEN_WHILE) {
		whileStatement()
	} else if match(TOKEN_LEFT_BRACE) {
		beginScope()
		block()
		endScope()
	} else {
		expressionStatement()
	}
}

func printStatement() {
	expression()
	consume(TOKEN_SEMICOLON, "Expect ';' after value.")
	emitOp(OP_PRINT)
}

func forStatement() {
	beginScope()
	consume(TOKEN_LEFT_PAREN, "Expect '(' after 'for'.")
	if match(TOKEN_SEMICOLON) {
		// no initializer
	} else if match(TOKEN_VAR) {
		varDecl()
	} else {
		expressionStatement()
	}

	loopStart := len(current.function.funcChunk.code)
	exitJump := -1
	if !match(TOKEN_SEMICOLON) {
		expression()
		consume(TOKEN_SEMICOLON, "Expect ';' after loop condition.")
		exitJump = emitJump(OP_JUMP_IF_FALSE)
		emitOp(OP_POP)
	}

	if !match(TOKEN_RIGHT_PAREN) {
		bodyJump := emitJump(OP_JUMP)
		incrementStart := len(current.function.funcChunk.code)
		expression()
		emitOp(OP_POP)
		consume(TOKEN_RIGHT_PAREN, "Expect ')' after for clauses.")
		emitLoop(loopStart)
		loopStart = incrementStart
		patchJump(bodyJump)
	}

	statement()
	emitLoop(loopStart)

	if exitJump != -1 {
		patchJump(exitJump)
		emitOp(OP_POP)
	}
	endScope()
}

func ifStatement() {
	consume(TOKEN_LEFT_PAREN, "Expect '(' after 'if'.")
	expression()
	consume(TOKEN_RIGHT_PAREN, "Expect ')' after condition.")

	thenJump := emitJump(OP_JUMP_IF_FALSE)
	emitOp(OP_POP)
	statement()

	elseJump := emitJump(OP_JUMP)
	patchJump(thenJump)
	emitOp(OP_POP)

	if match(TOKEN_ELSE) {
		statement()
	}
	patchJump(elseJump)
}

func returnStatement() {
	if current.funcType == TYPE_SCRIPT {
		errorAtMsg("Cannot return from top-level code.")
	}
	if match(TOKEN_SEMICOLON) {
		emitReturn()
	} else {
		if current.funcType == TYPE_INITIALIZER {
			errorAtMsg("Cannot return a value from an initializer.")
		}
		expression()
		consume(TOKEN_SEMICOLON, "Expect ';' after return value.")
		emitOp(OP_RETURN)
	}
}

func whileStatement() {
	loopStart := len(current.function.funcChunk.code)
	consume(TOKEN_LEFT_PAREN, "Expect '(' after 'while'.")
	expression()
	consume(TOKEN_RIGHT_PAREN, "Expect ')' after condition.")

	exitJump := emitJump(OP_JUMP_IF_FALSE)
	emitOp(OP_POP)
	statement()
	emitLoop(loopStart)

	patchJump(exitJump)
	emitOp(OP_POP)
}

func expressionStatement() {
	expression()
	consume(TOKEN_SEMICOLON, "Expect ';' after expression.")
	emitOp(OP_POP)
}

func block() {
	for !check(TOKEN_RIGHT_BRACE) && !check(TOKEN_EOF) {
		declaration()
	}
	consume(TOKEN_RIGHT_BRACE, "Expect '}' after block.")
}

func method() {
	consume(TOKEN_IDENTIFIER, "Expect method name.")
	nameConstant := identifierConstant(current.previous)

	fnType := TYPE_METHOD
	if current.previous.Lexeme == "init" {
		fnType = TYPE_INITIALIZER
	}
	function_(fnType)

	emitOp(OP_METHOD)
	emitShort(nameConstant)
}

// ---- expression parsing ----

func expression() {
	parsePrecedence(PREC_ASSIGNMENT)
}

func parsePrecedence(prec Precedence) {
	advance()
	prefixRule := getRule(current.previous.Type).prefix
	if prefixRule == nil {
		errorAtMsg("Expect expression.")
		return
	}
	canAssign := prec <= PREC_ASSIGNMENT
	prefixRule(canAssign)

	for prec <= getRule(current.current.Type).prec {
		advance()
		infixRule := getRule(current.previous.Type).infix
		if infixRule == nil {
			break
		}
		infixRule(canAssign)
	}

	if canAssign && match(TOKEN_EQUAL) {
		errorAtMsg("Invalid assignment target.")
	}
}

// ---- prefix/infix parse functions ----

func grouping(_ bool) {
	expression()
	consume(TOKEN_RIGHT_PAREN, "Expect ')' after expression.")
}

func call(_ bool) {
	argCount := argumentList()
	emitOp(OP_CALL)
	emitBytes(byte(argCount))
}

func dot(canAssign bool) {
	consume(TOKEN_IDENTIFIER, "Expect property name after '.'.")
	nameConstant := identifierConstant(current.previous)

	if canAssign && match(TOKEN_EQUAL) {
		expression()
		emitOp(OP_SET_PROPERTY)
		emitShort(nameConstant)
	} else if match(TOKEN_LEFT_PAREN) {
		argCount := argumentList()
		emitOp(OP_INVOKE)
		emitShort(nameConstant)
		emitBytes(byte(argCount))
	} else {
		emitOp(OP_GET_PROPERTY)
		emitShort(nameConstant)
	}
}

func literal(_ bool) {
	switch current.previous.Type {
	case TOKEN_FALSE:
		emitOp(OP_FALSE)
	case TOKEN_NIL:
		emitOp(OP_NIL)
	case TOKEN_TRUE:
		emitOp(OP_TRUE)
	}
}

func number(_ bool) {
	val := NumberVal(current.previous.Literal.(float64))
	emitOp(OP_CONSTANT)
	emitShort(makeConstant(val))
}

func string_(_ bool) {
	val := ObjVal(NewObjString(current.previous.Literal.(string)))
	emitOp(OP_CONSTANT)
	emitShort(makeConstant(val))
}

func variable(canAssign bool) {
	namedVariable(current.previous.Lexeme, canAssign)
}

func this_(_ bool) {
	if currentClass == nil {
		errorAtMsg("Cannot use 'this' outside of a class.")
		return
	}
	variable(false)
}

func super_(_ bool) {
	if currentClass == nil {
		errorAtMsg("Cannot use 'super' outside of a class.")
	} else if !currentClass.hasSuperclass {
		errorAtMsg("Cannot use 'super' in a class with no superclass.")
	}
	consume(TOKEN_DOT, "Expect '.' after 'super'.")
	consume(TOKEN_IDENTIFIER, "Expect superclass method name.")
	nameConstant := identifierConstant(current.previous)

	namedVariable("this", false)
	if match(TOKEN_LEFT_PAREN) {
		argCount := argumentList()
		namedVariable("super", false)
		emitOp(OP_SUPER_INVOKE)
		emitShort(nameConstant)
		emitBytes(byte(argCount))
	} else {
		namedVariable("super", false)
		emitOp(OP_GET_SUPER)
		emitShort(nameConstant)
	}
}

func unary(_ bool) {
	operatorType := current.previous.Type
	parsePrecedence(PREC_UNARY)

	switch operatorType {
	case TOKEN_BANG:
		emitOp(OP_NOT)
	case TOKEN_MINUS:
		emitOp(OP_NEGATE)
	}
}

func binary(_ bool) {
	operatorType := current.previous.Type
	rule := getRule(operatorType)
	parsePrecedence(Precedence(int(rule.prec) + 1))

	switch operatorType {
	case TOKEN_BANG_EQUAL:
		emitOps(OP_EQUAL, OP_NOT)
	case TOKEN_EQUAL_EQUAL:
		emitOp(OP_EQUAL)
	case TOKEN_GREATER:
		emitOp(OP_GREATER)
	case TOKEN_GREATER_EQUAL:
		emitOps(OP_LESS, OP_NOT)
	case TOKEN_LESS:
		emitOp(OP_LESS)
	case TOKEN_LESS_EQUAL:
		emitOps(OP_GREATER, OP_NOT)
	case TOKEN_PLUS:
		emitOp(OP_ADD)
	case TOKEN_MINUS:
		emitOp(OP_SUBTRACT)
	case TOKEN_STAR:
		emitOp(OP_MULTIPLY)
	case TOKEN_SLASH:
		emitOp(OP_DIVIDE)
	}
}

func and_(_ bool) {
	endJump := emitJump(OP_JUMP_IF_FALSE)
	emitOp(OP_POP)
	parsePrecedence(PREC_AND)
	patchJump(endJump)
}

func or_(_ bool) {
	elseJump := emitJump(OP_JUMP_IF_FALSE)
	endJump := emitJump(OP_JUMP)

	patchJump(elseJump)
	emitOp(OP_POP)

	parsePrecedence(PREC_OR)
	patchJump(endJump)
}

func argumentList() int {
	argCount := 0
	if !check(TOKEN_RIGHT_PAREN) {
		for {
			expression()
			argCount++
			if argCount > 255 {
				errorAtMsg("Can't have more than 255 arguments.")
			}
			if !match(TOKEN_COMMA) {
				break
			}
		}
	}
	consume(TOKEN_RIGHT_PAREN, "Expect ')' after arguments.")
	return argCount
}

// ---- variable helpers ----

func identifierConstant(token Token) int {
	val := ObjVal(NewObjString(token.Lexeme))
	return current.function.funcChunk.AddConstant(val)
}

func addLocal(name string) {
	locals := &current.locals
	*locals = append(*locals, Local{name: name, depth: current.scopeDepth})
}

func declareVariable() {
	if current.scopeDepth == 0 {
		return
	}
	name := current.previous.Lexeme
	locals := current.locals
	for i := len(locals) - 1; i >= 0; i-- {
		l := locals[i]
		if l.depth != -1 && l.depth < current.scopeDepth {
			break
		}
		if l.name == name {
			errorAtMsg("Variable with this name already declared in this scope.")
		}
	}
	addLocal(name)
}

func parseVariable(errorMsg string) int {
	consume(TOKEN_IDENTIFIER, errorMsg)
	declareVariable()
	if current.scopeDepth > 0 {
		return 0
	}
	return identifierConstant(current.previous)
}

func markInitialized() {
	if current.scopeDepth == 0 {
		return
	}
	locals := &current.locals
	(*locals)[len(*locals)-1].depth = current.scopeDepth
}

func defineVariable(global int) {
	if current.scopeDepth > 0 {
		markInitialized()
		return
	}
	emitOp(OP_DEFINE_GLOBAL)
	emitShort(global)
}

func namedVariable(name string, canAssign bool) {
	isLocal, isUpvalue := -1, -1
	_, locIdx, uvIdx := resolveLocal(current, name)
	if locIdx != -1 {
		isLocal = locIdx
	} else if uvIdx != -1 {
		isUpvalue = uvIdx
	}

	if canAssign && match(TOKEN_EQUAL) {
		expression()
		if isLocal != -1 {
			emitOp(OP_SET_LOCAL)
			emitBytes(byte(isLocal))
		} else if isUpvalue != -1 {
			emitOp(OP_SET_UPVALUE)
			emitBytes(byte(isUpvalue))
		} else {
			emitOp(OP_SET_GLOBAL)
			emitShort(identifierConstant(Token{Lexeme: name}))
		}
	} else {
		if isLocal != -1 {
			emitOp(OP_GET_LOCAL)
			emitBytes(byte(isLocal))
		} else if isUpvalue != -1 {
			emitOp(OP_GET_UPVALUE)
			emitBytes(byte(isUpvalue))
		} else {
			emitOp(OP_GET_GLOBAL)
			emitShort(identifierConstant(Token{Lexeme: name}))
		}
	}
}

func resolveLocal(compiler *Compiler, name string) (int, int, int) {
	if compiler != nil {
		for i := len(compiler.locals) - 1; i >= 0; i-- {
			local := compiler.locals[i]
			if name == local.name {
				return i, i, -1
			}
		}
	}
	if compiler.enclosing != nil {
		_, isLocal, isUpvalue := resolveLocal(compiler.enclosing, name)
		if isLocal != -1 {
			compiler.enclosing.locals[isLocal].isCaptured = true
			uv := Upvalue{index: isLocal, isLocal: true}
			compiler.upvalues = append(compiler.upvalues, uv)
			return -1, -1, len(compiler.upvalues) - 1
		}
		if isUpvalue != -1 {
			uv := Upvalue{index: isUpvalue, isLocal: false}
			compiler.upvalues = append(compiler.upvalues, uv)
			return -1, -1, len(compiler.upvalues) - 1
		}
	}
	return -1, -1, -1
}

func beginScope() {
	current.scopeDepth++
}

func endScope() {
	current.scopeDepth--
	for len(current.locals) > 0 && current.locals[len(current.locals)-1].depth > current.scopeDepth {
		if current.locals[len(current.locals)-1].isCaptured {
			emitOp(OP_CLOSE_UPVALUE)
		} else {
			emitOp(OP_POP)
		}
		current.locals = current.locals[:len(current.locals)-1]
	}
}

// ---- bytecode emission ----

func emitReturn() {
	if current.funcType == TYPE_INITIALIZER {
		emitOp(OP_GET_LOCAL)
		emitBytes(0)
	} else {
		emitOp(OP_NIL)
	}
	emitOp(OP_RETURN)
}

func emitOp(op OpCode) {
	current.function.funcChunk.Write(byte(op), current.previous.Line)
}

func emitOps(ops ...OpCode) {
	for _, op := range ops {
		emitOp(op)
	}
}

func emitBytes(b ...byte) {
	for _, bb := range b {
		current.function.funcChunk.Write(bb, current.previous.Line)
	}
}

func emitShort(v int) {
	emitBytes(byte(v>>8), byte(v))
}

func emitJump(instruction OpCode) int {
	emitOp(instruction)
	emitBytes(0xff, 0xff)
	return len(current.function.funcChunk.code) - 2
}

func emitLoop(loopStart int) {
	emitOp(OP_LOOP)
	offset := len(current.function.funcChunk.code) - loopStart + 2
	if offset > 65535 {
		errorAtMsg("Loop body too large.")
	}
	emitBytes(byte(offset>>8), byte(offset))
}

func patchJump(offset int) {
	jump := len(current.function.funcChunk.code) - offset - 2
	if jump > 65535 {
		errorAtMsg("Too much code to jump over.")
	}
	current.function.funcChunk.code[offset] = byte(jump >> 8)
	current.function.funcChunk.code[offset+1] = byte(jump)
}

func makeConstant(v Value) int {
	return current.function.funcChunk.AddConstant(v)
}

// ---- parse rules table ----

var rules []ParseRule

func initRules() {
	rules = []ParseRule{
		/* TOKEN_LEFT_PAREN    */ {grouping, call, PREC_CALL},
		/* TOKEN_RIGHT_PAREN   */ {nil, nil, PREC_NONE},
		/* TOKEN_LEFT_BRACE    */ {nil, nil, PREC_NONE},
		/* TOKEN_RIGHT_BRACE   */ {nil, nil, PREC_NONE},
		/* TOKEN_COMMA         */ {nil, nil, PREC_NONE},
		/* TOKEN_DOT           */ {nil, dot, PREC_CALL},
		/* TOKEN_MINUS         */ {unary, binary, PREC_TERM},
		/* TOKEN_PLUS          */ {nil, binary, PREC_TERM},
		/* TOKEN_SEMICOLON     */ {nil, nil, PREC_NONE},
		/* TOKEN_SLASH         */ {nil, binary, PREC_FACTOR},
		/* TOKEN_STAR          */ {nil, binary, PREC_FACTOR},
		/* TOKEN_BANG          */ {unary, nil, PREC_NONE},
		/* TOKEN_BANG_EQUAL    */ {nil, binary, PREC_EQUALITY},
		/* TOKEN_EQUAL         */ {nil, nil, PREC_NONE},
		/* TOKEN_EQUAL_EQUAL   */ {nil, binary, PREC_EQUALITY},
		/* TOKEN_GREATER       */ {nil, binary, PREC_COMPARISON},
		/* TOKEN_GREATER_EQUAL */ {nil, binary, PREC_COMPARISON},
		/* TOKEN_LESS          */ {nil, binary, PREC_COMPARISON},
		/* TOKEN_LESS_EQUAL    */ {nil, binary, PREC_COMPARISON},
		/* TOKEN_IDENTIFIER    */ {variable, nil, PREC_NONE},
		/* TOKEN_STRING        */ {string_, nil, PREC_NONE},
		/* TOKEN_NUMBER        */ {number, nil, PREC_NONE},
		/* TOKEN_AND           */ {nil, and_, PREC_AND},
		/* TOKEN_CLASS         */ {nil, nil, PREC_NONE},
		/* TOKEN_ELSE          */ {nil, nil, PREC_NONE},
		/* TOKEN_FALSE         */ {literal, nil, PREC_NONE},
		/* TOKEN_FOR           */ {nil, nil, PREC_NONE},
		/* TOKEN_FUN           */ {nil, nil, PREC_NONE},
		/* TOKEN_IF            */ {nil, nil, PREC_NONE},
		/* TOKEN_NIL           */ {literal, nil, PREC_NONE},
		/* TOKEN_OR            */ {nil, or_, PREC_OR},
		/* TOKEN_PRINT         */ {nil, nil, PREC_NONE},
		/* TOKEN_RETURN        */ {nil, nil, PREC_NONE},
		/* TOKEN_SUPER         */ {super_, nil, PREC_NONE},
		/* TOKEN_THIS          */ {this_, nil, PREC_NONE},
		/* TOKEN_TRUE          */ {literal, nil, PREC_NONE},
		/* TOKEN_VAR           */ {nil, nil, PREC_NONE},
		/* TOKEN_WHILE         */ {nil, nil, PREC_NONE},
		/* TOKEN_ERROR         */ {nil, nil, PREC_NONE},
		/* TOKEN_EOF           */ {nil, nil, PREC_NONE},
	}
}

func getRule(typ TokenType) ParseRule {
	if int(typ) >= len(rules) {
		return ParseRule{}
	}
	return rules[typ]
}
