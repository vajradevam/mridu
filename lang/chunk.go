package lang

type OpCode byte

const (
	OP_CONSTANT OpCode = iota
	OP_NIL
	OP_TRUE
	OP_FALSE
	OP_POP
	OP_DUP
	OP_DEFINE_GLOBAL
	OP_GET_GLOBAL
	OP_SET_GLOBAL
	OP_GET_LOCAL
	OP_SET_LOCAL
	OP_GET_UPVALUE
	OP_SET_UPVALUE
	OP_CLOSE_UPVALUE
	OP_EQUAL
	OP_GREATER
	OP_LESS
	OP_ADD
	OP_SUBTRACT
	OP_MULTIPLY
	OP_DIVIDE
	OP_NOT
	OP_NEGATE
	OP_PRINT
	OP_JUMP
	OP_JUMP_IF_FALSE
	OP_LOOP
	OP_CALL
	OP_CLOSURE
	OP_RETURN
	OP_CLASS
	OP_GET_PROPERTY
	OP_SET_PROPERTY
	OP_METHOD
	OP_INVOKE
	OP_INHERIT
	OP_GET_SUPER
	OP_SUPER_INVOKE
)

type Chunk struct {
	code      []byte
	constants []Value
	lines     []int
}

func NewChunk() *Chunk {
	return &Chunk{}
}

func (c *Chunk) Write(b byte, line int) {
	c.code = append(c.code, b)
	c.lines = append(c.lines, line)
}

func (c *Chunk) Write16(val uint16, line int) {
	c.Write(byte(val>>8), line)
	c.Write(byte(val), line)
}

func (c *Chunk) AddConstant(v Value) int {
	c.constants = append(c.constants, v)
	return len(c.constants) - 1
}

func (c *Chunk) Read16(ip int) uint16 {
	return uint16(c.code[ip])<<8 | uint16(c.code[ip+1])
}

func (c *Chunk) GetLine(ip int) int {
	if ip >= 0 && ip < len(c.lines) {
		return c.lines[ip]
	}
	return -1
}
