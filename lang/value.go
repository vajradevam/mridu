package lang

import "fmt"

type ValueType int

const (
	VAL_BOOL ValueType = iota
	VAL_NIL
	VAL_NUMBER
	VAL_OBJ
)

type Value struct {
	type_ ValueType

	// immediate data
	boolean bool
	number  float64

	// heap object
	obj *Object
}

func BoolVal(b bool) Value   { return Value{type_: VAL_BOOL, boolean: b} }
func NilVal() Value          { return Value{type_: VAL_NIL} }
func NumberVal(n float64) Value { return Value{type_: VAL_NUMBER, number: n} }
func ObjVal(o *Object) Value {
	if o == nil {
		return NilVal()
	}
	return Value{type_: VAL_OBJ, obj: o}
}

func AS_BOOL(v Value) bool      { return v.boolean }
func AS_NUMBER(v Value) float64 { return v.number }
func AS_OBJ(v Value) *Object    { return v.obj }

func IS_BOOL(v Value) bool   { return v.type_ == VAL_BOOL }
func IS_NIL(v Value) bool    { return v.type_ == VAL_NIL }
func IS_NUMBER(v Value) bool { return v.type_ == VAL_NUMBER }
func IS_OBJ(v Value) bool    { return v.type_ == VAL_OBJ }

func ValuesEqual(a, b Value) bool {
	if a.type_ != b.type_ {
		return false
	}
	switch a.type_ {
	case VAL_BOOL:
		return a.boolean == b.boolean
	case VAL_NIL:
		return true
	case VAL_NUMBER:
		return a.number == b.number
	case VAL_OBJ:
		if a.obj != nil && b.obj != nil && a.obj.objType == OBJ_STRING && b.obj.objType == OBJ_STRING {
			return a.obj.strValue == b.obj.strValue
		}
		return a.obj == b.obj
	}
	return false
}

type ObjType int

const (
	OBJ_STRING ObjType = iota
	OBJ_FUNCTION
	OBJ_NATIVE
	OBJ_CLOSURE
	OBJ_UPVALUE
	OBJ_CLASS
	OBJ_INSTANCE
	OBJ_BOUND_METHOD
)

type Object struct {
	objType ObjType

	// ObjString
	strValue string

	// ObjFunction
	funcArity    int
	funcChunk    Chunk
	funcName     string
	funcUpvalues int

	// ObjNative
	nativeFn func(argCount int, args []Value) Value

	// ObjClosure
	closureFn  *Object // *ObjFunction
	upvaluePtr []*Object // []*ObjUpvalue

	// ObjUpvalue
	upvalueLoc    *Value
	upvalueClosed Value
	upvalueSlot   int // absolute stack index for pointer comparison

	// Linked list for open upvalues
	next *Object

	// ObjClass
	className string
	methods   map[string]*Object // name -> *ObjFunction

	// ObjInstance
	instClass *Object // *ObjClass
	fields    map[string]Value

	// ObjBoundMethod
	boundRecv Value
	boundFn   *Object // *ObjClosure
}

func NewObjString(s string) *Object {
	return &Object{objType: OBJ_STRING, strValue: s}
}

func NewObjFunction(name string) *Object {
	return &Object{objType: OBJ_FUNCTION, funcName: name}
}

func NewObjNative(fn func(int, []Value) Value) *Object {
	return &Object{objType: OBJ_NATIVE, nativeFn: fn}
}

func NewObjClosure(fn *Object) *Object {
	upvs := make([]*Object, fn.funcUpvalues)
	return &Object{objType: OBJ_CLOSURE, closureFn: fn, upvaluePtr: upvs}
}

func NewObjUpvalue(slot *Value, absSlot int) *Object {
	return &Object{objType: OBJ_UPVALUE, upvalueLoc: slot, upvalueSlot: absSlot}
}

func NewObjClass(name string) *Object {
	return &Object{objType: OBJ_CLASS, className: name, methods: make(map[string]*Object)}
}

func NewObjInstance(klass *Object) *Object {
	return &Object{objType: OBJ_INSTANCE, instClass: klass, fields: make(map[string]Value)}
}

func NewObjBoundMethod(receiver Value, method *Object) *Object {
	return &Object{objType: OBJ_BOUND_METHOD, boundRecv: receiver, boundFn: method}
}

func AS_STRING(v Value) string   { return v.obj.strValue }
func AS_CSTRING(v Value) string  { return v.obj.strValue }
func AS_FUNCTION(v Value) *Object { return v.obj }
func AS_NATIVE(v Value) func(int, []Value) Value { return v.obj.nativeFn }
func AS_CLOSURE(v Value) *Object  { return v.obj }
func AS_CLASS(v Value) *Object    { return v.obj }
func AS_INSTANCE(v Value) *Object { return v.obj }
func AS_BOUND_METHOD(v Value) *Object { return v.obj }

func IS_STRING(v Value) bool {
	return IS_OBJ(v) && v.obj.objType == OBJ_STRING
}
func IS_FUNCTION(v Value) bool {
	return IS_OBJ(v) && v.obj.objType == OBJ_FUNCTION
}
func IS_NATIVE(v Value) bool {
	return IS_OBJ(v) && v.obj.objType == OBJ_NATIVE
}
func IS_CLOSURE(v Value) bool {
	return IS_OBJ(v) && v.obj.objType == OBJ_CLOSURE
}
func IS_CLASS(v Value) bool {
	return IS_OBJ(v) && v.obj.objType == OBJ_CLASS
}
func IS_INSTANCE(v Value) bool {
	return IS_OBJ(v) && v.obj.objType == OBJ_INSTANCE
}
func IS_BOUND_METHOD(v Value) bool {
	return IS_OBJ(v) && v.obj.objType == OBJ_BOUND_METHOD
}

func (v Value) String() string {
	switch v.type_ {
	case VAL_BOOL:
		return fmt.Sprintf("%t", v.boolean)
	case VAL_NIL:
		return "nil"
	case VAL_NUMBER:
		s := fmt.Sprintf("%g", v.number)
		return s
	case VAL_OBJ:
		return objString(v)
	}
	return "?"
}

func objString(v Value) string {
	o := v.obj
	switch o.objType {
	case OBJ_STRING:
		return o.strValue
	case OBJ_FUNCTION:
		if o.funcName == "" {
			return "<script>"
		}
		return fmt.Sprintf("<fn %s>", o.funcName)
	case OBJ_NATIVE:
		return fmt.Sprintf("<native fn>")
	case OBJ_CLOSURE:
		return objString(ObjVal(o.closureFn))
	case OBJ_UPVALUE:
		return "upvalue"
	case OBJ_CLASS:
		return o.className
	case OBJ_INSTANCE:
		return fmt.Sprintf("%s instance", o.instClass.className)
	case OBJ_BOUND_METHOD:
		return objString(ObjVal(o.boundFn))
	}
	return "?"
}

var Globals = make(map[string]Value)
