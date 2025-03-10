package gotemplate

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"gitee.com/llyb120/goscript"
)

func (t *SqlRender) lib() map[string]any {
	return map[string]any{
		"val": func(arg any) interface{} {
			ctx := t.sqlContext.GetContext()
			ctx.params = append(ctx.params, arg)
			return "?"
		},
		"each": func(arg any) string {
			// 必须是一个切片，通过反射进行判断
			if reflect.TypeOf(arg).Kind() != reflect.Slice {
				return ""
			}
			// 循环
			iter := reflect.ValueOf(arg)
			str := ""
			ctx := t.sqlContext.GetContext()
			for i := 0; i < iter.Len(); i++ {
				value := iter.Index(i)
				ctx.params = append(ctx.params, value.Interface())
				str += "?,"
			}
			str = strings.TrimSuffix(str, ",")
			if str == "" {
				fmt.Println("warn: each 没有找到数据")
				ctx.params = append(ctx.params, "__UNDEFINED__")
				return "?"
			}
			return str
		},
		"exist": func(arg any) bool {
			if arg == nil || arg == goscript.Undefined {
				return false
			}
			if arg == false || arg == 0 || arg == "" {
				return false
			}
			if reflect.TypeOf(arg).Kind() == reflect.Map || reflect.TypeOf(arg).Kind() == reflect.Slice {
				return reflect.ValueOf(arg).Len() > 0
			}
			return true
		},
		"use": func(alias, main, sub string) string {
			ctx := t.sqlContext.GetContext()
			ctx.currentUseScope = alias
			defer func() {
				ctx.currentUseScope = "default"
			}()
			if main == "" {
				main = ctx.fromTitle
			}
			sql := t.getSql(main, sub)
			if sql == "" {
				ctx.err = fmt.Errorf("没有找到模板 %s %s", main, sub)
				return ""
			}
			res, err := t.engine.doRender(ctx.inter, sql)
			if err != nil {
				ctx.err = err
				return ""
			}
			return res
		},
		"hook": func(name string, content string) string {
			ctx := t.sqlContext.GetContext()
			re := regexp.MustCompile(`@\{|@\}`)
			ctx.hooks[name] = re.ReplaceAllStringFunc(content, func(s string) string {
				if s == `@{` {
					return `{{`
				} else if s == `@}` {
					return `}}`
				}
				return s
			})
			return ""
		},
		"slot": func(name string, self string) string {
			ctx := t.sqlContext.GetContext()
			var code string
			var ok bool
			if code, ok = ctx.hooks[ctx.currentUseScope+"."+name]; !ok {
				// 对自身进行转义
				decodeCode(&self)
				code = self
			}
			res, err := t.engine.doRender(ctx.inter, code)
			if err != nil {
				ctx.err = err
				return ""
			}
			return res
		},
		"trim": func(target string, safe string, code string) string {
			decodeCode(&code)
			ctx := t.sqlContext.GetContext()
			res, err := t.engine.doRender(ctx.inter, code)
			if err != nil {
				ctx.err = err
				return ""
			}
			// trim target
			res = strings.TrimSpace(res)
			res = strings.TrimPrefix(res, target)
			res = strings.TrimSuffix(res, target)

			if strings.TrimSpace(res) == "" {
				return " " + safe + " "
			}
			// if target is empty use safe statement instead
			return " " + res + " "
		},
	}
}
