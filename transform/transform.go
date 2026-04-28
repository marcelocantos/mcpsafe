// Copyright 2026 Marcelo Cantos
// SPDX-License-Identifier: Apache-2.0

// Package transform loads Starlark scripts that define how requests are
// rewritten before being forwarded to the real backend.
package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.starlark.net/starlark"

	"github.com/marcelocantos/mcpsafe/keychain"
)

// Script is a loaded Starlark transform script.
type Script struct {
	thread  *starlark.Thread
	globals starlark.StringDict
	store   *keychain.Store
}

// Load reads and compiles a Starlark script file. The script must define
// a transform(req) function.
func Load(path string, store *keychain.Store) (*Script, error) {
	s := &Script{
		thread: &starlark.Thread{Name: "mcpsafe"},
		store:  store,
	}

	builtins := starlark.StringDict{
		"keychain": starlark.NewBuiltin("keychain", s.builtinKeychain),
	}

	globals, err := starlark.ExecFile(s.thread, path, nil, builtins)
	if err != nil {
		return nil, fmt.Errorf("loading starlark script %s: %w", path, err)
	}

	if _, ok := globals["transform"]; !ok {
		return nil, fmt.Errorf("starlark script %s does not define a transform function", path)
	}

	s.globals = globals
	return s, nil
}

// Apply runs the Starlark transform function against the given request,
// modifying it in place.
func (s *Script) Apply(req *http.Request) error {
	query := req.URL.Query()

	qDict := starlark.NewDict(len(query))
	for k, vs := range query {
		if len(vs) > 0 {
			if err := qDict.SetKey(starlark.String(k), starlark.String(vs[0])); err != nil {
				return err
			}
		}
	}

	hdrs := starlark.NewDict(16)
	for k, vs := range req.Header {
		if len(vs) > 0 {
			if err := hdrs.SetKey(starlark.String(k), starlark.String(vs[0])); err != nil {
				return err
			}
		}
	}

	// Parse JSON body if present.
	var bodyDict *starlark.Dict
	if req.Body != nil && isJSON(req.Header.Get("Content-Type")) {
		bodyBytes, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return fmt.Errorf("reading request body: %w", err)
		}
		if len(bodyBytes) > 0 {
			var jsonBody map[string]any
			if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
				bodyDict = starlark.NewDict(len(jsonBody))
				for k, v := range jsonBody {
					sv, err := goToStarlark(v)
					if err != nil {
						return fmt.Errorf("converting body field %q: %w", k, err)
					}
					bodyDict.SetKey(starlark.String(k), sv)
				}
			} else {
				// Not valid JSON — put raw body back.
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}
	}

	reqDict := starlark.NewDict(7)
	reqDict.SetKey(starlark.String("host"), starlark.String(req.Host))
	reqDict.SetKey(starlark.String("scheme"), starlark.String(req.URL.Scheme))
	reqDict.SetKey(starlark.String("path"), starlark.String(req.URL.Path))
	reqDict.SetKey(starlark.String("query"), qDict)
	reqDict.SetKey(starlark.String("headers"), hdrs)
	reqDict.SetKey(starlark.String("method"), starlark.String(req.Method))
	if bodyDict != nil {
		reqDict.SetKey(starlark.String("body"), bodyDict)
	}

	result, err := starlark.Call(s.thread, s.globals["transform"], starlark.Tuple{reqDict}, nil)
	if err != nil {
		return fmt.Errorf("starlark transform: %w", err)
	}

	out, ok := result.(*starlark.Dict)
	if !ok {
		return fmt.Errorf("transform must return a dict, got %s", result.Type())
	}

	if v, found, _ := out.Get(starlark.String("host")); found {
		if host, ok := v.(starlark.String); ok {
			req.Host = string(host)
			req.URL.Host = string(host)
		}
	}
	if v, found, _ := out.Get(starlark.String("scheme")); found {
		if scheme, ok := v.(starlark.String); ok {
			req.URL.Scheme = string(scheme)
		}
	}
	if v, found, _ := out.Get(starlark.String("path")); found {
		if path, ok := v.(starlark.String); ok {
			req.URL.Path = string(path)
		}
	}

	if v, found, _ := out.Get(starlark.String("query")); found {
		if qd, ok := v.(*starlark.Dict); ok {
			newQuery := url.Values{}
			for _, item := range qd.Items() {
				k, _ := starlark.AsString(item[0])
				v, _ := starlark.AsString(item[1])
				newQuery.Set(k, v)
			}
			req.URL.RawQuery = newQuery.Encode()
		}
	}

	if v, found, _ := out.Get(starlark.String("headers")); found {
		if hd, ok := v.(*starlark.Dict); ok {
			for _, item := range hd.Items() {
				k, _ := starlark.AsString(item[0])
				v, _ := starlark.AsString(item[1])
				req.Header.Set(k, v)
			}
		}
	}

	// Serialize body dict back to JSON.
	if v, found, _ := out.Get(starlark.String("body")); found {
		if bd, ok := v.(*starlark.Dict); ok {
			goMap := make(map[string]any, bd.Len())
			for _, item := range bd.Items() {
				k, _ := starlark.AsString(item[0])
				goMap[k] = starlarkToGo(item[1])
			}
			encoded, err := json.Marshal(goMap)
			if err != nil {
				return fmt.Errorf("encoding body: %w", err)
			}
			req.Body = io.NopCloser(bytes.NewReader(encoded))
			req.ContentLength = int64(len(encoded))
		}
	}

	return nil
}

func (s *Script) builtinKeychain(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var service, account string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &service, &account); err != nil {
		return nil, err
	}
	if account == "" {
		account = service
	}
	pw, err := s.store.Get(service, account)
	if err != nil {
		return nil, err
	}
	return starlark.String(pw), nil
}

func isJSON(contentType string) bool {
	return strings.HasPrefix(contentType, "application/json")
}

func goToStarlark(v any) (starlark.Value, error) {
	switch v := v.(type) {
	case nil:
		return starlark.None, nil
	case bool:
		return starlark.Bool(v), nil
	case float64:
		if v == float64(int64(v)) {
			return starlark.MakeInt64(int64(v)), nil
		}
		return starlark.Float(v), nil
	case string:
		return starlark.String(v), nil
	case []any:
		elems := make([]starlark.Value, len(v))
		for i, e := range v {
			var err error
			elems[i], err = goToStarlark(e)
			if err != nil {
				return nil, err
			}
		}
		return starlark.NewList(elems), nil
	case map[string]any:
		d := starlark.NewDict(len(v))
		for k, val := range v {
			sv, err := goToStarlark(val)
			if err != nil {
				return nil, err
			}
			d.SetKey(starlark.String(k), sv)
		}
		return d, nil
	default:
		return starlark.String(fmt.Sprint(v)), nil
	}
}

func starlarkToGo(v starlark.Value) any {
	switch v := v.(type) {
	case starlark.NoneType:
		return nil
	case starlark.Bool:
		return bool(v)
	case starlark.Int:
		if i, ok := v.Int64(); ok {
			return i
		}
		return v.String()
	case starlark.Float:
		return float64(v)
	case starlark.String:
		return string(v)
	case *starlark.List:
		out := make([]any, v.Len())
		for i := range v.Len() {
			out[i] = starlarkToGo(v.Index(i))
		}
		return out
	case *starlark.Dict:
		out := make(map[string]any, v.Len())
		for _, item := range v.Items() {
			k, _ := starlark.AsString(item[0])
			out[k] = starlarkToGo(item[1])
		}
		return out
	default:
		return v.String()
	}
}
