package eparser

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Token interface {
	Clone() Token
	String() string
}

// opToken represents operators
type opToken string

func (o opToken) Clone() Token {
	return o
}

func (o opToken) String() string {
	return string(o)
}

// unaryPlaceholderToken is only used for making it easier
// to handle unary operators as if they were binary.
//
// When parsing the expression it will just be dropped
type unaryPlaceholderToken struct{}

func (u unaryPlaceholderToken) Clone() Token {
	return u
}

func (unaryPlaceholderToken) String() string {
	return "UnaryToken"
}

// Function represents a custom function for our parser
type Function func(args []Token, scope mapToken) (Token, error)

// Type implements the Token interface
func (f Function) Clone() Token {
	return f
}

func (f Function) String() string {
	return "[function]"
}

// strToken represent string tokens
type strToken string

func (s strToken) Clone() Token {
	return s
}

func (s strToken) String() string {
	b, _ := json.Marshal(s)
	return string(b)
}

// intToken represent real numerical values
type intToken int

func (i intToken) Clone() Token {
	return i
}

func (i intToken) String() string {
	return strconv.Itoa(int(i))
}

// floatToken represent real numerical values
type floatToken float64

func (f floatToken) Clone() Token {
	return f
}

func (f floatToken) String() string {
	return strconv.FormatFloat(float64(f), 'f', -1, 64)
}

// boolToken represent boolean values
type boolToken bool

func (b boolToken) Clone() Token {
	return b
}

func (b boolToken) String() string {
	if bool(b) {
		return "true"
	}
	return "false"
}

// mapToken represent boolean values
type mapToken map[string]Token

func (m mapToken) Clone() Token {
	return m
}

func (m mapToken) String() string {
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Sprintf("error stringifying '%#v': %s", m, err)
	}

	return string(b)
}

func (m mapToken) getChildMap() mapToken {
	return mapToken{
		"$parent": m,
	}
}

// tupleToken represents tuples like in Python: (1, "foo", false)
type tupleToken []Token

func (t tupleToken) Clone() Token {
	return t
}

func (t tupleToken) String() string {
	items := []string{}
	for _, token := range t {
		items = append(items, token.String())
	}

	return "(" + strings.Join(items, ",") + ")"
}

// refToken is used to keep references
type refToken struct {
	// The value found at compilation time
	originalValue Token

	// The key used to reference this token at compilation time
	key varToken

	// The scope of variables available at compilation time:
	origin mapToken
}

func (r refToken) Clone() Token {
	return r
}

func (r refToken) String() string {
	return "&" + strings.Join(r.key, ".")
}

func (r refToken) Resolve(localScope map[string]Token) Token {
	// Local variables have no map of origin,
	// thus, require a localScope to be resolved:
	if r.origin == nil && localScope != nil {
		// Get the most recent value from the local scope:
		refValue := r.key.Resolve(localScope)
		if refValue != nil {
			// TODO(vingarcia): Consider cloning this value first
			return refValue
		}
	}

	// In last case return the compilation-time value:
	// TODO(vingarcia): Consider cloning this value first
	return r.originalValue
}

// varToken represent variable references
//
// A variable such as: `a.b['and c']`
// would be stored here as: []string{"a", "b", "and c"}
type varToken []string

func (v varToken) Clone() Token {
	return append(varToken{}, v...)
}

func (v varToken) String() string {
	out := ""
	for _, str := range v {
		onlyVarChars := isVarChar(rune(str[0]))
		if onlyVarChars {
			for _, c := range str[1:] {
				if !isVarChar(c) && !unicode.IsNumber(c) {
					onlyVarChars = false
					break
				}
			}
		}

		if onlyVarChars {
			out += "." + str
		} else {
			b, _ := json.Marshal(str)
			out += "[" + string(b) + "]"
		}
	}
	return out
}

func (v varToken) Resolve(vars map[string]Token) Token {
	value := vars[v[0]]
	for _, str := range v[1:] {
		m, ok := value.(mapToken)
		if !ok {
			return strToken(v.String())
		}

		value = m[str]
	}

	if value == nil {
		return strToken(v.String())
	}

	return value
}
