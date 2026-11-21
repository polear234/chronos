package object

// Traveller 带状态时间旅行的变量对象
// 包装基础对象，内部维护历史栈记录旧值，支持 rollback()
type Traveller struct {
	Current    Object
	History    []Object // 历史旧值栈，栈顶为最近一次赋值前的值
	MaxHistory int
}

// DefaultMaxHistory 默认最大历史深度，防止内存无限增长
const DefaultMaxHistory = 100

func NewTraveller(current Object) *Traveller {
	return &Traveller{
		Current:    current,
		History:    make([]Object, 0),
		MaxHistory: DefaultMaxHistory,
	}
}

func (t *Traveller) Type() ObjectType { return TRAVELLER_OBJ }
func (t *Traveller) Inspect() string {
	if t.Current != nil {
		return t.Current.Inspect()
	}
	return "nil"
}

// Set 更新当前值：把旧值压入历史栈（超过 MaxHistory 丢弃最老），再赋新值
func (t *Traveller) Set(newVal Object) {
	if t.MaxHistory <= 0 {
		t.MaxHistory = DefaultMaxHistory
	}
	t.History = append(t.History, t.Current)
	if len(t.History) > t.MaxHistory {
		// 丢弃最老的历史
		t.History = t.History[len(t.History)-t.MaxHistory:]
	}
	t.Current = newVal
}

// Rollback 回滚到上一次赋值前的值
// 返回旧值和是否成功（历史为空时返回 false）
func (t *Traveller) Rollback() (Object, bool) {
	if len(t.History) == 0 {
		return nil, false
	}
	oldVal := t.History[len(t.History)-1]
	t.History = t.History[:len(t.History)-1]
	t.Current = oldVal
	return oldVal, true
}
