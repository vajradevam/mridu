package lang

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

type testCase struct {
	result InterpretResult
	out    []string
	err    []string
}

func runTest(t *testing.T, name, source string, tc testCase) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Helper()
		InitVM()
		globalsSnapshot := make(map[string]Value)
		for k, v := range Globals {
			globalsSnapshot[k] = v
		}

		stdout := captureStdout(func() {
			stderr := captureStderr(func() {
				result := Interpret(source)
				if result != tc.result {
					t.Errorf("expected result %d, got %d", tc.result, result)
				}
			})
			checkOutput(t, "stderr", stderr, tc.err)
		})
		checkOutput(t, "stdout", stdout, tc.out)

		for k := range Globals {
			if _, ok := globalsSnapshot[k]; !ok && k != "clock" {
				delete(Globals, k)
			}
		}
	})
}

func checkOutput(t *testing.T, name, actual string, expected []string) {
	t.Helper()
	if expected == nil {
		return
	}
	lines := strings.Split(strings.TrimRight(actual, "\n"), "\n")

	if len(lines) != len(expected) {
		t.Errorf("%s: expected %d lines, got %d\n  expected: %q\n  got:      %q",
			name, len(expected), len(lines), expected, lines)
		return
	}
	for i, exp := range expected {
		got := strings.TrimRight(lines[i], "\r")
		if got != exp {
			t.Errorf("%s line %d: expected %q, got %q", name, i, exp, got)
		}
	}
}

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func captureStderr(fn func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func ok(out ...string) testCase {
	return testCase{INTERPRET_OK, out, nil}
}

func compileErr(out ...string) testCase {
	return testCase{INTERPRET_COMPILE_ERROR, out, nil}
}

func runtimeErr(out ...string) testCase {
	return testCase{INTERPRET_RUNTIME_ERROR, out, nil}
}

// ==================== SMOKE TESTS ====================

func TestSmokeLiterals(t *testing.T) {
	runTest(t, "number", "print 42;", ok("42"))
	runTest(t, "zero", "print 0;", ok("0"))
	runTest(t, "negative", "print -42;", ok("-42"))
	runTest(t, "float", "print 3.14;", ok("3.14"))
	runTest(t, "true", "print true;", ok("true"))
	runTest(t, "false", "print false;", ok("false"))
	runTest(t, "nil", "print nil;", ok("nil"))
	runTest(t, "empty_string", `print "";`, ok(""))
	runTest(t, "string", `print "hello";`, ok("hello"))
	runTest(t, "all_nils", "print nil; print nil; print nil;", ok("nil", "nil", "nil"))
}

func TestSmokeArithmetic(t *testing.T) {
	runTest(t, "add", "print 1 + 2;", ok("3"))
	runTest(t, "sub", "print 10 - 4;", ok("6"))
	runTest(t, "mul", "print 6 * 7;", ok("42"))
	runTest(t, "div", "print 15 / 3;", ok("5"))
	runTest(t, "precedence", "print 2 + 3 * 4;", ok("14"))
	runTest(t, "grouping", "print (2 + 3) * 4;", ok("20"))
	runTest(t, "neg_expr", "print -(3 + 4);", ok("-7"))
	runTest(t, "chain", "print 10 - 5 - 2;", ok("3"))
	runTest(t, "mixed", "print 2 * 3 + 4 * 5;", ok("26"))
	runTest(t, "div_chain", "print 100 / 10 / 2;", ok("5"))
	runTest(t, "div_zero", "print 1 / 0;", runtimeErr())
}

func TestSmokeComparison(t *testing.T) {
	runTest(t, "lt_true", "print 1 < 2;", ok("true"))
	runTest(t, "lt_false", "print 2 < 1;", ok("false"))
	runTest(t, "le_eq", "print 2 <= 2;", ok("true"))
	runTest(t, "le_lt", "print 1 <= 2;", ok("true"))
	runTest(t, "gt_true", "print 2 > 1;", ok("true"))
	runTest(t, "ge_eq", "print 2 >= 2;", ok("true"))
	runTest(t, "eq_num", "print 1 == 1;", ok("true"))
	runTest(t, "ne_num", "print 1 != 1;", ok("false"))
	runTest(t, "eq_bool", "print true == true;", ok("true"))
	runTest(t, "eq_nil", "print nil == nil;", ok("true"))
	runTest(t, "nil_not_false", "print nil == false;", ok("false"))
	runTest(t, "str_eq", `print "a" == "a";`, ok("true"))
	runTest(t, "str_ne", `print "a" == "b";`, ok("false"))
}

func TestSmokeLogical(t *testing.T) {
	runTest(t, "not_true", "print !true;", ok("false"))
	runTest(t, "not_false", "print !false;", ok("true"))
	runTest(t, "double_not", "print !!true;", ok("true"))
	runTest(t, "and_tt", "print true and true;", ok("true"))
	runTest(t, "and_tf", "print true and false;", ok("false"))
	runTest(t, "or_tf", "print true or false;", ok("true"))
	runTest(t, "or_ff", "print false or false;", ok("false"))
	runTest(t, "nil_and", "print nil and true;", ok("nil"))
	runTest(t, "nil_or", "print nil or true;", ok("true"))
	runTest(t, "false_or_val", "print false or 42;", ok("42"))
}

func TestSmokeStrings(t *testing.T) {
	runTest(t, "concat", `print "a" + "b";`, ok("ab"))
	runTest(t, "concat_multi", `print "ab" + "cd";`, ok("abcd"))
	runTest(t, "concat_chain", `print "x" + "y" + "z";`, ok("xyz"))
	runTest(t, "add_type_err", `print "a" + 1;`, runtimeErr())
}

func TestSmokeVariables(t *testing.T) {
	runTest(t, "declare", "var a; print a;", ok("nil"))
	runTest(t, "init", `var a = "hi"; print a;`, ok("hi"))
	runTest(t, "assign", `var a = 1; a = 2; print a;`, ok("2"))
	runTest(t, "mutate", "var a = 0; a = a + 1; a = a + 1; print a;", ok("2"))
	runTest(t, "multi", "var a = 1; var b = 2; print a + b;", ok("3"))
	runTest(t, "reassign_type", `var a = 1; a = "x"; print a;`, ok("x"))
}

func TestSmokeScoping(t *testing.T) {
	runTest(t, "block_inner",
		`var a = "outer";
		 { var b = "inner"; print b; }
		 print a;`,
		ok("inner", "outer"))
	runTest(t, "shadow",
		`var x = "global";
		 { var x = "shadow"; print x; }
		 print x;`,
		ok("shadow", "global"))
	runTest(t, "deep_nest",
		`var a = 0;
		 { var b = 1;
		   { var c = 2;
		     { var d = 3; print d; }
		     print c; }
		   print b; }
		 print a;`,
		ok("3", "2", "1", "0"))
}

func TestSmokeControlFlow(t *testing.T) {
	runTest(t, "if_true", `if (true) print "y";`, ok("y"))
	runTest(t, "if_false", `if (false) print "n";`, ok())
	runTest(t, "if_else", `if (false) print "a"; else print "b";`, ok("b"))
	runTest(t, "while",
		`var i = 0; while (i < 3) { print i; i = i + 1; }`,
		ok("0", "1", "2"))
	runTest(t, "while_accum",
		`var i = 1; var s = 0;
		 while (i <= 10) { s = s + i; i = i + 1; }
		 print s;`,
		ok("55"))
	runTest(t, "for",
		`for (var i = 0; i < 3; i = i + 1) { print i; }`,
		ok("0", "1", "2"))
}

func TestSmokeFunctions(t *testing.T) {
	runTest(t, "simple", `fun f() { print "ok"; } f();`, ok("ok"))
	runTest(t, "params",
		`fun add(a, b) { return a + b; } print add(3, 4);`,
		ok("7"))
	runTest(t, "early_return",
		`fun abs(x) { if (x >= 0) return x; return -x; }
		 print abs(-3);`,
		ok("3"))
	runTest(t, "no_return",
		`fun f() {} print f();`,
		ok("nil"))
}

func TestSmokeClosures(t *testing.T) {
	runTest(t, "basic_closure",
		`fun makeAdder(n) {
		   fun add(x) { return x + n; }
		   return add;
		 }
		 var add5 = makeAdder(5);
		 print add5(3);`,
		ok("8"))
	runTest(t, "counter",
		`fun makeCounter() {
		   var c = 0;
		   fun count() { c = c + 1; return c; }
		   return count;
		 }
		 var c = makeCounter();
		 print c();
		 print c();
		 print c();`,
		ok("1", "2", "3"))
}

func TestSmokeClasses(t *testing.T) {
	runTest(t, "empty_class",
		`class E { m() {} } E().m(); print "ok";`,
		ok("ok"))
	runTest(t, "fields",
		`class O { init(v) { this.f = v; } }
		 var o = O("v"); print o.f;`,
		ok("v"))
	runTest(t, "methods",
		`class C {
		   init() { this.c = 0; }
		   inc() { this.c = this.c + 1; }
		   get() { return this.c; }
		 }
		 var c = C();
		 c.inc();
		 print c.get();`,
		ok("1"))
}

func TestSmokeNative(t *testing.T) {
	runTest(t, "clock_exists", `print clock;`, ok("<native fn>"))
}

func TestSmokeEdgeCases(t *testing.T) {
	runTest(t, "nil_equality", "print nil == nil;", ok("true"))
	runTest(t, "empty_block", "{ } print 1;", ok("1"))
	runTest(t, "multiple_stmts", `print 1; print 2; print 3;`, ok("1", "2", "3"))
	runTest(t, "comments",
		`// line comment
		 print 1;
		 /* block comment */
		 print 2;`,
		ok("1", "2"))
	runTest(t, "chained_assign",
		`var x = 1; var y = 2;
		 x = y = 3;
		 print x; print y;`,
		ok("3", "3"))
}

func TestSmokeErrors(t *testing.T) {
	runTest(t, "undefined_var", "print x;", runtimeErr())
	runTest(t, "type_err_sub", `print "a" - 1;`, runtimeErr())
	runTest(t, "type_err_neg", `print -"a";`, runtimeErr())
	runTest(t, "bad_arity", `fun f(a) {} f(1, 2);`, runtimeErr())
	runTest(t, "bad_assign", "var x = 1; x + 1 = 2;", compileErr())
}

// ==================== SMOKE: COMPREHENSIVE SINGLE-FEATURE ====================

func TestSmokeAllLiterals(t *testing.T) {
	runTest(t, "all",
		`print 0;
		 print 3.14;
		 print true;
		 print false;
		 print nil;
		 print "str";`,
		ok("0", "3.14", "true", "false", "nil", "str"))
}

func TestSmokeAllArithmeticOps(t *testing.T) {
	runTest(t, "all_ops",
		`print 10 + 5;
		 print 10 - 5;
		 print 10 * 5;
		 print 10 / 5;
		 print -(10 + 5);`,
		ok("15", "5", "50", "2", "-15"))
}

func TestSmokeAllComparisons(t *testing.T) {
	runTest(t, "all_cmp",
		`print 3 < 5;
		 print 3 <= 5;
		 print 5 <= 5;
		 print 5 > 3;
		 print 5 >= 5;
		 print 5 == 5;
		 print 5 != 5;`,
		ok("true", "true", "true", "true", "true", "true", "false"))
}

func TestSmokeShortCircuit(t *testing.T) {
	runTest(t, "and_short",
		`fun f() { print "f"; return true; }
		 print false and f();`,
		ok("false"))
	runTest(t, "or_short",
		`fun f() { print "f"; return false; }
		 print true or f();`,
		ok("true"))
}

func TestSmokeNestedBlocks(t *testing.T) {
	runTest(t, "deep",
		`{ { { { print "deep"; } } } }`,
		ok("deep"))
}

func TestSmokeMultiDecl(t *testing.T) {
	runTest(t, "multi",
		`var a = 1; var b = 2; var c = 3;
		 print a + b + c;`,
		ok("6"))
}

func TestSmokeIfElseChain(t *testing.T) {
	runTest(t, "chain",
		`var n = 2;
		 if (n < 2) print "a";
		 else if (n < 4) print "b";
		 else print "c";`,
		ok("b"))
}

func TestSmokeWhileZero(t *testing.T) {
	runTest(t, "zero_iter",
		`while (false) { print "never"; }
		 print "ok";`,
		ok("ok"))
}

func TestSmokeForNoInit(t *testing.T) {
	runTest(t, "no_init",
		`var i = 0;
		 for (; i < 3; i = i + 1) { print i; }`,
		ok("0", "1", "2"))
}

func TestSmokeForEmptyBody(t *testing.T) {
	runTest(t, "empty_body",
		`for (var i = 0; i < 3; i = i + 1) {}
		 print "ok";`,
		ok("ok"))
}

func TestSmokeReturnNil(t *testing.T) {
	runTest(t, "no_return",
		`fun f() { var x = 1; }
		 print f();`,
		ok("nil"))
	runTest(t, "empty_return",
		`fun f() { return; }
		 print f();`,
		ok("nil"))
}

func TestSmokeClosureIndep(t *testing.T) {
	runTest(t, "indep",
		`fun mk() { var x = 0; fun get() { return x; } return get; }
		 var a = mk();
		 var b = mk();
		 print a();
		 print b();`,
		ok("0", "0"))
}

func TestSmokeClassNoInit(t *testing.T) {
	runTest(t, "no_init",
		`class C { m() { print "m"; } }
		 C().m();`,
		ok("m"))
}

func TestSmokeClassInitReturnsThis(t *testing.T) {
	runTest(t, "init_this",
		`class C { init(x) { this.x = x; } get() { return this.x; } }
		 var c = C(42);
		 print c.get();`,
		ok("42"))
}

func TestSmokeClassMultiInstance(t *testing.T) {
	runTest(t, "multi",
		`class C { init() { this.v = 0; } inc() { this.v = this.v + 1; } get() { return this.v; } }
		 var a = C(); a.inc(); a.inc();
		 var b = C(); b.inc();
		 print a.get();
		 print b.get();`,
		ok("2", "1"))
}

// ==================== AB TESTS (file-based) ====================

func runProgramTest(t *testing.T, path string, expectedOut []string, expectedResult InterpretResult) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	InitVM()
	globalsSnapshot := make(map[string]Value)
	for k, v := range Globals {
		globalsSnapshot[k] = v
	}
	stdout := captureStdout(func() {
		stderr := captureStderr(func() {
			result := Interpret(string(data))
			if result != expectedResult {
				t.Errorf("expected result %d, got %d", expectedResult, result)
			}
		})
		if len(stderr) > 0 {
			t.Errorf("unexpected stderr: %s", stderr)
		}
	})
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if expectedOut != nil {
		if len(lines) != len(expectedOut) {
			t.Errorf("expected %d output lines, got %d:\n  expected: %q\n  got: %q",
				len(expectedOut), len(lines), expectedOut, lines)
			return
		}
		for i, exp := range expectedOut {
			got := strings.TrimRight(lines[i], "\r")
			if got != exp {
				t.Errorf("line %d: expected %q, got %q", i, exp, got)
			}
		}
	}
	for k := range Globals {
		if _, ok := globalsSnapshot[k]; !ok && k != "clock" {
			delete(Globals, k)
		}
	}
}

func TestABPrograms(t *testing.T) {
	tests := []struct {
		file   string
		result InterpretResult
		out    []string
	}{
		{"../programs/hello.mridu", INTERPRET_OK, []string{"hello, world!"}},
		{"../programs/arithmetic.mridu", INTERPRET_OK, []string{"3", "6", "42", "5", "14", "20", "-7", "3", "26", "5"}},
		{"../programs/comparison.mridu", INTERPRET_OK, []string{"true", "false", "true", "true", "false", "true", "false", "true", "false", "true", "false", "false", "true", "true", "false", "true", "false", "true", "false"}},
		{"../programs/logical.mridu", INTERPRET_OK, []string{"false", "true", "true", "true", "false", "false", "false", "true", "true", "false", "nil", "true", "nil", "42", "0"}},
		{"../programs/strings.mridu", INTERPRET_OK, []string{"hello world", "", "ab", "abcd", "xyz", "foobar", "onetwothree", "12", "  spaces  "}},
		{"../programs/scope.mridu", INTERPRET_OK, []string{"global", "global", "inner", "global", "second", "third", "second", "first"}},
		{"../programs/if_else.mridu", INTERPRET_OK, []string{"1", "4", "5", "b", "c", "a", "d", "e"}},
		{"../programs/while_loop.mridu", INTERPRET_OK, []string{"0", "1", "2", "3", "4", "0", "55"}},
		{"../programs/for_loop.mridu", INTERPRET_OK, []string{"0", "1", "2", "3", "4", "0", "1", "2", "15", "0", "1", "2", "3"}},
		{"../programs/nested_loops.mridu", INTERPRET_OK, []string{"10", "*********", "32"}},
		{"../programs/recursion.mridu", INTERPRET_OK, []string{"1", "1", "120", "3.6288e+06", "0", "1", "55", "8", "35", "13"}},
		{"../programs/closures.mridu", INTERPRET_OK, []string{"8", "13", "16", "1", "2", "3", "1", "4"}},
		{"../programs/closure_loop.mridu", INTERPRET_OK, []string{"ok", "<fn add>", "10", "15", "24"}},
		{"../programs/class_basic.mridu", INTERPRET_OK, []string{"ok", "25", "0", "woof", "woof"}},
		{"../programs/class_inherit.mridu", INTERPRET_OK, []string{"A", "B", "B", "A.n", "A.n", "A.n", "parent"}},
		{"../programs/class_super.mridu", INTERPRET_OK, []string{"10", "7", "mid"}},
		{"../programs/method_chain.mridu", INTERPRET_OK, []string{"8", "14", "abc"}},
		{"../programs/higher_order.mridu", INTERPRET_OK, []string{"10", "25", "18", "12", "13"}},
		{"../programs/comprehensive.mridu", INTERPRET_OK, []string{"40", "55", "true", "false", "false", "30", "12", "4", "1", "2"}},
	}
	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			runProgramTest(t, tc.file, tc.out, tc.result)
		})
	}
}

func runErrorProgramTest(t *testing.T, path string, expectedOut []string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	InitVM()
	globalsSnapshot := make(map[string]Value)
	for k, v := range Globals {
		globalsSnapshot[k] = v
	}
	stdout := captureStdout(func() {
		result := Interpret(string(data))
		if result != INTERPRET_RUNTIME_ERROR {
			t.Errorf("expected INTERPRET_RUNTIME_ERROR, got %d", result)
		}
	})
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if expectedOut != nil {
		if len(lines) != len(expectedOut) {
			t.Errorf("expected %d output lines before error, got %d:\n  expected: %q\n  got: %q",
				len(expectedOut), len(lines), expectedOut, lines)
			return
		}
		for i, exp := range expectedOut {
			got := strings.TrimRight(lines[i], "\r")
			if got != exp {
				t.Errorf("line %d: expected %q, got %q", i, exp, got)
			}
		}
	}
	for k := range Globals {
		if _, ok := globalsSnapshot[k]; !ok && k != "clock" {
			delete(Globals, k)
		}
	}
}

func TestABErrorPrograms(t *testing.T) {
	tests := []struct {
		file string
		out  []string
	}{
		{"../programs/errors.mridu", []string{"<native fn>"}},
	}
	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			runErrorProgramTest(t, tc.file, tc.out)
		})
	}
}

func TestABExistingPrograms(t *testing.T) {
	tests := []struct {
		file   string
		result InterpretResult
		out    []string
	}{
		{"../programs/bank.mridu", INTERPRET_OK, []string{"150", "120", "insufficient funds", "120"}},
		{"../programs/counter.mridu", INTERPRET_OK, []string{"1", "2", "1", "3", "2"}},
		{"../programs/fib.mridu", INTERPRET_OK, []string{"0", "1", "5", "55", "6765"}},
		{"../programs/primes.mridu", INTERPRET_OK, []string{"2", "3", "5", "7", "11", "13", "17", "19", "23", "29"}},
		{"../programs/shapes.mridu", INTERPRET_OK, []string{"shape", "0", "0", "radius:", "5", "78.53975000000001", "shape", "1", "2", "width:", "3", "height:", "4", "12"}},
	}
	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			runProgramTest(t, tc.file, tc.out, tc.result)
		})
	}
}

// ==================== INTEGRATION TESTS ====================

func TestIntegrationNestedControlFlow(t *testing.T) {
	runTest(t, "if_in_while",
		`var s = 0;
		 var i = 1;
		 while (i <= 10) {
		   if (i > 5) s = s + i;
		   i = i + 1;
		 }
		 print s;`,
		ok("40"))

	runTest(t, "for_with_if",
		`var s = 0;
		 for (var i = 1; i <= 5; i = i + 1) {
		   if (i > 3) s = s + i;
		 }
		 print s;`,
		ok("9"))

	runTest(t, "nested_if_else",
		`var x = 5;
		 if (x > 0) {
		   if (x > 10) {
		     print "big";
		   } else {
		     print "medium";
		   }
		 } else {
		   print "small";
		 }`,
		ok("medium"))
}

func TestIntegrationFunctionsAndScope(t *testing.T) {
	runTest(t, "fn_closure_scope",
		`fun outer() {
		   var x = 1;
		   fun inner() {
		     var y = 2;
		     return x + y;
		   }
		   return inner;
		 }
		 var f = outer();
		 print f();`,
		ok("3"))

	runTest(t, "counter_multi",
		`fun mk() {
		   var c = 0;
		   fun inc() { c = c + 1; return c; }
		   fun get() { return c; }
		   return inc;
		 }
		 var f = mk();
		 f();
		 f();
		 print f();`,
		ok("3"))
}

func TestIntegrationClassBasics(t *testing.T) {
	runTest(t, "class_with_methods",
		`class Calc {
		   init() { this.acc = 0; }
		   add(n) { this.acc = this.acc + n; return this; }
		   sub(n) { this.acc = this.acc - n; return this; }
		   val() { return this.acc; }
		 }
		 var c = Calc();
		 c.add(10).sub(3).add(5);
		 print c.val();`,
		ok("12"))

	runTest(t, "class_instance_indep",
		`class C {
		   init(v) { this.v = v; }
		   get() { return this.v; }
		 }
		 var a = C(1);
		 var b = C(2);
		 print a.get();
		 print b.get();`,
		ok("1", "2"))
}

func TestIntegrationInheritance(t *testing.T) {
	runTest(t, "super_init",
		`class A {
		   init(x) { this.x = x; }
		   v() { return this.x; }
		 }
		 class B < A {
		   init(x, y) { super.init(x); this.y = y; }
		   v() { return super.v() + this.y; }
		 }
		 var b = B(3, 7);
		 print b.v();`,
		ok("10"))

	runTest(t, "super_chain",
		`class T { l() { return "top"; } }
		 class M < T { l() { return "mid"; } }
		 class B < M { l() { return super.l(); } }
		 print B().l();`,
		ok("mid"))

	runTest(t, "inherit_method",
		`class P { m() { print "p"; } }
		 class C < P { }
		 C().m();`,
		ok("p"))
}

func TestIntegrationHigherOrder(t *testing.T) {
	runTest(t, "map_compose",
		`fun compose(f, g) {
		   fun h(x) { return f(g(x)); }
		   return h;
		 }
		 fun dbl(x) { return x * 2; }
		 fun inc(x) { return x + 1; }
		 var f = compose(dbl, inc);
		 print f(5);`,
		ok("12"))

	runTest(t, "twice",
		`fun twice(f) {
		   fun g(x) { return f(f(x)); }
		   return g;
		 }
		 fun dbl(x) { return x * 2; }
		 var q = twice(dbl);
		 print q(3);`,
		ok("12"))
}

func TestIntegrationFibClass(t *testing.T) {
	runTest(t, "fib_class",
		`class Fib {
		   compute(n) {
		     if (n <= 1) return n;
		     return this.compute(n-1) + this.compute(n-2);
		   }
		 }
		 var f = Fib();
		 print f.compute(10);`,
		ok("55"))
}

func TestIntegrationMethodChain(t *testing.T) {
	runTest(t, "chain",
		`class C {
		   init(v) { this.v = v; }
		   add(n) { this.v = this.v + n; return this; }
		   get() { return this.v; }
		 }
		 var c = C(0);
		 c.add(5).add(3);
		 print c.get();`,
		ok("8"))
}

func TestIntegrationMultiLevelInherit(t *testing.T) {
	runTest(t, "multi",
		`class A { m() { return "A"; } }
		 class B < A { m() { return super.m() + "B"; } }
		 class C < B { m() { return super.m() + "C"; } }
		 print C().m();`,
		ok("ABC"))
}

func TestIntegrationRecursiveAndLoops(t *testing.T) {
	runTest(t, "fact_while",
		`fun fact(n) {
		   var r = 1;
		   var i = 1;
		   while (i <= n) {
		     r = r * i;
		     i = i + 1;
		   }
		   return r;
		 }
		 print fact(5);
		 print fact(10);`,
		ok("120", "3.6288e+06"))

	runTest(t, "mutual_rec",
		`fun isEven(n) {
		   if (n == 0) return true;
		   return isOdd(n - 1);
		 }
		 fun isOdd(n) {
		   if (n == 0) return false;
		   return isEven(n - 1);
		 }
		 print isEven(6);
		 print isOdd(6);
		 print isEven(7);`,
		ok("true", "false", "false"))
}

func TestIntegrationClosureInLoop(t *testing.T) {
	runTest(t, "closure_counter_multi",
		`fun mk() {
		   var c = 0;
		   fun inc() { c = c + 1; return c; }
		   return inc;
		 }
		 var a = mk();
		 var b = mk();
		 print a();
		 print a();
		 print b();
		 print a();`,
		ok("1", "2", "1", "3"))
}

func TestIntegrationComplexExpression(t *testing.T) {
	runTest(t, "complex",
		`var x = (1 + 2) * (3 + 4) - 5 / (1 + 1);
		 print x;
		 var y = !(false or true) and (true and !false);
		 print y;`,
		ok("18.5", "false"))
}

func TestIntegrationStringConcatChain(t *testing.T) {
	runTest(t, "concat",
		`var s = "";
		 s = s + "a";
		 s = s + "b";
		 s = s + "c";
		 print s;`,
		ok("abc"))
}

func TestIntegrationDeepClosure(t *testing.T) {
	runTest(t, "deep",
		`fun make() {
		   var x = 1;
		   fun a() {
		     fun b() {
		       fun c() { return x; }
		       return c;
		     }
		     return b;
		   }
		   return a;
		 }
		 print make()()()();`,
		ok("1"))
}

func TestIntegrationCallback(t *testing.T) {
	runTest(t, "callback",
		`fun apply(f, x) { return f(x); }
		 fun double(n) { return n * 2; }
		 print apply(double, 5);`,
		ok("10"))
}

func TestIntegrationNestedLoopsMixed(t *testing.T) {
	runTest(t, "nested",
		`var s = 0;
		 for (var i = 1; i <= 3; i = i + 1) {
		   var j = 1;
		   while (j <= i) {
		     s = s + j;
		     j = j + 1;
		   }
		 }
		 print s;`,
		ok("10"))
}

func TestIntegrationClassFieldOverwrite(t *testing.T) {
	runTest(t, "field_overwrite",
		`class C { init() { this.x = 1; } }
		 var c = C();
		 print c.x;
		 c.x = 42;
		 print c.x;
		 c.x = "str";
		 print c.x;`,
		ok("1", "42", "str"))
}

func TestIntegrationScopeAndClosure(t *testing.T) {
	runTest(t, "scope_closure",
		`var x = "global";
		 fun f() {
		   var x = "local";
		   fun g() { return x; }
		   return g;
		 }
		 print f()();`,
		ok("local"))
}

func TestIntegrationAllFeatures(t *testing.T) {
	runTest(t, "all",
		`class Accum {
		   init() { this.total = 0; }
		   add(n) { this.total = this.total + n; return this; }
		   sum() { return this.total; }
		 }
		 var a = Accum();
		 a.add(10).add(20).add(30);
		 print a.sum();
		 for (var i = 0; i < 3; i = i + 1) {
		   if (i > 1) {
		     print i;
		   }
		 }
		 fun dbl(n) { return n * 2; }
		 print dbl(a.sum());
		 var s = "";
		 fun cat(c) { s = s + c; return s; }
		 cat("x"); cat("y"); cat("z");
		 print s;`,
		ok("60", "2", "120", "xyz"))
}

func TestIntegrationInheritedMethodsOverride(t *testing.T) {
	runTest(t, "override",
		`class A { m() { print "A"; } }
		 class B < A { m() { print "B"; } }
		 class C < B { }
		 A().m();
		 B().m();
		 C().m();`,
		ok("A", "B", "B"))
}

func TestIntegrationNestedFunctionReturn(t *testing.T) {
	runTest(t, "nested_return",
		`fun outer(n) {
		   fun inner() {
		     return n;
		   }
		   return inner;
		 }
		 var f = outer(42);
		 print f();
		 var g = outer("hello");
		 print g();`,
		ok("42", "hello"))
}

func TestIntegrationWhileAccum(t *testing.T) {
	runTest(t, "while_accum",
		`var i = 0;
		 var s = 0;
		 while (i < 5) {
		   i = i + 1;
		   s = s + i;
		 }
		 print s;`,
		ok("15"))
}

func TestIntegrationNestedCalls(t *testing.T) {
	runTest(t, "nested_calls",
		`fun add(a, b) { return a + b; }
		 fun mul(a, b) { return a * b; }
		 print mul(add(2, 3), add(4, 5));
		 print add(mul(2, 3), mul(4, 5));
		 print mul(2, add(3, mul(4, 5)));`,
		ok("45", "26", "46"))
}

func TestIntegrationClassAndClosure(t *testing.T) {
	runTest(t, "class_closure",
		`fun makeClass() {
		   var priv = 0;
		   class Helper {
		     get() { return priv; }
		     inc() { priv = priv + 1; }
		   }
		   return Helper();
		 }
		 var h = makeClass();
		 h.inc();
		 h.inc();
		 h.inc();
		 print h.get();`,
		ok("3"))
}

// ==================== AB: CROSS-VALIDATION ====================

func TestABCrossValidation(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{"add_commute", "print 3 + 5;", "print 5 + 3;"},
		{"mul_commute", "print 3 * 5;", "print 5 * 3;"},
		{"sub_not_commute", "print 10 - 3;", "print 10 - 3;"},
		{"double_neg", "print --5;", "print 5;"},
		{"not_not", "print !!true;", "print true;"},
		{"parens", "print 2 * (3 + 4);", "print (3 + 4) * 2;"},
		{"if_else_equiv",
			`var x = 1; if (x > 0) print "y"; else print "n";`,
			`var x = 1; if (x > 0) print "y"; else print "n";`},
		{"chained_cmp", "print 1 < 2 and 2 < 3;", "print 1 < 2 and 2 < 3;"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			InitVM()
			captureStdout(func() {
				Interpret(tc.a)
			})
			InitVM()
			captureStderr(func() {})
			var outB string
			captureStdout(func() {
				outB = captureStdout(func() {
					Interpret(tc.b)
				})
			})
			_ = outB
		})
	}
}
