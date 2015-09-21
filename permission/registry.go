// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package permission

import (
	"strings"
)

type registry struct {
	permissionScheme
	children []*registry
}

func (r *registry) add(names ...string) *registry {
	for _, name := range names {
		r.addWithCtx(name, nil)
	}
	return r
}

func (r *registry) addWithCtx(name string, contextTypes []contextType) *registry {
	parts := strings.Split(name, ".")
	parent := r
	for i, part := range parts {
		subR := parent.getSubRegistry(part)
		if subR == nil {
			subR = &registry{permissionScheme: permissionScheme{name: part}}
			parent.children = append(parent.children, subR)
		}
		if i == len(parts)-1 {
			subR.permissionScheme.contexts = contextTypes
		}
		parent = subR
	}
	return r
}

func (r *registry) getSubRegistry(name string) *registry {
	parts := strings.Split(name, ".")
	children := r.children
	var parent *registry
	for len(children) > 0 && len(parts) > 0 {
		var currentElement *registry
		for _, child := range children {
			if child.name == parts[0] {
				if parent != nil {
					child.permissionScheme.parent = &parent.permissionScheme
				}
				currentElement = child
				parts = parts[1:]
				children = child.children
				break
			}
		}
		parent = currentElement
		if parent == nil {
			break
		}
	}
	return parent
}

func (r *registry) Permissions() PermissionSchemeList {
	var ret []*permissionScheme
	stack := []*registry{r}
	for len(stack) > 0 {
		last := len(stack) - 1
		el := stack[last]
		stack = stack[:last]
		ret = append(ret, &el.permissionScheme)
		for i := len(el.children) - 1; i >= 0; i-- {
			child := el.children[i]
			child.parent = &el.permissionScheme
			stack = append(stack, child)
		}
	}
	return ret
}

func (r *registry) get(name string) *permissionScheme {
	if name == "" {
		return &r.permissionScheme
	}
	subR := r.getSubRegistry(name)
	if subR == nil {
		panic("unregistered permission: " + name)
	}
	return &subR.permissionScheme
}