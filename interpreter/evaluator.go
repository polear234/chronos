package interpreter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"strings"

	"chronospp/ast"
	"chronospp/lexer"
	"chronospp/object"
	"chronospp/parser"
	"chronospp/token"
)

var (
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
	NIL   = &object.Nil{}
)

// thisKey 在环境中绑定当前实例的特殊键名
const thisKey = "__this__"

// superKey 在环境中绑定当前方法所属类的特殊键名，用于 super 从正确父类查找
const superKey = "__super_class__"

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj object.Object) bool {
	return obj != nil && obj.Type() == object.ERROR_OBJ
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case NIL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}

// Eval 求值入口
func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalProgram(node, env)
	case *ast.LetStatement:
		return evalLetStatement(node, env)
	case *ast.ConstStatement:
		return evalConstStatement(node, env)
	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)
	case *ast.BlockStatement:
		return evalBlockStatement(node, env)
	case *ast.IfExpression:
		return evalIfExpression(node, env)
	case *ast.WhileExpression:
		return evalWhileExpression(node, env)
	case *ast.ForExpression:
		return evalForExpression(node, env)
	case *ast.BreakStatement:
		return &object.Break{}
	case *ast.ContinueStatement:
		return &object.Continue{}
	case *ast.FunctionLiteral:
		return evalFunctionLiteral(node, env)
	case *ast.CallExpression:
		return evalCallExpression(node, env)
	case *ast.Identifier:
		return evalIdentifier(node, env)
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.FloatLiteral:
		return &object.Float{Value: node.Value}
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}
	case *ast.BooleanLiteral:
		return nativeBoolToBooleanObject(node.Value)
	case *ast.NilLiteral:
		return NIL
	case *ast.PrefixExpression:
		// &x：取地址，右侧必须是变量名（不求值）
		if node.Operator == "&" {
			ident, ok := node.Right.(*ast.Identifier)
			if !ok {
				return newError("取地址要写成 &变量名")
			}
			if _, defined := env.Get(ident.Value); !defined {
				return newError("变量 %s 还没定义，不能取地址", ident.Value)
			}
			return &object.Pointer{Env: env, Name: ident.Value}
		}
		// *p：解引用
		if node.Operator == "*" {
			val := Eval(node.Right, env)
			if isError(val) {
				return val
			}
			ptr, ok := val.(*object.Pointer)
			if !ok {
				return newError("* 只能用于指针，收到 %s", val.Type())
			}
			target, ok := ptr.Deref()
			if !ok {
				return newError("指针指向的变量 %s 已不存在", ptr.Name)
			}
			return target
		}
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)
	case *ast.InfixExpression:
		// C++ 风格数据流单独处理：
		//   prints << expr   → 输出
		//   inputs >> ident  → 读入到已有变量（右侧不求值，按变量名赋值）
		if node.Operator == "<<" || node.Operator == ">>" {
			return evalStreamExpression(node, env)
		}
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)
	case *ast.IndexExpression:
		return evalIndexExpression(node, env)
	case *ast.ArrayLiteral:
		return evalArrayLiteral(node, env)
	case *ast.HashLiteral:
		return evalHashLiteral(node, env)
	case *ast.AssignExpression:
		return evalAssignExpression(node, env)
	case *ast.PropertyExpression:
		return evalPropertyExpression(node, env)
	case *ast.MethodCallExpression:
		return evalMethodCallExpression(node, env)
	case *ast.ClassLiteral:
		return evalClassLiteral(node, env)
	case *ast.InterfaceLiteral:
		return evalInterfaceLiteral(node, env)
	case *ast.TransactionStatement:
		return evalTransactionStatement(node, env)
	case *ast.NewExpression:
		return evalNewExpression(node, env)
	case *ast.ThisExpression:
		return evalThisExpression(env)
	case *ast.SuperExpression:
		return evalSuperExpression(node, env)
	case *ast.ImportStatement:
		return evalImportStatement(node, env)
	case *ast.ExportStatement:
		return evalExportStatement(node, env)
	case *ast.TryStatement:
		return evalTryStatement(node, env)
	case *ast.ThrowStatement:
		return evalThrowStatement(node, env)
	case *ast.MatchExpression:
		return evalMatchExpression(node, env)
	}
	return NIL
}

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object
	for _, stmt := range program.Statements {
		result = Eval(stmt, env)
		switch r := result.(type) {
		case *object.ReturnValue:
			return r.Value
		case *object.Error:
			return r
		}
	}
	return result
}

func evalLetStatement(stmt *ast.LetStatement, env *object.Environment) object.Object {
	val := Eval(stmt.Value, env)
	if isError(val) {
		return val
	}

	// 类型校验：VarType 非空时检查初值类型是否匹配
	if stmt.VarType != "" {
		val = coerceToType(val, stmt.VarType)
		if err := checkVarType(stmt.VarType, val); err != nil {
			return err
		}
	}

	if stmt.IsTraveller {
		// traveller 修饰：包装成 Traveller 对象，支持 rollback
		env.Define(stmt.Name.Value, object.NewTraveller(val))
		// 记录类型约束
		env.DefineVarType(stmt.Name.Value, stmt.VarType)
		return NIL
	}

	if stmt.VarType != "" {
		env.DefineTyped(stmt.Name.Value, val, stmt.VarType)
	} else {
		env.Define(stmt.Name.Value, val)
		// 没有显式类型时按初值推断，让 inputs >> 知道该按什么读
		if inferred := inferVarType(val); inferred != "" {
			env.DefineVarType(stmt.Name.Value, inferred)
		}
	}
	return NIL
}

// inferVarType 根据值推断变量类型，供 inputs >> 这类需要类型信息的场景使用
func inferVarType(val object.Object) string {
	switch val.(type) {
	case *object.Integer:
		return "int"
	case *object.Float:
		return "float"
	case *object.String:
		return "string"
	case *object.Boolean:
		return "bool"
	}
	return ""
}

// coerceToType 在赋值给【类型化变量】时做隐式转换：
//
//	string "42"  → int 42
//	int 3        → float 3.0
//	int/float    → string "3"
//
// 转换失败时原样返回，交给 checkVarType 报类型错误。
func coerceToType(val object.Object, targetType string) object.Object {
	switch targetType {
	case "int":
		switch v := val.(type) {
		case *object.String:
			if n, err := strconv.ParseInt(strings.TrimSpace(v.Value), 10, 64); err == nil {
				return &object.Integer{Value: n}
			}
		case *object.Float:
			return &object.Integer{Value: int64(v.Value)}
		}
	case "float":
		switch v := val.(type) {
		case *object.String:
			if f, err := strconv.ParseFloat(strings.TrimSpace(v.Value), 64); err == nil {
				return &object.Float{Value: f}
			}
		case *object.Integer:
			return &object.Float{Value: float64(v.Value)}
		}
	}
	// 注意：不把 int/float 隐式转成 string —— 那样会掩盖 bug，
	// 也破坏"类型化变量赋值严格校验"的语义。
	return val
}

// checkVarType 校验对象类型是否匹配声明的类型标注
func checkVarType(varType string, val object.Object) *object.Error {
	switch varType {
	case "int":
		if val.Type() != object.INTEGER_OBJ {
			return newError("type mismatch: declared %s, got %s", varType, val.Type())
		}
	case "float":
		if val.Type() != object.FLOAT_OBJ && val.Type() != object.INTEGER_OBJ {
			return newError("type mismatch: declared %s, got %s", varType, val.Type())
		}
	case "string":
		if val.Type() != object.STRING_OBJ {
			return newError("type mismatch: declared %s, got %s", varType, val.Type())
		}
	case "bool":
		if val.Type() != object.BOOLEAN_OBJ {
			return newError("type mismatch: declared %s, got %s", varType, val.Type())
		}
	case "array":
		if val.Type() != object.ARRAY_OBJ {
			return newError("type mismatch: declared %s, got %s", varType, val.Type())
		}
	case "dict":
		if val.Type() != object.HASH_OBJ {
			return newError("type mismatch: declared %s, got %s", varType, val.Type())
		}
	default:
		// 未知类型标注：不强制校验（允许自定义类名等）
	}
	return nil
}

func evalConstStatement(stmt *ast.ConstStatement, env *object.Environment) object.Object {
	val := Eval(stmt.Value, env)
	if isError(val) {
		return val
	}
	env.DefineConst(stmt.Name.Value, val)
	return NIL
}

func evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object = NIL
	for _, stmt := range block.Statements {
		result = Eval(stmt, env)
		if result != nil {
			switch result.Type() {
			case object.RETURN_VALUE_OBJ, object.ERROR_OBJ, object.BREAK_OBJ, object.CONTINUE_OBJ:
				return result
			}
		}
	}
	return result
}

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}
	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	}
	return NIL
}

func evalWhileExpression(we *ast.WhileExpression, env *object.Environment) object.Object {
	for {
		condition := Eval(we.Condition, env)
		if isError(condition) {
			return condition
		}
		if !isTruthy(condition) {
			// 自然结束（含零次循环）：执行 else
			if we.Alternative != nil {
				return Eval(we.Alternative, env)
			}
			return NIL
		}
		result := Eval(we.Body, env)
		if isError(result) {
			return result
		}
		switch result.(type) {
		case *object.Break:
			// break 跳过 else
			return NIL
		case *object.Continue:
			continue
		case *object.ReturnValue:
			return result
		}
	}
}

func evalForExpression(fe *ast.ForExpression, env *object.Environment) object.Object {
	iterable := Eval(fe.Iterable, env)
	if isError(iterable) {
		return iterable
	}

	var elements []object.Object
	switch it := iterable.(type) {
	case *object.Array:
		elements = it.Elements
	case *object.String:
		runes := []rune(it.Value)
		elements = make([]object.Object, 0, len(runes))
		for _, r := range runes {
			elements = append(elements, &object.String{Value: string(r)})
		}
	case *object.Hash:
		for _, pair := range it.Pairs {
			elements = append(elements, pair.Key)
		}
	default:
		return newError("for loop requires iterable, got %s", iterable.Type())
	}

	for _, elem := range elements {
		loopEnv := object.NewEnclosedEnvironment(env)
		loopEnv.Define(fe.Variable.Value, elem)
		result := Eval(fe.Body, loopEnv)
		if isError(result) {
			return result
		}
		switch result.(type) {
		case *object.Break:
			// break 跳过 else
			return NIL
		case *object.Continue:
			continue
		case *object.ReturnValue:
			return result
		}
	}
	// 自然结束（含空迭代）：执行 else
	if fe.Alternative != nil {
		return Eval(fe.Alternative, env)
	}
	return NIL
}

func evalFunctionLiteral(fl *ast.FunctionLiteral, env *object.Environment) object.Object {
	params := make([]*ast.Identifier, len(fl.Parameters))
	copy(params, fl.Parameters)
	fn := &object.Function{
		Parameters: params,
		Body:       fl.Body,
		Env:        env,
		Name:       fl.Name,
		ReturnType: fl.ReturnType,
	}
	// 命名函数：把函数名绑定到当前环境，支持递归调用
	if fl.Name != "" {
		// 有参数类型时注册为重载；同时保持普通绑定（兼容旧行为）
		hasTypeHint := false
		for _, p := range fl.Parameters {
			if p.TypeHint != "" {
				hasTypeHint = true
				break
			}
		}
		if hasTypeHint {
			env.DefineOverload(fn)
			// 只在第一次定义时绑定名字，后续重载不覆盖
			if _, exists := env.Get(fl.Name); !exists {
				env.Define(fl.Name, fn)
			}
		} else {
			env.Define(fl.Name, fn)
		}
	}
	return fn
}

func evalCallExpression(ce *ast.CallExpression, env *object.Environment) object.Object {
	// 特殊高阶 API 拦截：需要 env 才能执行 callback
	if id, ok := ce.Function.(*ast.Identifier); ok {
		switch id.Value {
		case "withSnapshot":
			return evalWithSnapshot(ce.Arguments, env)
		case "mark":
			// mark() / mark("名字")：记录当前所有变量
			name := "t" + strconv.FormatInt(time.Now().UnixNano(), 36)
			if len(ce.Arguments) > 0 {
				nameVal := Eval(ce.Arguments[0], env)
				if isError(nameVal) {
					return nameVal
				}
				name = nameVal.Inspect()
			}
			return env.SnapshotMark(name)

		case "restore":
			// restore(mark)：恢复到某个时间点
			if len(ce.Arguments) != 1 {
				return newError("restore 需要一个时间点参数")
			}
			markVal := Eval(ce.Arguments[0], env)
			if isError(markVal) {
				return markVal
			}
			mp, ok := markVal.(*object.MarkPoint)
			if !ok {
				return newError("restore 的参数要是 mark() 返回的时间点")
			}
			count := env.RestoreMark(mp)
			return &object.Integer{Value: int64(count)}

		case "diff":
			// diff(mark)：返回与时间点之间变化的变量名
			if len(ce.Arguments) != 1 {
				return newError("diff 需要一个时间点参数")
			}
			markVal := Eval(ce.Arguments[0], env)
			if isError(markVal) {
				return markVal
			}
			mp, ok := markVal.(*object.MarkPoint)
			if !ok {
				return newError("diff 的参数要是 mark() 返回的时间点")
			}
			changed := env.DiffMark(mp)
			out := make([]object.Object, len(changed))
			for i, name := range changed {
				out[i] = &object.String{Value: name}
			}
			return &object.Array{Elements: out}
		case "Timeline":
			return evalTimelineConstruct(ce.Arguments, env)
		case "start":
			// 并行执行多个函数，返回结果数组（仅在本机命令行版本真正并行）
			if _, defined := env.Get(id.Value); !defined {
				return evalStartCall(ce.Arguments, env)
			}
		case "map", "filter", "forEach", "reduce", "find":
			// 数组高阶函数的「全局函数形态」：map(arr, fn)
			// 仅当用户没有自定义同名函数时才生效，保证可覆盖
			if _, defined := env.Get(id.Value); !defined {
				return evalGlobalArrayHOF(id.Value, ce.Arguments, env)
			}
		}

		// 函数重载：按实参类型查找匹配的版本
		if env.HasOverloads(id.Value) {
			// 先求值所有实参（为了拿到类型）
			args := make([]object.Object, len(ce.Arguments))
			for i, a := range ce.Arguments {
				args[i] = Eval(a, env)
				if isError(args[i]) {
					return args[i]
				}
			}
			if fn := env.LookupOverload(id.Value, args); fn != nil {
				return applyFunction(fn, args, env)
			}
			// 没匹配上，继续走普通调用（会报参数类型错误）
		}
	}

	fn := Eval(ce.Function, env)
	if isError(fn) {
		return fn
	}
	args := evalExpressions(ce.Arguments, env)
	if len(args) == 1 && isError(args[0]) {
		return args[0]
	}
	return applyFunction(fn, args, env)
}

// evalGlobalArrayHOF 实现 map(arr, fn) / filter(arr, fn) / reduce(arr, fn, init?)
// 这类「全局函数形态」，让管道符可以写成 arr |> map(fn(x){...})
func evalGlobalArrayHOF(name string, argExprs []ast.Expression, env *object.Environment) object.Object {
	if len(argExprs) < 2 {
		return newError("%s expects at least 2 arguments (array, function), got %d", name, len(argExprs))
	}
	arrVal := Eval(argExprs[0], env)
	if isError(arrVal) {
		return arrVal
	}
	arr, ok := arrVal.(*object.Array)
	if !ok {
		return newError("%s: first argument must be array, got %s", name, arrVal.Type())
	}
	rest := evalExpressions(argExprs[1:], env)
	if len(rest) == 1 && isError(rest[0]) {
		return rest[0]
	}
	return evalArrayHigherOrderMethod(arr, name, rest, env)
}

// evalWithSnapshot 实现 withSnapshot(targets, callback)：
// 记录 targets（数组/字典）的当前状态 → 执行 callback → 原地恢复。
func evalWithSnapshot(argExprs []ast.Expression, env *object.Environment) object.Object {
	if len(argExprs) != 2 {
		return newError("withSnapshot expects 2 arguments (targets, callback), got %d", len(argExprs))
	}

	// 求值 targets（必须是一个数组，元素为 Array 或 Hash）
	targetsObj := Eval(argExprs[0], env)
	if isError(targetsObj) {
		return targetsObj
	}
	targetsArr, ok := targetsObj.(*object.Array)
	if !ok {
		return newError("withSnapshot first argument must be an array of targets, got %s", targetsObj.Type())
	}

	// 对每个 target 取快照，并保存 target 引用（原地恢复用）
	type snapEntry struct {
		target object.Object
		snap   *object.Snapshot
	}
	entries := make([]snapEntry, 0, len(targetsArr.Elements))
	for _, t := range targetsArr.Elements {
		snap := object.CaptureSnapshot(t)
		if snap == nil {
			return newError("withSnapshot target must be array or hash, got %s", t.Type())
		}
		entries = append(entries, snapEntry{target: t, snap: snap})
	}

	// 求值并执行 callback（无参数调用）
	callback := Eval(argExprs[1], env)
	if isError(callback) {
		return callback
	}
	result := applyFunction(callback, nil, env)

	// 无论 callback 成功或报错，都恢复所有 targets
	for _, e := range entries {
		object.RestoreSnapshot(e.target, e.snap)
	}

	return result
}

// evalTimelineConstruct 实现 Timeline(target)：创建版本历史对象。
func evalTimelineConstruct(argExprs []ast.Expression, env *object.Environment) object.Object {
	if len(argExprs) != 1 {
		return newError("Timeline expects 1 argument (target), got %d", len(argExprs))
	}
	target := Eval(argExprs[0], env)
	if isError(target) {
		return target
	}
	if object.CaptureSnapshot(target) == nil {
		return newError("Timeline target must be array or hash, got %s", target.Type())
	}
	return &object.Timeline{Target: target, History: []object.Snapshot{}}
}

func evalExpressions(exps []ast.Expression, env *object.Environment) []object.Object {
	result := make([]object.Object, 0, len(exps))
	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}
	return result
}

// applyFunction 调用函数/方法/内置函数
func applyFunction(fn object.Object, args []object.Object, env *object.Environment) object.Object {
	switch fn := fn.(type) {
	case *object.Function:
		// 参数个数与类型检查
		if err := checkFunctionArgs(fn, args); err != nil {
			return err
		}
		extended := extendFunctionEnv(fn, args)
		result := unwrapReturnValue(Eval(fn.Body, extended))
		if isError(result) {
			return result
		}
		// 返回类型检查（只在声明了类型时）
		if fn.ReturnType != "" && fn.ReturnType != "void" && fn.ReturnType != "any" {
			checked := coerceToType(result, fn.ReturnType)
			if err := checkVarType(fn.ReturnType, checked); err != nil {
				name := fn.Name
				if name == "" {
					name = "匿名函数"
				}
				return newError("%s 声明返回 %s，实际返回 %s",
					name, fn.ReturnType, result.Type())
			}
			return checked
		}
		// void 函数不该有返回值
		if fn.ReturnType == "void" && result != NIL {
			name := fn.Name
			if name == "" {
				name = "匿名函数"
			}
			return newError("%s 声明为 void，不应有返回值", name)
		}
		return result
	case *object.Builtin:
		return fn.Fn(args...)
	case *object.Method:
		// 方法调用：把接收者绑定为 this，方法所属类绑定为 super 查找起点
		extended := extendFunctionEnv(fn.Fn, args)
		extended.Define(thisKey, fn.Receiver)
		extended.Define(superKey, fn.Class)
		return unwrapReturnValue(Eval(fn.Fn.Body, extended))
	case *object.Class:
		// 类本身可被调用，作为构造函数（工厂模式）
		instance := &object.Instance{Class: fn, Fields: make(map[string]object.Object)}
		for name, val := range fn.Properties {
			instance.Fields[name] = val
		}
		if init, defClass := lookupMethod(fn, "init"); init != nil {
			extended := extendFunctionEnv(init, args)
			extended.Define(thisKey, instance)
			extended.Define(superKey, defClass)
			result := unwrapReturnValue(Eval(init.Body, extended))
			if isError(result) {
				return result
			}
		}
		return instance
	default:
		return newError("not a function: %s", fn.Type())
	}
}

// checkFunctionArgs 校验参数个数和类型，返回错误对象或 nil
func checkFunctionArgs(fn *object.Function, args []object.Object) object.Object {
	name := fn.Name
	if name == "" {
		name = "匿名函数"
	}
	if len(args) != len(fn.Parameters) {
		return newError("%s 需要 %d 个参数，收到 %d 个",
			name, len(fn.Parameters), len(args))
	}
	for i, param := range fn.Parameters {
		if param.TypeHint == "" || i >= len(args) {
			continue
		}
		val := args[i]
		if val == NIL {
			continue
		}
		converted := coerceToType(val, param.TypeHint)
		if err := checkVarType(param.TypeHint, converted); err != nil {
			return newError("%s 的第 %d 个参数 %s 声明为 %s，收到 %s",
				name, i+1, param.Value, param.TypeHint, val.Type())
		}
	}
	return nil
}

func extendFunctionEnv(fn *object.Function, args []object.Object) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)
	for i, param := range fn.Parameters {
		var val object.Object
		if i < len(args) {
			val = args[i]
		} else {
			val = NIL
		}
		// C++ 风格参数类型：int a → 传参时做隐式转换
		if param.TypeHint != "" && val != NIL {
			val = coerceToType(val, param.TypeHint)
		}
		env.Define(param.Value, val)
		if param.TypeHint != "" {
			env.DefineVarType(param.Value, param.TypeHint)
		}
	}
	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if rv, ok := obj.(*object.ReturnValue); ok {
		return rv.Value
	}
	return obj
}

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		// traveller 变量读取时透明解包为当前值
		if trav, isTrav := val.(*object.Traveller); isTrav {
			return trav.Current
		}
		return val
	}
	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}
	return newError("identifier not found: %s", node.Value)
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!", "not":
		return nativeBoolToBooleanObject(!isTruthy(right))
	case "-":
		switch r := right.(type) {
		case *object.Integer:
			return &object.Integer{Value: -r.Value}
		case *object.Float:
			return &object.Float{Value: -r.Value}
		default:
			return newError("unknown operator: -%s", right.Type())
		}
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

// evalStreamExpression 处理 C++ 风格数据流：
//   prints << expr       把值输出到标准输出
//   inputs >> name       从标准输入读一行，写进已声明的变量
// inputs 右侧必须是已定义类型的变量名，否则报错。
func evalStreamExpression(node *ast.InfixExpression, env *object.Environment) object.Object {
	switch node.Operator {
	case "<<":
		// prints << x           左操作数是 prints/cout 标识符
		// prints << a << b      链式：左操作数本身是流表达式，先执行它
		leftIdent, isIdent := node.Left.(*ast.Identifier)
		if !isIdent {
			if inner, ok := node.Left.(*ast.InfixExpression); ok && inner.Operator == "<<" {
				// 先把左边那串输出掉
				if r := evalStreamExpression(inner, env); isError(r) {
					return r
				}
			} else {
				return newError("数据流输出要写成 prints << 值")
			}
		} else if leftIdent.Value != "prints" && leftIdent.Value != "cout" {
			return newError("数据流输出请用 prints << 值，收到 %s <<", leftIdent.Value)
		}
		// endl / flush：C++ 风格换行符，不求值成变量
		if id, ok := node.Right.(*ast.Identifier); ok {
			if id.Value == "endl" {
				fmt.Fprint(outputWriter, "\n")
				return NIL
			}
			if id.Value == "flush" {
				if f, ok := outputWriter.(interface{ Sync() error }); ok {
					f.Sync()
				}
				return NIL
			}
		}
		val := Eval(node.Right, env)
		if isError(val) {
			return val
		}
		// C++ 的 cout << 不换行；要换行用 endl 或自己写 "\n"
		fmt.Fprint(outputWriter, val.Inspect())
		return NIL

	case ">>":
		// inputs >> name：右侧必须是变量名，不能先求值
		ident, ok := node.Right.(*ast.Identifier)
		if !ok {
			return newError("inputs >> 右边要写变量名，不能是表达式")
		}
		// 变量必须先声明类型（用户明确要求）
		varType := env.GetVarType(ident.Value)
		if varType == "" {
			if _, defined := env.Get(ident.Value); !defined {
				return newError("inputs >> %s：变量还没声明，请先写 let %s = 0", ident.Value, ident.Value)
			}
			return newError("inputs >> %s：变量 %s 没有类型，请先写成 let %s = 0 这样带初值", ident.Value, ident.Value, ident.Value)
		}

		raw := ReadInput("")
		var val object.Object
		switch varType {
		case "int":
			n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
			if err != nil {
				return newError("inputs >> %s：期望整数，收到 %q", ident.Value, raw)
			}
			val = &object.Integer{Value: n}
		case "float":
			f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
			if err != nil {
				return newError("inputs >> %s：期望小数，收到 %q", ident.Value, raw)
			}
			val = &object.Float{Value: f}
		case "bool":
			t := strings.TrimSpace(raw)
			val = nativeBoolToBooleanObject(t == "true" || t == "1" || t == "yes")
		default:
			val = &object.String{Value: raw}
		}
		env.Set(ident.Value, val)
		return NIL
	}
	return NIL
}

func evalInfixExpression(operator string, left, right object.Object) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case isNumber(left) && isNumber(right):
		return evalNumberInfixExpression(operator, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case operator == "&&" || operator == "and":
		return nativeBoolToBooleanObject(logicalTruthy(left) && logicalTruthy(right))
	case operator == "||" || operator == "or":
		return nativeBoolToBooleanObject(logicalTruthy(left) || logicalTruthy(right))
	case operator == "==":
		return nativeBoolToBooleanObject(objectsEqual(left, right))
	case left.Type() == object.ARRAY_OBJ && right.Type() == object.ARRAY_OBJ:
		return evalArrayInfixExpression(operator, left, right)
	// 字符串 + 数字：自动转字符串拼接
	case left.Type() == object.STRING_OBJ && (right.Type() == object.INTEGER_OBJ || right.Type() == object.FLOAT_OBJ) && operator == "+":
		return &object.String{Value: left.(*object.String).Value + right.Inspect()}
	case (left.Type() == object.INTEGER_OBJ || left.Type() == object.FLOAT_OBJ) && right.Type() == object.STRING_OBJ && operator == "+":
		return &object.String{Value: left.Inspect() + right.(*object.String).Value}
	// 字符串 * 整数：重复
	case left.Type() == object.STRING_OBJ && right.Type() == object.INTEGER_OBJ && operator == "*":
		return &object.String{Value: strings.Repeat(left.(*object.String).Value, int(right.(*object.Integer).Value))}
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.STRING_OBJ && operator == "*":
		return &object.String{Value: strings.Repeat(right.(*object.String).Value, int(left.(*object.Integer).Value))}
	// 布尔 + 布尔：true=1 false=0
	case left.Type() == object.BOOLEAN_OBJ && right.Type() == object.BOOLEAN_OBJ && (operator == "+" || operator == "*"):
		lv, rv := 0, 0
		if left.(*object.Boolean).Value { lv = 1 }
		if right.(*object.Boolean).Value { rv = 1 }
		if operator == "+" {
			return &object.Integer{Value: int64(lv + rv)}
		}
		return &object.Integer{Value: int64(lv * rv)}
	case operator == "!=":
		return nativeBoolToBooleanObject(!objectsEqual(left, right))
	default:
		// 运算符重载：查左操作数实例的方法 operator<符号>
		if result, ok := tryOperatorOverload(operator, left, right); ok {
			return result
		}
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	}
}

// tryOperatorOverload 尝试用实例的运算符方法处理二元运算
// 约定：类里定义 fn operator+(other) { ... } 即可重载 +
func tryOperatorOverload(operator string, left, right object.Object) (object.Object, bool) {
	inst, ok := left.(*object.Instance)
	if !ok {
		return nil, false
	}

	// 运算符 → 方法名映射
	methodName := "operator" + operator
	method, _ := lookupMethod(inst.Class, methodName)
	if method == nil {
		return nil, false
	}

	// 调用方法，this 绑定到左操作数，参数是右操作数
	extended := object.NewEnclosedEnvironment(method.Env)
	extended.Define(thisKey, inst)
	if len(method.Parameters) > 0 {
		extended.Define(method.Parameters[0].Value, right)
	}
	result := unwrapReturnValue(Eval(method.Body, extended))
	if isError(result) {
		return result, true
	}
	// 返回类型检查
	if method.ReturnType != "" && method.ReturnType != "void" && method.ReturnType != "any" {
		converted := coerceToType(result, method.ReturnType)
		if err := checkVarType(method.ReturnType, converted); err != nil {
			return newError("%s 声明返回 %s，实际返回 %s",
				methodName, method.ReturnType, result.Type()), true
		}
		return converted, true
	}
	return result, true
}

// evalArrayInfixExpression 数组的 + 运算（拼接）
func evalArrayInfixExpression(operator string, left, right object.Object) object.Object {
	l := left.(*object.Array)
	r := right.(*object.Array)
	switch operator {
	case "+":
		out := make([]object.Object, 0, len(l.Elements)+len(r.Elements))
		out = append(out, l.Elements...)
		out = append(out, r.Elements...)
		return &object.Array{Elements: out}
	}
	return newError("数组不支持 %s 运算", operator)
}

// logicalTruthy 用于 && / || 的真假判定：
// 数字 0 为假（与整数运算里的 l != 0 保持一致），其余沿用 isTruthy。
func logicalTruthy(obj object.Object) bool {
	switch o := obj.(type) {
	case *object.Integer:
		return o.Value != 0
	case *object.Float:
		return o.Value != 0
	default:
		return isTruthy(obj)
	}
}

func isNumber(obj object.Object) bool {
	return obj.Type() == object.INTEGER_OBJ || obj.Type() == object.FLOAT_OBJ
}

func numberValue(obj object.Object) float64 {
	switch o := obj.(type) {
	case *object.Integer:
		return float64(o.Value)
	case *object.Float:
		return o.Value
	}
	return 0
}

func evalIntegerInfixExpression(operator string, left, right object.Object) object.Object {
	l := left.(*object.Integer).Value
	r := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: l + r}
	case "-":
		return &object.Integer{Value: l - r}
	case "*":
		return &object.Integer{Value: l * r}
	case "/":
		if r == 0 {
			return newError("division by zero")
		}
		return &object.Integer{Value: l / r}
	case "%":
		if r == 0 {
			return newError("modulo by zero")
		}
		return &object.Integer{Value: l % r}
	case "**":
		return &object.Integer{Value: powInt(l, r)}
	case "<":
		return nativeBoolToBooleanObject(l < r)
	case ">":
		return nativeBoolToBooleanObject(l > r)
	case "<=":
		return nativeBoolToBooleanObject(l <= r)
	case ">=":
		return nativeBoolToBooleanObject(l >= r)
	case "==":
		return nativeBoolToBooleanObject(l == r)
	case "!=":
		return nativeBoolToBooleanObject(l != r)
	case "&&", "and":
		return nativeBoolToBooleanObject(l != 0 && r != 0)
	case "||", "or":
		return nativeBoolToBooleanObject(l != 0 || r != 0)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalNumberInfixExpression(operator string, left, right object.Object) object.Object {
	l := numberValue(left)
	r := numberValue(right)

	switch operator {
	case "+":
		return &object.Float{Value: l + r}
	case "-":
		return &object.Float{Value: l - r}
	case "*":
		return &object.Float{Value: l * r}
	case "/":
		if r == 0 {
			return newError("division by zero")
		}
		return &object.Float{Value: l / r}
	case "%":
		return &object.Float{Value: float64(int64(l) % int64(r))}
	case "**":
		return &object.Float{Value: powFloat(l, r)}
	case "<":
		return nativeBoolToBooleanObject(l < r)
	case ">":
		return nativeBoolToBooleanObject(l > r)
	case "<=":
		return nativeBoolToBooleanObject(l <= r)
	case ">=":
		return nativeBoolToBooleanObject(l >= r)
	case "==":
		return nativeBoolToBooleanObject(l == r)
	case "!=":
		return nativeBoolToBooleanObject(l != r)
	case "&&", "and":
		return nativeBoolToBooleanObject(l != 0 && r != 0)
	case "||", "or":
		return nativeBoolToBooleanObject(l != 0 || r != 0)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func powInt(base, exp int64) int64 {
	if exp < 0 {
		return 0
	}
	result := int64(1)
	for exp > 0 {
		if exp%2 == 1 {
			result *= base
		}
		base *= base
		exp /= 2
	}
	return result
}

func powFloat(base, exp float64) float64 {
	result := 1.0
	for i := int64(0); i < int64(exp); i++ {
		result *= base
	}
	return result
}

func evalStringInfixExpression(operator string, left, right object.Object) object.Object {
	l := left.(*object.String).Value
	r := right.(*object.String).Value

	switch operator {
	case "+":
		return &object.String{Value: l + r}
	case "==":
		return nativeBoolToBooleanObject(l == r)
	case "!=":
		return nativeBoolToBooleanObject(l != r)
	case "<":
		return nativeBoolToBooleanObject(l < r)
	case ">":
		return nativeBoolToBooleanObject(l > r)
	case "<=":
		return nativeBoolToBooleanObject(l <= r)
	case ">=":
		return nativeBoolToBooleanObject(l >= r)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalArrayLiteral(al *ast.ArrayLiteral, env *object.Environment) object.Object {
	elements := evalExpressions(al.Elements, env)
	if len(elements) == 1 && isError(elements[0]) {
		return elements[0]
	}
	return &object.Array{Elements: elements}
}

func evalIndexExpression(ie *ast.IndexExpression, env *object.Environment) object.Object {
	left := Eval(ie.Left, env)
	if isError(left) {
		return left
	}
	index := Eval(ie.Index, env)
	if isError(index) {
		return index
	}

	switch l := left.(type) {
	case *object.Array:
		idx, ok := index.(*object.Integer)
		if !ok {
			return newError("array index must be integer, got %s", index.Type())
		}
		// 负索引从末尾开始计数（Python 风格）
		actualIdx := idx.Value
		if actualIdx < 0 {
			actualIdx = int64(len(l.Elements)) + actualIdx
		}
		if actualIdx < 0 || actualIdx >= int64(len(l.Elements)) {
			return NIL
		}
		return l.Elements[actualIdx]
	case *object.String:
		idx, ok := index.(*object.Integer)
		if !ok {
			return newError("string index must be integer, got %s", index.Type())
		}
		runes := []rune(l.Value)
		actualIdx := idx.Value
		if actualIdx < 0 {
			actualIdx = int64(len(runes)) + actualIdx
		}
		if actualIdx < 0 || actualIdx >= int64(len(runes)) {
			return NIL
		}
		return &object.String{Value: string(runes[actualIdx])}
	case *object.Hash:
		key, ok := index.(object.Hashable)
		if !ok {
			return newError("unusable as hash key: %s", index.Type())
		}
		if pair, ok := l.Pairs[key.HashKey()]; ok {
			return pair.Value
		}
		return NIL
	case *object.Instance:
		key, ok := index.(*object.String)
		if !ok {
			return newError("instance field access requires string, got %s", index.Type())
		}
		if val, ok := l.Fields[key.Value]; ok {
			return val
		}
		if m, defClass := lookupMethod(l.Class, key.Value); m != nil {
			return &object.Method{Fn: m, Receiver: l, Class: defClass}
		}
		return NIL
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

func evalHashLiteral(hl *ast.HashLiteral, env *object.Environment) object.Object {
	pairs := make(map[object.HashKey]object.HashPair)
	for k, v := range hl.Pairs {
		key := Eval(k, env)
		if isError(key) {
			return key
		}
		hk, ok := key.(object.Hashable)
		if !ok {
			return newError("unusable as hash key: %s", key.Type())
		}
		value := Eval(v, env)
		if isError(value) {
			return value
		}
		pairs[hk.HashKey()] = object.HashPair{Key: key, Value: value}
	}
	return &object.Hash{Pairs: pairs}
}

func evalAssignExpression(ae *ast.AssignExpression, env *object.Environment) object.Object {
	value := Eval(ae.Value, env)
	if isError(value) {
		return value
	}

	// 复合赋值（+= -= *= /=）需要先读取旧值
	if ae.Token.Type != token.ASSIGN {
		old := evalAssignTarget(ae.Left, env)
		if isError(old) {
			return old
		}
		op := compoundOperator(ae.Token.Type)
		value = evalInfixExpression(op, old, value)
		if isError(value) {
			return value
		}
	}

	return evalAssignTarget(ae.Left, env, value)
}

// compoundOperator 将复合赋值 token 转换为对应中缀运算符
func compoundOperator(t token.TokenType) string {
	switch t {
	case token.PLUS_ASSIGN:
		return "+"
	case token.MINUS_ASSIGN:
		return "-"
	case token.MUL_ASSIGN:
		return "*"
	case token.DIV_ASSIGN:
		return "/"
	default:
		return "+"
	}
}

// evalAssignTarget 求值赋值目标（读或写）
func evalAssignTarget(left ast.Expression, env *object.Environment, newVals ...object.Object) object.Object {
	assigning := len(newVals) > 0
	var newVal object.Object
	if assigning {
		newVal = newVals[0]
	}

	switch l := left.(type) {
	case *ast.PrefixExpression:
		// *p = 值：通过指针写
		if l.Operator == "*" {
			ptrVal := Eval(l.Right, env)
			if isError(ptrVal) {
				return ptrVal
			}
			ptr, ok := ptrVal.(*object.Pointer)
			if !ok {
				return newError("* 左边只能是指针，收到 %s", ptrVal.Type())
			}
			if !assigning {
				// 读取
				target, ok := ptr.Deref()
				if !ok {
					return newError("指针指向的变量 %s 已不存在", ptr.Name)
				}
				return target
			}
			// 写入：做类型检查
			if varType := ptr.Env.GetVarType(ptr.Name); varType != "" {
				newVal = coerceToType(newVal, varType)
				if err := checkVarType(varType, newVal); err != nil {
					return err
				}
			}
			if result := ptr.Env.Set(ptr.Name, newVal); isError(result) {
				return result
			}
			return newVal
		}
		return newError("不支持对 %s 赋值", l.Operator)

	case *ast.Identifier:
		if assigning {
			// 类型校验
			if varType := env.GetVarType(l.Value); varType != "" {
				newVal = coerceToType(newVal, varType)
				if err := checkVarType(varType, newVal); err != nil {
					return err
				}
			}
			// 若变量是 Traveller，走其 Set()（自动压历史栈）
			if existing, ok := env.Get(l.Value); ok {
				if trav, isTrav := existing.(*object.Traveller); isTrav {
					trav.Set(newVal)
					return newVal
				}
			}
			// env.Set 可能返回 const 校验错误
			if result := env.Set(l.Value, newVal); isError(result) {
				return result
			}
			return newVal
		}
		if val, ok := env.Get(l.Value); ok {
			// traveller 变量读取时透明解包
			if trav, isTrav := val.(*object.Traveller); isTrav {
				return trav.Current
			}
			return val
		}
		return newError("identifier not found: %s", l.Value)

	case *ast.PropertyExpression:
		obj := Eval(l.Object, env)
		if isError(obj) {
			return obj
		}
		inst, ok := obj.(*object.Instance)
		if !ok {
			return newError("cannot access property of %s", obj.Type())
		}
		if assigning {
			// 泛型特化后的字段类型检查
			if ft := inst.FieldTypes[l.Property]; ft != "" && ft != "any" {
				converted := coerceToType(newVal, ft)
				if err := checkVarType(ft, converted); err != nil {
					return newError("字段 %s 声明为 %s，收到 %s",
						l.Property, ft, newVal.Type())
				}
				newVal = converted
			}
			inst.Fields[l.Property] = newVal
			return newVal
		}
		if val, ok := inst.Fields[l.Property]; ok {
			return val
		}
		return NIL

	case *ast.IndexExpression:
		obj := Eval(l.Left, env)
		if isError(obj) {
			return obj
		}
		index := Eval(l.Index, env)
		if isError(index) {
			return index
		}
		switch o := obj.(type) {
		case *object.Array:
			idx, ok := index.(*object.Integer)
			if !ok {
				return newError("array index must be integer")
			}
			if idx.Value < 0 || idx.Value >= int64(len(o.Elements)) {
				return newError("index out of bounds")
			}
			if assigning {
				o.Elements[idx.Value] = newVal
				return newVal
			}
			return o.Elements[idx.Value]
		case *object.Hash:
			key, ok := index.(object.Hashable)
			if !ok {
				return newError("unusable as hash key")
			}
			if assigning {
				o.Pairs[key.HashKey()] = object.HashPair{Key: index, Value: newVal}
				return newVal
			}
			if pair, ok := o.Pairs[key.HashKey()]; ok {
				return pair.Value
			}
			return NIL
		default:
			return newError("index assignment not supported on %s", obj.Type())
		}
	default:
		return newError("invalid assignment target")
	}
}

// lookupMethod 沿继承链向上查找方法（当前类 → 父类 → 祖父类...）
// 返回方法及其实际定义所在的类
func lookupMethod(class *object.Class, name string) (*object.Function, *object.Class) {
	for c := class; c != nil; c = c.SuperClass {
		if m, ok := c.Methods[name]; ok {
			return m, c
		}
	}
	return nil, nil
}

func evalPropertyExpression(pe *ast.PropertyExpression, env *object.Environment) object.Object {
	// super.property 特殊处理
	if _, isSuper := pe.Object.(*ast.SuperExpression); isSuper {
		return evalSuperProperty(pe.Property, env)
	}
	obj := Eval(pe.Object, env)
	if isError(obj) {
		return obj
	}
	switch o := obj.(type) {
	case *object.Instance:
		if val, ok := o.Fields[pe.Property]; ok {
			return val
		}
		if m, defClass := lookupMethod(o.Class, pe.Property); m != nil {
			return &object.Method{Fn: m, Receiver: o, Class: defClass}
		}
		return NIL
	case *object.Class:
		if m, ok := o.Methods[pe.Property]; ok {
			return m
		}
		if val, ok := o.Properties[pe.Property]; ok {
			return val
		}
		return NIL
	case *object.Module:
		if val, ok := o.Env.Get(pe.Property); ok {
			// traveller 变量透明解包
			if trav, isTrav := val.(*object.Traveller); isTrav {
				return trav.Current
			}
			return val
		}
		return NIL
	case *object.Hash:
		key := &object.String{Value: pe.Property}
		if pair, ok := o.Pairs[key.HashKey()]; ok {
			return pair.Value
		}
		return NIL
	default:
		return newError("property access not supported on %s", obj.Type())
	}
}

func evalMethodCallExpression(mce *ast.MethodCallExpression, env *object.Environment) object.Object {
	// super.method(...) 特殊处理
	if _, isSuper := mce.Object.(*ast.SuperExpression); isSuper {
		m := evalSuperProperty(mce.Method, env)
		if isError(m) {
			return m
		}
		args := evalExpressions(mce.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(m, args, env)
	}

	// 变量回滚：x.rollback()
	// 若 x 是 Traveller 对象，走其历史栈；否则走 Environment 的历史栈
	if id, isIdent := mce.Object.(*ast.Identifier); isIdent && mce.Method == "rollback" {
		if len(mce.Arguments) > 0 {
			return newError("rollback takes no arguments")
		}
		if existing, ok := env.Get(id.Value); ok {
			if trav, isTrav := existing.(*object.Traveller); isTrav {
				oldVal, hasHistory := trav.Rollback()
				if !hasHistory {
					return newError("Traveller history is empty")
				}
				return oldVal
			}
		}
		// 普通变量的环境级回滚
		oldVal, ok := env.Rollback(id.Value)
		if !ok {
			return newError("cannot rollback %s: no previous value", id.Value)
		}
		return oldVal
	}

	// 时间旅行 API：x.history() / x.at(n) / x.changes() / x.step() / x.reset()
	if id, isIdent := mce.Object.(*ast.Identifier); isIdent {
		if existing, ok := env.Get(id.Value); ok {
			if trav, isTrav := existing.(*object.Traveller); isTrav {
				switch mce.Method {
				case "history":
					items := trav.HistoryList()
					out := make([]object.Object, len(items))
					copy(out, items)
					return &object.Array{Elements: out}

				case "at":
					if len(mce.Arguments) != 1 {
						return newError("at() 需要一个索引")
					}
					idxVal := Eval(mce.Arguments[0], env)
					if isError(idxVal) {
						return idxVal
					}
					n, ok := idxVal.(*object.Integer)
					if !ok {
						return newError("at() 的索引要是整数")
					}
					v, found := trav.At(int(n.Value))
					if !found {
						return newError("历史索引 %d 超出范围（共 %d 步）", n.Value, trav.Step())
					}
					return v

				case "changes":
					changes := trav.Changes()
					out := make([]object.Object, 0, len(changes))
					for _, c := range changes {
						pairs := map[object.HashKey]object.HashPair{}
						k1 := &object.String{Value: "from"}
						k2 := &object.String{Value: "to"}
						pairs[k1.HashKey()] = object.HashPair{Key: k1, Value: c[0]}
						pairs[k2.HashKey()] = object.HashPair{Key: k2, Value: c[1]}
						out = append(out, &object.Hash{Pairs: pairs})
					}
					return &object.Array{Elements: out}

				case "step":
					return &object.Integer{Value: int64(trav.Step())}

				case "reset":
					trav.Reset()
					return NIL
				}
			}
		}
	}

	obj := Eval(mce.Object, env)
	if isError(obj) {
		return obj
	}
	args := evalExpressions(mce.Arguments, env)
	if len(args) == 1 && isError(args[0]) {
		return args[0]
	}

	switch o := obj.(type) {
	case *FfiLib:
		// FFI 库方法：lib.call("sqrt", 16.0) / lib.declare(...)
		return ffiLibCall(o, mce.Method, args)

	case *object.Instance:
		if m, defClass := lookupMethod(o.Class, mce.Method); m != nil {
			return applyFunction(&object.Method{Fn: m, Receiver: o, Class: defClass}, args, env)
		}
		// 支持实例字段上的可调用对象（动态方法/闭包）
		if field, ok := o.Fields[mce.Method]; ok {
			if fn, ok := field.(*object.Function); ok {
				return applyFunction(&object.Method{Fn: fn, Receiver: o, Class: o.Class}, args, env)
			}
			return applyFunction(field, args, env)
		}
		return newError("method not found: %s", mce.Method)
	case *object.Class:
		if m, ok := o.Methods[mce.Method]; ok {
			return applyFunction(m, args, env)
		}
		return newError("method not found: %s", mce.Method)
	case *object.String:
		if m, ok := stringMethods[mce.Method]; ok {
			return m.Fn(append([]object.Object{obj}, args...)...)
		}
		return newError("method not found: %s", mce.Method)
	case *object.Array:
		// 高阶方法：需要执行用户传入的函数，单独处理
		switch mce.Method {
		case "map", "filter", "forEach", "reduce", "find":
			return evalArrayHigherOrderMethod(o, mce.Method, args, env)
		}
		if m, ok := arrayMethods[mce.Method]; ok {
			return m.Fn(append([]object.Object{obj}, args...)...)
		}
		return newError("method not found: %s", mce.Method)
	case *object.Hash:
		if m, ok := hashMethods[mce.Method]; ok {
			return m.Fn(append([]object.Object{obj}, args...)...)
		}
		return newError("method not found: %s", mce.Method)
	case *object.Timeline:
		return evalTimelineMethod(o, mce.Method, args)
	case *object.Module:
		// 模块内函数/可调用对象：从模块环境取出并调用
		if val, ok := o.Env.Get(mce.Method); ok {
			if trav, isTrav := val.(*object.Traveller); isTrav {
				val = trav.Current
			}
			return applyFunction(val, args, env)
		}
		return newError("module %s has no member: %s", o.Name, mce.Method)
	default:
		return newError("method call not supported on %s", obj.Type())
	}
}

// evalTimelineMethod 实现 Timeline 的 commit() / undo()
func evalTimelineMethod(t *object.Timeline, method string, args []object.Object) object.Object {
	if len(args) != 0 {
		return newError("%s takes no arguments", method)
	}
	switch method {
	case "commit":
		snap := object.CaptureSnapshot(t.Target)
		if snap == nil {
			return newError("timeline target is no longer array or hash")
		}
		t.History = append(t.History, *snap)
		return NIL
	case "undo":
		if len(t.History) == 0 {
			return newError("timeline undo: no committed state to restore")
		}
		last := t.History[len(t.History)-1]
		t.History = t.History[:len(t.History)-1]
		object.RestoreSnapshot(t.Target, &last)
		return t.Target
	default:
		return newError("unknown timeline method: %s", method)
	}
}

// evalArrayHigherOrderMethod 实现数组的高阶方法（map/filter/forEach/reduce/find）
// 这些方法需要执行用户传入的函数，所以必须拿到 env
func evalArrayHigherOrderMethod(a *object.Array, method string, args []object.Object, env *object.Environment) object.Object {
	if len(args) == 0 {
		return newError("%s expects a function argument", method)
	}
	callback := args[0]

	switch method {
	case "map":
		result := make([]object.Object, 0, len(a.Elements))
		for _, e := range a.Elements {
			r := applyFunction(callback, []object.Object{e}, env)
			if isError(r) {
				return r
			}
			result = append(result, r)
		}
		return &object.Array{Elements: result}

	case "filter":
		result := []object.Object{}
		for _, e := range a.Elements {
			r := applyFunction(callback, []object.Object{e}, env)
			if isError(r) {
				return r
			}
			if isTruthy(r) {
				result = append(result, e)
			}
		}
		return &object.Array{Elements: result}

	case "forEach":
		for _, e := range a.Elements {
			r := applyFunction(callback, []object.Object{e}, env)
			if isError(r) {
				return r
			}
		}
		return NIL

	case "find":
		for _, e := range a.Elements {
			r := applyFunction(callback, []object.Object{e}, env)
			if isError(r) {
				return r
			}
			if isTruthy(r) {
				return e
			}
		}
		return NIL

	case "reduce":
		// reduce(fn(acc, item), initial?) 或 reduce(fn(acc, item))
		if len(a.Elements) == 0 {
			if len(args) >= 2 {
				return args[1]
			}
			return newError("reduce of empty array with no initial value")
		}
		startIdx := 0
		var acc object.Object
		if len(args) >= 2 {
			acc = args[1]
		} else {
			acc = a.Elements[0]
			startIdx = 1
		}
		for i := startIdx; i < len(a.Elements); i++ {
			r := applyFunction(callback, []object.Object{acc, a.Elements[i]}, env)
			if isError(r) {
				return r
			}
			acc = r
		}
		return acc

	default:
		return newError("unknown array method: %s", method)
	}
}

func evalClassLiteral(cl *ast.ClassLiteral, env *object.Environment) object.Object {
	var superClass *object.Class
	if cl.SuperClass != nil {
		superVal := Eval(cl.SuperClass, env)
		if isError(superVal) {
			return superVal
		}
		if sc, ok := superVal.(*object.Class); ok {
			superClass = sc
		} else {
			return newError("superclass must be a class, got %s", superVal.Type())
		}
		// 继承环检测：沿父类链查找，若链上已存在与新类同名的类，则构成环
		if hasInheritanceCycle(superClass, cl.Name.Value) {
			return newError("inheritance cycle detected: class %s extends %s", cl.Name.Value, superClass.Name)
		}
	}

	methods := make(map[string]*object.Function)
	for name, fn := range cl.Methods {
		methods[name] = &object.Function{
			Parameters: fn.Parameters,
			Body:       fn.Body,
			Env:        env,
			Name:       name,
		}
	}

	properties := make(map[string]object.Object)
	for name, prop := range cl.Properties {
		val := Eval(prop, env)
		if isError(val) {
			return val
		}
		properties[name] = val
	}

	class := &object.Class{
		Name:       cl.Name.Value,
		SuperClass: superClass,
		Methods:    methods,
		Properties: properties,
		TypeParams: cl.TypeParams,
	}

	// 检查 implements 的接口是否都实现了
	for _, ifaceIdent := range cl.Interfaces {
		ifaceVal, ok := env.Get(ifaceIdent.Value)
		if !ok {
			return newError("类 %s 声明实现接口 %s，但该接口不存在",
				cl.Name.Value, ifaceIdent.Value)
		}
		iface, ok := ifaceVal.(*object.Interface)
		if !ok {
			return newError("%s 不是接口（%s）", ifaceIdent.Value, ifaceVal.Type())
		}
		// 逐个检查方法
		for _, methodName := range iface.Methods {
			if _, defClass := lookupMethod(class, methodName); defClass == nil {
				return newError("类 %s 声明实现接口 %s，但缺少方法 %s()",
					cl.Name.Value, iface.Name, methodName)
			}
		}
	}

	env.Define(cl.Name.Value, class)
	return class
}

// evalTransactionStatement 执行事务块：
// 先记录快照，执行块内代码；若抛错则恢复快照并重新抛出
func evalTransactionStatement(ts *ast.TransactionStatement, env *object.Environment) object.Object {
	snap := env.SnapshotMark("transaction")

	result := Eval(ts.Body, env)

	// 出错：回滚后把错误往外抛
	if isError(result) {
		env.RestoreMark(snap)
		return result
	}

	// throw 语句返回的是错误对象，同样回滚
	if err, ok := result.(*object.Error); ok {
		env.RestoreMark(snap)
		return err
	}

	return result
}

// evalInterfaceLiteral 求值接口声明
func evalInterfaceLiteral(il *ast.InterfaceLiteral, env *object.Environment) object.Object {
	if il.Name == nil {
		return newError("interface 缺少名字")
	}
	methods := make([]string, 0, len(il.Methods))
	for _, m := range il.Methods {
		methods = append(methods, m.Name)
	}
	iface := &object.Interface{
		Name:    il.Name.Value,
		Methods: methods,
	}
	env.Define(il.Name.Value, iface)
	return iface
}

// hasInheritanceCycle 沿父类链查找，若链上存在名为 className 的类则返回 true（说明会构成环）
func hasInheritanceCycle(super *object.Class, className string) bool {
	seen := make(map[*object.Class]bool)
	for super != nil {
		// 按名字判断：新类的父类链上若已有同名类，定义后必然成环
		if super.Name == className {
			return true
		}
		// 按指针去重，防止遍历时自身成环导致死循环（防御性）
		if seen[super] {
			return true
		}
		seen[super] = true
		super = super.SuperClass
	}
	return false
}

func evalNewExpression(ne *ast.NewExpression, env *object.Environment) object.Object {
	classObj := Eval(ne.Class, env)
	if isError(classObj) {
		return classObj
	}
	class, ok := classObj.(*object.Class)
	if !ok {
		return newError("new requires a class, got %s", classObj.Type())
	}

	// 泛型实例化：new Box<int>(5)
	if len(class.TypeParams) > 0 {
		if len(ne.TypeArgs) == 0 {
			return newError("泛型类 %s 需要类型参数，写成 new %s<%s>(...)",
				class.Name, class.Name, strings.Join(class.TypeParams, ", "))
		}
		if len(ne.TypeArgs) != len(class.TypeParams) {
			return newError("%s 需要 %d 个类型参数，收到 %d 个",
				class.Name, len(class.TypeParams), len(ne.TypeArgs))
		}
		// 生成特化后的类
		specialized := specializeClass(class, ne.TypeArgs)
		args := evalExpressions(ne.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return instantiate(specialized, args)
	}

	if len(ne.TypeArgs) > 0 {
		return newError("%s 不是泛型类，不能带类型参数", class.Name)
	}

	args := evalExpressions(ne.Arguments, env)
	if len(args) == 1 && isError(args[0]) {
		return args[0]
	}
	return instantiate(class, args)
}

// specializeClass 把泛型类按类型实参特化：T → int
// 返回一个浅拷贝，参数类型和字段类型里的 T 都被替换
func specializeClass(class *object.Class, typeArgs []string) *object.Class {
	// 建替换表：T → int
	subst := make(map[string]string, len(class.TypeParams))
	for i, tp := range class.TypeParams {
		if i < len(typeArgs) {
			subst[tp] = typeArgs[i]
		}
	}

	replace := func(typeName string) string {
		if v, ok := subst[typeName]; ok {
			return v
		}
		// 处理 T* 这种指针类型
		if strings.HasSuffix(typeName, "*") {
			base := strings.TrimSuffix(typeName, "*")
			if v, ok := subst[base]; ok {
				return v + "*"
			}
		}
		return typeName
	}

	// 复制方法，替换参数类型和返回类型
	methods := make(map[string]*object.Function, len(class.Methods))
	for name, m := range class.Methods {
		params := make([]*ast.Identifier, len(m.Parameters))
		for i, p := range m.Parameters {
			cp := *p
			if cp.TypeHint != "" {
				cp.TypeHint = replace(cp.TypeHint)
			}
			params[i] = &cp
		}
		methods[name] = &object.Function{
			Parameters: params,
			Body:       m.Body,
			Env:        m.Env,
			Name:       m.Name,
			ReturnType: replace(m.ReturnType),
		}
	}

	// 从 init 方法的参数类型推断字段类型：
	// fn init(v: T) 里的 T 特化成 int，则 this.v 也应该是 int
	fieldTypes := make(map[string]string)
	if initFn, ok := class.Methods["init"]; ok {
		for _, p := range initFn.Parameters {
			if p.TypeHint == "" {
				continue
			}
			// 只记录能替换成具体类型的
			replaced := replace(p.TypeHint)
			if replaced != p.TypeHint {
				fieldTypes[p.Value] = replaced
			}
		}
	}

	// 名字带上类型实参，方便调试
	specName := class.Name + "<" + strings.Join(typeArgs, ", ") + ">"
	return &object.Class{
		Name:       specName,
		SuperClass: class.SuperClass,
		Methods:    methods,
		Properties: class.Properties,
		TypeParams: class.TypeParams,
		TypeArgs:   typeArgs,
		FieldTypes: fieldTypes,
	}
}

func instantiate(class *object.Class, args []object.Object) object.Object {
	instance := &object.Instance{
		Class:      class,
		Fields:     make(map[string]object.Object),
		FieldTypes: class.FieldTypes,
	}
	for name, val := range class.Properties {
		instance.Fields[name] = val
	}
	if init, defClass := lookupMethod(class, "init"); init != nil {
		extended := extendFunctionEnv(init, args)
		extended.Define(thisKey, instance)
		extended.Define(superKey, defClass)
		result := unwrapReturnValue(Eval(init.Body, extended))
		if isError(result) {
			return result
		}
	}
	return instance
}

func evalThisExpression(env *object.Environment) object.Object {
	if val, ok := env.Get(thisKey); ok {
		return val
	}
	return newError("'this' used outside of method")
}

func evalSuperExpression(se *ast.SuperExpression, env *object.Environment) object.Object {
	return newError("'super' must be followed by a method/property access")
}

// evalSuperProperty 获取超类的方法或属性（绑定当前实例为 this）
func evalSuperProperty(name string, env *object.Environment) object.Object {
	thisObj, ok := env.Get(thisKey)
	if !ok {
		return newError("'super' used outside of method")
	}
	inst, ok := thisObj.(*object.Instance)
	if !ok {
		return newError("'super' used outside of instance method")
	}

	// 当前方法所属的类（由 applyFunction 绑定），super 从它的父类开始找
	var curClass *object.Class
	if c, ok := env.Get(superKey); ok {
		if cc, ok := c.(*object.Class); ok {
			curClass = cc
		}
	}
	if curClass == nil {
		curClass = inst.Class
	}

	if curClass.SuperClass == nil {
		return newError("class %s has no superclass", curClass.Name)
	}
	if m, defClass := lookupMethod(curClass.SuperClass, name); m != nil {
		return &object.Method{Fn: m, Receiver: inst, Class: defClass}
	}
	if val, ok := curClass.SuperClass.Properties[name]; ok {
		return val
	}
	return newError("superclass has no member: %s", name)
}

func evalImportStatement(is *ast.ImportStatement, env *object.Environment) object.Object {
	if is.Module == nil {
		return newError("import requires a module path string")
	}
	modPath := is.Module.Value

	// 先尝试按文件解析（用户自己的模块优先）
	fullPath := resolveModulePath(modPath, env.ImportDir)

	// 没有对应文件时才用内置模块：import "os" / import "math"
	if fullPath == "" {
		if mod := lookupBuiltinModule(modPath); mod != nil {
			bindName := strings.TrimSuffix(filepath.Base(modPath), filepath.Ext(modPath))
			if is.Alias != nil {
				bindName = is.Alias.Value
			}
			env.Define(bindName, mod)
			return mod
		}
		return newError("cannot resolve import path: %s", modPath)
	}

	// 循环 import 检测
	if env.ImportChain[fullPath] {
		return newError("circular import detected: %s", modPath)
	}

	// 读取并解析模块文件（虚拟文件系统优先，WASM 环境用）
	src, err := readModuleFile(fullPath)
	if err != nil {
		return newError("cannot read module %s: %v", modPath, err)
	}

	l := lexer.New(string(src))
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return newError("parse error in module %s: %v", modPath, p.Errors()[0])
	}

	// 在全新的独立环境中执行
	modEnv := object.NewEnvironment()
	modEnv.ImportDir = filepath.Dir(fullPath)
	modEnv.ImportChain = make(map[string]bool)
	for k := range env.ImportChain {
		modEnv.ImportChain[k] = true
	}
	modEnv.ImportChain[fullPath] = true

	result := Eval(program, modEnv)
	if isError(result) {
		return newError("error executing module %s: %s", modPath, result.Inspect())
	}

	// 打包成 Module 对象
	module := &object.Module{
		Name:    filepath.Base(modPath),
		Env:     modEnv,
		Exports: modEnv.Snapshot(),
	}

	// 确定绑定名：as 别名优先，否则用文件名（去扩展名）
	bindName := ""
	if is.Alias != nil {
		bindName = is.Alias.Value
	} else {
		bindName = strings.TrimSuffix(filepath.Base(modPath), filepath.Ext(modPath))
	}
	if bindName == "" {
		bindName = strings.TrimSuffix(modPath, filepath.Ext(modPath))
	}

	env.Define(bindName, module)
	return module
}

// resolveModulePath 把 import 路径解析成实际文件路径
// 优先 .chpp（新后缀），兼容旧版 .px。
func resolveModulePath(modPath, baseDir string) string {
	// 已是绝对路径
	if filepath.IsAbs(modPath) {
		if moduleFileExists(modPath) {
			return modPath
		}
		for _, ext := range []string{".chpp", ".chpp", ".px"} {
			if moduleFileExists(modPath + ext) {
				return modPath + ext
			}
		}
		return ""
	}

	// 相对路径：基于 baseDir 解析
	dir := baseDir
	if dir == "" {
		dir = "."
	}
	candidates := []string{filepath.Join(dir, modPath)}
	for _, ext := range []string{".chpp", ".chpp", ".px"} {
		candidates = append(candidates,
			filepath.Join(dir, modPath+ext),
			filepath.Join(dir, modPath, "index"+ext),
		)
	}
	for _, c := range candidates {
		if moduleFileExists(c) {
			return c
		}
	}

	// 本地找不到时，查已安装的第三方包：~/.chronos/packages/<name>/
	// 支持 import "mypkg" 和 import "mypkg/sub"
	if home, err := os.UserHomeDir(); err == nil {
		pkgDir := filepath.Join(home, ".chronos", "packages", modPath)
		if moduleFileExists(pkgDir) {
			return pkgDir
		}
		for _, ext := range []string{".chpp", ".chpp", ".px"} {
			if moduleFileExists(pkgDir + ext) {
				return pkgDir + ext
			}
			indexPath := filepath.Join(pkgDir, "index"+ext)
			if moduleFileExists(indexPath) {
				return indexPath
			}
		}
		// 包有 manifest 时按 manifest.main 找
		mfPath := filepath.Join(pkgDir, "manifest.json")
		if data, err := os.ReadFile(mfPath); err == nil {
			var mf struct {
				Main string `json:"main"`
			}
			if json.Unmarshal(data, &mf) == nil && mf.Main != "" {
				mainPath := filepath.Join(pkgDir, mf.Main)
				if moduleFileExists(mainPath) {
					return mainPath
				}
			}
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func evalExportStatement(es *ast.ExportStatement, env *object.Environment) object.Object {
	if es.Value != nil {
		return Eval(es.Value, env)
	}
	return NIL
}

func evalTryStatement(ts *ast.TryStatement, env *object.Environment) object.Object {
	result := Eval(ts.Body, env)

	if isError(result) && ts.CatchBody != nil {
		catchEnv := object.NewEnclosedEnvironment(env)
		if ts.CatchVar != nil {
			// 异常对象绑定为错误消息字符串（而不是 Error 对象本身，
			// 这样 catch 块内可以正常打印、拼接，避免 Error 短路传播）
			errMsg := result.(*object.Error).Message
			catchEnv.Define(ts.CatchVar.Value, &object.String{Value: errMsg})
		}
		result = Eval(ts.CatchBody, catchEnv)
	}

	if ts.FinallyBody != nil {
		finalResult := Eval(ts.FinallyBody, env)
		// finally 的错误优先
		if isError(finalResult) {
			return finalResult
		}
	}
	return result
}

func evalThrowStatement(ts *ast.ThrowStatement, env *object.Environment) object.Object {
	val := Eval(ts.Value, env)
	if isError(val) {
		return val
	}
	// 抛出的值包装为错误对象
	switch v := val.(type) {
	case *object.String:
		return newError("%s", v.Value)
	default:
		return newError("%s", val.Inspect())
	}
}

func evalMatchExpression(me *ast.MatchExpression, env *object.Environment) object.Object {
	value := Eval(me.Value, env)
	if isError(value) {
		return value
	}
	for _, c := range me.Cases {
		condValue := Eval(c.Condition, env)
		if isError(condValue) {
			return condValue
		}
		if objectsEqual(value, condValue) {
			return Eval(c.Body, env)
		}
	}
	if me.Default != nil {
		return Eval(me.Default, env)
	}
	return NIL
}

func objectsEqual(a, b object.Object) bool {
	if a.Type() != b.Type() {
		return false
	}
	switch av := a.(type) {
	case *object.Integer:
		return av.Value == b.(*object.Integer).Value
	case *object.Float:
		return av.Value == b.(*object.Float).Value
	case *object.String:
		return av.Value == b.(*object.String).Value
	case *object.Boolean:
		return av.Value == b.(*object.Boolean).Value
	case *object.Nil:
		return true
	default:
		return a == b
	}
}

func nativeBoolToBooleanObject(b bool) *object.Boolean {
	if b {
		return TRUE
	}
	return FALSE
}
