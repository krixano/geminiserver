package titan


import (
	"strings"
)

type (
	router struct {
		tree   *node
		routes map[string]*Route
		titan    *Titan
	}
	node struct {
		kind     kind
		label    byte
		prefix   string
		parent   *node
		children children
		ppath    string
		pnames   []string
		handler  HandlerFunc
	}
	kind     uint8
	children []*node
)

const (
	skind kind = iota
	pkind
	akind
)

func newRouter(t *Titan) *router {
	return &router{
		tree:   &node{},
		routes: map[string]*Route{},
		titan:    t,
	}
}

func (r *router) add(path string, h HandlerFunc) {
	// Validate path
	if path == "" {
		path = "/"
	}

	if path[0] != '/' {
		path = "/" + path
	}

	pnames := []string{} // Param names
	ppath := path        // Pristine path

	for i, l := 0, len(path); i < l; i++ {
		if path[i] == ':' {
			j := i + 1

			r.insert(path[:i], nil, skind, "", nil)

			for ; i < l && path[i] != '/'; i++ {
			}

			pnames = append(pnames, path[j:i])
			path = path[:j] + path[i:]
			i, l = j, len(path)

			if i == l {
				r.insert(path[:i], h, pkind, ppath, pnames)
			} else {
				r.insert(path[:i], nil, pkind, "", nil)
			}
		} else if path[i] == '*' {
			r.insert(path[:i], nil, skind, "", nil)
			pnames = append(pnames, "*")
			r.insert(path[:i+1], h, akind, ppath, pnames)
		}
	}

	r.insert(path, h, skind, ppath, pnames)
}

func (r *router) insert(path string, h HandlerFunc, t kind, ppath string, pnames []string) {
	// Adjust max param
	l := len(pnames)
	if *r.titan.maxParam < l {
		*r.titan.maxParam = l
	}

	cn := r.tree // Current node as root
	if cn == nil {
		panic("titan: invalid tree")
	}

	search := path

	for {
		sl := len(search)
		pl := len(cn.prefix)
		l := 0

		// LCP
		max := pl
		if sl < max {
			max = sl
		}

		for ; l < max && search[l] == cn.prefix[l]; l++ {
		}

		switch {
		case l == 0:
			// At root node
			cn.label = search[0]
			cn.prefix = search

			if h != nil {
				cn.kind = t
				cn.handler = h
				cn.ppath = ppath
				cn.pnames = pnames
			}
		case l < pl:
			// Split node
			n := newNode(cn.kind, cn.prefix[l:], cn, cn.children, cn.handler, cn.ppath, cn.pnames)

			// Update parent path for all children to new node
			for _, child := range cn.children {
				child.parent = n
			}

			// Reset parent node
			cn.kind = skind
			cn.label = cn.prefix[0]
			cn.prefix = cn.prefix[:l]
			cn.children = nil
			cn.ppath = ""
			cn.pnames = nil

			cn.addChild(n)

			if l == sl {
				// At parent node
				cn.kind = t
				cn.handler = h
				cn.ppath = ppath
				cn.pnames = pnames
			} else {
				// Create child node
				n = newNode(t, search[l:], cn, nil, nil, ppath, pnames)
				n.handler = h
				cn.addChild(n)
			}
		case l < sl:
			search = search[l:]
			c := cn.findChildWithLabel(search[0])

			if c != nil {
				// Go deeper
				cn = c
				continue
			}
			// Create child node
			n := newNode(t, search, cn, nil, nil, ppath, pnames)
			n.handler = h
			cn.addChild(n)
		case h != nil:
			// Node already exists
			cn.handler = h
			cn.ppath = ppath

			if len(cn.pnames) == 0 {
				cn.pnames = pnames
			}
		}

		return
	}
}

func newNode(t kind, pre string, p *node, c children, h HandlerFunc, ppath string, pnames []string) *node {
	return &node{
		kind:     t,
		label:    pre[0],
		prefix:   pre,
		parent:   p,
		children: c,
		ppath:    ppath,
		pnames:   pnames,
		handler:  h,
	}
}

func (n *node) addChild(c *node) {
	n.children = append(n.children, c)
}

func (n *node) findChild(l byte, t kind) *node {
	for _, c := range n.children {
		if c.label == l && c.kind == t {
			return c
		}
	}

	return nil
}

func (n *node) findChildWithLabel(l byte) *node {
	for _, c := range n.children {
		if c.label == l {
			return c
		}
	}

	return nil
}

func (n *node) findChildByKind(t kind) *node {
	for _, c := range n.children {
		if c.kind == t {
			return c
		}
	}

	return nil
}

// find lookup a handler registered for path. It also parses URL for path
// parameters and load them into context.
func (r *router) find(path string, c Context) {
	ctx := c.(*context)
	ctx.path = path
	cn := r.tree // Current node as root

	var (
		search  = path
		child   *node         // Child node
		n       int           // Param counter
		nk      kind          // Next kind
		nn      *node         // Next node
		ns      string        // Next search
		pvalues = ctx.pvalues // Use the internal slice so the interface can keep the illusion of a dynamic slice
	)

	// Search order static > param > any
	for {
		if search == "" {
			break
		}

		pl := 0 // Prefix length
		l := 0  // LCP length

		if cn.label != ':' {
			sl := len(search)
			pl = len(cn.prefix)

			// LCP
			max := pl
			if sl < max {
				max = sl
			}

			for ; l < max && search[l] == cn.prefix[l]; l++ {
			}
		}

		if l == pl {
			// Continue search
			search = search[l:]
			// Finish routing if no remaining search and we are on an leaf node
			if search == "" && (nn == nil || cn.parent == nil || cn.ppath != "") {
				break
			}
		}

		// Attempt to go back up the tree on no matching prefix or no remaining search
		if l != pl || search == "" {
			// Handle special case of trailing slash route with existing any route (see #1526)
			if path[len(path)-1] == '/' && cn.findChildByKind(akind) != nil {
				goto Any
			}

			if nn == nil {
				return // Not found
			}

			cn = nn
			search = ns

			if nk == pkind {
				goto Param
			} else if nk == akind {
				goto Any
			}
		}

		// Static node
		if child = cn.findChild(search[0], skind); child != nil {
			// Save next
			if cn.prefix[len(cn.prefix)-1] == '/' {
				nk = pkind
				nn = cn
				ns = search
			}

			cn = child

			continue
		}

	Param:
		// Param node
		if child = cn.findChildByKind(pkind); child != nil {
			if len(pvalues) == n {
				continue
			}

			// Save next
			if cn.prefix[len(cn.prefix)-1] == '/' {
				nk = akind
				nn = cn
				ns = search
			}

			cn = child
			i, l := 0, len(search)
			for ; i < l && search[i] != '/'; i++ {
			}
			pvalues[n] = search[:i]
			n++
			search = search[i:]
			continue
		}

	Any:
		// Any node
		if cn = cn.findChildByKind(akind); cn != nil {
			// If any node is found, use remaining path for pvalues
			pvalues[len(cn.pnames)-1] = search
			break
		}

		// No node found, continue at stored next node
		// or find nearest "any" route
		if nn != nil {
			// No next node to go down in routing (issue #954)
			// Find nearest "any" route going up the routing tree
			search = ns
			// Consider param route one level up only
			if cn = nn.findChildByKind(pkind); cn != nil {
				pos := strings.IndexByte(ns, '/')
				if pos == -1 {
					// If no slash is remaining in search string set param value
					pvalues[len(cn.pnames)-1] = search
					break
				} else if pos > 0 {
					// Otherwise continue route processing with restored next node
					cn = nn
					nn = nil
					ns = ""
					goto Param
				}
			}
			// No param route found, try to resolve nearest any route
			for {
				np := nn.parent

				if cn = nn.findChildByKind(akind); cn != nil {
					break
				}

				if np == nil {
					break // no further parent nodes in tree, abort
				}

				var str strings.Builder

				str.WriteString(nn.prefix)
				str.WriteString(search)
				search = str.String()
				nn = np
			}

			if cn != nil { // use the found "any" route and update path
				pvalues[len(cn.pnames)-1] = search
				break
			}
		}

		return // Not found
	}

	ctx.handler = cn.handler
	ctx.path = cn.ppath
	ctx.pnames = cn.pnames

	// NOTE: Slow zone...
	if ctx.handler == nil {
		ctx.handler = NotFoundHandler

		// Dig further for any
		if cn = cn.findChildByKind(akind); cn == nil {
			return
		}

		if cn.handler != nil {
			ctx.handler = cn.handler
		} else {
			ctx.handler = NotFoundHandler
		}

		ctx.path = cn.ppath
		ctx.pnames = cn.pnames
		pvalues[len(cn.pnames)-1] = ""
	}
}
