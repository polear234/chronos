//go:build !js

package interpreter

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"chronospp/object"
)

// newRequestsModule 构造 Python 风格的 requests 模块
//   import "requests"
//   let r = requests.get("https://example.com")
//   print(r["status"])
//   print(r["text"])
func newRequestsModule() *object.Module {
	env := object.NewEnvironment()

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
	}

	// 构造响应哈希
	makeResponse := func(resp *http.Response, body []byte) object.Object {
		pairs := map[object.HashKey]object.HashPair{}
		put := func(k string, v object.Object) {
			key := &object.String{Value: k}
			pairs[key.HashKey()] = object.HashPair{Key: key, Value: v}
		}
		put("status", &object.Integer{Value: int64(resp.StatusCode)})
		put("ok", nativeBool(resp.StatusCode >= 200 && resp.StatusCode < 300))
		put("text", &object.String{Value: string(body)})
		put("url", &object.String{Value: resp.Request.URL.String()})

		// 响应头
		hdrPairs := map[object.HashKey]object.HashPair{}
		for k, vals := range resp.Header {
			hk := &object.String{Value: strings.ToLower(k)}
			hdrPairs[hk.HashKey()] = object.HashPair{
				Key:   hk,
				Value: &object.String{Value: strings.Join(vals, ", ")},
			}
		}
		put("headers", &object.Hash{Pairs: hdrPairs})
		return &object.Hash{Pairs: pairs}
	}

	// 把参数里的 headers / params 转成 Go 结构
	extractHeaders := func(o object.Object) map[string]string {
		out := map[string]string{}
		if h, ok := o.(*object.Hash); ok {
			for _, pair := range h.Pairs {
				k := pair.Key.(*object.String).Value
				out[k] = pair.Value.Inspect()
			}
		}
		return out
	}

	doRequest := func(method, rawURL string, headers map[string]string, body io.Reader) object.Object {
		req, err := http.NewRequest(method, rawURL, body)
		if err != nil {
			return newError("requests: 无效的 URL: %v", err)
		}
		// 默认 User-Agent
		if _, ok := headers["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "Chronospp/1.0")
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			return newError("requests: 请求失败: %v", err)
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20)) // 最多 32MB
		if err != nil {
			return newError("requests: 读取响应失败: %v", err)
		}
		return makeResponse(resp, data)
	}

	// ── get(url, headers?) ──────────────
	env.Define("get", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("requests.get 需要 URL")
			}
			u := args[0].Inspect()
			headers := map[string]string{}
			if len(args) > 1 {
				headers = extractHeaders(args[1])
			}
			return doRequest("GET", u, headers, nil)
		},
	})

	// ── post(url, data?, headers?) ──────
	env.Define("post", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("requests.post 需要 URL")
			}
			u := args[0].Inspect()
			headers := map[string]string{}
			var body io.Reader

			if len(args) > 1 && args[1] != NIL {
				// 哈希 → JSON，字符串 → 原样
				switch v := args[1].(type) {
				case *object.Hash:
					// 转成 JSON
					m := map[string]any{}
					for _, pair := range v.Pairs {
						k := pair.Key.(*object.String).Value
						m[k] = objectToGo(pair.Value)
					}
					b, err := json.Marshal(m)
					if err != nil {
						return newError("requests.post: 序列化失败: %v", err)
					}
					body = bytes.NewReader(b)
					headers["Content-Type"] = "application/json"
				case *object.String:
					body = strings.NewReader(v.Value)
				default:
					body = strings.NewReader(args[1].Inspect())
				}
			}

			if len(args) > 2 {
				for k, val := range extractHeaders(args[2]) {
					headers[k] = val
				}
			}
			return doRequest("POST", u, headers, body)
		},
	})

	// ── put / delete / head ─────────────
	for _, method := range []string{"put", "delete", "head"} {
		m := strings.ToUpper(method)
		name := method
		env.Define(name, &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 {
					return newError("requests.%s 需要 URL", name)
				}
				u := args[0].Inspect()
				headers := map[string]string{}
				var body io.Reader
				if len(args) > 1 && args[1] != NIL && m != "HEAD" {
					body = strings.NewReader(args[1].Inspect())
				}
				if len(args) > 2 {
					for k, v := range extractHeaders(args[2]) {
						headers[k] = v
					}
				}
				return doRequest(m, u, headers, body)
			},
		})
	}

	// ── 便捷：getJson(url) 直接返回解析好的哈希/数组 ──
	env.Define("getJson", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("requests.getJson 需要 URL")
			}
			u := args[0].Inspect()
			headers := map[string]string{"Accept": "application/json"}
			resp := doRequest("GET", u, headers, nil)
			if isError(resp) {
				return resp
			}
			h, ok := resp.(*object.Hash)
			if !ok {
				return newError("requests.getJson: 内部错误")
			}
			// 取 text 字段解析
			var text string
			for _, pair := range h.Pairs {
				if k, ok := pair.Key.(*object.String); ok && k.Value == "text" {
					if s, ok := pair.Value.(*object.String); ok {
						text = s.Value
					}
				}
			}
			return jsonToObject(text)
		},
	})

	// ── 便捷：download(url, path) ───────
	env.Define("download", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("requests.download 需要 URL 和目标路径")
			}
			u := args[0].Inspect()
			path := args[1].Inspect()

			req, err := http.NewRequest("GET", u, nil)
			if err != nil {
				return newError("requests.download: %v", err)
			}
			req.Header.Set("User-Agent", "Chronospp/1.0")
			resp, err := client.Do(req)
			if err != nil {
				return newError("requests.download: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				return newError("requests.download: HTTP %d", resp.StatusCode)
			}
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return newError("requests.download: %v", err)
			}
			if err := writeFileBytes(path, data); err != nil {
				return newError("requests.download: %v", err)
			}
			return &object.Integer{Value: int64(len(data))}
		},
	})

	// ── 便捷：urlEncode / quote ─────────
	env.Define("quote", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("requests.quote 需要字符串")
			}
			return &object.String{Value: url.QueryEscape(args[0].Inspect())}
		},
	})

	return &object.Module{
		Name:    "requests",
		Env:     env,
		Exports: env.Snapshot(),
	}
}

// objectToGo 把 Chronos 对象转成 Go 的 any（用于 JSON 序列化）
func objectToGo(o object.Object) any {
	switch v := o.(type) {
	case *object.Integer:
		return v.Value
	case *object.Float:
		return v.Value
	case *object.String:
		return v.Value
	case *object.Boolean:
		return v.Value
	case *object.Array:
		out := make([]any, len(v.Elements))
		for i, e := range v.Elements {
			out[i] = objectToGo(e)
		}
		return out
	case *object.Hash:
		m := map[string]any{}
		for _, pair := range v.Pairs {
			k := pair.Key.Inspect()
			m[k] = objectToGo(pair.Value)
		}
		return m
	}
	return nil
}

// jsonToObject 把 JSON 文本转成 Chronos 对象
func jsonToObject(text string) object.Object {
	var v any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		return newError("JSON 解析失败: %v", err)
	}
	return goToObject(v)
}

func goToObject(v any) object.Object {
	switch x := v.(type) {
	case nil:
		return NIL
	case bool:
		return nativeBool(x)
	case float64:
		// 整数就返回整数
		if x == float64(int64(x)) {
			return &object.Integer{Value: int64(x)}
		}
		return &object.Float{Value: x}
	case string:
		return &object.String{Value: x}
	case []any:
		elems := make([]object.Object, len(x))
		for i, e := range x {
			elems[i] = goToObject(e)
		}
		return &object.Array{Elements: elems}
	case map[string]any:
		pairs := map[object.HashKey]object.HashPair{}
		for k, val := range x {
			key := &object.String{Value: k}
			pairs[key.HashKey()] = object.HashPair{Key: key, Value: goToObject(val)}
		}
		return &object.Hash{Pairs: pairs}
	}
	return &object.String{Value: fmt.Sprint(v)}
}

// writeFileBytes 写二进制文件（供 download 用）
func writeFileBytes(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
