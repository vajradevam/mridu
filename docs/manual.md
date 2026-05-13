# mridu Language Manual

mridu is a dynamically-typed, object-oriented scripting language with first-class functions, closures, and classes. It runs on a bytecode virtual machine.

## 1. Running mridu

```
mridu [script]
```

- **REPL mode**: `mridu` — interactive prompt (`>`).
- **File mode**: `mridu path/to/script.mridu` — run a script.

Exit codes: `0` OK, `65` compile error, `70` runtime error, `74` file not found.

## 2. Comments

```
// Line comment

/* Block comment */
/* Nested /* block */ comments supported */
```

## 3. Data Types

### 3.1 Numbers

IEEE 754 double-precision floating point. Written as integers or decimals:

```
42
0
-3.14
0.5
100.0
```

### 3.2 Strings

Double-quoted, no escape sequences:

```
"hello"
""
"multi-
line"
```

Concatenation with `+`:

```
"a" + "b"  →  "ab"
```

### 3.3 Booleans

```
true
false
```

### 3.4 Nil

```
nil
```

Represents absence of a value. Falsy in conditionals (along with `false`).

### 3.5 Truthiness

Only `false` and `nil` are falsy. All other values — numbers (including `0`), strings (including `""`), functions, classes, instances — are truthy.

## 4. Variables

### 4.1 Declaration

```
var name;           // initialized to nil
var name = value;
```

### 4.2 Assignment

```
name = new_value;
```

### 4.3 Scope

Variables are lexically scoped. Inner scopes can shadow outer names.

```
var x = "global";
{
  var x = "shadow";
  print x;   // "shadow"
}
print x;     // "global"
```

Global variables are stored in a global hash map. Local variables live on the VM stack.

## 5. Expressions

### 5.1 Arithmetic

```
+    add (also string concatenation)
-    subtract (also unary negate)
*    multiply
/    divide (runtime error on division by zero)
```

Arithmetic operators require number operands (except `+` which accepts two strings).

### 5.2 Comparison

```
==    equal
!=    not equal
<     less than
<=    less than or equal
>     greater than
>=    greater than or equal
```

Comparison operators (`<`, `<=`, `>`, `>=`) require number operands. `==` and `!=` work on any type (value equality).

Equality rules:
- Same type, same value → equal
- Different types → not equal
- Strings compared by value
- Objects (functions, classes, instances) compared by identity (pointer)

### 5.3 Logical

```
!    logical not
and  logical and (short-circuit)
or   logical or  (short-circuit)
```

`and` returns the first falsy operand or the last operand. `or` returns the first truthy operand or the last operand.

```
true and false  → false
nil and true    → nil
false or 42     → 42
nil or true     → true
```

### 5.4 Grouping

Parentheses override precedence:

```
(2 + 3) * 4  →  20
```

### 5.5 Precedence (highest to lowest)

| Associativity | Operators        |
|---------------|------------------|
| Right         | `=`              |
| Left          | `or`             |
| Left          | `and`            |
| Left          | `==` `!=`        |
| Left          | `<` `<=` `>` `>=`|
| Left          | `+` `-`          |
| Left          | `*` `/`          |
| Right         | `!` `-`          |
| Left          | `()` `.`         |

## 6. Statements

### 6.1 Expression Statement

```
expression;
```

The value is discarded.

### 6.2 Print

```
print expression;
```

Prints the value to stdout followed by a newline.

### 6.3 Blocks

```
{
  declarations...
}
```

Creates a new lexical scope.

### 6.4 If

```
if (condition) statement;
if (condition) statement1; else statement2;
```

The condition is truthiness-tested. The `else` clause binds to the nearest `if`.

```
if (n < 2) print "a";
else if (n < 4) print "b";
else print "c";
```

### 6.5 While

```
while (condition) statement;
```

### 6.6 For

```
for (initializer; condition; increment) statement;
```

All three clauses are optional:

```
for (; i < 10; i = i + 1) {}
for (;;) { }       // infinite loop
for (var i = 0; i < 5; i = i + 1) { print i; }
```

The initializer can be a `var` declaration or an expression statement.

### 6.7 Return

```
return;
return expression;
```

Only allowed inside functions. `return` with no value returns `nil`. In an `init` method (constructor), returning a value is an error.

## 7. Functions

### 7.1 Declaration

```
fun functionName(param1, param2, ...) {
  body...
}
```

Up to 255 parameters.

### 7.2 Calling

```
functionName(arg1, arg2, ...);
```

Argument count must match parameter count exactly (runtime error on mismatch).

### 7.3 Return Value

Functions return `nil` by default (if execution falls off the end).

```
fun f() {}    // returns nil
fun g() { return 42; }
```

### 7.4 First-Class

Functions are objects that can be passed around:

```
fun apply(f, x) { return f(x); }
fun double(n) { return n * 2; }
print apply(double, 5);   // 10
```

## 8. Closures

Functions capture the surrounding lexical scope. Captured variables survive even after the enclosing function returns.

```
fun makeCounter() {
  var count = 0;
  fun inc() {
    count = count + 1;
    return count;
  }
  return inc;
}

var c = makeCounter();
print c();   // 1
print c();   // 2
print c();   // 3
```

Multiple closures can independently capture the same variable or different instances of it.

## 9. Classes

### 9.1 Declaration

```
class ClassName {
  method1() { ... }
  method2(param) { ... }
}
```

### 9.2 Instantiation

```
var obj = ClassName();
```

Calling a class creates a new instance. If the class defines an `init` method, it is called with the provided arguments.

### 9.3 Constructor (init)

```
class Point {
  init(x, y) {
    this.x = x;
    this.y = y;
  }
}
```

The `init` method is called automatically on instantiation. It cannot return a value. The instance is the result of the class call regardless of what `init` returns.

### 9.4 Methods

Methods defined on a class are shared by all instances. Calling a method binds `this` to the receiver:

```
class Calc {
  init() { this.acc = 0; }
  add(n) { this.acc = this.acc + n; return this; }
  val()  { return this.acc; }
}

var c = Calc();
c.add(10).add(5);
print c.val();   // 15
```

Methods can be chained when they return `this`.

### 9.5 Fields

Instance fields are dynamically created by assignment to `this`:

```
this.name = value;
```

Reading an undefined field is a runtime error.

### 9.6 This

`this` refers to the current instance. Can only be used inside methods.

## 10. Inheritance

### 10.1 Single Inheritance

```
class Subclass < Superclass {
  ...
}
```

The subclass inherits all methods from the superclass. Overridden methods shadow the superclass version.

### 10.2 Super Calls

Use `super.method(...)` to call an overridden method on the superclass:

```
class A {
  init(x) { this.x = x; }
  v() { return this.x; }
}
class B < A {
  init(x, y) { super.init(x); this.y = y; }
  v() { return super.v() + this.y; }
}
```

### 10.3 Self-Inheritance

A class cannot inherit from itself: `class C < C {}` is a compile error.

## 11. Native Functions

### 11.1 clock

```
clock
```

Returns the current time in seconds as a float (millisecond precision). Available globally.

## 12. Limits

| Limit | Value |
|-------|-------|
| Max function/method parameters | 255 |
| Max call arguments | 255 |
| VM stack slots | 256 |
| Max call frames (stack depth) | 64 |
| Jump/loop offset | 65535 bytes |
| Identifier characters | `[a-zA-Z_]` only |
| String escape sequences | none |

## 13. Error Handling

### 13.1 Compile Errors

Common compile errors:
- `Expect expression.` — missing expression
- `Expect ';' after value.` — missing semicolon
- `Expect ')' after expression.` — unbalanced parens
- `Expect '{' before function body.` — missing brace
- `Cannot return from top-level code.` — `return` outside function
- `Cannot return a value from an initializer.` — value return from `init`
- `Cannot use 'this' outside of a class.` — `this` in global/function scope
- `Cannot use 'super' outside of a class.` — `super` in global/function scope
- `Cannot use 'super' in a class with no superclass.` — `super` without inheritance
- `Invalid assignment target.` — assigning to non-assignable expression
- `Variable with this name already declared in this scope.` — duplicate local variable
- `A class cannot inherit from itself.` — self-inheritance

### 13.2 Runtime Errors

Common runtime errors:
- `Undefined variable 'x'.` — reading/writing an undefined global
- `Operands must be numbers.` — arithmetic/comparison on wrong types
- `Operands must be two numbers or two strings.` — `+` on mixed types
- `Division by zero.` — division by zero
- `Can only call functions and classes.` — calling non-callable value
- `Expected N arguments but got M.` — argument count mismatch
- `Only instances have properties.` — accessing `.` on non-instance
- `Only instances have fields.` — setting `.` on non-instance
- `Undefined property 'x'.` — accessing undefined field/method
- `Stack overflow.` — call frame limit (64) exceeded

## 14. Differences from Other Languages

- No `break` or `continue` in loops.
- No `switch` statement.
- No array/list types.
- No increment (`++`) or decrement (`--`) operators.
- No compound assignment (`+=`, `-=`, etc.).
- No escape sequences in string literals — backslash is literal.
- No ternary operator (`? :`).
- No module/import system.
- No garbage collection.
- Numbers are always double-precision floats (no integer type).

## 15. Example

```
fun fib(n) {
  if (n <= 1) return n;
  return fib(n - 1) + fib(n - 2);
}

class Accum {
  init() { this.total = 0; }
  add(n) { this.total = this.total + n; return this; }
  sum()  { return this.total; }
}

var a = Accum();
a.add(5).add(10).add(15);
print a.sum();          // 30

print fib(10);          // 55

for (var i = 0; i < 5; i = i + 1) {
  if (i > 2) print i;   // 3, 4
}
```
