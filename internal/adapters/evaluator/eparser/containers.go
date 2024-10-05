package eparser

import (
	"strings"

	"github.com/vingarcia/insights"
)

// TokenList represents a list data type
type TokenList []Token

// NewTokenList is an internal constructor that matches the signature
// of the type `Function`
func NewTokenList(args []Token, scope mapToken) (Token, error) {
	return TokenList(args), nil
}

func (t TokenList) Clone() Token {
	return t
}

// TODO(vingarcia): Consider how to handle an infinite loop
// in case the list contains itself
func (t TokenList) String() string {
	tokens := []string{}
	for _, token := range t {
		tokens = append(tokens, token.String())
	}
	return "[" + strings.Join(tokens, ",") + "]"
}

// TokenMap represents a map data type
type TokenMap map[string]Token

// NewTokenMap is an internal constructor that matches the signature
// of the type `Function`
func NewTokenMap(args []Token, scope mapToken) (Token, error) {
	m := TokenMap{}
	for _, v := range args {
		kv, notAKVPair := v.(KeyValuePair)
		if !notAKVPair {
			return nil, insights.SyntaxErr("map constructor expects only `key: value` pairs", map[string]any{
				"invalidArgument": v,
			})
		}

		_, alreadyExists := m[kv.Key]
		if alreadyExists {
			return nil, insights.SyntaxErr("duplicate key in map literal", map[string]any{
				"key": kv.Key,
			})
		}

		m[kv.Key] = kv.Value
	}

	return m, nil
}

func (t TokenMap) Clone() Token {
	return t
}

// TODO(vingarcia): Consider how to handle an infinite loop
// in case the map contains itself
func (t TokenMap) String() string {
	kvPairs := []string{}
	for k, v := range t {
		kvPairs = append(kvPairs, k+":"+v.String())
	}
	return "{" + strings.Join(kvPairs, ",") + "}"
}

type KeyValuePair struct {
	Key   string
	Value Token
}

func (k KeyValuePair) Clone() Token {
	return k
}

func (k KeyValuePair) String() string {
	return k.Key + ":" + k.Value.String()
}
