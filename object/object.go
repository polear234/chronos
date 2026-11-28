package object

// ObjectType 对象类型标识
type ObjectType string

const (
	INTEGER_OBJ      = "INTEGER"
	FLOAT_OBJ        = "FLOAT"
	STRING_OBJ       = "STRING"
	BOOLEAN_OBJ      = "BOOLEAN"
	NIL_OBJ          = "NIL"
	RETURN_VALUE_OBJ = "RETURN_VALUE"
	ERROR_OBJ        = "ERROR"
	FUNCTION_OBJ     = "FUNCTION"
	BUILTIN_OBJ      = "BUILTIN"
	ARRAY_OBJ        = "ARRAY"
	HASH_OBJ         = "HASH"
	CLASS_OBJ        = "CLASS"
	INSTANCE_OBJ     = "INSTANCE"
	METHOD_OBJ       = "METHOD"
	MODULE_OBJ       = "MODULE"
	BREAK_OBJ        = "BREAK"
	CONTINUE_OBJ     = "CONTINUE"
	TIMELINE_OBJ     = "TIMELINE"
	TRAVELLER_OBJ    = "TRAVELLER"
	POINTER_OBJ      = "POINTER"
)

// Object 所有运行时对象的统一接口
type Object interface {
	Type() ObjectType
	Inspect() string
}

// Hashable 可作为哈希键的对象
type Hashable interface {
	HashKey() HashKey
}

// HashKey 哈希键
type HashKey struct {
	Type  ObjectType
	Value uint64
}
