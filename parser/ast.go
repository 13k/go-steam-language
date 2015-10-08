package parser

import (
	"fmt"
)

type Symbol struct {
	Value string
	Scope Node
	Node  Node
}

type Node interface {
	Name() string
	NamePath() []string
	Parent() Node
	SetParent(Node)
	Path() []Node
	Ancestors() []Node
	Children() []Node
	AddChild(Node)
	AdoptChildren(Node)
	ClearChildren()
	Symbols() []*Symbol
	CreateSymbol(string, Node) *Symbol
	AddSymbol(*Symbol)
	FindSymbol(string, bool) *Symbol
	FindNestedSymbol([]string) *Symbol
	ImportSymbols(Node)
	ClearSymbols()
}

type symbolTable map[string]*Symbol

func (t symbolTable) Symbols() []*Symbol {
	var symbols []*Symbol

	for _, s := range t {
		symbols = append(symbols, s)
	}

	return symbols
}

func (t symbolTable) Clear() {
	for k, _ := range t {
		delete(t, k)
	}
}

type node struct {
	parent   Node
	owner    Node
	children []Node
	symbols  symbolTable
}

func NewNode(parent Node) Node {
	return newNode(parent, nil)
}

func newNode(parent Node, owner Node) *node {
	n := &node{
		parent:  parent,
		owner:   owner,
		symbols: make(symbolTable),
	}

	if parent != nil {
		parent.AddChild(n)
	}

	return n
}

func (n *node) Name() string {
	if n.owner != nil {
		return n.owner.Name()
	}

	return fmt.Sprintf("%p", n)
}

func (n *node) NamePath() []string {
	var namepath []string
	path := n.Path()

	for _, node := range path {
		namepath = append(namepath, node.Name())
	}

	return namepath
}

func (n *node) Ancestors() []Node {
	var path []Node

	if n.Parent() != nil {
		path = append(path, n.Parent().Ancestors()...)
		path = append(path, n.Parent())
	}

	return path
}

func (n *node) Path() []Node {
	path := n.Ancestors()

	if n.owner != nil {
		path = append(path, n.owner)
	} else {
		path = append(path, n)
	}

	return path
}

func (n *node) Parent() Node {
	return n.parent
}

func (n *node) SetParent(parent Node) {
	n.parent = parent
}

func (n *node) Children() []Node {
	return n.children
}

func (n *node) AddChild(child Node) {
	child.SetParent(n)
	n.children = append(n.children, child)
}

func (n *node) AdoptChildren(other Node) {
	for _, child := range other.Children() {
		n.AddChild(child)
	}

	other.ClearChildren()
}

func (n *node) ClearChildren() {
	n.children = nil
}

func (n *node) CreateSymbol(value string, node Node) *Symbol {
	sym := &Symbol{Value: value, Node: node}
	n.AddSymbol(sym)
	return sym
}

func (n *node) AddSymbol(s *Symbol) {
	if s == nil {
		panic(fmt.Errorf("Trying to add nil symbol to node %v", n.NamePath()))
	}

	if s.Value == "" {
		panic(fmt.Errorf("Trying to add empty symbol to node %v", n.NamePath()))
	}

	if _, ok := n.symbols[s.Value]; ok {
		panic(fmt.Errorf("Trying to add existing symbol %q to node %v", s.Value, n.NamePath()))
	}

	//fmt.Printf("Adding symbol %q to node %v\n", s.Value, n.NamePath())
	s.Scope = n
	n.symbols[s.Value] = s
}

func (n *node) FindSymbol(value string, create bool) *Symbol {
	if sym := n.symbols[value]; sym != nil {
		return sym
	}

	if n.parent != nil {
		if sym := n.parent.FindSymbol(value, create); sym != nil {
			return sym
		}
	}

	if create {
		return n.CreateSymbol(value, n.owner)
	}

	return nil
}

func (n *node) FindNestedSymbol(path []string) *Symbol {
	var sym *Symbol
	var node Node = n
	last := len(path) - 1

	for i, value := range path {
		sym = node.FindSymbol(value, i == last)

		if sym == nil {
			return nil
		}

		var next Node

		if sym.Node != nil {
			next = sym.Node
		} else {
			next = sym.Scope
		}

		node = next
	}

	return sym
}

func (n *node) ImportSymbols(other Node) {
	for _, sym := range other.Symbols() {
		func() {
			defer func() {
				recover()
			}()

			n.AddSymbol(sym)
		}()
	}
}

func (n *node) Symbols() []*Symbol {
	return n.symbols.Symbols()
}

func (n *node) ClearSymbols() {
	n.symbols.Clear()
}

type baseNode struct {
	Node
	Value []byte
	owner Node
}

func newBaseNode(parent Node, owner Node) *baseNode {
	return &baseNode{Node: newNode(parent, owner), owner: owner}
}

func (n *baseNode) Symbol() *Symbol {
	return &Symbol{Value: string(n.Value), Node: n.owner}
}

func (n *baseNode) Name() string {
	return string(n.Value)
}

type ClassNode struct {
	*baseNode
	Qualifier *Symbol
}

func NewClassNode(parent Node) *ClassNode {
	n := &ClassNode{}
	n.baseNode = newBaseNode(parent, n)
	return n
}

type EnumNode struct {
	*baseNode
	Flags     bool
	Qualifier *Symbol
}

func NewEnumNode(parent Node) *EnumNode {
	n := &EnumNode{}
	n.baseNode = newBaseNode(parent, n)
	return n
}

type PropertyNode struct {
	*baseNode
	Flags          string
	FlagsOpt       *Symbol
	Type           *Symbol
	Default        []*Symbol
	Obsolete       bool
	ObsoleteReason string
}

func NewPropertyNode(parent Node) *PropertyNode {
	n := &PropertyNode{}
	n.baseNode = newBaseNode(parent, n)
	return n
}

func (n *PropertyNode) AddDefault(s *Symbol) {
	if s == nil {
		panic(fmt.Errorf("Trying to add nil symbol to PropertyNode %v", n.NamePath()))
	}

	n.Default = append(n.Default, s)
}
