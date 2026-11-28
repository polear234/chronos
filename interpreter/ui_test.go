package interpreter

import "testing"

func TestUIEmptyAndReset(t *testing.T) {
	// 1. 没有元素时返回空串
	ResetUI()
	if html := GetUIHTML(); html != "" {
		t.Errorf("空 UI 应该返回空串，得到 %d 字节", len(html))
	}

	// 2. 加元素后有 HTML
	ResetUI()
	uiState.Root = append(uiState.Root, &UIElement{Tag: "title", Text: "测试"})
	html := GetUIHTML()
	if html == "" {
		t.Error("有元素时应该有 HTML")
	}
	if !contains(html, "测试") {
		t.Error("HTML 应该包含文本")
	}

	// 3. Reset 后恢复空
	ResetUI()
	if html := GetUIHTML(); html != "" {
		t.Error("Reset 后应该返回空串")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
