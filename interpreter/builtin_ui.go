package interpreter

import (
	"fmt"
	"strings"
	"sync"

	"chronospp/object"
)

// ── UI 元素树 ──────────────────────────────

// UIElement 一个 UI 组件
type UIElement struct {
	Tag      string            // "title" "text" "button" "input" "list" "progress" "select" "checkbox" "slider" "image" "row" "column" "card"
	Text     string
	Attrs    map[string]string
	Children []*UIElement
	Callback *object.Function   // 点击/输入时的回调
	CallbackID string          // 回调注册 ID
}

// UIState 全局 UI 状态
type UIState struct {
	mu      sync.Mutex
	Root    []*UIElement        // 顶层元素列表
	Callbacks map[string]*object.Function  // 回调注册表
	Seq     int                // 回调 ID 计数器
	Output  string             // 生成的 HTML
}

var uiState = &UIState{
	Callbacks: make(map[string]*object.Function),
}

// ResetUI 清空 UI 状态（供 WASM 每次执行前调用）
func ResetUI() { resetUI() }

// resetUI 清空 UI 状态
func resetUI() {
	uiState.Root = nil
	uiState.Callbacks = make(map[string]*object.Function)
	uiState.Seq = 0
}

// registerCallback 注册一个回调，返回 ID
func registerCallback(fn *object.Function) string {
	uiState.Seq++
	id := fmt.Sprintf("cb_%d", uiState.Seq)
	uiState.Callbacks[id] = fn
	return id
}

// InvokeUICallback 由外部（WASM/HTTP）调用回调
func InvokeUICallback(id string, arg string) {
	uiState.mu.Lock()
	fn, ok := uiState.Callbacks[id]
	uiState.mu.Unlock()
	if !ok {
		return
	}
	// 调用回调函数
	env := fn.Env
	extended := object.NewEnclosedEnvironment(env)
	if len(fn.Parameters) > 0 {
		extended.Define(fn.Parameters[0].Value, &object.String{Value: arg})
	} else {
		extended.Define("value", &object.String{Value: arg})
	}
	Eval(fn.Body, extended)
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
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  background: #f5f5f5; color: #333; padding: 16px; max-width: 600px; margin: 0 auto; }
h1 { font-size: 24px; margin-bottom: 16px; }
p, .text { margin: 8px 0; line-height: 1.6; }
button { background: #007AFF; color: #fff; border: none; padding: 12px 24px;
  border-radius: 8px; font-size: 16px; cursor: pointer; margin: 8px 0; }
button:active { background: #0056CC; }
input, select { width: 100%; padding: 12px; border: 1px solid #ddd; border-radius: 8px;
  font-size: 16px; margin: 8px 0; }
ul { margin: 8px 0; padding-left: 24px; }
li { margin: 4px 0; }
.progress-bar { width: 100%; height: 20px; background: #e0e0e0; border-radius: 10px; margin: 8px 0; }
.progress-fill { height: 100%; background: #007AFF; border-radius: 10px; transition: width 0.3s; }
.row { display: flex; gap: 8px; margin: 8px 0; }
.column { display: flex; flex-direction: column; gap: 8px; margin: 8px 0; }
.card { background: #fff; border-radius: 12px; padding: 16px; margin: 8px 0; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
.checkbox-label { display: flex; align-items: center; gap: 8px; margin: 8px 0; }
.slider { width: 100%; margin: 8px 0; }
label { font-weight: 500; margin: 4px 0; display: block; }
img { max-width: 100%; border-radius: 8px; margin: 8px 0; }
</style>
</head><body>
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
</body></html>`)

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
		sb.WriteString(indent + "<h1>" + escHTML(el.Text) + "</h1>\n")
	case "text":
		sb.WriteString(indent + "<p class='text'>" + escHTML(el.Text) + "</p>\n")
	case "button":
		cbID := el.CallbackID
		if cbID != "" {
			sb.WriteString(fmt.Sprintf(`%s<button onclick="chronosCallback('%s', this.textContent)">%s</button>`+"\n",
				indent, cbID, escHTML(el.Text)))
		} else {
			sb.WriteString(indent + "<button>" + escHTML(el.Text) + "</button>\n")
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
		sb.WriteString(fmt.Sprintf(`%s<div class="checkbox-label"><input type="checkbox"%s`, indent, checked))
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
		sb.WriteString(fmt.Sprintf(`%s<input type="range" class="slider" min="%s" max="%s" value="%s"`, indent, min, max, val))
		if cbID != "" {
			sb.WriteString(fmt.Sprintf(` oninput="chronosCallback('%s', this.value)"`, cbID))
		}
		sb.WriteString(" />\n")
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
		sb.WriteString(fmt.Sprintf(`%s<div class="progress-bar"><div class="progress-fill" style="width:%s%%"></div></div>`+"\n",
			indent, val))
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
			return NIL
		},
	})

	// ui.title("文本") 大标题
	env.Define("title", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 { return newError("ui.title 需要文本") }
			uiState.Root = append(uiState.Root, &UIElement{
				Tag: "title", Text: args[0].Inspect(),
			})
			return NIL
		},
	})

	// ui.text("文本") 普通文本
	env.Define("text", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 { return newError("ui.text 需要文本") }
			uiState.Root = append(uiState.Root, &UIElement{
				Tag: "text", Text: args[0].Inspect(),
			})
			return NIL
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
			uiState.Root = append(uiState.Root, el)
			return NIL
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
			uiState.Root = append(uiState.Root, el)
			return NIL
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
			uiState.Root = append(uiState.Root, el)
			return NIL
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
			uiState.Root = append(uiState.Root, el)
			return NIL
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
			uiState.Root = append(uiState.Root, el)
			return NIL
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
			uiState.Root = append(uiState.Root, el)
			return NIL
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
			uiState.Root = append(uiState.Root, el)
			return NIL
		},
	})

	// ui.image("url", "alt") 图片
	env.Define("image", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			el := &UIElement{Tag: "image", Attrs: map[string]string{}}
			if len(args) > 0 { el.Attrs["src"] = args[0].Inspect() }
			if len(args) > 1 { el.Text = args[1].Inspect() }
			uiState.Root = append(uiState.Root, el)
			return NIL
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

// UIElement 实现 Object 接口（用于 ui.row/ui.card 传入子元素）
func (el *UIElement) Type() object.ObjectType { return "UI_ELEMENT" }
func (el *UIElement) Inspect() string {
	return "<ui " + el.Tag + ">"
}
