package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	keyword = []string{"class", "constructor", "function", "method", "field", "static", "var", "int", "char", "boolean",
		"void", "true", "false", "null", "this", "let", "do", "if", "else", "while", "return"}
	symbols    = []byte{'{', '}', '[', ']', '(', ')', '.', ',', ';', '+', '-', '*', '/', '&', '|', '<', '>', '=', '~'}
	ops        = []string{"+", "-", "*", "/", "&", "|", "<", ">", "="}
	keywordSet = make(map[string]bool)
	symbolsSet = make(map[byte]bool)
	opsSet     = make(map[string]bool)
)

func main() {
	// init
	initMaps()
	targetFiles := getFiles(os.Args[1])

	for _, targetFile := range targetFiles {
		// create new output file
		out, err := os.OpenFile(createOutput(targetFile), os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			panic(err)
		}
		tokenizer := buildTokenizer(targetFile)
		e := buildCompilationEngine(tokenizer, out)
		e.compileClass(0)
		out.Close()
	}
}

/*
return the paths of all the jack files
*/
func initMaps() {
	for _, str := range keyword {
		keywordSet[str] = true
	}
	for _, b := range symbols {
		symbolsSet[b] = true
	}
	for _, op := range ops {
		opsSet[op] = true

	}
}

func getFiles(target string) []string {
	inputFiles := make([]string, 0)
	err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}
		if !info.IsDir() && filepath.Ext(path) == ".jack" {
			inputFiles = append(inputFiles, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return inputFiles
}

// FIXME: wrong filename
func createOutput(target string) string {
	d := filepath.Dir(target)

	ext := filepath.Ext(target)

	base := filepath.Base(target)

	var o string
	oFormat := ".xml"
	if isDir(target) {
		o = filepath.Join(d, base, "Main.xml")
	} else {
		o = filepath.Join(d, strings.Replace(base, ext, oFormat, 1))
	}
	return o
}

func isDir(f string) bool {
	osf, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	defer osf.Close()
	info, err := osf.Stat()
	if err != nil {
		panic(err)
	}
	return info.IsDir()
}

/**
Tokenizer
*/

const (
	TokenTypeKeyword = iota
	TokenTypeSymbol
	TokenTypeIdentifier
	TokenTypeIntConst
	TokenTypeStringConst
)

type Tokenizer struct {
	fileContent string
	cursor      int
	curr        string
	currType    int
}

func buildTokenizer(filePath string) *Tokenizer {
	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)
	cb := make([]byte, 0)
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		if line[0] == '/' || line[0] == '*' {
			continue
		}
		if idx := strings.Index(line, "//"); idx > 0 {
			line = line[:idx]
		}
		cb = append(cb, []byte(line)...)
		cb = append(cb, '\n')
	}
	return &Tokenizer{
		fileContent: string(cb),
		cursor:      0,
	}
}

func (t *Tokenizer) hasMoreTokens() bool {
	return t.cursor <= len(t.fileContent)-1
}

func (t *Tokenizer) advance() {
	read := make([]byte, 0)

	isDuringStr := false

	for t.hasMoreTokens() {
		curByte := t.fileContent[t.cursor]

		if isDuringStr {
			read = append(read, curByte)
			t.cursor += 1
			if curByte != '"' {
				continue
			} else {
				isDuringStr = false
				break
			}
		} else {
			if curByte == '"' {
				if len(read) > 0 {
					break
				} else {
					read = append(read, curByte)
					isDuringStr = true
					t.cursor += 1
					continue
				}
			}
			if symbolsSet[curByte] {
				if len(read) > 0 {
					break
				} else {
					read = append(read, curByte)
					t.cursor += 1
					break
				}
			}
			if curByte == ' ' || curByte == '\n' {
				t.cursor += 1
				if len(read) > 0 {
					break
				} else {
					continue
				}
			}
			read = append(read, curByte)
			t.cursor += 1
		}

	}
	t.curr = string(read)
	t.currType = getType(t.curr)
}

func (t *Tokenizer) tokenType() int {
	return t.currType
}

func (t *Tokenizer) getCur() string {
	return t.curr
}

func getType(cur string) int {
	if keywordSet[cur] {
		return TokenTypeKeyword
	}
	if len(cur) == 1 && symbolsSet[cur[0]] {
		return TokenTypeSymbol
	}
	if isConstantInteger(cur) {
		return TokenTypeIntConst
	}
	if cur[0] == '"' && cur[len(cur)-1] == '"' {
		return TokenTypeStringConst
	}
	return TokenTypeIdentifier
}

func isConstantInteger(cur string) bool {
	if cur == "" {
		return true
	}
	if cur[0] >= '0' && cur[0] <= '9' {
		return isConstantInteger(cur[1:])
	} else {
		return false
	}
}

/**
CompilationEngine
*/

type CompilationEngine struct {
	Tokenizer *Tokenizer
	Out       *os.File
}

func buildCompilationEngine(tokenizer *Tokenizer, out *os.File) *CompilationEngine {
	return &CompilationEngine{
		Tokenizer: tokenizer,
		Out:       out,
	}
}

func (e *CompilationEngine) compile() {
	for e.Tokenizer.hasMoreTokens() {
		if e.Tokenizer.getCur() == "class" {
			e.compileClass(0)
		} else {
			panic("not expect: " + e.Tokenizer.getCur())
		}
	}
}

func (e *CompilationEngine) compileClass(depth int) {
	e.Tokenizer.advance()
	e.writePureTag("class", true, depth)
	e.writeTag("keyword", "class", depth+1)

	e.Tokenizer.advance()
	e.writeTag("identifier", e.Tokenizer.getCur(), depth+1)
	e.Tokenizer.advance()

	e.writeTag("symbol", e.Tokenizer.getCur(), depth+1)
	e.Tokenizer.advance()

	for e.Tokenizer.hasMoreTokens() {
		switch e.Tokenizer.getCur() {
		case "static", "field":
			e.compileClassVarDec(depth + 1)
		case "method", "function", "constructor":
			e.compileSubroutine(depth + 1)
		case "}":
			e.writeTag("symbol", "}", depth+1)
			e.writePureTag("class", false, depth)
			return
		default:
			// should not happen
			panic("compile class error: " + e.Tokenizer.getCur())
		}
	}
}

func (e *CompilationEngine) compileClassVarDec(depth int) {
	e.writePureTag("classVarDec", true, depth)
	// should be field or static
	e.writeTag("keyword", e.Tokenizer.getCur(), depth+1)
	e.Tokenizer.advance()

	e.writeTag("identifier", e.Tokenizer.getCur(), depth+1)
	e.Tokenizer.advance()

	e.writeTag("identifier", e.Tokenizer.getCur(), depth+1)
	e.Tokenizer.advance() // should be ;

	for e.Tokenizer.getCur() != ";" {
		// should be ","
		e.writeTag("symbol", e.Tokenizer.getCur(), depth+1)
		e.Tokenizer.advance()
		e.writeTag("identifier", e.Tokenizer.getCur(), depth+1)
		e.Tokenizer.advance()
	}

	e.writeTag("symbol", ";", depth+1)
	e.Tokenizer.advance()
	e.writePureTag("classVarDec", false, depth)
	return
}
func (e *CompilationEngine) compileSubroutine(depth int) {
	e.writePureTag("subroutineDec", true, depth)
	// should be method, function or constructor
	e.writeTag("keyword", e.Tokenizer.getCur(), depth+1)
	// return value
	e.Tokenizer.advance()
	if e.Tokenizer.tokenType() == TokenTypeKeyword {
		e.writeTag("keyword", e.Tokenizer.getCur(), depth+1)
	} else {
		e.writeTag("identifier", e.Tokenizer.getCur(), depth+1)
	}
	// function name
	e.Tokenizer.advance()
	e.writeTag("identifier", e.Tokenizer.getCur(), depth+1)
	// (
	e.Tokenizer.advance()
	e.writeTag("symbol", e.Tokenizer.getCur(), depth+1)

	e.Tokenizer.advance()
	// param list
	e.compileParameterList(depth + 1)
	// )
	e.writeTag("symbol", ")", depth+1)
	e.Tokenizer.advance()
	// {body}
	e.compileSubroutineBody(depth + 1)
	e.writePureTag("subroutineDec", false, depth)
}

func (e *CompilationEngine) compileParameterList(depth int) {
	e.writePureTag("parameterList", true, depth)
	for {
		cur := e.Tokenizer.getCur()
		if cur == ")" {
			break
		}
		switch e.Tokenizer.tokenType() {
		case TokenTypeIdentifier:
			e.writeTag("identifier", cur, depth+1)
		case TokenTypeKeyword:
			e.writeTag("keyword", cur, depth+1)
		case TokenTypeSymbol:
			e.writeTag("symbol", cur, depth+1)
		}
		e.Tokenizer.advance()
	}
	e.writePureTag("parameterList", false, depth)
}

func (e *CompilationEngine) compileSubroutineBody(depth int) {
	e.writePureTag("subroutineBody", true, depth)
	for {
		cur := e.Tokenizer.getCur()
		switch cur {
		case "{":
			e.writeTag("symbol", "{", depth+1)
			e.Tokenizer.advance()
		case "var":
			e.compileVarDec(depth + 1)
		case "}":
			e.writeTag("symbol", "}", depth+1)
			e.writePureTag("subroutineBody", false, depth)
			e.Tokenizer.advance()
			return
		default:
			e.compileStatements(depth + 1)
		}
	}
}

func (e *CompilationEngine) compileVarDec(depth int) {
	e.writePureTag("varDec", true, depth)
	for {
		cur := e.Tokenizer.getCur()
		if cur == ";" {
			e.writeTag("symbol", ";", depth+1)
			break
		} else {
			switch e.Tokenizer.tokenType() {
			case TokenTypeSymbol:
				e.writeTag("symbol", cur, depth+1)
			case TokenTypeIdentifier:
				e.writeTag("identifier", cur, depth+1)
			case TokenTypeKeyword:
				e.writeTag("keyword", cur, depth+1)
			}
			e.Tokenizer.advance()
		}
	}
	e.Tokenizer.advance()
	e.writePureTag("varDec", false, depth)
}

func (e *CompilationEngine) compileStatements(depth int) {
	e.writePureTag("statements", true, depth)
	for {
		cur := e.Tokenizer.getCur()
		switch cur {
		case "let":
			e.compileLet(depth + 1)
		case "do":
			e.compileDo(depth + 1)
		case "if":
			e.compileIf(depth + 1)
		case "return":
			e.compileReturn(depth + 1)
		case "while":
			e.compileWhile(depth + 1)
		default:
			// "}"
			e.writePureTag("statements", false, depth)
			return
		}
	}
}

func (e *CompilationEngine) compileLet(depth int) {
	e.writePureTag("letStatement", true, depth)
	assert(e.Tokenizer.getCur(), "let")
	e.writeTag("keyword", "let", depth+1)
	e.Tokenizer.advance()
	e.writeTag("identifier", e.Tokenizer.getCur(), depth+1)
	e.Tokenizer.advance()
	if e.Tokenizer.getCur() == "[" {
		e.writeTag("symbol", "[", depth+1)
		e.Tokenizer.advance()
		e.compileExpression(depth + 1)
		assert(e.Tokenizer.getCur(), "]")
		e.writeTag("symbol", "]", depth+1)
		e.Tokenizer.advance()
	}
	e.writeTag("symbol", "=", depth+1)
	e.Tokenizer.advance()
	e.compileExpression(depth + 1)

	assert(e.Tokenizer.getCur(), ";")
	e.writeTag("symbol", ";", depth+1)
	e.Tokenizer.advance()
	e.writePureTag("letStatement", false, depth)
}

func (e *CompilationEngine) compileIf(depth int) {
	e.writePureTag("ifStatement", true, depth)
	e.writeTag("keyword", "if", depth+1)
	e.Tokenizer.advance()
	e.writeTag("symbol", "(", depth+1)
	e.Tokenizer.advance()
	e.compileExpression(depth + 1)
	e.writeTag("symbol", ")", depth+1)
	e.Tokenizer.advance()
	e.writeTag("symbol", "{", depth+1)
	e.Tokenizer.advance()
	e.compileStatements(depth + 1)
	e.writeTag("symbol", "}", depth+1)
	e.Tokenizer.advance()
	if e.Tokenizer.getCur() == "else" {
		e.writeTag("keyword", "else", depth+1)
		e.Tokenizer.advance()
		e.writeTag("symbol", "{", depth+1)
		e.Tokenizer.advance()
		e.compileStatements(depth + 1)
		e.writeTag("symbol", "}", depth+1)
		e.Tokenizer.advance()
	}
	e.writePureTag("ifStatement", false, depth)
}

func (e *CompilationEngine) compileWhile(depth int) {
	e.writePureTag("whileStatement", true, depth)
	e.writeTag("keyword", "while", depth+1)
	e.Tokenizer.advance()
	e.writeTag("symbol", "(", depth+1)
	e.Tokenizer.advance()
	e.compileExpression(depth + 1)
	e.writeTag("symbol", ")", depth+1)
	e.Tokenizer.advance()
	e.writeTag("symbol", "{", depth+1)
	e.Tokenizer.advance()
	e.compileStatements(depth + 1)
	e.writeTag("symbol", "}", depth+1)
	e.Tokenizer.advance()
	e.writePureTag("whileStatement", false, depth)
}

func (e *CompilationEngine) compileDo(depth int) {
	e.writePureTag("doStatement", true, depth)
	e.writeTag("keyword", "do", depth+1)
	e.Tokenizer.advance()
	// subroutine call
	e.writeTag("identifier", e.Tokenizer.getCur(), depth+1)
	e.Tokenizer.advance()
	if e.Tokenizer.getCur() == "." {
		e.writeTag("symbol", ".", depth+1)
		e.Tokenizer.advance()
		e.writeTag("identifier", e.Tokenizer.getCur(), depth+1)
		e.Tokenizer.advance()
	}
	e.writeTag("symbol", "(", depth+1)
	e.Tokenizer.advance()
	e.compileExpressionList(depth + 2)
	e.writeTag("symbol", ")", depth+1)
	e.Tokenizer.advance()
	e.writeTag("symbol", ";", depth+1)
	e.Tokenizer.advance()
	e.writePureTag("doStatement", false, depth)
}
func (e *CompilationEngine) compileReturn(depth int) {
	e.writePureTag("returnStatement", true, depth)
	e.writeTag("keyword", "return", depth)
	e.Tokenizer.advance()
	if e.Tokenizer.getCur() != ";" {
		e.compileExpression(depth + 1)
	}
	e.writeTag("symbol", ";", depth+1)
	e.Tokenizer.advance()
	e.writePureTag("returnStatement", false, depth)
}

func (e *CompilationEngine) compileExpression(depth int) {
	e.writePureTag("expression", true, depth)
	e.compileTerm(depth + 1)
	for opsSet[e.Tokenizer.getCur()] {
		e.writeTag("symbol", e.Tokenizer.getCur(), depth+1)
		e.Tokenizer.advance()
		e.compileTerm(depth + 1)
	}
	e.writePureTag("expression", false, depth)
}

func (e *CompilationEngine) compileTerm(depth int) {
	e.writePureTag("term", true, depth)
	switch e.Tokenizer.tokenType() {
	case TokenTypeIntConst:
		e.writeTag("integerConstant", e.Tokenizer.getCur(), depth+1)
		e.Tokenizer.advance()
	case TokenTypeStringConst:
		cur := e.Tokenizer.getCur()
		cur = cur[1 : len(cur)-1]
		e.writeTag("stringConstant", cur, depth+1)
		e.Tokenizer.advance()
	case TokenTypeKeyword:
		e.writeTag("keyword", e.Tokenizer.getCur(), depth+1)
		e.Tokenizer.advance()
	case TokenTypeSymbol:
		if e.Tokenizer.getCur() == "(" {
			e.writeTag("symbol", "(", depth+1)
			e.Tokenizer.advance()
			e.compileExpression(depth + 1)
			e.writeTag("symbol", ")", depth+1)
			e.Tokenizer.advance()
		} else {
			e.writeTag("symbol", e.Tokenizer.getCur(), depth+1)
			e.Tokenizer.advance()
			e.compileTerm(depth + 1)
		}
	case TokenTypeIdentifier:
		cur := e.Tokenizer.getCur()
		e.Tokenizer.advance()
		ahead := e.Tokenizer.getCur()
		switch ahead {
		case "[":
			e.writeTag("identifier", cur, depth+1)
			e.writeTag("symbol", ahead, depth+1)
			e.Tokenizer.advance()
			e.compileExpression(depth + 1)
			e.writeTag("symbol", "]", depth+1)
			e.Tokenizer.advance()
		case ".", "(":
			// subroutine call
			e.writeTag("identifier", cur, depth+1)
			if ahead == "." {
				e.writeTag("symbol", ".", depth+1)
				e.Tokenizer.advance()
				e.writeTag("identifier", e.Tokenizer.getCur(), depth+1)
				e.Tokenizer.advance()
			}
			e.writeTag("symbol", "(", depth+1)
			e.Tokenizer.advance()
			e.compileExpressionList(depth + 2)
			e.writeTag("symbol", ")", depth+1)
			e.Tokenizer.advance()
		default:
			e.writeTag("identifier", cur, depth+1)
		}
	}
	e.writePureTag("term", false, depth)
}
func (e *CompilationEngine) compileExpressionList(depth int) int {
	e.writePureTag("expressionList", true, depth)
	for {
		cur := e.Tokenizer.getCur()
		if cur == ")" {
			break
		}
		if cur == "." {
			e.writeTag("symbol", ".", depth+1)
			e.Tokenizer.advance()
			continue
		}
		if cur == "," {
			e.writeTag("symbol", ",", depth+1)
			e.Tokenizer.advance()
			continue
		}
		e.compileExpression(depth + 1)
	}
	e.writePureTag("expressionList", false, depth)
	return 0
}

func (e *CompilationEngine) writeTag(tag string, content string, stackDepth ...int) {
	blank := strings.Repeat("  ", stackDepth[0])
	var line string
	if tag == "symbol" {
		line = fmt.Sprintf("<%s> %s </%s>\n", tag, modifySymbol(content), tag)
	} else {
		if tag == "identifier" && keywordSet[content] {
			tag = "keyword"
		}
		line = fmt.Sprintf("<%s> %s </%s>\n", tag, content, tag)
	}
	_, _ = e.Out.WriteString(blank)
	_, _ = e.Out.WriteString(line)
}

func (e *CompilationEngine) writePureTag(tag string, isStartTag bool, stackDepth ...int) {
	blank := strings.Repeat("  ", stackDepth[0])
	var line string
	if isStartTag {
		line = fmt.Sprintf("<%s>\n", tag)
	} else {
		line = fmt.Sprintf("</%s>\n", tag)
	}
	_, _ = e.Out.WriteString(blank)
	_, _ = e.Out.WriteString(line)
}

func modifySymbol(s string) string {
	if s == "<" {
		return "&lt;"
	}
	if s == ">" {
		return "&gt;"
	}
	if s == "&" {
		return "&amp;"
	}
	return s
}

func assert(cur string, compare string) {
	if cur == compare {
		return
	}
	panic(cur + "vs" + compare)
}
