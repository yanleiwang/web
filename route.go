package web

import (
	"fmt"
	"regexp"
	"strings"
)

type router struct {
	// trees 是按照 HTTP 方法来组织的
	// 如 GET => *node
	trees map[string]*node
}

func newRouter() router {
	return router{
		trees: map[string]*node{},
	}
}

// addRoute 注册路由。
// method 是 HTTP 方法
// - 已经注册了的路由，无法被覆盖。例如 /user/home 注册两次，会冲突
// - path 必须以 / 开始并且结尾不能有 /，中间也不允许有连续的 /
// - 不能在同一个位置注册不同的参数路由，例如 /user/:id 和 /user/:name 冲突
// - 不能在同一个位置同时注册通配符路由和参数路由，例如 /user/:id 和 /user/* 冲突
// - 同名路径参数，在路由匹配的时候，值会被覆盖。例如 /user/:id/abc/:id，那么 /user/123/abc/456 最终 id = 456
func (r *router) addRoute(method string, path string, handler HandleFunc) {
	if path == "" {
		panic("web: 路由是空字符串")
	}

	if path[0] != '/' {
		panic("web: 路由必须以 / 开头")
	}

	if path != "/" && path[len(path)-1] == '/' {
		panic("web: 路由不能以 / 结尾")
	}

	root, ok := r.trees[method]
	if !ok {
		root = &node{path: "/", typ: nodeTypeStatic}
		r.trees[method] = root
	}

	if path == "/" {
		if root.handler != nil {
			panic("web: 路由冲突[/]")
		}
		root.handler = handler
		return
	}

	seqs := strings.Split(path[1:], "/")
	for _, s := range seqs {
		if s == "" {
			panic(fmt.Sprintf("web: 非法路由。不允许使用 //a/b, /a//b 之类的路由, [%s]", path))
		}

		root = root.childOrCreate(s)
	}

	if root.handler != nil {
		panic(fmt.Sprintf("web: 路由冲突[%s]", path))
	}
	root.handler = handler

}

// findRoute 查找对应的节点
// 注意，返回的 node 内部 HandleFunc 不为 nil 才算是注册了路由
func (r *router) findRoute(method string, path string) (*matchInfo, bool) {
	panic("implement me")
}

type nodeType int

const (
	// 静态路由
	nodeTypeStatic = iota
	// 正则路由
	nodeTypeReg
	// 路径参数路由
	nodeTypeParam
	// 通配符路由
	nodeTypeAny
)

// node 代表路由树的节点
// 路由树的匹配顺序是：
// 1. 静态完全匹配
// 2. 正则匹配，形式 :param_name(reg_expr)
// 3. 路径参数匹配：形式 :param_name
// 4. 通配符匹配：*
// 这是不回溯匹配
type node struct {
	typ nodeType

	path string
	// children 子节点
	// 子节点的 path => node
	children map[string]*node
	// handler 命中路由之后执行的逻辑
	handler HandleFunc

	// 通配符 * 表达的节点，任意匹配
	starChild *node

	paramChild *node
	// 正则路由和参数路由都会使用这个字段
	paramName string

	// 正则表达式
	regChild *node
	regExpr  *regexp.Regexp
}

// child 返回子节点
// 第一个返回值 *node 是命中的节点
// 第二个返回值 bool 代表是否命中
func (n *node) childOf(path string) (*node, bool) {
	panic("implement me")
}

// childOrCreate 查找子节点，如果不存在则创建一个
// 首先会判断 path 是不是通配符路径
// 其次判断 path 是不是参数路径，即以 : 开头的路径
// 最后会从 children 里面查找，
// 如果没有找到，那么会创建一个新的节点，并且保存在 node 里面
func (n *node) childOrCreate(path string) *node {
	//通配符路径
	if path == "*" {
		if n.starChild == nil {
			if n.regChild != nil {
				panic("web: 非法路由，已有正则路由。不允许同时注册通配符路由和正则路由 [*]")
			}

			if n.paramChild != nil {
				panic("web: 非法路由，已有路径参数路由。不允许同时注册通配符路由和参数路由 [*]")
			}

			n.starChild = &node{path: "*", typ: nodeTypeAny}
		}
		return n.starChild
	}

	if path[0] == ':' {
		// 正则路径
		if ok, _ := regexp.MatchString("^:.+\\(.+\\)$", path); ok {
			if n.paramChild != nil {
				panic(fmt.Sprintf("web: 非法路由，已有路径参数路由。不允许同时注册正则路由和参数路由 [%s]", path))
			}

			if n.starChild != nil {
				panic(fmt.Sprintf("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和正则路由 [%s]", path))
			}

			if n.regChild != nil {
				if n.regChild.path != path {
					panic(fmt.Sprintf("web: 路由冲突, 正则路由冲突, 已有:[%s], 新注册:[%s]", n.regChild.path, path))
				}
			} else {
				idx := strings.Index(path, "(")
				paraName := path[1:idx]
				expr := path[idx+1 : len(path)-1]
				regExpr, _ := regexp.Compile(expr)
				n.regChild = &node{path: path, typ: nodeTypeReg, regExpr: regExpr, paramName: paraName}
			}
			return n.regChild
		} else {
			// 参数路径
			if n.regChild != nil {
				panic(fmt.Sprintf("web: 非法路由，已有正则路由。不允许同时注册正则路由和参数路由 [%s]", path))
			}

			if n.starChild != nil {
				panic(fmt.Sprintf("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和参数路由 [%s]", path))
			}

			if n.paramChild != nil {
				if n.paramChild.path != path {
					panic(fmt.Sprintf("web: 路由冲突，参数路由冲突，已有 %s，新注册 %s", n.paramChild.path, path))
				}
			} else {
				n.paramChild = &node{path: path, typ: nodeTypeParam, paramName: path[1:]}
			}
			return n.paramChild
		}
	}

	// 静态匹配
	if n.children == nil {
		n.children = map[string]*node{}
	}
	target, ok := n.children[path]
	if !ok {
		target = &node{path: path, typ: nodeTypeStatic}
		n.children[path] = target
	}
	return target
}

type matchInfo struct {
	n          *node
	pathParams map[string]string
}

func (m *matchInfo) addValue(key string, value string) {
	if m.pathParams == nil {
		// 大多数情况，参数路径只会有一段
		m.pathParams = map[string]string{key: value}
	}
	m.pathParams[key] = value
}
