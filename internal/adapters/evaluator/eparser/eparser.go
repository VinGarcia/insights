package eparser

import (
	"fmt"
	"strconv"
	"unicode"

	"github.com/vingarcia/insights"
	"github.com/vingarcia/insights/internal/adapters/evaluator"
)

type ParsingCtx struct {
	currentLine    int
	lastLineStart  int
	lastTokenWasOp bool
}

func (p *ParsingCtx) HandleNewLine(newLineRuneIdx int) {
	p.currentLine++
	p.lastLineStart = newLineRuneIdx + 1
}

func (p ParsingCtx) FormatLineCol(i int) string {
	return strconv.Itoa(p.currentLine) + ":" + strconv.Itoa(i-p.lastLineStart)
}

// Parse will decode the input expression into a Reverse
// Polish notation for easy future evaluation.
func Parse(strExpr string, vars map[string]Token) (_ evaluator.Expression, err error) {
	if len(strExpr) == 0 {
		return nil, fmt.Errorf("cannot build an expression from an empty string")
	}

	expr := []rune(strExpr)

	var rpnBuilder RPNBuilder

	parsingCtx := ParsingCtx{
		currentLine:    0,
		lastLineStart:  0,
		lastTokenWasOp: false,
	}

	i := consumeSpaces(expr, 0, &parsingCtx)

	// Each iteration of this loop should produce a token or an operator
	for ; i < len(expr) && expr[i] != ';'; i++ {
		switch {
		case unicode.IsNumber(expr[i]):
			var num Token
			i, num, err = parseNumber(expr, i)
			if err != nil {
				return nil, err
			}

			rpnBuilder.handleToken(num)

		case isVarChar(expr[i]):
			var varName string
			i, varName = parseVar(expr, i)
			rpnBuilder.handleToken(varToken([]string{varName}))

			parser := reservedWordParsers[varName]
			if parser != nil {
				i, err = parser(expr, &parsingCtx, &rpnBuilder, i)
				if err != nil {
					return nil, err
				}
			} else {
				token := vars[varName]
				if token != nil {
					// Save a reference token:
					// TODO(vingarcia): Consider cloning the token here
					err := rpnBuilder.handleToken(refToken{
						key:           varToken{varName},
						originalValue: token,
					})
					if err != nil {
						return nil, err
					}
				} else {
					// Save the variable name:
					rpnBuilder.handleToken(varToken([]string{varName}))
				}
			}
		case expr[i] == '\'' || expr[i] == '"':
			// If it is a string literal, parse it and
			// add to the output queue.
			quote := expr[i]
			formattedPos := parsingCtx.FormatLineCol(i)

			i++
			str := []rune{}
			for i < len(expr) && expr[i] != quote && expr[i] != '\n' {
				if expr[i] == '\\' {
					switch expr[i+1] {
					case 'n':
						i += 2
						str = append(str, '\n')

					case 't':
						i += 2
						str = append(str, '\t')

					default:
						switch expr[i+1] {
						case '"', '\'':
							i++
						case '\n':
							i++
							parsingCtx.HandleNewLine(i)
						}
						str = append(str, expr[i])
						i++
					}
				} else {
					str = append(str, expr[i])
					i++
				}
			}

			if expr[i] != quote {
				return nil, insights.SyntaxErr("string literal not terminated", map[string]any{
					"startedAt": formattedPos,
				})
			}
			i++
			rpnBuilder.handleToken(strToken(string(str)))
		default:
			// Otherwise, the variable is an operator or parenthesis.
			switch expr[i] {
			case '(':
				// If it is a function call:
				if rpnBuilder.lastTokenWasOp == "no" {
					// This counts as a bracket and as an operator:
					rpnBuilder.handleOp("()")
					// Add it as a bracket to the op stack:
				}
				rpnBuilder.openBracket("(")
				i++
			case '[':
				if rpnBuilder.lastTokenWasOp == "no" {
					// If it is an operator:
					rpnBuilder.handleOp("[]")
				} else {
					// If it is the list constructor:
					// Add the list constructor to the rpn:
					rpnBuilder.handleToken(Function(NewTokenList))

					// We make the program see it as a normal function call:
					rpnBuilder.handleOp("()")
				}
				// Add it as a bracket to the op stack:
				rpnBuilder.openBracket("[")
				i++
			case '{':
				// Add a map constructor call to the rpn:
				rpnBuilder.handleToken(Function(NewTokenMap))

				// We make the program see it as a normal function call:
				rpnBuilder.handleOp("()")
				rpnBuilder.openBracket("{")
				i++
			case ')':
				rpnBuilder.closeBracket("(")
				i++
			case ']':
				rpnBuilder.closeBracket("[")
				i++
			case '}':
				rpnBuilder.closeBracket("{")
				i++
			default:
				{
					// Then the token is an operator

					start := i
					opRunes := []rune{expr[i]}
					i++
					// These ops are here to serve as ending characters so that expressions
					// such as: `10 *-3` don't interpret *- as a single operator when its actually 2.
					opStartingChars := map[rune]bool{
						'+': true, '-': true, '\'': true, '"': true,
						'(': true, ')': true, '[': true, ']': true, '{': true, '}': true,
						'_': true,
					}
					for i < len(expr) && unicode.IsPunct(expr[i]) && !opStartingChars[expr[i]] {
						opRunes = append(opRunes, expr[i])
						i++
					}
					op := string(opRunes)

					// Evaluate the meaning of this operator in the following order:
					// 1. Is it a reserved word?
					// 2. Is it a valid operator?
					// 3. Is there a character parser for its first character?
					parser, isReservedWord := reservedWordParsers[op]
					if isReservedWord {
						// Parse reserved operators:
						i, err = parser(expr, &parsingCtx, &rpnBuilder, i)
						if err != nil {
							return nil, err
						}
					} else if _, isKnownOp := opPrecedence[op]; isKnownOp {
						rpnBuilder.handleOp(op)
						// Maybe just the first character is an operator:
					} else if parser, isReservedWord := reservedWordParsers[op[0:1]]; isReservedWord {
						i = start + 1
						i, err = parser(expr, &parsingCtx, &rpnBuilder, i)
						if err != nil {
							return nil, err
						}
					} else {
						return nil, insights.SyntaxErr("unrecognized operator", map[string]any{
							"op":  op,
							"pos": parsingCtx.FormatLineCol(i),
						})
					}
				}
			}
		}

		i = consumeSpaces(expr, i, &parsingCtx)
	}

	rpn, err := rpnBuilder.FinishAndReturnRPN(expr, i, parsingCtx)
	if err != nil {
		return nil, err
	}

	fmt.Println(rpn)
	return nil, nil
}

// EvaluationData contains the context used during
// evaluation and is passed as argument to all
// operator and custom operator functions,
// which allows the operators to take advantage
// of that
type EvaluationData struct {
	Vars mapToken

	LeftRef  refToken
	RightRef refToken
}

// Evaluate will copy
func Evaluate(originalRpn []Token, vars map[string]Token) (_ Token, err error) {
	var left, right Token
	data := EvaluationData{
		Vars: vars,
	}

	rpn := copyRPN(originalRpn)

	evalStack := []Token{}

	l := len(rpn)
	for ; l > 0; l-- {
		token := rpn[l-1]
		op, isOperator := token.(opToken)
		if !isOperator {
			if v, isVar := token.(varToken); isVar {
				token = v.Resolve(data.Vars)
			}

			evalStack = append(evalStack, token)
			continue
		}

		// If it got here it's an operator:
		evalStack, left, right, err = popLeftAndRightOperands(evalStack)
		if err != nil {
			return nil, err
		}

		switch v := right.(type) {
		case refToken:
			data.RightRef = v
			right = v.Resolve(data.Vars)
		case varToken:
			data.RightRef = refToken{key: v}
		default:
			data.RightRef = refToken{}
		}

		switch v := left.(type) {
		case refToken:
			data.LeftRef = v
			left = v.Resolve(data.Vars)
		case varToken:
			data.LeftRef = refToken{key: v}
		default:
			data.LeftRef = refToken{}
		}

		if fn, ok := left.(Function); ok && op == "()" {
			var args tupleToken
			if tuple, ok := right.(tupleToken); ok {
				args = tuple
			} else {
				// A tuple with a single element, which might be a unaryPlaceholder:
				args = tupleToken{right}
			}

			var fnReceiver = data.Vars
			if data.LeftRef.origin != nil {
				fnReceiver = data.LeftRef.origin
			}

			resp, err := execFunc(fnReceiver, fn, args, data.Vars)
			if err != nil {
				return nil, insights.RuntimeErr("error parsing function", map[string]any{
					"error": err,
				})
			}

			evalStack = append(evalStack, resp)
		} else {
			// * * * * * All other operations * * * * * //

			// TODO(vingarcia): Copy the exec_operator func from cparse (it's more complex than this):
			resp, err := operators[op](left, right, op, &data)
			if err != nil {
				return nil, insights.RuntimeErr("operation error", map[string]any{
					"error": err,
				})
			}

			evalStack = append(evalStack, resp)
		}
	}

	return nil, nil
}

func popLeftAndRightOperands(evalStack []Token) (updatedStack []Token, left Token, right Token, _ error) {
	if len(evalStack) < 2 {
		return nil, nil, nil, insights.InternalErr("missing operands for operator", map[string]any{
			"evalStack": evalStack,
		})
	}

	l := len(evalStack)
	right = (evalStack)[l-1]
	left = (evalStack)[l-2]
	return (evalStack)[:l-2], left, right, nil
}

func copyRPN(rpn []Token) (copy []Token) {
	for _, token := range rpn {
		copy = append(copy, token.Clone())
	}

	return copy
}

func consumeSpaces(expr []rune, index int, parsingCtx *ParsingCtx) (newIndex int) {
	for i := index; i < len(expr); i++ {
		if expr[i] == '\n' {
			parsingCtx.HandleNewLine(i)
		}
		if !unicode.IsSpace(expr[i]) {
			return i
		}
	}

	return 0
}

// isVarChar checks if a character is the first character of a variable:
func isVarChar(c rune) bool {
	return unicode.IsLetter(c) || c == '_'
}

func parseVar(expr []rune, index int) (newIndex int, varName string) {
	// parseVar assumes the first character is already a valid starting
	// character for a varname, so we skip it:
	for i := index + 1; i < len(expr); i++ {
		if !isVarChar(expr[i]) && !unicode.IsNumber(expr[i]) && expr[i] != '_' {
			return i, string(expr[index:i])
		}
	}

	return len(expr), string(expr[index:])
}

var hexValidChars = map[rune]bool{
	'0': true, '1': true, '2': true, '3': true, '4': true,
	'5': true, '6': true, '7': true, '8': true, '9': true,
	'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true,
	'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true,
}

func isValidHexDigit(r rune) bool {
	return hexValidChars[r]
}

func parseNumber(expr []rune, index int) (newIndex int, token Token, err error) {
	base := 10

	i := index
	if expr[i] == '0' {
		// Handle hexadecimal numbers such as 0x10
		if expr[i+1] == 'x' {
			base = 16
			// Skip the '0x' characters
			i += 2
			// Handle octal numbers such as 010
		} else if unicode.IsNumber(expr[i+1]) {
			base = 8
			// Skip the '0' character
			i++
		}
	}

	isNumberFn := unicode.IsNumber
	if base == 16 {
		isNumberFn = isValidHexDigit
	}

	isFloat := false

	// Find the end of the numerical literal:
	for ; i < len(expr) && isNumberFn(expr[i]); i++ {
		// Consume the decimal part of the number:
		if expr[i] == '.' {
			i++
			isFloat = true

			for i < len(expr) && isNumberFn(expr[i]) {
				i++
			}

			break
		}
	}

	if isFloat {
		if base != 10 {
			return 0, nil, insights.SyntaxErr("only base 10 literals can have decimals", map[string]any{
				"literal": string(expr[index:i]),
			})
		}

		num, err := strconv.ParseFloat(string(expr[index:i]), 64)
		if err != nil {
			panic(
				fmt.Errorf("unexpected error parsing a previously validated number '%s': %w",
					string(expr[index:i]), err,
				),
			)
		}

		return i, floatToken(num), nil
	}

	num, err := strconv.ParseInt(string(expr[index:i]), base, 64)
	if err != nil {
		return 0, nil, insights.SyntaxErr("error parsing numeric literal", map[string]any{
			"literal": expr[index:i],
			"error":   err,
		})
	}

	return i, intToken(num), nil
}

func execFunc(this mapToken, fn Function, args tupleToken, vars mapToken) (Token, error) {
	vars = vars.getChildMap()
	fn(args, mapToken{
		"$parent": vars,
		"this":    this,
	})

	return nil, nil
}
