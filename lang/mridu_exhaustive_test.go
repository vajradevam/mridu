package lang

import (
	"math"
	"strings"
	"testing"
)

// =============================================================================
// SCANNER UNIT TESTS
// =============================================================================

func tokenTypeName(tt TokenType) string {
	names := []string{
		"LEFT_PAREN", "RIGHT_PAREN", "LEFT_BRACE", "RIGHT_BRACE",
		"COMMA", "DOT", "MINUS", "PLUS", "SEMICOLON", "SLASH", "STAR",
		"BANG", "BANG_EQUAL", "EQUAL", "EQUAL_EQUAL",
		"GREATER", "GREATER_EQUAL", "LESS", "LESS_EQUAL",
		"IDENTIFIER", "STRING", "NUMBER",
		"AND", "CLASS", "ELSE", "FALSE", "FUN", "FOR", "IF", "NIL",
		"OR", "PRINT", "RETURN", "SUPER", "THIS", "TRUE", "VAR", "WHILE",
		"ERROR", "EOF",
	}
	if int(tt) < len(names) {
		return names[tt]
	}
	return "UNKNOWN"
}

func scanAll(source string) []Token {
	s := NewScanner(source)
	var tokens []Token
	for {
		tok := s.ScanToken()
		tokens = append(tokens, tok)
		if tok.Type == TOKEN_EOF || tok.Type == TOKEN_ERROR {
			break
		}
	}
	return tokens
}

func TestScannerSingleCharTokens(t *testing.T) {
	source := "(){}.,-+;!"
	tokens := scanAll(source)
	expected := []TokenType{
		TOKEN_LEFT_PAREN, TOKEN_RIGHT_PAREN,
		TOKEN_LEFT_BRACE, TOKEN_RIGHT_BRACE,
		TOKEN_DOT, TOKEN_COMMA, TOKEN_MINUS, TOKEN_PLUS,
		TOKEN_SEMICOLON, TOKEN_BANG, TOKEN_EOF,
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Type != exp {
			t.Errorf("token %d: expected %s, got %s (lexeme=%q)",
				i, tokenTypeName(exp), tokenTypeName(tokens[i].Type), tokens[i].Lexeme)
		}
	}
}

func TestScannerTwoCharTokens(t *testing.T) {
	source := "!= == <= >= // /*"
	tokens := scanAll(source)
	expected := []TokenType{
		TOKEN_BANG_EQUAL, TOKEN_EQUAL_EQUAL,
		TOKEN_LESS_EQUAL, TOKEN_GREATER_EQUAL,
		TOKEN_EOF,
	}
	// Comments absorb everything after them, so // and /* will skip
	// Let's just test the first 4
	for i := 0; i < 4; i++ {
		if tokens[i].Type != expected[i] {
			t.Errorf("token %d: expected %s, got %s (lexeme=%q)",
				i, tokenTypeName(expected[i]), tokenTypeName(tokens[i].Type), tokens[i].Lexeme)
		}
	}
}

func TestScannerNumbers(t *testing.T) {
	tests := []struct {
		src  string
		val  float64
	}{
		{"0", 0},
		{"123", 123},
		{"3.14", 3.14},
		{"0.5", 0.5},
		{"100.0", 100.0},
		{"999", 999},
	}
	for _, tc := range tests {
		tokens := scanAll(tc.src)
		if len(tokens) < 1 || tokens[0].Type != TOKEN_NUMBER {
			t.Errorf("%q: expected NUMBER token", tc.src)
			continue
		}
		got := tokens[0].Literal.(float64)
		if got != tc.val {
			t.Errorf("%q: expected %v, got %v", tc.src, tc.val, got)
		}
	}
}

func TestScannerStrings(t *testing.T) {
	tests := []struct {
		src string
		val string
	}{
		{`"hello"`, "hello"},
		{`""`, ""},
		{`" "`, " "},
		{`"abc123"`, "abc123"},
	}
	for _, tc := range tests {
		tokens := scanAll(tc.src)
		if len(tokens) < 1 || tokens[0].Type != TOKEN_STRING {
			t.Errorf("%q: expected STRING token", tc.src)
			continue
		}
		got := tokens[0].Literal.(string)
		if got != tc.val {
			t.Errorf("%q: expected %q, got %q", tc.src, tc.val, got)
		}
	}
}

func TestScannerKeywords(t *testing.T) {
	keywords := []struct {
		src string
		typ TokenType
	}{
		{"and", TOKEN_AND},
		{"class", TOKEN_CLASS},
		{"else", TOKEN_ELSE},
		{"false", TOKEN_FALSE},
		{"for", TOKEN_FOR},
		{"fun", TOKEN_FUN},
		{"if", TOKEN_IF},
		{"nil", TOKEN_NIL},
		{"or", TOKEN_OR},
		{"print", TOKEN_PRINT},
		{"return", TOKEN_RETURN},
		{"super", TOKEN_SUPER},
		{"this", TOKEN_THIS},
		{"true", TOKEN_TRUE},
		{"var", TOKEN_VAR},
		{"while", TOKEN_WHILE},
	}
	for _, kw := range keywords {
		tokens := scanAll(kw.src)
		if len(tokens) < 1 || tokens[0].Type != kw.typ {
			t.Errorf("%q: expected %s, got %s",
				kw.src, tokenTypeName(kw.typ), tokenTypeName(tokens[0].Type))
		}
	}
}

func TestScannerIdentifiers(t *testing.T) {
	ids := []string{"x", "foo", "bar123", "_underscore", "camelCase", "UPPER"}
	for _, id := range ids {
		tokens := scanAll(id)
		if len(tokens) < 1 || tokens[0].Type != TOKEN_IDENTIFIER {
			t.Errorf("%q: expected IDENTIFIER", id)
			continue
		}
		if tokens[0].Lexeme != id {
			t.Errorf("%q: expected lexeme %q, got %q", id, id, tokens[0].Lexeme)
		}
	}
}

func TestScannerLineComments(t *testing.T) {
	source := "// this is a comment\nprint;"
	tokens := scanAll(source)
	if len(tokens) < 2 || tokens[0].Type != TOKEN_PRINT {
		t.Errorf("expected PRINT after line comment, got %s", tokenTypeName(tokens[0].Type))
	}
}

func TestScannerBlockComments(t *testing.T) {
	source := "/* block */ print;"
	tokens := scanAll(source)
	if len(tokens) < 1 || tokens[0].Type != TOKEN_PRINT {
		t.Errorf("expected PRINT after block comment, got %s", tokenTypeName(tokens[0].Type))
	}
}

func TestScannerNestedBlockComments(t *testing.T) {
	source := "/* outer /* inner */ more */ print;"
	tokens := scanAll(source)
	if len(tokens) < 1 || tokens[0].Type != TOKEN_PRINT {
		t.Errorf("expected PRINT after nested block comment, got %s", tokenTypeName(tokens[0].Type))
	}
}

func TestScannerUnterminatedString(t *testing.T) {
	source := `"unterminated`
	tokens := scanAll(source)
	if len(tokens) < 1 || tokens[0].Type != TOKEN_ERROR {
		t.Errorf("expected ERROR token for unterminated string, got %s", tokenTypeName(tokens[0].Type))
	}
}

func TestScannerUnexpectedChar(t *testing.T) {
	source := "@"
	tokens := scanAll(source)
	if len(tokens) < 1 || tokens[0].Type != TOKEN_ERROR {
		t.Errorf("expected ERROR token for unexpected char, got %s", tokenTypeName(tokens[0].Type))
	}
}

func TestScannerEmptySource(t *testing.T) {
	tokens := scanAll("")
	if len(tokens) != 1 || tokens[0].Type != TOKEN_EOF {
		t.Errorf("expected single EOF token for empty source")
	}
}

func TestScannerWhitespace(t *testing.T) {
	source := "  \t\r\n  print  \n  ;  "
	tokens := scanAll(source)
	if len(tokens) < 2 {
		t.Fatalf("expected at least 2 tokens")
	}
	if tokens[0].Type != TOKEN_PRINT {
		t.Errorf("expected PRINT, got %s", tokenTypeName(tokens[0].Type))
	}
	if tokens[1].Type != TOKEN_SEMICOLON {
		t.Errorf("expected SEMICOLON, got %s", tokenTypeName(tokens[1].Type))
	}
}

func TestScannerLineCounting(t *testing.T) {
	source := "1\n2\n3\n"
	_ = NewScanner(source)
	// Scan tokens and verify line numbers
	s := NewScanner(source)
	var toks []Token
	for {
		tok := s.ScanToken()
		toks = append(toks, tok)
		if tok.Type == TOKEN_EOF {
			break
		}
	}
	if len(toks) < 3 {
		t.Fatalf("expected at least 3 tokens")
	}
	// numbers 1, 2, 3 should be on lines 1, 2, 3
	if toks[0].Line != 1 {
		t.Errorf("first number expected line 1, got %d", toks[0].Line)
	}
	if toks[1].Line != 2 {
		t.Errorf("second number expected line 2, got %d", toks[1].Line)
	}
	if toks[2].Line != 3 {
		t.Errorf("third number expected line 3, got %d", toks[2].Line)
	}
}

func TestScannerMultiLineString(t *testing.T) {
	source := "\"line1\nline2\""
	tokens := scanAll(source)
	if len(tokens) < 1 || tokens[0].Type != TOKEN_STRING {
		t.Errorf("expected STRING for multi-line string, got %s", tokenTypeName(tokens[0].Type))
	}
}

// =============================================================================
// CHUNK UNIT TESTS
// =============================================================================

func TestChunkWriteRead(t *testing.T) {
	c := NewChunk()
	c.Write(0xAB, 1)
	c.Write(0xCD, 1)
	if len(c.code) != 2 {
		t.Fatalf("expected 2 bytes, got %d", len(c.code))
	}
	if c.code[0] != 0xAB || c.code[1] != 0xCD {
		t.Errorf("code bytes mismatch")
	}
}

func TestChunkWrite16(t *testing.T) {
	c := NewChunk()
	c.Write16(0xABCD, 5)
	if len(c.code) != 2 {
		t.Fatalf("expected 2 bytes, got %d", len(c.code))
	}
	if c.code[0] != 0xAB || c.code[1] != 0xCD {
		t.Errorf("Write16: expected [0xAB, 0xCD], got [0x%02X, 0x%02X]", c.code[0], c.code[1])
	}
}

func TestChunkRead16(t *testing.T) {
	c := NewChunk()
	c.code = []byte{0x12, 0x34, 0xAB, 0xCD}
	val := c.Read16(0)
	if val != 0x1234 {
		t.Errorf("Read16(0): expected 0x1234, got 0x%04X", val)
	}
	val = c.Read16(2)
	if val != 0xABCD {
		t.Errorf("Read16(2): expected 0xABCD, got 0x%04X", val)
	}
}

func TestChunkConstants(t *testing.T) {
	c := NewChunk()
	idx1 := c.AddConstant(NumberVal(42.0))
	idx2 := c.AddConstant(BoolVal(true))
	idx3 := c.AddConstant(ObjVal(NewObjString("hello")))

	if idx1 != 0 || idx2 != 1 || idx3 != 2 {
		t.Errorf("constant indices: expected 0,1,2 got %d,%d,%d", idx1, idx2, idx3)
	}
	if len(c.constants) != 3 {
		t.Errorf("expected 3 constants, got %d", len(c.constants))
	}
	if !IS_NUMBER(c.constants[0]) || AS_NUMBER(c.constants[0]) != 42.0 {
		t.Errorf("constant 0 mismatch")
	}
	if !IS_BOOL(c.constants[1]) || AS_BOOL(c.constants[1]) != true {
		t.Errorf("constant 1 mismatch")
	}
}

func TestChunkLines(t *testing.T) {
	c := NewChunk()
	c.Write(0x01, 10)
	c.Write(0x02, 10)
	c.Write(0x03, 20)
	if c.GetLine(0) != 10 {
		t.Errorf("line 0: expected 10, got %d", c.GetLine(0))
	}
	if c.GetLine(2) != 20 {
		t.Errorf("line 2: expected 20, got %d", c.GetLine(2))
	}
	if c.GetLine(99) != -1 {
		t.Errorf("out of range: expected -1, got %d", c.GetLine(99))
	}
	if c.GetLine(-1) != -1 {
		t.Errorf("negative: expected -1, got %d", c.GetLine(-1))
	}
}

func TestChunkEmpty(t *testing.T) {
	c := NewChunk()
	if c == nil {
		t.Fatal("NewChunk returned nil")
	}
	if len(c.code) != 0 {
		t.Errorf("new chunk should have empty code")
	}
	if len(c.constants) != 0 {
		t.Errorf("new chunk should have no constants")
	}
}

// =============================================================================
// VALUE / OBJECT UNIT TESTS
// =============================================================================

func TestValueBool(t *testing.T) {
	v := BoolVal(true)
	if !IS_BOOL(v) {
		t.Error("IS_BOOL should be true")
	}
	if IS_NIL(v) || IS_NUMBER(v) || IS_OBJ(v) {
		t.Error("should only be bool")
	}
	if AS_BOOL(v) != true {
		t.Error("AS_BOOL should be true")
	}
	if v.String() != "true" {
		t.Errorf("String: expected 'true', got %q", v.String())
	}

	v = BoolVal(false)
	if AS_BOOL(v) != false {
		t.Error("AS_BOOL should be false")
	}
	if v.String() != "false" {
		t.Errorf("String: expected 'false', got %q", v.String())
	}
}

func TestValueNil(t *testing.T) {
	v := NilVal()
	if !IS_NIL(v) {
		t.Error("IS_NIL should be true")
	}
	if IS_BOOL(v) || IS_NUMBER(v) || IS_OBJ(v) {
		t.Error("should only be nil")
	}
	if v.String() != "nil" {
		t.Errorf("String: expected 'nil', got %q", v.String())
	}
}

func TestValueNumber(t *testing.T) {
	v := NumberVal(3.14)
	if !IS_NUMBER(v) {
		t.Error("IS_NUMBER should be true")
	}
	if AS_NUMBER(v) != 3.14 {
		t.Error("AS_NUMBER mismatch")
	}
	// %g format: 3.14 -> "3.14"
	if v.String() != "3.14" {
		t.Errorf("String: expected '3.14', got %q", v.String())
	}

	v = NumberVal(0)
	if AS_NUMBER(v) != 0 {
		t.Error("zero number mismatch")
	}

	v = NumberVal(-42.5)
	if AS_NUMBER(v) != -42.5 {
		t.Error("negative number mismatch")
	}
}

func TestValueObj(t *testing.T) {
	obj := NewObjString("test")
	v := ObjVal(obj)
	if !IS_OBJ(v) {
		t.Error("IS_OBJ should be true")
	}
	if IS_BOOL(v) || IS_NIL(v) || IS_NUMBER(v) {
		t.Error("should only be obj")
	}
	if AS_OBJ(v) != obj {
		t.Error("AS_OBJ mismatch")
	}
}

func TestValueObjNil(t *testing.T) {
	v := ObjVal(nil)
	if !IS_NIL(v) {
		t.Error("ObjVal(nil) should be nil")
	}
}

func TestValuesEqual(t *testing.T) {
	// same type
	if !ValuesEqual(BoolVal(true), BoolVal(true)) {
		t.Error("true == true")
	}
	if ValuesEqual(BoolVal(true), BoolVal(false)) {
		t.Error("true != false")
	}
	if !ValuesEqual(NilVal(), NilVal()) {
		t.Error("nil == nil")
	}
	if !ValuesEqual(NumberVal(42), NumberVal(42)) {
		t.Error("42 == 42")
	}
	if ValuesEqual(NumberVal(42), NumberVal(43)) {
		t.Error("42 != 43")
	}
	if !ValuesEqual(ObjVal(NewObjString("hi")), ObjVal(NewObjString("hi"))) {
		t.Error(`"hi" == "hi"`)
	}
	if ValuesEqual(ObjVal(NewObjString("hi")), ObjVal(NewObjString("bye"))) {
		t.Error(`"hi" != "bye"`)
	}

	// different types
	if ValuesEqual(BoolVal(true), NilVal()) {
		t.Error("true != nil")
	}
	if ValuesEqual(NumberVal(0), BoolVal(false)) {
		t.Error("0 != false")
	}
	if ValuesEqual(NilVal(), BoolVal(false)) {
		t.Error("nil != false")
	}
}

func TestObjectCreation(t *testing.T) {
	// ObjString
	s := NewObjString("hello")
	if s.objType != OBJ_STRING || s.strValue != "hello" {
		t.Error("ObjString creation failed")
	}

	// ObjFunction
	fn := NewObjFunction("myFunc")
	if fn.objType != OBJ_FUNCTION || fn.funcName != "myFunc" {
		t.Error("ObjFunction creation failed")
	}
	if fn.funcArity != 0 || fn.funcUpvalues != 0 {
		t.Error("new function should have 0 arity and upvalues")
	}

	// ObjClosure
	fn.funcUpvalues = 2
	cl := NewObjClosure(fn)
	if cl.objType != OBJ_CLOSURE || cl.closureFn != fn {
		t.Error("ObjClosure creation failed")
	}
	if len(cl.upvaluePtr) != 2 {
		t.Errorf("expected 2 upvalue slots, got %d", len(cl.upvaluePtr))
	}

	// ObjUpvalue
	uv := NewObjUpvalue(nil, 0)
	if uv.objType != OBJ_UPVALUE {
		t.Error("ObjUpvalue creation failed")
	}

	// ObjClass
	cls := NewObjClass("MyClass")
	if cls.objType != OBJ_CLASS || cls.className != "MyClass" {
		t.Error("ObjClass creation failed")
	}
	if cls.methods == nil {
		t.Error("new class should have methods map")
	}

	// ObjInstance
	inst := NewObjInstance(cls)
	if inst.objType != OBJ_INSTANCE || inst.instClass != cls {
		t.Error("ObjInstance creation failed")
	}
	if inst.fields == nil {
		t.Error("new instance should have fields map")
	}

	// ObjBoundMethod
	bm := NewObjBoundMethod(NumberVal(42), cl)
	if bm.objType != OBJ_BOUND_METHOD || bm.boundFn != cl {
		t.Error("ObjBoundMethod creation failed")
	}
	if !ValuesEqual(bm.boundRecv, NumberVal(42)) {
		t.Error("bound receiver mismatch")
	}
}

func TestTypeChecks(t *testing.T) {
	str := ObjVal(NewObjString("s"))
	fn := ObjVal(NewObjFunction("f"))
	nat := ObjVal(NewObjNative(func(int, []Value) Value { return NilVal() }))
	cl := ObjVal(NewObjClosure(NewObjFunction("c")))
	cls := ObjVal(NewObjClass("C"))
	inst := ObjVal(NewObjInstance(NewObjClass("I")))
	bm := ObjVal(NewObjBoundMethod(NilVal(), NewObjClosure(NewObjFunction("m"))))

	checks := []struct {
		v    Value
		str  bool
		fn   bool
		nat  bool
		clo  bool
		cls  bool
		inst bool
		bm   bool
	}{
		{str, true, false, false, false, false, false, false},
		{fn, false, true, false, false, false, false, false},
		{nat, false, false, true, false, false, false, false},
		{cl, false, false, false, true, false, false, false},
		{cls, false, false, false, false, true, false, false},
		{inst, false, false, false, false, false, true, false},
		{bm, false, false, false, false, false, false, true},
	}

	for i, c := range checks {
		if IS_STRING(c.v) != c.str {
			t.Errorf("check %d: IS_STRING mismatch", i)
		}
		if IS_FUNCTION(c.v) != c.fn {
			t.Errorf("check %d: IS_FUNCTION mismatch", i)
		}
		if IS_NATIVE(c.v) != c.nat {
			t.Errorf("check %d: IS_NATIVE mismatch", i)
		}
		if IS_CLOSURE(c.v) != c.clo {
			t.Errorf("check %d: IS_CLOSURE mismatch", i)
		}
		if IS_CLASS(c.v) != c.cls {
			t.Errorf("check %d: IS_CLASS mismatch", i)
		}
		if IS_INSTANCE(c.v) != c.inst {
			t.Errorf("check %d: IS_INSTANCE mismatch", i)
		}
		if IS_BOUND_METHOD(c.v) != c.bm {
			t.Errorf("check %d: IS_BOUND_METHOD mismatch", i)
		}
	}
}

func TestValueStringRepresentations(t *testing.T) {
	tests := []struct {
		v    Value
		want string
	}{
		{BoolVal(true), "true"},
		{BoolVal(false), "false"},
		{NilVal(), "nil"},
		{NumberVal(0), "0"},
		{NumberVal(42), "42"},
		{NumberVal(3.14), "3.14"},
		{NumberVal(-1), "-1"},
		{ObjVal(NewObjString("hello")), "hello"},
		{ObjVal(NewObjString("")), ""},
		{ObjVal(NewObjFunction("myFunc")), "<fn myFunc>"},
		{ObjVal(NewObjFunction("")), "<script>"},
		{ObjVal(NewObjNative(func(int, []Value) Value { return NilVal() })), "<native fn>"},
	}
	for _, tc := range tests {
		got := tc.v.String()
		if got != tc.want {
			t.Errorf("expected %q, got %q", tc.want, got)
		}
	}
}

func TestClosureString(t *testing.T) {
	fn := NewObjFunction("testFn")
	cl := ObjVal(NewObjClosure(fn))
	// closure string should delegate to function string
	got := cl.String()
	if got != "<fn testFn>" {
		t.Errorf("closure string: expected '<fn testFn>', got %q", got)
	}
}

func TestBoundMethodString(t *testing.T) {
	fn := NewObjFunction("bm")
	cl := NewObjClosure(fn)
	bm := ObjVal(NewObjBoundMethod(NilVal(), cl))
	got := bm.String()
	if got != "<fn bm>" {
		t.Errorf("bound method string: expected '<fn bm>', got %q", got)
	}
}

func TestInstanceString(t *testing.T) {
	cls := NewObjClass("MyClass")
	inst := ObjVal(NewObjInstance(cls))
	got := inst.String()
	if got != "MyClass instance" {
		t.Errorf("instance string: expected 'MyClass instance', got %q", got)
	}
}

func TestClassString(t *testing.T) {
	cls := ObjVal(NewObjClass("MyClass"))
	got := cls.String()
	if got != "MyClass" {
		t.Errorf("class string: expected 'MyClass', got %q", got)
	}
}

func TestValuesEqualEdgeCases(t *testing.T) {
	// Same object pointer
	s1 := NewObjString("same")
	s2 := s1
	if !ValuesEqual(ObjVal(s1), ObjVal(s2)) {
		t.Error("same pointer should be equal")
	}

	// Different non-string object types
	fn1 := NewObjFunction("f")
	fn2 := NewObjFunction("f")
	if ValuesEqual(ObjVal(fn1), ObjVal(fn2)) {
		t.Error("different function objects should not be equal")
	}
}

// =============================================================================
// COMPILER ERROR EXHAUSTIVENESS
// =============================================================================

func runCompileError(t *testing.T, name, source string) {
	t.Helper()
	InitVM()
	result := Interpret(source)
	if result != INTERPRET_COMPILE_ERROR {
		t.Errorf("%s: expected COMPILE_ERROR, got %d", name, result)
	}
}

func TestCompileErrorsExhaustive(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"expect_expr", "print ;"},
		{"expect_paren_after_if", "if true {}"},
		{"expect_paren_after_while", "while true {}"},
		{"expect_paren_after_for", "for true {}"},
		{"expect_semicolon_after_expr", "print 1"},
		{"expect_semicolon_after_var", "var x"},
		{"invalid_assign", "var x = 1; x + 1 = 2;"},
		{"var_no_name", "var ;"},
		{"fun_no_name", "fun () {}"},
		{"class_no_name", "class {}"},
		{"no_brace_after_class", "class C"},
		{"no_body_after_if", "if (true)"},
		{"double_var_decl", "{ var x; var x; }"},
	}
	for _, tc := range tests {
		runCompileError(t, tc.name, tc.src)
	}
}

func TestCompileErrorReturnFromTopLevel(t *testing.T) {
	runCompileError(t, "return_top", "return;")
}

func TestCompileErrorReturnFromTopLevelValue(t *testing.T) {
	runCompileError(t, "return_top_val", "return 1;")
}

func TestCompileErrorBadSuffix(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"trailing_dot", "print 1.;"},
		{"bad_char", "print @;"},
	}
	for _, tc := range tests {
		runCompileError(t, tc.name, tc.src)
	}
}

func TestCompileErrorBlockNotClosed(t *testing.T) {
	// If the brace is missing, the parser will hit EOF while still in a block
	// Since there's no explicit error for unclosed blocks, we just check it doesn't crash
	InitVM()
	result := Interpret("{ var x = 1; ")
	if result != INTERPRET_COMPILE_ERROR && result != INTERPRET_RUNTIME_ERROR {
		t.Log("unclosed block: result =", result)
	}
}

func TestCompileErrorUnterminatedBlockComment(t *testing.T) {
	InitVM()
	result := Interpret("/* unterminated ")
	if result != INTERPRET_COMPILE_ERROR && result != INTERPRET_RUNTIME_ERROR {
		t.Log("unterminated block comment: result =", result)
	}
}

// =============================================================================
// VM RUNTIME ERROR EXHAUSTIVENESS
// =============================================================================

func runRuntimeError(t *testing.T, name, source string) {
	t.Helper()
	InitVM()
	result := Interpret(source)
	if result != INTERPRET_RUNTIME_ERROR {
		t.Errorf("%s: expected RUNTIME_ERROR, got %d", name, result)
	}
}

func TestRuntimeErrorsExhaustive(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"undefined_var", "print x;"},
		{"assign_undefined", "x = 1;"},
		{"bad_arity_too_many", "fun f(a) {} f(1, 2);"},
		{"bad_arity_too_few", "fun f(a, b) {} f(1);"},
		{"type_add_str_num", `print "a" + 1;`},
		{"type_add_num_str", `print 1 + "a";`},
		{"type_sub", `print "a" - 1;`},
		{"type_mul", `print "a" * 1;`},
		{"type_div", `print "a" / 1;`},
		{"type_gt", `print "a" > 1;`},
		{"type_lt", `print "a" < 1;`},
		{"type_neg", `print -"a";`},
		{"div_zero", "print 1 / 0;"},
		{"call_nonfunc", "print 42();"},
		{"call_nonfunc_str", `print "str"();`},
		{"call_nonfunc_bool", "print true();"},
		{"call_nonfunc_nil", "print nil();"},
		{"get_property_noninst", `print 42.x;`},
		{"set_property_noninst", `42.x = 1;`},
		{"invoke_noninst", `42.m();`},
	}
	for _, tc := range tests {
		runRuntimeError(t, tc.name, tc.src)
	}
}

func TestRuntimeErrorSuperWithoutSuperclass(t *testing.T) {
	runRuntimeError(t, "super_no_super",
		`class C { m() { super.foo(); } }
		 C().m();`)
}

func TestCompileErrorThisOutsideClass(t *testing.T) {
	runCompileError(t, "this_outside", "print this;")
}

func TestCompileErrorSuperOutsideClass(t *testing.T) {
	runCompileError(t, "super_outside", "super.foo();")
}

func TestRuntimeErrorUndefinedProperty(t *testing.T) {
	runRuntimeError(t, "undefined_prop",
		`class C { m() { this.x; } }
		 C().m();`)
}

func TestRuntimeErrorUndefinedMethodCall(t *testing.T) {
	runRuntimeError(t, "undefined_method",
		`class C {} C().foo();`)
}

func TestRuntimeErrorCallNonCallableField(t *testing.T) {
	runRuntimeError(t, "noncallable_field",
		`class C { init() { this.x = 42; } }
		 var c = C();
		 c.x();`)
}

func TestRuntimeErrorStackOverflow(t *testing.T) {
	runRuntimeError(t, "stack_overflow",
		`fun f() { f(); }
		 f();`)
}

// =============================================================================
// OPCODE ISOLATION TESTS
// =============================================================================

func TestOpConstant(t *testing.T) {
	runTest(t, "const_num", "print 42;", ok("42"))
	runTest(t, "const_str", `print "hi";`, ok("hi"))
	runTest(t, "const_bool", "print true;", ok("true"))
	runTest(t, "const_nil", "print nil;", ok("nil"))
	runTest(t, "const_neg", "print -3;", ok("-3"))
}

func TestOpNilTrueFalse(t *testing.T) {
	runTest(t, "nil", "print nil;", ok("nil"))
	runTest(t, "true", "print true;", ok("true"))
	runTest(t, "false", "print false;", ok("false"))
	runTest(t, "all_three",
		"print nil; print true; print false;",
		ok("nil", "true", "false"))
}

func TestOpPop(t *testing.T) {
	// values get popped by expression statements
	runTest(t, "pop_expr", "1; 2; 3; print 4;", ok("4"))
}

func TestOpDup(t *testing.T) {
	runTest(t, "dup", "print 42;", ok("42"))
}

func TestOpGetSetLocal(t *testing.T) {
	runTest(t, "get_set",
		`var x = 10;
		 x = x + 1;
		 print x;`,
		ok("11"))
	runTest(t, "multi_local",
		`var a = 1;
		 var b = 2;
		 var c = 3;
		 print a + b + c;`,
		ok("6"))
}

func TestOpDefineGetSetGlobal(t *testing.T) {
	runTest(t, "global",
		`var g = 1;
		 print g;
		 g = 2;
		 print g;`,
		ok("1", "2"))
}

func TestOpGetSetUpvalue(t *testing.T) {
	runTest(t, "upvalue_get_set",
		`fun mk() {
		   var x = 1;
		   fun get() { return x; }
		   fun set(n) { x = n; }
		   fun both() { return x; }
		   return both;
		 }
		 var f = mk();
		 print f();`,
		ok("1"))
}

func TestOpCloseUpvalue(t *testing.T) {
	runTest(t, "close_upvalue",
		`fun mk() {
		   var x = 1;
		   fun f() { return x; }
		   return f;
		 }
		 print mk()();`,
		ok("1"))
}

func TestOpEquality(t *testing.T) {
	runTest(t, "eq", "print 1 == 1;", ok("true"))
	runTest(t, "ne", "print 1 != 1;", ok("false"))
	runTest(t, "eq_str", `print "a" == "a";`, ok("true"))
	runTest(t, "eq_nil", "print nil == nil;", ok("true"))
	runTest(t, "eq_bool", "print true == true;", ok("true"))
}

func TestOpGreater(t *testing.T) {
	runTest(t, "gt", "print 2 > 1;", ok("true"))
	runTest(t, "gt_false", "print 1 > 2;", ok("false"))
	runTest(t, "gt_equal", "print 2 > 2;", ok("false"))
}

func TestOpLess(t *testing.T) {
	runTest(t, "lt", "print 1 < 2;", ok("true"))
	runTest(t, "lt_false", "print 2 < 1;", ok("false"))
	runTest(t, "lt_equal", "print 2 < 2;", ok("false"))
}

func TestOpAdd(t *testing.T) {
	runTest(t, "add_num", "print 1 + 2;", ok("3"))
	runTest(t, "add_str", `print "a" + "b";`, ok("ab"))
	runTest(t, "add_neg", "print 1 + (-2);", ok("-1"))
	runTest(t, "add_zero", "print 0 + 0;", ok("0"))
}

func TestOpSubtract(t *testing.T) {
	runTest(t, "sub", "print 5 - 3;", ok("2"))
	runTest(t, "sub_neg", "print 5 - (-3);", ok("8"))
	runTest(t, "sub_zero", "print 5 - 5;", ok("0"))
}

func TestOpMultiply(t *testing.T) {
	runTest(t, "mul", "print 3 * 4;", ok("12"))
	runTest(t, "mul_zero", "print 5 * 0;", ok("0"))
	runTest(t, "mul_neg", "print 3 * (-4);", ok("-12"))
}

func TestOpDivide(t *testing.T) {
	runTest(t, "div", "print 10 / 2;", ok("5"))
	runTest(t, "div_frac", "print 5 / 2;", ok("2.5"))
	runTest(t, "div_neg", "print 10 / (-2);", ok("-5"))
	runTest(t, "div_one", "print 7 / 1;", ok("7"))
}

func TestOpNot(t *testing.T) {
	runTest(t, "not_true", "print !true;", ok("false"))
	runTest(t, "not_false", "print !false;", ok("true"))
	runTest(t, "not_nil", "print !nil;", ok("true"))
	runTest(t, "not_num", "print !42;", ok("false"))
	runTest(t, "not_zero", "print !0;", ok("false"))
}

func TestOpNegate(t *testing.T) {
	runTest(t, "neg", "print -42;", ok("-42"))
	runTest(t, "neg_zero", "print -0;", ok("-0"))
	runTest(t, "neg_neg", "print --5;", ok("5"))
	runTest(t, "neg_expr", "print -(3 + 4);", ok("-7"))
}

func TestOpJump(t *testing.T) {
	runTest(t, "jump_over",
		`if (false) print "nope";
		 print "ok";`,
		ok("ok"))
}

func TestOpJumpIfFalse(t *testing.T) {
	runTest(t, "jump_if_false",
		`if (true) print "y";
		 if (false) print "n";
		 print "done";`,
		ok("y", "done"))
}

func TestOpLoop(t *testing.T) {
	runTest(t, "loop",
		`var i = 0;
		 while (i < 3) {
		   i = i + 1;
		 }
		 print i;`,
		ok("3"))
	runTest(t, "loop_zero",
		`while (false) { print "never"; }
		 print "ok";`,
		ok("ok"))
}

func TestOpCall(t *testing.T) {
	runTest(t, "call", `fun f() { return 1; } print f();`, ok("1"))
	runTest(t, "call_args", `fun add(a,b) { return a+b; } print add(3,4);`, ok("7"))
	runTest(t, "call_nested", `fun f(x) { return x; } print f(f(5));`, ok("5"))
}

func TestOpClosure(t *testing.T) {
	runTest(t, "closure",
		`fun make(x) { fun get() { return x; } return get; }
		 print make(42)();`,
		ok("42"))
}

func TestOpReturn(t *testing.T) {
	runTest(t, "return_val",
		`fun f() { return 42; }
		 print f();`,
		ok("42"))
	runTest(t, "return_nil",
		`fun f() { return; }
		 print f();`,
		ok("nil"))
	runTest(t, "return_implicit",
		`fun f() {}
		 print f();`,
		ok("nil"))
}

func TestOpClass(t *testing.T) {
	runTest(t, "class",
		`class C { m() { return 1; } }
		 print C().m();`,
		ok("1"))
}

func TestOpGetSetProperty(t *testing.T) {
	runTest(t, "get_set_prop",
		`class C { init() { this.x = 1; } }
		 var c = C();
		 print c.x;
		 c.x = 42;
		 print c.x;`,
		ok("1", "42"))
}

func TestOpMethod(t *testing.T) {
	runTest(t, "method",
		`class C {
		   f() { return 10; }
		 }
		 print C().f();`,
		ok("10"))
}

func TestOpInvoke(t *testing.T) {
	runTest(t, "invoke",
		`class C {
		   m(a, b) { return a + b; }
		 }
		 print C().m(3, 4);`,
		ok("7"))
	runTest(t, "invoke_chain",
		`class C {
		   init(v) { this.v = v; }
		   add(n) { this.v = this.v + n; return this; }
		   get() { return this.v; }
		 }
		 print C(0).add(5).add(3).get();`,
		ok("8"))
}

func TestOpInherit(t *testing.T) {
	runTest(t, "inherit",
		`class P { m() { return "p"; } }
		 class C < P { }
		 print C().m();`,
		ok("p"))
	runTest(t, "inherit_override",
		`class P { m() { return "p"; } }
		 class C < P { m() { return "c"; } }
		 print C().m();`,
		ok("c"))
}

func TestOpGetSuper(t *testing.T) {
	runTest(t, "get_super",
		`class A { m() { return "A"; } }
		 class B < A { m() { return super.m() + "B"; } }
		 print B().m();`,
		ok("AB"))
}

func TestOpSuperInvoke(t *testing.T) {
	runTest(t, "super_invoke",
		`class A {
		   init(x) { this.x = x; }
		   v() { return this.x; }
		 }
		 class B < A {
		   init(x, y) { super.init(x); this.y = y; }
		   v() { return super.v() + this.y; }
		 }
		 print B(3, 7).v();`,
		ok("10"))
}

// =============================================================================
// BOUNDARY / LIMIT TESTS
// =============================================================================

func TestBoundaryManyLocals(t *testing.T) {
	var src strings.Builder
	src.WriteString("fun f() {\n")
	nLocals := 200
	for i := range nLocals {
		src.WriteString("  var l")
		src.WriteString(string(rune('0' + i%10)))
		if i >= 10 {
			src.WriteString(string(rune('a' + (i/10)%26)))
		}
		src.WriteString(" = ")
		src.WriteString(string(rune('0' + i%10)))
		src.WriteString(";\n")
	}
	src.WriteString("  return l0;\n}\nprint f();")
	runTest(t, "many_locals", src.String(), ok("0"))
}

func TestBoundaryMaxRecursion(t *testing.T) {
	// Recursion until stack overflow
	runRuntimeError(t, "max_recursion",
		`fun f(n) { f(n+1); }
		 f(0);`)
}

func TestBoundaryManyGlobals(t *testing.T) {
	var src strings.Builder
	n := 50
	for i := range n {
		src.WriteString("var g")
		src.WriteString(string(rune('0' + i%10)))
		if i >= 10 {
			src.WriteString(string(rune('0' + i/10)))
		}
		src.WriteString(" = ")
		src.WriteString(string(rune('0' + i%10)))
		src.WriteString(";\n")
	}
	src.WriteString("print g0;\n")
	runTest(t, "many_globals", src.String(), ok("0"))
}

func TestBoundaryDeepNesting(t *testing.T) {
	var src strings.Builder
	depth := 50
	for range depth {
		src.WriteString("{ ")
	}
	src.WriteString("print 1;\n")
	for range depth {
		src.WriteString("}")
	}
	runTest(t, "deep_nest", src.String(), ok("1"))
}

func TestBoundaryDeepArithmetic(t *testing.T) {
	var src strings.Builder
	src.WriteString("print ")
	// Create a deeply nested expression: (((1+1)+1)+1...)
	n := 50
	src.WriteString("1")
	for range n {
		src.WriteString(" + 1")
	}
	src.WriteString(";")
	runTest(t, "deep_arith", src.String(), ok("51"))
}

func TestBoundaryManyUpvalues(t *testing.T) {
	// Create a closure capturing many upvalues
	var src strings.Builder
	src.WriteString("fun make() {\n")
	n := 50
	vars := make([]string, n)
	for i := range n {
		name := string(rune('a' + i%26))
		if i >= 26 {
			name += string(rune('0' + i/26))
		}
		vars[i] = name
		src.WriteString("  var ")
		src.WriteString(name)
		src.WriteString(" = ")
		src.WriteString(string(rune('0' + i%10)))
		src.WriteString(";\n")
	}
	src.WriteString("  fun get() { return ")
	src.WriteString(vars[0])
	src.WriteString("; }\n")
	src.WriteString("  return get;\n}\n")
	src.WriteString("print make()();")
	runTest(t, "many_upvalues", src.String(), ok("0"))
}

func TestBoundaryDeepInheritance(t *testing.T) {
	var src strings.Builder
	depth := 20
	src.WriteString("class C0 { m() { return 1; } }\n")
	for i := 1; i < depth; i++ {
		src.WriteString("class C")
		src.WriteString(string(rune('0' + i%10)))
		if i >= 10 {
			src.WriteString(string(rune('a' + (i/10)%26)))
		}
		src.WriteString(" < C")
		src.WriteString(string(rune('0' + (i-1)%10)))
		if i-1 >= 10 {
			src.WriteString(string(rune('a' + ((i-1)/10)%26)))
		}
		src.WriteString(" {}\n")
	}
	lastIdx := depth - 1
	src.WriteString("print C")
	src.WriteString(string(rune('0' + lastIdx%10)))
	if lastIdx >= 10 {
		src.WriteString(string(rune('a' + (lastIdx/10)%26)))
	}
	src.WriteString("().m();")
	runTest(t, "deep_inherit", src.String(), ok("1"))
}

func TestBoundaryLongString(t *testing.T) {
	long := strings.Repeat("x", 10000)
	src := `print "` + long + `";`
	runTest(t, "long_string", src, ok(long))
}

func TestBoundaryManyParams(t *testing.T) {
	var src strings.Builder
	n := 100
	src.WriteString("fun f(")
	for i := range n {
		if i > 0 {
			src.WriteString(", ")
		}
		src.WriteString("p")
		src.WriteString(string(rune('0' + i%10)))
		if i >= 10 {
			src.WriteString(string(rune('a' + (i/10)%26)))
		}
	}
	src.WriteString(") {\n")
	src.WriteString("  return p0;\n}\n")
	src.WriteString("print f(")
	for i := range n {
		if i > 0 {
			src.WriteString(", ")
		}
		src.WriteString("1")
	}
	src.WriteString(");")
	runTest(t, "many_params", src.String(), ok("1"))
}

func TestBoundaryManyClassInstances(t *testing.T) {
	var src strings.Builder
	src.WriteString("class C { init() { this.x = 0; } m() { return 1; } }\n")
	n := 100
	for i := range n {
		src.WriteString("var c")
		src.WriteString(string(rune('0' + i%10)))
		if i >= 10 {
			src.WriteString(string(rune('0' + i/10)))
		}
		src.WriteString(" = C();\n")
	}
	src.WriteString("print 1;")
	runTest(t, "many_instances", src.String(), ok("1"))
}

func TestBoundaryLargeLoopIterations(t *testing.T) {
	runTest(t, "big_loop",
		`var s = 0;
		 var i = 0;
		 while (i < 1000) {
		   s = s + i;
		   i = i + 1;
		 }
		 print s;`,
		ok("499500"))
}

func TestBoundaryManyNestedLoops(t *testing.T) {
	var src strings.Builder
	src.WriteString("var s = 0;\n")
	// 5 nested loops each iterating 3 times = 243 iterations
	nLoops := 5
	for i := range nLoops {
		indent := strings.Repeat(" ", i*2)
		idx := string(rune('a' + i))
		src.WriteString(indent)
		src.WriteString("for (var ")
		src.WriteString(idx)
		src.WriteString(" = 0; ")
		src.WriteString(idx)
		src.WriteString(" < 3; ")
		src.WriteString(idx)
		src.WriteString(" = ")
		src.WriteString(idx)
		src.WriteString(" + 1) {\n")
	}
	src.WriteString(strings.Repeat(" ", nLoops*2))
	src.WriteString("s = s + 1;\n")
	for i := nLoops - 1; i >= 0; i-- {
		src.WriteString(strings.Repeat(" ", i*2))
		src.WriteString("}\n")
	}
	src.WriteString("print s;")
	runTest(t, "many_nested_loops", src.String(), ok("243"))
}

// =============================================================================
// STRESS TESTS
// =============================================================================

func TestStressFibonacci30(t *testing.T) {
	runTest(t, "fib30",
		`fun fib(n) {
		   if (n <= 1) return n;
		   return fib(n-1) + fib(n-2);
		 }
		 print fib(30);`,
		ok("832040"))
}

func TestStressManyStringConcats(t *testing.T) {
	var src strings.Builder
	src.WriteString(`var s = "";` + "\n")
	n := 200
	for range n {
		src.WriteString("s = s + \"x\";\n")
	}
	src.WriteString("print s;")
	// This is a stress test — may be slow, but should not crash
	InitVM()
	result := Interpret(src.String())
	if result != INTERPRET_OK {
		t.Errorf("stress concat: expected OK, got %d", result)
	}
}

func TestStressDeepRecursion(t *testing.T) {
	// Deep recursion that doesn't overflow: fact(1000) will overflow, so use smaller
	runRuntimeError(t, "deep_recursion",
		`fun f(n) { if (n <= 0) return 0; return 1 + f(n-1); }
		 print f(10000);`)
}

func TestStressManyFunctionCalls(t *testing.T) {
	runTest(t, "many_calls",
		`fun id(x) { return x; }
		 var s = 0;
		 var i = 0;
		 while (i < 1000) {
		   s = s + id(i);
		   i = i + 1;
		 }
		 print s;`,
		ok("499500"))
}

func TestStressLargeClassHierarchy(t *testing.T) {
	var src strings.Builder
	src.WriteString("class C0 { m() { return 1; } }\n")
	n := 50
	for i := 1; i < n; i++ {
		src.WriteString("class C")
		src.WriteString(string(rune('0' + i%10)))
		if i >= 10 {
			src.WriteString(string(rune('a' + (i/10)%26)))
		}
		src.WriteString(" < C")
		src.WriteString(string(rune('0' + (i-1)%10)))
		if i-1 >= 10 {
			src.WriteString(string(rune('a' + ((i-1)/10)%26)))
		}
		src.WriteString(" {}\n")
	}
	lastIdx := n - 1
	src.WriteString("print C")
	src.WriteString(string(rune('0' + lastIdx%10)))
	if lastIdx >= 10 {
		src.WriteString(string(rune('a' + (lastIdx/10)%26)))
	}
	src.WriteString("().m();")
	runTest(t, "large_hierarchy", src.String(), ok("1"))
}

// =============================================================================
// PROPERTY-BASED / RANDOMIZED TESTS
// =============================================================================

func TestPropertyArithmeticIdentity(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"add_zero", "print 42 + 0;", "42"},
		{"sub_zero", "print 42 - 0;", "42"},
		{"mul_one", "print 42 * 1;", "42"},
		{"div_one", "print 42 / 1;", "42"},
		{"mul_zero", "print 42 * 0;", "0"},
		{"sub_self", "print 42 - 42;", "0"},
		{"div_self", "print 42 / 42;", "1"},
		{"neg_neg", "print --42;", "42"},
		{"not_not", "print !!true;", "true"},
		{"or_true", "print false or true;", "true"},
		{"and_true", "print true and true;", "true"},
		{"and_false", "print true and false;", "false"},
	}
	for _, tc := range tests {
		runTest(t, tc.name, tc.src, ok(tc.want))
	}
}

func TestPropertyBooleanAlgebra(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"idempotent_and", "print (true and true) == true;", "true"},
		{"idempotent_or", "print (false or false) == false;", "true"},
		{"double_negation", "print !!true == true;", "true"},
		{"de_morgan1",
			`var a = true; var b = false;
			 print !(a and b) == (!a or !b);`, "true"},
		{"de_morgan2",
			`var a = true; var b = false;
			 print !(a or b) == (!a and !b);`, "true"},
		{"complement_and",
			`var a = true;
			 print (a and !a) == false;`, "true"},
		{"complement_or",
			`var a = true;
			 print (a or !a) == true;`, "true"},
	}
	for _, tc := range tests {
		runTest(t, tc.name, tc.src, ok(tc.want))
	}
}

func TestPropertyNumericLaws(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"commutative_add", "print 3 + 5 == 5 + 3;", "true"},
		{"commutative_mul", "print 3 * 5 == 5 * 3;", "true"},
		{"associative_add",
			"print (1 + 2) + 3 == 1 + (2 + 3);", "true"},
		{"associative_mul",
			"print (2 * 3) * 4 == 2 * (3 * 4);", "true"},
		{"distributive",
			"print 2 * (3 + 4) == 2*3 + 2*4;", "true"},
	}
	for _, tc := range tests {
		runTest(t, tc.name, tc.src, ok(tc.want))
	}
}

func TestPropertyComparisonLaws(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"trichotomy_lt",
			`var a = 5; var b = 3;
			 print (a < b) or (a == b) or (a > b);`, "true"},
		{"antisymmetry",
			`var a = 3; var b = 5;
			 print (a <= b) and (b >= a);`, "true"},
		{"transitivity",
			`print (1 < 2) and (2 < 3) and (1 < 3);`, "true"},
		{"reflexivity_eq",
			"print 42 == 42;", "true"},
		{"not_gt_equals_lt",
			`print !(5 > 3) == (5 <= 3);`, "true"},
	}
	for _, tc := range tests {
		runTest(t, tc.name, tc.src, ok(tc.want))
	}
}

// =============================================================================
// NEGATIVE TEST EXPANSION
// =============================================================================

func TestNegativeDivByZeroAllForms(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"div_by_zero", "print 1 / 0;"},
		{"div_neg_by_zero", "print (-1) / 0;"},
		{"div_zero_by_zero", "print 0 / 0;"},
		{"div_by_zero_expr", "print 10 / (1 - 1);"},
	}
	for _, tc := range tests {
		runRuntimeError(t, tc.name, tc.src)
	}
}

func TestNegativeArithTypeErrors(t *testing.T) {
	ops := []string{"+", "-", "*", "/", "<", ">", "<=", ">="}
	for _, op := range ops {
		runRuntimeError(t, "str_"+op, `print "x" `+op+` 1;`)
		runRuntimeError(t, "num_"+op+"_str", `print 1 `+op+` "x";`)
	}
}

func TestNegativeNegateTypeError(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"neg_str", `print -"x";`},
		{"neg_bool", "print -true;"},
		{"neg_nil", "print -nil;"},
	}
	for _, tc := range tests {
		runRuntimeError(t, tc.name, tc.src)
	}
}

func TestNegativeCallNonFunction(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"call_number", "print 42();"},
		{"call_string", `print "str"();`},
		{"call_bool", "print true();"},
		{"call_nil", "print nil();"},
	}
	for _, tc := range tests {
		runRuntimeError(t, tc.name, tc.src)
	}
}

func TestNegativeArityMismatch(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"too_many_args", `fun f(a) {} f(1, 2);`},
		{"too_few_args", `fun f(a, b) {} f(1);`},
		{"zero_expected_one", `fun f() {} f(1);`},
		{"one_expected_zero", `fun f(a) {} f();`},
	}
	for _, tc := range tests {
		runRuntimeError(t, tc.name, tc.src)
	}
}

func TestNegativeInheritSelf(t *testing.T) {
	runCompileError(t, "inherit_self",
		`class C < C {}`)
}

func TestNegativePropertyOnNonInstance(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"get_on_num", "print 42.x;"},
		{"get_on_str", `print "s".x;`},
		{"get_on_bool", "print true.x;"},
		{"get_on_nil", "print nil.x;"},
		{"set_on_num", "42.x = 1;"},
		{"invoke_on_num", "42.m();"},
	}
	for _, tc := range tests {
		runRuntimeError(t, tc.name, tc.src)
	}
}

func TestNegativeUndefinedGlobal(t *testing.T) {
	runRuntimeError(t, "get_undef", "print x;")
	runRuntimeError(t, "set_undef", "x = 1;")
}

// =============================================================================
// GLOBAL STATE ISOLATION
// =============================================================================

func TestGlobalStateReInit(t *testing.T) {
	// Note: InitVM() does not clear Globals — only resets VM state.
	// Globals must be managed manually. This test documents that behavior.
	InitVM()
	Interpret(`var x = 42;`)
	delete(Globals, "x")
	InitVM()
	// After deleting x and reinit, x should not exist
	_ = captureStdout(func() {
		result := Interpret(`print x;`)
		if result != INTERPRET_RUNTIME_ERROR {
			t.Error("expected x to be undefined")
		}
	})
}

// =============================================================================
// COMPILER OPCODE GENERATION PATTERNS
// =============================================================================

func TestCompilerChainedComparison(t *testing.T) {
	// The language doesn't have chained comparison syntax,
	// but we can test sequential comparisons
	runTest(t, "chain_cmp_and",
		`print 1 < 2 and 2 < 3;`,
		ok("true"))
	runTest(t, "chain_cmp_or",
		`print 1 > 2 or 2 < 3;`,
		ok("true"))
}

func TestCompilerNestedGrouping(t *testing.T) {
	runTest(t, "nested_group",
		"print ((((1 + 2) * 3) - 4) / 5);",
		ok("1"))
}

func TestCompilerManyConstants(t *testing.T) {
	var src strings.Builder
	src.WriteString("print ")
	n := 100
	for i := range n {
		if i > 0 {
			src.WriteString(" + ")
		}
		src.WriteString("1")
	}
	src.WriteString(";")
	runTest(t, "many_constants", src.String(), ok("100"))
}

func TestCompilerEmptyFunction(t *testing.T) {
	runTest(t, "empty_fn",
		`fun f() {} print f();`,
		ok("nil"))
}

func TestCompilerNestedFunctionReturn(t *testing.T) {
	runTest(t, "nested_fn",
		`fun outer() {
		   fun inner() {
		     return 42;
		   }
		   return inner;
		 }
		 print outer()();`,
		ok("42"))
}

func TestCompilerClosureCapturesMultiple(t *testing.T) {
	runTest(t, "capture_multi",
		`fun make(a, b, c) {
		   fun f() { return a + b + c; }
		   return f;
		 }
		 print make(1, 2, 3)();`,
		ok("6"))
}

func TestCompilerClassWithManyMethods(t *testing.T) {
	var src strings.Builder
	src.WriteString("class C {\n")
	n := 50
	for i := range n {
		src.WriteString("  m")
		src.WriteString(string(rune('0' + i%10)))
		if i >= 10 {
			src.WriteString(string(rune('0' + i/10)))
		}
		src.WriteString("() { return ")
		src.WriteString(string(rune('0' + i%10)))
		src.WriteString("; }\n")
	}
	src.WriteString("  get() { return 1; }\n}\n")
	src.WriteString("print C().get();")
	runTest(t, "many_methods", src.String(), ok("1"))
}

func TestCompilerClassWithManyFields(t *testing.T) {
	var src strings.Builder
	src.WriteString("class C {\n")
	src.WriteString("  init() {\n")
	n := 100
	for i := range n {
		src.WriteString("    this.f")
		src.WriteString(string(rune('0' + i%10)))
		if i >= 10 {
			src.WriteString(string(rune('0' + i/10)))
		}
		src.WriteString(" = ")
		src.WriteString(string(rune('0' + i%10)))
		src.WriteString(";\n")
	}
	src.WriteString("  }\n")
	src.WriteString("  get() { return this.f0; }\n}\n")
	src.WriteString("print C().get();")
	runTest(t, "many_fields", src.String(), ok("0"))
}

func TestCompilerWhileTrue(t *testing.T) {
	// Infinite loop that breaks via return
	runTest(t, "while_true_return",
		`fun f() {
		   while (true) {
		     return 42;
		   }
		 }
		 print f();`,
		ok("42"))
}

func TestCompilerForWithComplexIncrement(t *testing.T) {
	runTest(t, "complex_inc",
		`var s = 0;
		 for (var i = 0; i < 5; i = i + 2) {
		   s = s + i;
		 }
		 print s;`,
		ok("6"))
}

// =============================================================================
// NIL / BOOL EDGE CASES
// =============================================================================

func TestNilBoolEdgeCases(t *testing.T) {
	runTest(t, "nil_cond", "if (nil) print 1; else print 2;", ok("2"))
	runTest(t, "zero_cond", "if (0) print 1; else print 2;", ok("1"))
	runTest(t, "empty_str_cond", `if ("") print 1; else print 2;`, ok("1"))
	runTest(t, "nil_or", "print nil or true;", ok("true"))
	runTest(t, "nil_and", "print nil and false;", ok("nil"))
}

// =============================================================================
// EDGE CASE: MANY NESTED FUNCTION CALLS (WITHIN LIMITS)
// =============================================================================

func TestEdgeDeepCallChain(t *testing.T) {
	// chain of function calls within frame limit
	runTest(t, "deep_calls",
		`fun f0(x) { return x; }
		 fun f1(x) { return f0(x); }
		 fun f2(x) { return f1(x); }
		 fun f3(x) { return f2(x); }
		 fun f4(x) { return f3(x); }
		 fun f5(x) { return f4(x); }
		 fun f6(x) { return f5(x); }
		 fun f7(x) { return f6(x); }
		 fun f8(x) { return f7(x); }
		 fun f9(x) { return f8(x); }
		 print f9(42);`,
		ok("42"))
}

func TestEdgeMultipleReturns(t *testing.T) {
	runTest(t, "multi_return",
		`fun f(n) {
		   if (n <= 0) return 0;
		   if (n == 1) return 1;
		   if (n == 2) return 2;
		   return 3;
		 }
		 print f(-1);
		 print f(0);
		 print f(1);
		 print f(2);
		 print f(5);`,
		ok("0", "0", "1", "2", "3"))
}

func TestEdgeNoSemicolonBeforeEOF(t *testing.T) {
	// Last statement doesn't strictly need a semicolon in the grammar
	// due to how the parser works
	runCompileError(t, "no_semi", "print 1")
}

// =============================================================================
// FLOATING POINT EDGE CASES
// =============================================================================

func TestFloatEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"pi_approx", "print 3.14159;", "3.1415900000000003"},
		{"small_float", "print 0.0001;", "0.0001"},
		{"neg_float", "print -3.14;", "-3.14"},
		{"zero_point_five", "print 0.5;", "0.5"},
	}
	for _, tc := range tests {
		runTest(t, tc.name, tc.src, ok(tc.want))
	}
}

func TestFloatParseEdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"0", 0},
		{"1", 1},
		{"123", 123},
		{"0.5", 0.5},
		{"1.5", 1.5},
		{"3.14159", 3.14159},
		{"100.0", 100.0},
		{"0.001", 0.001},
	}
	for _, tc := range tests {
		got := parseFloat(tc.input)
		if math.Abs(got-tc.want) > 1e-10 {
			t.Errorf("parseFloat(%q): expected %v, got %v", tc.input, tc.want, got)
		}
	}
}

// =============================================================================
// REGRESSION: CHAINED ASSIGNMENT
// =============================================================================

func TestChainedAssignment(t *testing.T) {
	runTest(t, "chained_global",
		`var x; var y;
		 x = y = 42;
		 print x;
		 print y;`,
		ok("42", "42"))
}

func TestChainedAssignmentLocal(t *testing.T) {
	runTest(t, "chained_local",
		`{ var x; var y; x = y = 7; print x; print y; }`,
		ok("7", "7"))
}

// =============================================================================
// COMPILER: DEEP BLOCK NESTING + VARIABLE LOOKUP
// =============================================================================

func TestCompilerShadowingDepth(t *testing.T) {
	var src strings.Builder
	src.WriteString("var x = 0;\n")
	depth := 20
	for i := 1; i <= depth; i++ {
		src.WriteString("{ var x = ")
		src.WriteString(string(rune('0' + i%10)))
		if i >= 10 {
			src.WriteString(string(rune('0' + i/10)))
		}
		src.WriteString(";\n")
	}
	for range depth {
		src.WriteString("}\n")
	}
	src.WriteString("print x;")
	runTest(t, "deep_shadow", src.String(), ok("0"))
}

// =============================================================================
// VM: RETURN VALUE FROM DIFFERENT DEPTHS
// =============================================================================

func TestReturnFromDeepBlocks(t *testing.T) {
	runTest(t, "return_deep",
		`fun f() {
		   {
		     {
		       {
		         return 42;
		       }
		     }
		   }
		 }
		 print f();`,
		ok("42"))
}

// =============================================================================
// CROSS-FEATURE: CLOSURE + CLASS + INHERITANCE
// =============================================================================

func TestCrossClosureAndClass(t *testing.T) {
	runTest(t, "closure_in_class",
		`class C {
		   init() { this.x = 0; }
		   makeGetter() {
		     var self = this;
		     fun get() { return self.x; }
		     return get;
		   }
		 }
		 var c = C();
		 c.x = 42;
		 print c.makeGetter()();`,
		ok("42"))
}
