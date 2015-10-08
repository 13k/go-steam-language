package parser

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	openQualifierToken  = &Token{Op: OpOperator, Value: []byte("<")}
	closeQualifierToken = &Token{Op: OpOperator, Value: []byte(">")}
	openScopeToken      = &Token{Op: OpOperator, Value: []byte("{")}
	closeScopeToken     = &Token{Op: OpOperator, Value: []byte("}")}
	assignmentToken     = &Token{Op: OpOperator, Value: []byte("=")}
	binaryOrToken       = &Token{Op: OpOperator, Value: []byte("|")}
	obsoleteToken       = &Token{Op: OpIdentifier, Value: []byte("obsolete")}
	flagsToken          = &Token{Op: OpIdentifier, Value: []byte("flags")}
)

type Analyzer struct {
	t        *Tokenizer
	tokens   *TokenQueue
	filename string
}

func NewAnalyzer(t *Tokenizer, f string) *Analyzer {
	return &Analyzer{
		t:        t,
		filename: f,
	}
}

func (a *Analyzer) Errorf(row, col int, format string, v ...interface{}) error {
	var values []interface{}

	if row > 0 || col > 0 {
		format = "%d:%d: " + format
		values = append([]interface{}{row, col}, v...)
	}

	if a.filename != "" {
		format = "%s:" + format
		values = append([]interface{}{a.filename}, values...)
	}

	return fmt.Errorf(format, values...)
}

func (a *Analyzer) Analyze() (Node, error) {
	defer func() {
		fmt.Println(a.filename)
	}()

	if a.t == nil {
		return nil, fmt.Errorf("Uninitialized Analyzer")
	}

	root := NewNode(nil)
	tokens, err := a.t.Tokenize()

	if err != nil {
		return root, err
	}

	a.tokens = tokens
	t := a.tokens.Dequeue()

	for t != nil {
		if t.Error != nil {
			return root, t.Error
		}

		if err := a.handleToken(t, root); err != nil {
			return root, err
		}

		t = a.tokens.Dequeue()
	}

	return root, nil
}

func (a *Analyzer) handleToken(t *Token, root Node) error {
	switch t.Op {
	case OpPreprocess:
		return a.handlePreprocessToken(t, root)
	case OpIdentifier:
		return a.handleIdentifierToken(t, root)
	default:
		return a.Errorf(t.Row, t.Col, "Invalid token %q", t.Raw)
	}
}

func (a *Analyzer) handlePreprocessToken(t *Token, root Node) error {
	nextToken, err := a.expectOp(OpString)

	if err != nil {
		return err
	}

	if t.ValueString() == "import" {
		return a.importFile(string(nextToken.Value), root)
	}

	return nil
}

func (a *Analyzer) handleIdentifierToken(t *Token, root Node) error {
	switch t.ValueString() {
	case "class":
		return a.analyzeClass(root)
	case "enum":
		return a.analyzeEnum(root)
	default:
		return a.Errorf(t.Row, t.Col, "Invalid token %q", t.Raw)
	}
}

func (a *Analyzer) analyzeClass(root Node) error {
	node := NewClassNode(root)
	name, err := a.expectOp(OpIdentifier)

	if err != nil {
		return err
	}

	node.Value = name.Value
	root.AddSymbol(node.Symbol())
	qualifiers, err := a.getQualifierIdentifier()

	if err != nil {
		return err
	}

	node.Qualifier = root.FindNestedSymbol(tokenStringValues(qualifiers))

	if err := a.analyzeScope(node); err != nil {
		return err
	}

	if _, err := a.expectOp(OpTerminator); err != nil {
		return err
	}

	return nil
}

func (a *Analyzer) analyzeEnum(root Node) error {
	node := NewEnumNode(root)
	name, err := a.expectOp(OpIdentifier)

	if err != nil {
		return err
	}

	node.Value = name.Value
	root.AddSymbol(node.Symbol())
	qualifiers, err := a.getQualifierIdentifier()

	if err != nil {
		return err
	}

	node.Qualifier = root.FindNestedSymbol(tokenStringValues(qualifiers))

	if flag := a.optionalToken(flagsToken); flag != nil {
		node.Flags = true
	}

	if err := a.analyzeScope(node); err != nil {
		return err
	}

	if _, err := a.expectOp(OpTerminator); err != nil {
		return err
	}

	return nil
}

func (a *Analyzer) analyzeScope(root Node) error {
	if _, err := a.expectToken(openScopeToken); err != nil {
		return err
	}

	closeScope := a.optionalToken(closeScopeToken)

	for closeScope == nil {
		if err := a.analyzeProperty(root); err != nil {
			return err
		}

		closeScope = a.optionalToken(closeScopeToken)
	}

	return nil
}

func (a *Analyzer) analyzeProperty(root Node) error {
	node := NewPropertyNode(root)
	t1, err := a.expectOp(OpIdentifier)

	if err != nil {
		return err
	}

	qualifiers, err := a.getQualifierIdentifier()

	if err != nil {
		return err
	}

	t2 := a.optionalOp(OpIdentifier)
	t3 := a.optionalOp(OpIdentifier)

	var (
		nodeValue  []byte
		typeSymbol string
		flags      string
	)

	if t3 != nil {
		nodeValue = t3.Value
		typeSymbol = t2.ValueString()
		flags = t1.ValueString()
	} else if t2 != nil {
		nodeValue = t2.Value
		typeSymbol = t1.ValueString()
	} else {
		nodeValue = t1.Value
	}

	node.Value = nodeValue
	node.FlagsOpt = root.FindNestedSymbol(tokenStringValues(qualifiers))
	root.AddSymbol(node.Symbol())

	if typeSymbol != "" {
		node.Type = root.FindSymbol(typeSymbol, true)
	}

	node.Flags = flags

	if assignment := a.optionalToken(assignmentToken); assignment != nil {
		for {
			tokens, err := a.getNamespacedIdentifier()

			if err != nil {
				return err
			}

			sym := node.FindNestedSymbol(tokenStringValues(tokens))
			node.AddDefault(sym)

			if t := a.optionalToken(binaryOrToken); t != nil {
				continue
			}

			break
		}
	}

	if _, err := a.expectOp(OpTerminator); err != nil {
		return err
	}

	if obsolete := a.optionalToken(obsoleteToken); obsolete != nil {
		node.Obsolete = true

		if obsoleteReason := a.optionalOp(OpString); obsoleteReason != nil {
			node.ObsoleteReason = obsoleteReason.ValueString()
		} else {
			a.optionalOp(OpTerminator)
		}
	}

	return nil
}

func (a *Analyzer) expectOp(op OpCode) (*Token, error) {
	t := a.tokens.Peek()

	if t == nil {
		return nil, a.Errorf(-1, -1, "EOF")
	}

	if t.Op != op {
		return nil, a.Errorf(t.Row, t.Col, "Unexpected token %q", t.Raw)
	}

	return a.tokens.Dequeue(), nil
}

func (a *Analyzer) expectToken(t1 *Token) (*Token, error) {
	t2 := a.tokens.Peek()

	if t2 == nil {
		return nil, a.Errorf(-1, -1, "EOF")
	}

	if !t1.Equal(t2) {
		return nil, a.Errorf(t2.Row, t2.Col, "Unexpected token %q", t2.Raw)
	}

	return a.tokens.Dequeue(), nil
}

func (a *Analyzer) optionalOp(op OpCode) *Token {
	t := a.tokens.Peek()

	if t == nil {
		return nil
	}

	if t.Op != op {
		return nil
	}

	return a.tokens.Dequeue()
}

func (a *Analyzer) optionalToken(t1 *Token) *Token {
	t2 := a.tokens.Peek()

	if t2 == nil {
		return nil
	}

	if !t1.Equal(t2) {
		return nil
	}

	return a.tokens.Dequeue()
}

func (a *Analyzer) getNamespacedIdentifier() ([]*Token, error) {
	var result []*Token

	id, err := a.expectOp(OpIdentifier)

	if err != nil {
		return nil, err
	}

	result = append(result, id)

	ns := a.optionalOp(OpNamespace)

	if ns != nil {
		id, err = a.expectOp(OpIdentifier)

		if err != nil {
			return nil, err
		}

		result = append(result, id)
	}

	return result, nil
}

func (a *Analyzer) getQualifierIdentifier() ([]*Token, error) {
	openQualifier := a.optionalToken(openQualifierToken)

	if openQualifier == nil {
		return nil, nil
	}

	qualifiers, err := a.getNamespacedIdentifier()

	if err != nil {
		return nil, err
	}

	if _, err := a.expectToken(closeQualifierToken); err != nil {
		return nil, err
	}

	return qualifiers, nil
}

func (a *Analyzer) importFile(filename string, root Node) error {
	var dir string

	if a.filename != "" {
		dir = filepath.Dir(a.filename)
	}

	input := filepath.Join(dir, filename)
	f, err := os.Open(input)

	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(f)

	if err != nil {
		return err
	}

	f.Close()
	t := NewTokenizer(data)
	importAnalyzer := NewAnalyzer(t, input)
	importRoot, err := importAnalyzer.Analyze()

	if err != nil {
		return err
	}

	root.AdoptChildren(importRoot)
	root.ImportSymbols(importRoot)

	return nil
}
