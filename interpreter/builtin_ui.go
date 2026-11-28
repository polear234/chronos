package interpreter

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"chronospp/object"
)

// ── UI 元素树 ──────────────────────────────

// UIElement 一个 UI 组件
type UIElement struct {
	Tag        string            // "title" "text" "button" "input" "list" "progress" "select" "checkbox" "slider" "image" "row" "column" "card"
	Text       string
	Attrs      map[string]string
	Children   []*UIElement
	Callback   *object.Function   // 点击/输入时的回调
	CallbackID string            // 回调注册 ID
	ElemID      string            // 元素唯一 ID（用于 update）
	callbackEnv *object.Environment  // 回调执行环境
	dirty       bool               // 是否被 update 过（局部更新用）
}

func (el *UIElement) Type() object.ObjectType { return "UI_ELEMENT" }
func (el *UIElement) Inspect() string {
	return "<ui " + el.Tag + " id=" + el.ElemID + ">"
}

// Update 更新元素文本（返回新值供链式使用）
func (el *UIElement) Update(text string) {
	el.Text = text
	el.dirty = true
}

// UIUpdate 一个元素的局部更新
type UIUpdate struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// CollectUpdates 收集并清空所有 dirty 元素（供前端局部更新）
func CollectUpdates() []UIUpdate {
	out := []UIUpdate{}
	walkElements(uiState.Root, &out)
	return out
}

func walkElements(els []*UIElement, out *[]UIUpdate) {
	for _, el := range els {
		if el.dirty {
			*out = append(*out, UIUpdate{ID: el.ElemID, Text: el.Text})
			el.dirty = false
		}
		walkElements(el.Children, out)
	}
}

// UIState 全局 UI 状态
type UIState struct {
	mu        sync.Mutex
	Root      []*UIElement                  // 顶层元素列表
	Callbacks map[string]*object.Function   // 回调注册表
	Elements  map[string]*UIElement         // 元素 ID → 元素（update 用）
	Seq       int                           // ID 计数器
	needFullRedraw bool                     // 结构性变化时标记需要整页重绘
}

var uiState = &UIState{
	Callbacks: make(map[string]*object.Function),
	Elements:  make(map[string]*UIElement),
}

// ResetUI 清空 UI 状态（供 WASM 每次执行前调用）
func ResetUI() { resetUI() }

// resetUI 清空 UI 状态
func resetUI() {
	uiState.Root = nil
	uiState.Callbacks = make(map[string]*object.Function)
	uiState.Elements = make(map[string]*UIElement)
	uiState.Seq = 0
	uiState.needFullRedraw = false
}

// registerElement 注册元素，分配 ID
func registerElement(el *UIElement) *UIElement {
	uiState.Seq++
	el.ElemID = fmt.Sprintf("e%d", uiState.Seq)
	uiState.Elements[el.ElemID] = el
	return el
}

// registerCallback 注册一个回调，返回 ID
func registerCallback(fn *object.Function) string {
	uiState.Seq++
	id := fmt.Sprintf("cb_%d", uiState.Seq)
	uiState.Callbacks[id] = fn
	return id
}

// InvokeUICallbackResult 回调结果
type InvokeUICallbackResult struct {
	Updates  []UIUpdate `json:"updates"`            // 局部更新
	FullHTML string     `json:"fullHtml,omitempty"` // 整页重绘（ui.reset 时才用）
}

// InvokeUICallback 由外部（WASM/HTTP）调用回调
// 返回 JSON：默认只给局部更新，避免重建 DOM 打断交互
func InvokeUICallback(id string, arg string) string {
	uiState.mu.Lock()
	fn, ok := uiState.Callbacks[id]
	uiState.mu.Unlock()
	if !ok {
		return "{}"
	}

	// 回调执行环境：优先用注册时的环境
	env := fn.Env
	if env == nil {
		env = object.NewEnvironment()
	}
	extended := object.NewEnclosedEnvironment(env)
	if len(fn.Parameters) > 0 {
		extended.Define(fn.Parameters[0].Value, &object.String{Value: arg})
	} else {
		extended.Define("value", &object.String{Value: arg})
	}

	// 回调开始前清标记：只有回调里真的新增元素才整页重绘
	uiState.needFullRedraw = false
	Eval(fn.Body, extended)

	// 收集被 update 过的元素
	updates := CollectUpdates()

	// 只有元素集合发生结构性变化（调用过 ui.xxx 新增元素）时才整页重绘
	result := InvokeUICallbackResult{Updates: updates}

	// 检测：如果回调里调用了 ui.xxx() 或 ui.reset()，需要整页重绘
	if uiState.needFullRedraw {
		result.FullHTML = GetUIHTML()
		uiState.needFullRedraw = false
	}

	b, _ := json.Marshal(result)
	return string(b)
}

// GetUIHTML 返回生成的 HTML（供 WebView/浏览器渲染）
func GetUIHTML() string {
	// 没有添加任何 UI 元素时返回空串，调用方据此判断是否要开界面
	if len(uiState.Root) == 0 {
		return ""
	}
	return generateHTML(uiState.Root)
}

// ── HTML 生成 ──────────────────────────────

func generateHTML(elements []*UIElement) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html><head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
.chronos-ui * { box-sizing: border-box; margin: 0; padding: 0; }
.chronos-ui { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  background: #f5f5f5; color: #333; padding: 16px; max-width: 600px; margin: 0 auto;
  min-height: 100vh; }
.chronos-ui h1 { font-size: 24px; margin-bottom: 16px; }
.chronos-ui p, .chronos-ui .text { margin: 8px 0; line-height: 1.6; }
.chronos-ui button { background: #007AFF; color: #fff; border: none; padding: 12px 24px;
  border-radius: 8px; font-size: 16px; cursor: pointer; margin: 8px 0; }
.chronos-ui button:active { background: #0056CC; }
.chronos-ui input, .chronos-ui select { width: 100%; padding: 12px; border: 1px solid #ddd;
  border-radius: 8px; font-size: 16px; margin: 8px 0; }
.chronos-ui ul { margin: 8px 0; padding-left: 24px; }
.chronos-ui li { margin: 4px 0; }
.chronos-ui .progress-bar { width: 100%; height: 20px; background: #e0e0e0;
  border-radius: 10px; margin: 8px 0; }
.chronos-ui .progress-fill { height: 100%; background: #007AFF; border-radius: 10px;
  transition: width 0.3s; }
.chronos-ui .row { display: flex; gap: 8px; margin: 8px 0; }
.chronos-ui .column { display: flex; flex-direction: column; gap: 8px; margin: 8px 0; }
.chronos-ui .card { background: #fff; border-radius: 12px; padding: 16px; margin: 8px 0;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
.chronos-ui .checkbox-label { display: flex; align-items: center; gap: 8px; margin: 8px 0; }
.chronos-ui .checkbox-label input { width: auto; margin: 0; }
.chronos-ui .slider-wrap { display: flex; align-items: center; gap: 12px; margin: 8px 0; }
.chronos-ui .slider { flex: 1; width: auto; margin: 0; height: 6px;
  -webkit-appearance: none; appearance: none; background: #ddd; border-radius: 3px; outline: none; }
.chronos-ui .slider::-webkit-slider-thumb { -webkit-appearance: none; appearance: none;
  width: 22px; height: 22px; border-radius: 50%; background: #007AFF; cursor: pointer;
  box-shadow: 0 2px 6px rgba(0,122,255,0.4); }
.chronos-ui .slider::-moz-range-thumb { width: 22px; height: 22px; border-radius: 50%;
  background: #007AFF; cursor: pointer; border: none; }
.chronos-ui .slider-val { min-width: 42px; text-align: right; font-weight: 600;
  font-size: 15px; color: #007AFF; font-variant-numeric: tabular-nums; }
.chronos-ui label { font-weight: 500; margin: 4px 0; display: block; }
.chronos-ui img { max-width: 100%; border-radius: 8px; margin: 8px 0; }
</style>
</head><body><div class="chronos-ui">
`)

	for _, el := range elements {
		writeElement(&sb, el, 0)
	}

	// JavaScript 回调桥
	sb.WriteString(`
<script>
function chronosCallback(id, value) {
  // 尝试调用 WASM 桥
  if (typeof Module !== 'undefined' && Module.chronosUICallback) {
    Module.chronosUICallback(id, value);
    return;
  }
  // 尝试调用全局桥
  if (typeof window.chronosUICallback === 'function') {
    window.chronosUICallback(id, value);
    return;
  }
  // HTTP 桥（桌面端 chpp serve）
  fetch('/__ui_callback', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({id: id, value: value})
  }).catch(function(){});
}
</script>
</div></body></html>`)

	return sb.String()
}

func writeElement(sb *strings.Builder, el *UIElement, depth int) {
	indent := strings.Repeat("  ", depth)
	attrs := ""
	for k, v := range el.Attrs {
		attrs += fmt.Sprintf(` %s="%s"`, k, v)
	}

	switch el.Tag {
	case "title":
		sb.WriteString(indent + "<h1 id='" + el.ElemID + "'>" + escHTML(el.Text) + "</h1>\n")
	case "text":
		sb.WriteString(indent + "<p class='text' id='" + el.ElemID + "'>" + escHTML(el.Text) + "</p>\n")
	case "button":
		cbID := el.CallbackID
		if cbID != "" {
			sb.WriteString(fmt.Sprintf(`%s<button id='%s' onclick="chronosCallback('%s', this.textContent)">%s</button>`+"\n",
				indent, el.ElemID, cbID, escHTML(el.Text)))
		} else {
			sb.WriteString(indent + "<button id='" + el.ElemID + "'>" + escHTML(el.Text) + "</button>\n")
		}
	case "input":
		label := el.Attrs["label"]
		if label != "" {
			sb.WriteString(indent + "<label>" + escHTML(label) + "</label>\n")
		}
		cbID := el.CallbackID
		ph := el.Text
		if ph == "" {
			ph = "请输入..."
		}
		typ := el.Attrs["type"]
		if typ == "" {
			typ = "text"
		}
		if cbID != "" {
			sb.WriteString(fmt.Sprintf(`%s<input type="%s" placeholder="%s" onchange="chronosCallback('%s', this.value)" />`+"\n",
				indent, typ, escHTML(ph), cbID))
		} else {
			sb.WriteString(fmt.Sprintf(`%s<input type="%s" placeholder="%s" />`+"\n",
				indent, typ, escHTML(ph)))
		}
	case "select":
		label := el.Attrs["label"]
		if label != "" {
			sb.WriteString(indent + "<label>" + escHTML(label) + "</label>\n")
		}
		cbID := el.CallbackID
		sb.WriteString(indent + "<select")
		if cbID != "" {
			sb.WriteString(fmt.Sprintf(` onchange="chronosCallback('%s', this.value)"`, cbID))
		}
		sb.WriteString(">\n")
		for _, child := range el.Children {
			sb.WriteString(indent + "  <option>" + escHTML(child.Text) + "</option>\n")
		}
		sb.WriteString(indent + "</select>\n")
	case "checkbox":
		cbID := el.CallbackID
		checked := ""
		if el.Attrs["checked"] == "true" {
			checked = " checked"
		}
		sb.WriteString(fmt.Sprintf(`%s<div class="checkbox-label"><input type="checkbox" id='%s'%s`, indent, el.ElemID, checked))
		if cbID != "" {
			sb.WriteString(fmt.Sprintf(` onchange="chronosCallback('%s', this.checked ? '1' : '0')"`, cbID))
		}
		sb.WriteString(" /><span>" + escHTML(el.Text) + "</span></div>\n")
	case "slider":
		label := el.Attrs["label"]
		if label != "" {
			sb.WriteString(indent + "<label>" + escHTML(label) + "</label>\n")
		}
		cbID := el.CallbackID
		min := el.Attrs["min"]
		if min == "" {
			min = "0"
		}
		max := el.Attrs["max"]
		if max == "" {
			max = "100"
		}
		val := el.Attrs["value"]
		if val == "" {
			val = "50"
		}
		// 滑块 + 实时数值显示
		sb.WriteString(fmt.Sprintf(`%s<div class="slider-wrap"><input type="range" class="slider" id='%s' min="%s" max="%s" value="%s" step="1"`,
			indent, el.ElemID, min, max, val))
		if cbID != "" {
			// 边滑边回调：立即更新数值显示，回调节流 80ms
			sb.WriteString(fmt.Sprintf(` oninput="`+
				`(function(el){`+
				`  var out=el.parentNode.querySelector('.slider-val');`+
				`  if(out)out.textContent=el.value;`+
				`  if(window.__csTick)clearTimeout(window.__csTick);`+
				`  window.__csTick=setTimeout(function(){chronosCallback('%s', el.value)},80);`+
				`})(this)"`, cbID))
		} else {
			sb.WriteString(` oninput="(function(el){var out=el.parentNode.querySelector('.slider-val');if(out)out.textContent=el.value})(this)"`)
		}
		sb.WriteString(fmt.Sprintf(` /><span class="slider-val">%s</span></div>`+"\n", val))
	case "list":
		sb.WriteString(indent + "<ul>\n")
		for _, child := range el.Children {
			sb.WriteString(indent + "  <li>" + escHTML(child.Text) + "</li>\n")
		}
		sb.WriteString(indent + "</ul>\n")
	case "progress":
		val := el.Attrs["value"]
		if val == "" {
			val = "0"
		}
		sb.WriteString(fmt.Sprintf(`%s<div class="progress-bar" id='%s'><div class="progress-fill" style="width:%s%%"></div></div>`+"\n",
			indent, el.ElemID, val))
	case "image":
		src := el.Attrs["src"]
		sb.WriteString(fmt.Sprintf(`%s<img src="%s" alt="%s" />`+"\n", indent, src, escHTML(el.Text)))
	case "row":
		sb.WriteString(indent + "<div class='row'>\n")
		for _, child := range el.Children {
			writeElement(sb, child, depth+1)
		}
		sb.WriteString(indent + "</div>\n")
	case "column":
		sb.WriteString(indent + "<div class='column'>\n")
		for _, child := range el.Children {
			writeElement(sb, child, depth+1)
		}
		sb.WriteString(indent + "</div>\n")
	case "card":
		sb.WriteString(indent + "<div class='card'>\n")
		for _, child := range el.Children {
			writeElement(sb, child, depth+1)
		}
		sb.WriteString(indent + "</div>\n")
	default:
		sb.WriteString(indent + "<div>" + escHTML(el.Text) + "</div>\n")
	}
}

func escHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// newUiModule 构造 UI 模块
func newUiModule() *object.Module {
	env := object.NewEnvironment()

	// ui.reset() 清空界面
	env.Define("reset", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			resetUI()
			uiState.needFullRedraw = true
			return NIL
		},
	})

	// ui.title("文本") 大标题
	env.Define("title", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 { return newError("ui.title 需要文本") }
			el := registerElement(&UIElement{Tag: "title", Text: args[0].Inspect()})
			uiState.Root = append(uiState.Root, el)
			uiState.needFullRedraw = true
			return el
		},
	})

	// ui.text("文本") 普通文本
	env.Define("text", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 { return newError("ui.text 需要文本") }
			el := registerElement(&UIElement{Tag: "text", Text: args[0].Inspect()})
			uiState.Root = append(uiState.Root, el)
			uiState.needFullRedraw = true
			return el
		},
	})

	// ui.button("按钮", fn) 可点击按钮
	env.Define("button", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 { return newError("ui.button 需要文本") }
			el := &UIElement{Tag: "button", Text: args[0].Inspect()}
			if len(args) > 1 {
				if fn, ok := args[1].(*object.Function); ok {
					el.Callback = fn
					el.CallbackID = registerCallback(fn)
				}
			}
			registerElement(el)
			uiState.Root = append(uiState.Root, el)
			uiState.needFullRedraw = true
			return el
		},
	})

	// ui.input("提示", fn) 输入框（回车/失焦触发）
	env.Define("input", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			el := &UIElement{
				Tag: "input",
				Text: "",
				Attrs: map[string]string{"type": "text"},
			}
			if len(args) > 0 {
				el.Text = args[0].Inspect()
			}
			if len(args) > 1 {
				if fn, ok := args[1].(*object.Function); ok {
					el.Callback = fn
					el.CallbackID = registerCallback(fn)
				}
			}
			registerElement(el)
			uiState.Root = append(uiState.Root, el)
			uiState.needFullRedraw = true
			return el
		},
	})

	// ui.checkbox("标签", checked, fn) 复选框
	env.Define("checkbox", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			el := &UIElement{Tag: "checkbox", Attrs: map[string]string{}}
			if len(args) > 0 { el.Text = args[0].Inspect() }
			if len(args) > 1 {
				if b, ok := args[1].(*object.Boolean); ok && b.Value {
					el.Attrs["checked"] = "true"
				}
			}
			if len(args) > 2 {
				if fn, ok := args[2].(*object.Function); ok {
					el.Callback = fn
					el.CallbackID = registerCallback(fn)
				}
			}
			registerElement(el)
			uiState.Root = append(uiState.Root, el)
			uiState.needFullRedraw = true
			return el
		},
	})

	// ui.slider("标签", min, max, value, fn) 滑块
	env.Define("slider", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			el := &UIElement{
				Tag: "slider",
				Attrs: map[string]string{
					"min": "0", "max": "100", "value": "50",
				},
			}
			if len(args) > 0 { el.Attrs["label"] = args[0].Inspect() }
			if len(args) > 1 { el.Attrs["min"] = args[1].Inspect() }
			if len(args) > 2 { el.Attrs["max"] = args[2].Inspect() }
			if len(args) > 3 { el.Attrs["value"] = args[3].Inspect() }
			if len(args) > 4 {
				if fn, ok := args[4].(*object.Function); ok {
					el.Callback = fn
					el.CallbackID = registerCallback(fn)
				}
			}
			registerElement(el)
			uiState.Root = append(uiState.Root, el)
			uiState.needFullRedraw = true
			return el
		},
	})

	// ui.select("标签", ["选项1","选项2"], fn) 下拉框
	env.Define("select", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			el := &UIElement{
				Tag: "select",
				Attrs: map[string]string{},
			}
			if len(args) > 0 { el.Attrs["label"] = args[0].Inspect() }
			if len(args) > 1 {
				if arr, ok := args[1].(*object.Array); ok {
					for _, item := range arr.Elements {
						el.Children = append(el.Children, &UIElement{
							Tag: "option", Text: item.Inspect(),
						})
					}
				}
			}
			if len(args) > 2 {
				if fn, ok := args[2].(*object.Function); ok {
					el.Callback = fn
					el.CallbackID = registerCallback(fn)
				}
			}
			registerElement(el)
			uiState.Root = append(uiState.Root, el)
			uiState.needFullRedraw = true
			return el
		},
	})

	// ui.list(["a","b","c"]) 列表
	env.Define("list", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			el := &UIElement{Tag: "list"}
			if len(args) > 0 {
				if arr, ok := args[0].(*object.Array); ok {
					for _, item := range arr.Elements {
						el.Children = append(el.Children, &UIElement{
							Tag: "li", Text: item.Inspect(),
						})
					}
				}
			}
			registerElement(el)
			uiState.Root = append(uiState.Root, el)
			uiState.needFullRedraw = true
			return el
		},
	})

	// ui.progress(75) 进度条
	env.Define("progress", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			el := &UIElement{
				Tag: "progress",
				Attrs: map[string]string{"value": "0"},
			}
			if len(args) > 0 {
				el.Attrs["value"] = args[0].Inspect()
			}
			registerElement(el)
			uiState.Root = append(uiState.Root, el)
			uiState.needFullRedraw = true
			return el
		},
	})

	// ui.image("url", "alt") 图片
	env.Define("image", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			el := &UIElement{Tag: "image", Attrs: map[string]string{}}
			if len(args) > 0 { el.Attrs["src"] = args[0].Inspect() }
			if len(args) > 1 { el.Text = args[1].Inspect() }
			registerElement(el)
			uiState.Root = append(uiState.Root, el)
			uiState.needFullRedraw = true
			return el
		},
	})

	// ui.row(...) 横排容器
	env.Define("row", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			el := &UIElement{Tag: "row"}
			for _, a := range args {
				if child, ok := a.(*UIElement); ok {
					el.Children = append(el.Children, child)
				}
			}
			uiState.Root = append(uiState.Root, el)
			return NIL
		},
	})

	// ui.card(...) 卡片容器
	env.Define("card", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			el := &UIElement{Tag: "card"}
			for _, a := range args {
				if child, ok := a.(*UIElement); ok {
					el.Children = append(el.Children, child)
				}
			}
			uiState.Root = append(uiState.Root, el)
			return NIL
		},
	})

	// ui.show() 返回 HTML 字符串（WASM/桌面端渲染）
	env.Define("show", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: GetUIHTML()}
		},
	})

	// ui.html() 同 show，别名
	env.Define("html", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: GetUIHTML()}
		},
	})

	return &object.Module{
		Name:    "ui",
		Env:     env,
		Exports: env.Snapshot(),
	}
}

