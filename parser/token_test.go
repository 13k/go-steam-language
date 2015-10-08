package parser

import (
	"testing"
)

var (
	token1 = &Token{Op: OpWhitespace, Value: []byte("\n"), Row: 1, Col: 1}
	token2 = &Token{Op: OpTerminator, Value: []byte(";"), Row: 1, Col: 1}
	token3 = &Token{Op: OpString, Value: []byte("hello"), Row: 1, Col: 1}
	token4 = &Token{Op: OpComment, Value: []byte("this is a comment"), Row: 1, Col: 1}
	token5 = &Token{Op: OpIdentifier, Value: []byte("id"), Row: 1, Col: 1}
	token6 = &Token{Op: OpPreprocess, Value: []byte("import"), Row: 1, Col: 1}
	token7 = &Token{Op: OpOperator, Value: []byte("<"), Row: 1, Col: 1}
	token8 = &Token{Op: OpInvalid, Value: []byte("&"), Row: 1, Col: 1}
)

func TestTokenEqual(t *testing.T) {
	token := &Token{Op: OpString, Value: []byte("hello")}

	if !token3.Equal(token) {
		t.Fatalf("expected Equal() to return true but returned false")
	}

	token.Op = OpWhitespace

	if token3.Equal(token) {
		t.Fatalf("expected Equal() to return false but returned true")
	}
}

func TestTokenValueEqual(t *testing.T) {
	token := &Token{Op: OpString, Value: []byte("hello")}

	if !token.ValueEqual([]byte("hello")) {
		t.Fatalf("expected ValueEqual() to return true but returned false")
	}
}

func TestTokenValueEqualString(t *testing.T) {
	token := &Token{Op: OpString, Value: []byte("hello")}

	if !token.ValueEqualString("hello") {
		t.Fatalf("expected ValueEqualString() to return true but returned false")
	}
}

func TestTokenValueString(t *testing.T) {
	token := &Token{Op: OpString, Value: []byte("hello")}

	if token.ValueString() != "hello" {
		t.Fatalf("mismatch: got %q, but expected %q", token.ValueString(), "hello")
	}
}

func TestTokenStringValues(t *testing.T) {
	tokens := []*Token{token1, token2, token3}
	expected := []string{"\n", ";", "hello"}
	values := tokenStringValues(tokens)

	for i, s := range values {
		if s != expected[i] {
			t.Fatalf("mismatch: got %q, but expected %q", s, expected[i])
		}
	}
}

func TestTokenQueueEnqueue(t *testing.T) {
	tokenQueue := NewTokenQueue()
	tokenQueue.enqueue(token6)
	tokenQueue.enqueue(token3)
	tokenQueue.enqueue(token2)

	if tokenQueue.list.Len() != 3 {
		t.Fatalf("expected tokenQueue to have %d items but it has %d", 3, tokenQueue.list.Len())
	}
}

func TestTokenQueueDequeue(t *testing.T) {
	tokenQueue := NewTokenQueue()
	tokenQueue.enqueue(token6)
	tokenQueue.enqueue(token3)
	tokenQueue.enqueue(token2)

	token := tokenQueue.Dequeue()

	if token == nil {
		t.Fatalf("expected token to not be nil but it is")
	}

	if tokenQueue.list.Len() != 2 {
		t.Fatalf("expected tokenQueue to have %d items but it has %d", 2, tokenQueue.list.Len())
	}

	if !token6.Equal(token) {
		t.Fatalf("expected token6 to be Equal() to token but it is not")
	}

	token = tokenQueue.Dequeue()

	if token == nil {
		t.Fatalf("expected token to not be nil but it is")
	}

	if tokenQueue.list.Len() != 1 {
		t.Fatalf("expected tokenQueue to have %d item but it has %d", 1, tokenQueue.list.Len())
	}

	if !token3.Equal(token) {
		t.Fatalf("expected token6 to be Equal() to token but it is not")
	}

	token = tokenQueue.Dequeue()

	if token == nil {
		t.Fatalf("expected token to not be nil but it is")
	}

	if tokenQueue.list.Len() != 0 {
		t.Fatalf("expected tokenQueue to have %d items but it has %d", 0, tokenQueue.list.Len())
	}

	if !token2.Equal(token) {
		t.Fatalf("expected token6 to be Equal() to token but it is not")
	}

	for i := 0; i < 3; i++ {
		token = tokenQueue.Dequeue()

		if token != nil {
			t.Fatalf("expected token to be nil but it is not")
		}
	}
}

func TestTokenQueuePeek(t *testing.T) {
	tokenQueue := NewTokenQueue()
	tokenQueue.enqueue(token6)
	token := tokenQueue.Peek()

	if token == nil {
		t.Fatalf("expected token to not be nil but it is")
	}

	if !token6.Equal(token) {
		t.Fatalf("expected token6 to be Equal() to token but it is not")
	}

	tokenQueue = NewTokenQueue()
	token = tokenQueue.Peek()

	if token != nil {
		t.Fatalf("expected token to be nil but it is not")
	}
}

func TestTokenQueueLen(t *testing.T) {
	tokenQueue := NewTokenQueue()
	tokenQueue.enqueue(token6)
	tokenQueue.enqueue(token3)
	tokenQueue.enqueue(token2)

	for i := 0; i < 4; i++ {
		n1 := tokenQueue.Len()
		n2 := tokenQueue.list.Len()

		if n1 != n2 {
			t.Fatalf("expected Len() to return %d but it returned %d", n2, n1)
		}

		tokenQueue.Dequeue()
	}
}

func TestTokenizerTokenize(t *testing.T) {
	data := []byte(`
		// import file.steamd
		#import "file.steamd"

		class MyClass<MyEnum::EnumProperty> {
			const uint C = 1;
			byte<20> x;
			y = C | MyEnum2::y;
			old string; obsolete "this is not used anymore"
		};
	`)

	expectedQ := NewTokenQueue()
	expectedQ.enqueue(&Token{Op: OpPreprocess, Value: []byte("import")})
	expectedQ.enqueue(&Token{Op: OpString, Value: []byte("file.steamd")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("class")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("MyClass")})
	expectedQ.enqueue(&Token{Op: OpOperator, Value: []byte("<")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("MyEnum")})
	expectedQ.enqueue(&Token{Op: OpNamespace, Value: []byte("::")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("EnumProperty")})
	expectedQ.enqueue(&Token{Op: OpOperator, Value: []byte(">")})
	expectedQ.enqueue(&Token{Op: OpOperator, Value: []byte("{")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("const")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("uint")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("C")})
	expectedQ.enqueue(&Token{Op: OpOperator, Value: []byte("=")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("1")})
	expectedQ.enqueue(&Token{Op: OpTerminator, Value: []byte(";")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("byte")})
	expectedQ.enqueue(&Token{Op: OpOperator, Value: []byte("<")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("20")})
	expectedQ.enqueue(&Token{Op: OpOperator, Value: []byte(">")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("x")})
	expectedQ.enqueue(&Token{Op: OpTerminator, Value: []byte(";")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("y")})
	expectedQ.enqueue(&Token{Op: OpOperator, Value: []byte("=")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("C")})
	expectedQ.enqueue(&Token{Op: OpOperator, Value: []byte("|")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("MyEnum2")})
	expectedQ.enqueue(&Token{Op: OpNamespace, Value: []byte("::")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("y")})
	expectedQ.enqueue(&Token{Op: OpTerminator, Value: []byte(";")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("old")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("string")})
	expectedQ.enqueue(&Token{Op: OpTerminator, Value: []byte(";")})
	expectedQ.enqueue(&Token{Op: OpIdentifier, Value: []byte("obsolete")})
	expectedQ.enqueue(&Token{Op: OpString, Value: []byte("this is not used anymore")})
	expectedQ.enqueue(&Token{Op: OpOperator, Value: []byte("}")})
	expectedQ.enqueue(&Token{Op: OpTerminator, Value: []byte(";")})

	tokenizer := NewTokenizer(data)
	tokens, err := tokenizer.Tokenize()

	if err != nil {
		t.Fatalf("not expected error %v", err)
	}

	if tokens.Len() != expectedQ.Len() {
		t.Fatalf("expected %d tokens, got %d", expectedQ.Len(), tokens.Len())
	}

	expected := expectedQ.Dequeue()
	token := tokens.Dequeue()

	for expected != nil {
		if !expected.Equal(token) {
			t.Fatalf("mismatch:\nexpected: Token{Op: %s, Value: %q}\ngot: Token{Op: %s, Value: %q}", expected.Op.String(), expected.Value, token.Op.String(), token.Value)
		}

		expected = expectedQ.Dequeue()
		token = tokens.Dequeue()
	}

	data = []byte(``)
	tokenizer = NewTokenizer(data)
	q, err := tokenizer.Tokenize()

	if err != nil {
		t.Fatalf("not expected error %v", err)
	}

	if q.Len() != 0 {
		t.Fatalf("expected %d tokens, got %d", 0, q.Len())
	}
}

func TestCountRunes(t *testing.T) {
	data := []byte("a\nb\r\ncdéfgåí界")
	rows, cols, err := countRunes(data)

	if err != nil {
		t.Fatalf("not expected error %v", err)
	}

	if rows != 2 || cols != 8 {
		t.Fatalf("mismatch: expected %d rows and %d columns, got %d rows and %d columns", 2, 8, rows, cols)
	}
}
