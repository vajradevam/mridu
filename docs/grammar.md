# mridu Grammar

## Notation

- `|` alternation
- `*` zero or more
- `+` one or more
- `?` optional
- `"..."` literal token
- `/* ... */` comment

## Lexical Grammar

```
letter        ::= "a"..."z" | "A"..."Z" | "_"
digit         ::= "0"..."9"
identifier    ::= letter (letter | digit)*
integer       ::= digit+
float         ::= digit+ "." digit+
number        ::= integer | float
string        ::= '"' char* '"'         /* no escape sequences */
char          ::= any Unicode byte except '"' and '\n'
newline       ::= '\n'
whitespace    ::= (' ' | '\t' | '\r' | newline)+
comment       ::= "//" any_byte* newline
block_comment ::= "/*" (block_comment | any_byte)* "*/"
              /* supports nesting */
```

## Keywords

```
and    class  else   false  for    fun    if
nil    or     print  return super  this   true
var    while
```

## Punctuation

```
(       )       {       }       ,       .
-       +       ;       /       *
!       !=      =       ==      <       <=
>       >=
```

## Syntactic Grammar

```
program        ::= declaration*
declaration    ::= class_decl
                 | fun_decl
                 | var_decl
                 | statement

class_decl     ::= "class" identifier ("<" identifier)?
                   "{" method* "}"

fun_decl       ::= "fun" function

function       ::= identifier "(" parameters? ")" block

parameters     ::= identifier ("," identifier)*

method         ::= identifier "(" parameters? ")" block

var_decl       ::= "var" identifier ("=" expression)? ";"

statement      ::= expr_stmt
                 | print_stmt
                 | block_stmt
                 | if_stmt
                 | while_stmt
                 | for_stmt
                 | return_stmt

expr_stmt      ::= expression ";"

print_stmt     ::= "print" expression ";"

block_stmt     ::= "{" declaration* "}"

if_stmt        ::= "if" "(" expression ")" statement
                   ("else" statement)?

while_stmt     ::= "while" "(" expression ")" statement

for_stmt       ::= "for" "("
                   (var_decl | expr_stmt | ";")
                   expression? ";"
                   expression?
                   ")" statement

return_stmt    ::= "return" expression? ";"

expression     ::= assignment

assignment     ::= (call ".")? identifier "=" assignment
                 | logic_or

logic_or       ::= logic_and ("or" logic_and)*

logic_and      ::= equality ("and" equality)*

equality       ::= comparison (("==" | "!=") comparison)*

comparison     ::= term ((">" | ">=" | "<" | "<=") term)*

term           ::= factor (("-" | "+") factor)*

factor         ::= unary (("/" | "*") unary)*

unary          ::= ("!" | "-") unary
                 | call

call           ::= primary ("(" arguments? ")"
                            | "." identifier
                            | "." identifier "(" arguments? ")")*

arguments      ::= expression ("," expression)*

primary        ::= "true"
                 | "false"
                 | "nil"
                 | "this"
                 | "super" "." identifier
                 | "super" "." identifier "(" arguments? ")"
                 | number
                 | string
                 | identifier
                 | "(" expression ")"
                 | "(" expression ")" arguments
                 /* class instantiation: ClassName(args) */
```

## Precedence Table (lowest to highest)

| Precedence | Associativity | Operators                  |
|-----------|---------------|----------------------------|
| Assignment | Right       | `=`                        |
| Or        | Left         | `or`                       |
| And       | Left         | `and`                      |
| Equality  | Left         | `==` `!=`                  |
| Comparison| Left         | `<` `<=` `>` `>=`          |
| Term      | Left         | `+` `-`                    |
| Factor    | Left         | `*` `/`                    |
| Unary     | Right        | `!` `-`                    |
| Call      | Left         | `()` `.` `.method()`       |
| Primary   | —            | literals, grouping         |

## Limits

- **Parameters**: max 255 per function/method
- **Arguments**: max 255 per call
- **Stack slots**: 256 (`STACK_MAX`)
- **Call frames**: 64 (`FRAMES_MAX`)
- **Jump offset**: max 65535 bytes
- **Loop offset**: max 65535 bytes
- **Identifier characters**: `[a-zA-Z_]` only
- **No escape sequences** in strings
