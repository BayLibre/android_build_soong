package parser

//import (
//	"bytes"
//	"fmt"
//)
//
//const eof = -1
//
//type token int
//
//const (
//	token_word token = iota
//	token_newline
//	token_punctuation
//	token_subst
//)
//
//type androidMkModule struct {
//	assignments map[string]string
//}
//
//type androidMkFile struct {
//	assignments map[string]string
//	modules     []androidMkModule
//	includes    []string
//}
//
//type parseState struct {
//	module     androidMkModule
//	b          []byte
//	stateStart int
//	result     *androidMkFile
//}
//
//type stateFunc func(*parseState, int, rune) (stateFunc, error)
//
//func ParseAndroidMkFile(b []byte) (*androidMkFile, error) {
//	r := bytes.NewReader(b)
//	printLines(r)
//	return nil, nil
//	//	parseState := &parseState{
//	//		b: b,
//	//		result: &androidMkFile{
//	//			assignments: make(map[string]string),
//	//		},
//	//	}
//	//
//	//	state := parseStart
//	//	var err error
//	//	for i := 0; i < len(b); i++ {
//	//		r := rune(b[i])
//	//		state, err = state(parseState, i, r)
//	//		if err != nil {
//	//			return nil, err
//	//		}
//	//	}
//	//
//	//	_, err = state(parseState, len(parseState.b), eof)
//	//	if err != nil {
//	//		return nil, err
//	//	}
//	//
//	//	return parseState.result, nil
//}
//
//func parseStart(state *parseState, i int, r rune) (stateFunc, error) {
//	switch {
//	case isValidIdentifier(r):
//		state.stateStart = i
//		return parseWord, nil
//	case r == '$':
//		state.stateStart = i
//		return parseSubst, nil
//	case r == '#':
//		return parseComment, nil
//	case r == ' ', r == '\t':
//		return parseStart, nil
//	case r == '\\':
//		return parseEscape, nil
//	case r == ':', r == '=', r == '+':
//		state.stateStart = i
//		return parsePunctuation, nil
//	case r == '\n':
//		emit(token_newline, nil)
//		return parseStart, nil
//	default:
//		return nil, fmt.Errorf("unexpected character %c", r)
//	}
//}
//
//func parseWord(state *parseState, i int, r rune) (stateFunc, error) {
//	switch {
//	case isValidIdentifier(r):
//		return parseWord, nil
//	default:
//		emit(token_word, state.b[state.stateStart:i])
//		return parseStart(state, i, r)
//	}
//}
//
//func parseComment(state *parseState, i int, r rune) (stateFunc, error) {
//	switch {
//	case r == '\n':
//		emit(token_newline, nil)
//		return parseStart, nil
//	default:
//		return parseComment, nil
//	}
//}
//
//func parseEscape(state *parseState, i int, r rune) (stateFunc, error) {
//	switch {
//	case r == '\n':
//		return parseStart, nil
//	case r == '"', r == '#':
//		return parseWord, nil
//	default:
//		return nil, fmt.Errorf("Unexpected escaped character %c", r)
//	}
//}
//
//func parsePunctuation(state *parseState, i int, r rune) (stateFunc, error) {
//	switch {
//	case r == ':', r == '=', r == '+':
//		return parsePunctuation, nil
//	default:
//		emit(token_punctuation, state.b[state.stateStart:i])
//		return parseStart(state, i, r)
//	}
//}
//
//func parseSubst(state *parseState, i int, r rune) (stateFunc, error) {
//	switch {
//	case r == '(':
//		return parseSubstParen, nil
//	case r == '{':
//		return parseSubstBracket, nil
//	default:
//		return nil, fmt.Errorf("Unexpected character %c after $", r)
//	}
//}
//
//func parseSubstParen(state *parseState, i int, r rune) (stateFunc, error) {
//	switch {
//	case r == ')':
//		emit(token_subst, state.b[state.stateStart:i+1])
//		return parseStart, nil
//	case r == '(', r == '{', r == '$':
//		return nil, fmt.Errorf("Unsupported character %c in substitution", r)
//	default:
//		return parseSubstParen, nil
//	}
//}
//
//func parseSubstBracket(state *parseState, i int, r rune) (stateFunc, error) {
//	switch {
//	case r == '}':
//		emit(token_subst, state.b[state.stateStart:i+1])
//		return parseStart, nil
//	case r == '(', r == '{', r == '$':
//		return nil, fmt.Errorf("Unsupported character %c in substitution", r)
//	default:
//		return parseSubstParen, nil
//	}
//}
//
//func isValidIdentifier(r rune) bool {
//	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
//		r == '_' || r == '-' || r == '.' || r == '/'
//}
//
//func emit(t token, b []byte) {
//	fmt.Println(t, string(b))
//}
