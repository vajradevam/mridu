package lang

import (
	"fmt"
	"os"
	"time"
	"unsafe"
)

const STACK_MAX = 256
const FRAMES_MAX = 64

type CallFrame struct {
	closure *Object
	ip      int
	slots   int // start index in the VM stack
}

type VM struct {
	frames      [FRAMES_MAX]CallFrame
	frameCount  int
	stack       [STACK_MAX]Value
	sp          int // stack pointer (next free slot)
	openUpvalues *Object // linked list of open upvalues
}

var vm VM

func InitVM() {
	vm.frameCount = 0
	vm.sp = 0
	vm.openUpvalues = nil

	// Define native functions
	Globals["clock"] = ObjVal(NewObjNative(clockNative))
}

func push(value Value) {
	vm.stack[vm.sp] = value
	vm.sp++
}

func pop() Value {
	vm.sp--
	return vm.stack[vm.sp]
}

func peek(distance int) Value {
	return vm.stack[vm.sp-1-distance]
}

func resetStack() {
	vm.sp = 0
	vm.frameCount = 0
	vm.openUpvalues = nil
}

func runtimeError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintln(os.Stderr)
	for i := vm.frameCount - 1; i >= 0; i-- {
		frame := &vm.frames[i]
		closure := frame.closure
		fn := closure.closureFn
		ip := frame.ip - 1
		line := fn.funcChunk.GetLine(ip)
		name := fn.funcName
		if name == "" {
			fmt.Fprintf(os.Stderr, "[line %d] in script\n", line)
		} else {
			fmt.Fprintf(os.Stderr, "[line %d] in %s()\n", line, name)
		}
	}
	resetStack()
}

func isFalsey(value Value) bool {
	return IS_NIL(value) || (IS_BOOL(value) && !AS_BOOL(value))
}

func clockNative(argCount int, args []Value) Value {
	return NumberVal(float64(time.Now().UnixMilli()) / 1000.0)
}

func run() InterpretResult {
	frame := &vm.frames[vm.frameCount-1]

	for {
		// Debug: uncomment to trace
		// disassembleInstruction(&frame.closure.closureFn.funcChunk, frame.ip)

		switch OpCode(frame.closure.closureFn.funcChunk.code[frame.ip]) {

		case OP_CONSTANT: {
			constant := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			frame.ip += 3
			push(frame.closure.closureFn.funcChunk.constants[constant])
			break
		}

		case OP_NIL:
			push(NilVal())
			frame.ip++
			break

		case OP_TRUE:
			push(BoolVal(true))
			frame.ip++
			break

		case OP_FALSE:
			push(BoolVal(false))
			frame.ip++
			break

		case OP_POP:
			pop()
			frame.ip++
			break

		case OP_DUP: {
			val := peek(0)
			push(val)
			frame.ip++
			break
		}

		case OP_GET_LOCAL: {
			slot := int(frame.closure.closureFn.funcChunk.code[frame.ip+1])
			frame.ip += 2
			push(vm.stack[frame.slots+slot])
			break
		}

		case OP_SET_LOCAL: {
			slot := int(frame.closure.closureFn.funcChunk.code[frame.ip+1])
			frame.ip += 2
			vm.stack[frame.slots+slot] = peek(0)
			break
		}

		case OP_DEFINE_GLOBAL: {
			constant := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			frame.ip += 3
			name := AS_CSTRING(frame.closure.closureFn.funcChunk.constants[constant])
			Globals[name] = peek(0)
			pop()
			break
		}

		case OP_SET_GLOBAL: {
			constant := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			frame.ip += 3
			name := AS_CSTRING(frame.closure.closureFn.funcChunk.constants[constant])
			_, ok := Globals[name]
			if !ok {
				runtimeError("Undefined variable '%s'.", name)
				return INTERPRET_RUNTIME_ERROR
			}
			Globals[name] = peek(0)
			break
		}

		case OP_GET_GLOBAL: {
			constant := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			frame.ip += 3
			name := AS_CSTRING(frame.closure.closureFn.funcChunk.constants[constant])
			val, ok := Globals[name]
			if !ok {
				runtimeError("Undefined variable '%s'.", name)
				return INTERPRET_RUNTIME_ERROR
			}
			push(val)
			break
		}

		case OP_GET_UPVALUE: {
			slot := int(frame.closure.closureFn.funcChunk.code[frame.ip+1])
			frame.ip += 2
			upvalue := frame.closure.upvaluePtr[slot]
			if upvalue.upvalueLoc != nil {
				push(*upvalue.upvalueLoc)
			} else {
				push(upvalue.upvalueClosed)
			}
			break
		}

		case OP_SET_UPVALUE: {
			slot := int(frame.closure.closureFn.funcChunk.code[frame.ip+1])
			frame.ip += 2
			upvalue := frame.closure.upvaluePtr[slot]
			if upvalue.upvalueLoc != nil {
				*upvalue.upvalueLoc = peek(0)
			} else {
				upvalue.upvalueClosed = peek(0)
			}
			break
		}

		case OP_CLOSE_UPVALUE: {
			closeUpvalues(&vm.stack[vm.sp-1])
			pop()
			frame.ip++
			break
		}

		case OP_EQUAL: {
			b := pop()
			a := pop()
			push(BoolVal(ValuesEqual(a, b)))
			frame.ip++
			break
		}

		case OP_GREATER: {
			b := pop()
			a := pop()
			if !IS_NUMBER(a) || !IS_NUMBER(b) {
				runtimeError("Operands must be numbers.")
				return INTERPRET_RUNTIME_ERROR
			}
			push(BoolVal(AS_NUMBER(a) > AS_NUMBER(b)))
			frame.ip++
			break
		}

		case OP_LESS: {
			b := pop()
			a := pop()
			if !IS_NUMBER(a) || !IS_NUMBER(b) {
				runtimeError("Operands must be numbers.")
				return INTERPRET_RUNTIME_ERROR
			}
			push(BoolVal(AS_NUMBER(a) < AS_NUMBER(b)))
			frame.ip++
			break
		}

		case OP_ADD: {
			b := pop()
			a := pop()
			if IS_NUMBER(a) && IS_NUMBER(b) {
				push(NumberVal(AS_NUMBER(a) + AS_NUMBER(b)))
			} else if IS_STRING(a) && IS_STRING(b) {
				push(ObjVal(NewObjString(AS_CSTRING(a) + AS_CSTRING(b))))
			} else {
				runtimeError("Operands must be two numbers or two strings.")
				return INTERPRET_RUNTIME_ERROR
			}
			frame.ip++
			break
		}

		case OP_SUBTRACT: {
			b := pop()
			a := pop()
			if !IS_NUMBER(a) || !IS_NUMBER(b) {
				runtimeError("Operands must be numbers.")
				return INTERPRET_RUNTIME_ERROR
			}
			push(NumberVal(AS_NUMBER(a) - AS_NUMBER(b)))
			frame.ip++
			break
		}

		case OP_MULTIPLY: {
			b := pop()
			a := pop()
			if !IS_NUMBER(a) || !IS_NUMBER(b) {
				runtimeError("Operands must be numbers.")
				return INTERPRET_RUNTIME_ERROR
			}
			push(NumberVal(AS_NUMBER(a) * AS_NUMBER(b)))
			frame.ip++
			break
		}

		case OP_DIVIDE: {
			b := pop()
			a := pop()
			if !IS_NUMBER(a) || !IS_NUMBER(b) {
				runtimeError("Operands must be numbers.")
				return INTERPRET_RUNTIME_ERROR
			}
			if AS_NUMBER(b) == 0 {
				runtimeError("Division by zero.")
				return INTERPRET_RUNTIME_ERROR
			}
			push(NumberVal(AS_NUMBER(a) / AS_NUMBER(b)))
			frame.ip++
			break
		}

		case OP_NOT:
			push(BoolVal(isFalsey(pop())))
			frame.ip++
			break

		case OP_NEGATE: {
			val := peek(0)
			if !IS_NUMBER(val) {
				runtimeError("Operand must be a number.")
				return INTERPRET_RUNTIME_ERROR
			}
			push(NumberVal(-AS_NUMBER(pop())))
			frame.ip++
			break
		}

		case OP_PRINT: {
			fmt.Println(peek(0).String())
			pop()
			frame.ip++
			break
		}

		case OP_JUMP: {
			offset := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			frame.ip += offset + 3
			break
		}

		case OP_JUMP_IF_FALSE: {
			offset := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			frame.ip += 3
			if isFalsey(peek(0)) {
				frame.ip += offset
			}
			break
		}

		case OP_LOOP: {
			offset := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			frame.ip -= offset
			frame.ip += 3
			break
		}

		case OP_CALL: {
			argCount := int(frame.closure.closureFn.funcChunk.code[frame.ip+1])
			frame.ip += 2
			if !callValue(peek(argCount), argCount) {
				return INTERPRET_RUNTIME_ERROR
			}
			frame = &vm.frames[vm.frameCount-1]
			break
		}

		case OP_CLOSURE: {
			constant := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			frame.ip += 3
			fn := AS_FUNCTION(frame.closure.closureFn.funcChunk.constants[constant])
			closure := NewObjClosure(fn)
			push(ObjVal(closure))

			for i := 0; i < fn.funcUpvalues; i++ {
				isLocal := frame.closure.closureFn.funcChunk.code[frame.ip]
				index := int(frame.closure.closureFn.funcChunk.code[frame.ip+1])
				frame.ip += 2

				if isLocal == 1 {
					closure.upvaluePtr[i] = captureUpvalue(&vm.stack[frame.slots+index])
				} else {
					closure.upvaluePtr[i] = frame.closure.upvaluePtr[index]
				}
			}
			break
		}

		case OP_RETURN: {
			result := pop()
			closeUpvalues(&vm.stack[frame.slots])
			vm.frameCount--
			if vm.frameCount == 0 {
				pop()
				return INTERPRET_OK
			}
			vm.sp = frame.slots
			push(result)
			frame = &vm.frames[vm.frameCount-1]
			break
		}

		case OP_CLASS: {
			constant := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			frame.ip += 3
			name := AS_CSTRING(frame.closure.closureFn.funcChunk.constants[constant])
			klass := NewObjClass(name)
			push(ObjVal(klass))
			break
		}

		case OP_GET_PROPERTY: {
			constant := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			frame.ip += 3
			name := AS_CSTRING(frame.closure.closureFn.funcChunk.constants[constant])
			val := peek(0)
			if !IS_INSTANCE(val) {
				runtimeError("Only instances have properties.")
				return INTERPRET_RUNTIME_ERROR
			}
			instance := AS_INSTANCE(val)
			if prop, ok := instance.fields[name]; ok {
				pop() // instance
				push(prop)
			} else {
				method := bindMethod(instance.instClass, name)
				if method == nil {
					runtimeError("Undefined property '%s'.", name)
					return INTERPRET_RUNTIME_ERROR
				}
				pop() // instance
				push(ObjVal(method))
			}
			break
		}

		case OP_SET_PROPERTY: {
			constant := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			frame.ip += 3
			name := AS_CSTRING(frame.closure.closureFn.funcChunk.constants[constant])
			val := peek(1) // instance is below value
			if !IS_INSTANCE(val) {
				runtimeError("Only instances have fields.")
				return INTERPRET_RUNTIME_ERROR
			}
			instance := AS_INSTANCE(val)
			instance.fields[name] = peek(0)
			val = pop() // value
			pop()        // instance
			push(val)
			break
		}

		case OP_METHOD: {
			constant := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			frame.ip += 3
			name := AS_CSTRING(frame.closure.closureFn.funcChunk.constants[constant])
			method := AS_CLOSURE(peek(0))
			klass := AS_CLASS(peek(1))
			klass.methods[name] = method
			pop()
			break
		}

		case OP_INVOKE: {
			constant := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			argCount := int(frame.closure.closureFn.funcChunk.code[frame.ip+3])
			frame.ip += 4
			name := AS_CSTRING(frame.closure.closureFn.funcChunk.constants[constant])
			receiver := peek(argCount)
			if !IS_INSTANCE(receiver) {
				runtimeError("Only instances have methods.")
				return INTERPRET_RUNTIME_ERROR
			}
			instance := AS_INSTANCE(receiver)
			if prop, ok := instance.fields[name]; ok {
				if !IS_CLOSURE(prop) && !IS_NATIVE(prop) && !IS_BOUND_METHOD(prop) {
					runtimeError("Can only call functions and classes.")
					return INTERPRET_RUNTIME_ERROR
				}
				push(prop)
				callValue(prop, argCount)
				frame = &vm.frames[vm.frameCount-1]
				break
			}
			closure, ok := instance.instClass.methods[name]
			if !ok {
				runtimeError("Undefined property '%s'.", name)
				return INTERPRET_RUNTIME_ERROR
			}
			if !callValue(ObjVal(closure), argCount) {
				return INTERPRET_RUNTIME_ERROR
			}
			frame = &vm.frames[vm.frameCount-1]
			break
		}

		case OP_INHERIT: {
			superclassVal := peek(1)
			subclassVal := peek(0)
			if !IS_CLASS(superclassVal) || !IS_CLASS(subclassVal) {
				runtimeError("Superclass must be a class.")
				return INTERPRET_RUNTIME_ERROR
			}
			superclass := AS_CLASS(superclassVal)
			subclass := AS_CLASS(subclassVal)
			for name, method := range superclass.methods {
				subclass.methods[name] = method
			}
			pop() // subclass
			frame.ip++
			break
		}

	case OP_GET_SUPER: {
		constant := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
		frame.ip += 3
		name := AS_CSTRING(frame.closure.closureFn.funcChunk.constants[constant])
			superclass := AS_CLASS(pop())
			method := bindMethod(superclass, name)
			if method == nil {
				runtimeError("Undefined property '%s'.", name)
				return INTERPRET_RUNTIME_ERROR
			}
			push(ObjVal(method))
			break
		}

		case OP_SUPER_INVOKE: {
			constant := int(frame.closure.closureFn.funcChunk.Read16(frame.ip + 1))
			argCount := int(frame.closure.closureFn.funcChunk.code[frame.ip+3])
			frame.ip += 4
			name := AS_CSTRING(frame.closure.closureFn.funcChunk.constants[constant])
			superclass := AS_CLASS(pop())
			closure, ok := superclass.methods[name]
			if !ok {
				runtimeError("Undefined property '%s'.", name)
				return INTERPRET_RUNTIME_ERROR
			}
			receiver := peek(argCount)
			if !IS_INSTANCE(receiver) {
				runtimeError("Only instances have methods.")
				return INTERPRET_RUNTIME_ERROR
			}
			callValue(ObjVal(closure), argCount)
			frame = &vm.frames[vm.frameCount-1]
			break
		}
		}
	}
}

func callValue(callee Value, argCount int) bool {
	if IS_OBJ(callee) {
		switch callee.obj.objType {
		case OBJ_BOUND_METHOD:
			bound := AS_BOUND_METHOD(callee)
			vm.stack[vm.sp-argCount-1] = bound.boundRecv
			return callClosure(bound.boundFn, argCount)
		case OBJ_CLASS:
			klass := AS_CLASS(callee)
			instance := NewObjInstance(klass)
			vm.stack[vm.sp-argCount-1] = ObjVal(instance)
			initializer, ok := klass.methods["init"]
			if ok {
				return callClosure(initializer, argCount)
			}
			if argCount != 0 {
				runtimeError("Expected 0 arguments but got %d.", argCount)
				return false
			}
			return true
		case OBJ_CLOSURE:
			return callClosure(callee.obj, argCount)
		case OBJ_NATIVE:
			native := AS_NATIVE(callee)
			args := make([]Value, argCount)
			for i := argCount - 1; i >= 0; i-- {
				args[i] = vm.stack[vm.sp-argCount+i]
			}
			result := native(argCount, args)
			vm.sp -= argCount + 1
			push(result)
			return true
		}
	}
	runtimeError("Can only call functions and classes.")
	return false
}

func callClosure(closure *Object, argCount int) bool {
	fn := closure.closureFn
	if argCount != fn.funcArity {
		runtimeError("Expected %d arguments but got %d.", fn.funcArity, argCount)
		return false
	}
	if vm.frameCount == FRAMES_MAX {
		runtimeError("Stack overflow.")
		return false
	}
	frame := &vm.frames[vm.frameCount]
	vm.frameCount++
	frame.closure = closure
	frame.ip = 0
	frame.slots = vm.sp - argCount - 1
	return true
}

func captureUpvalue(local *Value) *Object {
	absSlot := int(uintptr(unsafe.Pointer(local))-uintptr(unsafe.Pointer(&vm.stack[0]))) / int(unsafe.Sizeof(Value{}))
	upvalue := vm.openUpvalues
	for upvalue != nil {
		if upvalue.upvalueLoc == local {
			return upvalue
		}
		upvalue = upvalue.next
	}

	uv := NewObjUpvalue(local, absSlot)
	uv.next = vm.openUpvalues
	vm.openUpvalues = uv
	return uv
}

func closeUpvalues(last *Value) {
	for vm.openUpvalues != nil {
		upvalue := vm.openUpvalues
		if upvalue.upvalueLoc == nil {
			vm.openUpvalues = upvalue.next
			continue
		}
		// Compare stack slot indices
		lastSlot := int(uintptr(unsafe.Pointer(last))-uintptr(unsafe.Pointer(&vm.stack[0]))) / int(unsafe.Sizeof(Value{}))
		if upvalue.upvalueSlot < lastSlot {
			break
		}
		upvalue.upvalueClosed = *upvalue.upvalueLoc
		upvalue.upvalueLoc = nil
		vm.openUpvalues = upvalue.next
	}
}

func bindMethod(klass *Object, name string) *Object {
	method, ok := klass.methods[name]
	if !ok {
		return nil
	}
	return NewObjBoundMethod(peek(0), method)
}

type InterpretResult int

const (
	INTERPRET_OK InterpretResult = iota
	INTERPRET_COMPILE_ERROR
	INTERPRET_RUNTIME_ERROR
)

func Interpret(source string) InterpretResult {
	function := compile(source)
	if function == nil {
		return INTERPRET_COMPILE_ERROR
	}

	push(ObjVal(NewObjClosure(function)))
	callValue(peek(0), 0)

	result := run()
	return result
}


