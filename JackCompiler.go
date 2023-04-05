package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	labelCount = 0
)

func main() {
	initMaps()
	targetFiles := getFiles(os.Args[1])

	for _, targetFile := range targetFiles {
		// create new output file
		/*
			out, err := os.OpenFile(createOutput(targetFile, ".xml"), os.O_CREATE|os.O_WRONLY, os.ModePerm)
			if err != nil {
				panic(err)
			}
			tokenizer := buildTokenizer(targetFile)
			buildCompilationEngine(tokenizer, out).compileClass(0)
			out.Close()
		*/
		out, err := os.OpenFile(createVmOutput(targetFile), os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			panic(err)
		}
		tokenizer := buildTokenizer(targetFile)
		buildCompilationEngine2(tokenizer, out).compileClass()
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

func createXmlOutput(target string) string {
	return createOutput(target, ".xml")
}

func createVmOutput(target string) string {
	return createOutput(target, ".vm")
}

func createOutput(target string, extension string) string {
	d := filepath.Dir(target)

	ext := filepath.Ext(target)

	base := filepath.Base(target)

	var o string
	if isDir(target) {
		o = filepath.Join(d, base, "Main"+extension)
	} else {
		o = filepath.Join(d, strings.Replace(base, ext, extension, 1))
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
	//fmt.Println(t.curr)
}

func (t *Tokenizer) tokenType() int {
	return t.currType
}

func (t *Tokenizer) getCur() string {
	return t.curr
}

func (t *Tokenizer) advanceN(n int) {
	if n <= 0 {
		return
	}
	if n > 0 {
		t.advance()
		t.advanceN(n - 1)
	}
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

/*
Compilation Engine2
*/

type CompilationEngine2 struct {
	t                     *Tokenizer
	w                     *VMWriter
	classTable            *SymbolTable
	methodTable           *SymbolTable
	currentClassName      string
	currentSubroutineType string
	currentSubroutineName string
}

func buildCompilationEngine2(tokenizer *Tokenizer, out *os.File) *CompilationEngine2 {
	return &CompilationEngine2{
		t:           tokenizer,
		w:           buildVMWriter(out),
		classTable:  buildSymbolTable(SymbolTableClassLevel),
		methodTable: buildSymbolTable(SymbolTableSubroutineLevel),
	}
}

func (e *CompilationEngine2) compileClass() {
	e.classTable.reset()
	e.t.advanceN(2) // class, class name
	e.currentClassName = e.t.getCur()
	e.t.advanceN(2) // {, nextToken
	for e.t.hasMoreTokens() {
		cur := e.t.getCur()
		switch cur {
		case "static", "field":
			e.compileClassVarDec()
		case "method", "function", "constructor":
			e.compileSubroutine()
		case "}":
			return
		// ending the class, is there anything to do ?
		default:
			panic("should not be anything else")
		}
	}
}

func (e *CompilationEngine2) compileClassVarDec() {
	// cur is field or static
	segmentKind := e.t.getCur()
	e.t.advance()
	thisType := e.t.getCur()
	e.t.advance()
	varName := e.t.getCur()
	e.t.advance()
	e.classTable.define(varName, thisType, kind(segmentKind))
	for e.t.getCur() != ";" {
		e.t.advance()
		e.classTable.define(e.t.getCur(), thisType, kind(segmentKind))
		e.t.advance()
	}
	e.t.advance()
}

func (e *CompilationEngine2) compileSubroutine() {
	e.methodTable.reset()
	// (function | method | constructor)
	e.currentSubroutineType = e.t.getCur()
	e.t.advance()
	// (return type | void)
	e.t.advance()

	e.currentSubroutineName = e.t.getCur()
	e.t.advance() // `(`
	e.t.advance()
	e.compileParameterList() // updating symbol table
	e.t.advanceN(2)          // skip ) {
	e.compileSubroutineBody()
	e.t.advance() // skip }
}

// update the subroutine level symbol table
func (e *CompilationEngine2) compileParameterList() {
	if e.currentSubroutineName == "method" {
		e.methodTable.define("this", e.currentClassName, SegKindArg)
	}
	if e.t.getCur() == ")" {
		return
	}

	for {
		paramType := e.t.getCur() // param type
		e.t.advance()             // skip param type
		paramName := e.t.getCur()
		e.t.advance() // skip param name
		// fill symbol table
		e.methodTable.define(paramName, paramType, SegKindArg)
		if e.t.getCur() == ")" {
			return
		} else {
			assert(e.t.getCur(), ",")
			e.t.advance()
		}
	}
}

func (e *CompilationEngine2) compileSubroutineBody() {
	for e.t.getCur() == "var" {
		e.compileVarDec()
	}
	// write function according to var number
	localCount := e.methodTable.varCount(SegKindVar)
	funcFullName := fmt.Sprintf("%s.%s", e.currentClassName, e.currentSubroutineName)
	e.w.writeFunction(funcFullName, localCount)

	switch e.currentSubroutineType {
	case "function":
		// do nothing
	case "method":
		// push argument 0
		e.w.writePush(SegmentArgument, 0)
		// pop pointer 0
		e.w.writePop(SegmentPointer, 0)
	case "constructor":
		// push constant {argCount}
		e.w.writePush(SegmentConstant, e.methodTable.varCount(SegKindArg))
		// call memory to alloc
		e.w.writeCall("Memory.alloc", 1)
		// pop pointer 0
		e.w.writePop(SegmentPointer, 0)
	}

	// start to process statements inside a function
	e.compileStatements()
}

// fill the local segment of subroutine symbol table
func (e *CompilationEngine2) compileVarDec() {
	// skip "var"
	e.t.advance()
	// type of the var
	localType := e.t.getCur()
	e.t.advance()
	// name of the var
	e.methodTable.define(e.t.getCur(), localType, SegKindVar)
	e.t.advance()

	for e.t.getCur() != ";" {
		// skip ,
		e.t.advance()
		e.methodTable.define(e.t.getCur(), localType, SegKindVar)
		e.t.advance()
	}
	e.t.advance()
}

func (e *CompilationEngine2) compileStatements() {
	for {
		switch e.t.getCur() {
		case "let":
			e.compileLet()
		case "if":
			e.compileIf()
		case "while":
			e.compileWhile()
		case "do":
			e.compileDo()
		case "return":
			e.compileReturn()
		case "}":
			return
		}
	}
}

func (e *CompilationEngine2) compileLet() {
	// skip let
	e.t.advance()

	cur := e.t.getCur()
	e.t.advance()
	if e.t.getCur() == "[" {
		e.t.advance()
		e.pushIdentifier(cur)
		e.compileExpression()
		e.w.writeArithmetic(CommandAdd)
		e.t.advance() // skip ]
		e.t.advance() // skip =
		e.compileExpression()
		e.w.writePop(SegmentTemp, 0)
		e.w.writePop(SegmentPointer, 1)
		e.w.writePush(SegmentTemp, 0)
		e.w.writePop(SegmentThat, 0)
	} else {
		assert(e.t.getCur(), "=")
		e.t.advance() // skip =
		e.compileExpression()
		e.popIdentifier(cur)
	}
	// skip ;
	assert(e.t.getCur(), ";")
	e.t.advance()
}

func (e *CompilationEngine2) compileIf() {
	c := getCurrLabelCount()

	// skip if, (
	e.t.advanceN(2)

	e.compileExpression()
	e.w.writeArithmetic(CommandNot)

	e.w.writeIf("IF_FALSE" + c)
	e.t.advance() // skip )
	e.t.advance() // skip {
	e.compileStatements()
	e.w.writeGoto("OUT" + c)

	// skip }
	e.t.advance()
	if e.t.getCur() == "else" {
		e.t.advanceN(2) // skip else {
		e.compileStatements()
		e.t.advance() // skip }
	}
	e.w.writeLabel("IF_FALSE" + c)
	e.w.writeLabel("OUT" + c)
}

func (e *CompilationEngine2) compileWhile() {
	c := getCurrLabelCount()

	// label L1
	e.w.writeLabel("WHILE" + c)
	e.t.advance() // skip while
	e.t.advance() // skip (
	e.compileExpression()
	e.w.writeArithmetic(CommandNot)
	e.w.writeIf("OUT" + c)

	e.t.advanceN(2) // skip ) {

	e.compileStatements()
	e.w.writeGoto("WHILE" + c)

	e.w.writeLabel("OUT" + c)

	e.t.advance() // skip }
}

func (e *CompilationEngine2) compileDo() {
	e.t.advance() // skip do
	functionFullName := ""
	for e.t.getCur() != "(" {
		functionFullName += e.t.getCur()
		e.t.advance()
	}
	e.t.advance() // skip (
	paramCount := e.compileExpressionList()
	e.w.writeCall(functionFullName, paramCount)
	e.w.writePop(SegmentTemp, 0)
	e.t.advance() // skip ;
}

func (e *CompilationEngine2) compileReturn() {
	e.t.advance() // skip return
	if e.t.getCur() == ";" {
		// return ;
		e.w.writePush(SegmentConstant, 0)
		e.w.writeReturn()
		e.t.advance() // skip ;
		return
	}
	e.compileExpression()
	e.w.writeReturn()
	e.t.advance() // skip ;
}

func (e *CompilationEngine2) compileExpression() bool {
	isArrayExpression := false
	isArrayExpression = isArrayExpression || e.compileTerm()
	for opsSet[e.t.getCur()] {
		op := e.t.getCur()
		e.t.advance()
		e.compileTerm()
		switch op {
		case "+":
			e.w.writeArithmetic(CommandAdd)
		case "-":
			e.w.writeArithmetic(CommandSub)
		case "*":
			e.w.writeArithmetic("call Math.multiply 2")
		case "/":
			e.w.writeArithmetic("call Math.divide 2")
		case "<":
			e.w.writeArithmetic(CommandLt)
		case ">":
			e.w.writeArithmetic(CommandGt)
		case "=":
			e.w.writeArithmetic(CommandEq)
		case "&":
			e.w.writeArithmetic(CommandAnd)
		case "|":
			e.w.writeArithmetic(CommandOr)
		default:
			panic("does not support " + op)
		}
		isArrayExpression = false
	}
	return isArrayExpression
}

func (e *CompilationEngine2) compileTerm() bool {
	switch e.t.tokenType() {
	case TokenTypeIntConst:
		num, _ := strconv.ParseInt(e.t.getCur(), 10, 64)
		e.w.writePush(SegmentConstant, int(num))
		e.t.advance()
	case TokenTypeKeyword:
		switch e.t.getCur() {
		case "null", "false":
			e.w.writePush(SegmentConstant, 0)
		case "true":
			e.w.writePush(SegmentConstant, 1)
			e.w.writeArithmetic(CommandNot)
		case "this":
			e.pushIdentifier(e.t.getCur())
		}
		e.t.advance()
	case TokenTypeSymbol:
		if e.t.getCur() == "(" {
			e.t.advance() // skip (
			e.compileExpression()
			e.t.advance() // skip )
			break
		} else {
			// unaryOp
			op := e.t.getCur()
			e.t.advance()
			e.compileTerm()
			switch op {
			case "-":
				e.w.writeArithmetic(CommandNeg)
			case "~":
				e.w.writeArithmetic(CommandNot)
			default:
				panic("not supported unaryOp: " + op)
			}
		}
	case TokenTypeStringConst:
		cur := e.t.getCur()
		// should allocate memory for the string
		e.w.writePush(SegmentConstant, len(cur)-2) // minus the length of ""
		e.w.writeCall("String.new", 1)
		for i := 1; i < len(cur)-1; i++ {
			char := cur[i]
			e.w.writePush(SegmentConstant, int(char))
			e.w.writeCall("String.appendChar", 2)
		}
		e.t.advance()
	case TokenTypeIdentifier:
		cur := e.t.getCur()
		e.t.advance()
		ahead := e.t.getCur()
		switch ahead {
		case "[":
			// that is to say `cur` is an arr
			e.t.advance() // skip [
			e.compileExpression()
			assert(e.t.getCur(), "]")
			e.t.advance() // skip ]
			// cur is an array
			// find in class symbol table, then find in method symbol table
			e.pushIdentifier(cur)
			e.w.writeArithmetic(CommandAdd)
			e.w.writePop(SegmentPointer, 1)
			e.w.writePush(SegmentThat, 0)
			//e.w.writeArithmetic(CommandAdd)
			return true
		case ".", "(":
			// subroutine call
			funcFullName := cur
			if ahead == "." {
				funcFullName += "."
				e.t.advance()
				funcFullName += e.t.getCur()
				e.t.advance()
			}
			e.t.advance() // skip (
			paramsCount := e.compileExpressionList()
			e.w.writeCall(funcFullName, paramsCount)
		default:
			e.pushIdentifier(cur)
			// do nothing
		}
	}
	return false
}

func (e *CompilationEngine2) compileExpressionList() int {
	expCount := 0
	for {
		cur := e.t.getCur()
		switch cur {
		case ",":
			e.t.advance()
		case ")":
			e.t.advance()
			return expCount
		default:
			expCount += 1
			e.compileExpression()
		}
	}
}

func (e *CompilationEngine2) dealWithIdentifier(cur string, f func(segment Segment, int2 int)) {
	if e.classTable.indexOf(cur) >= 0 {
		kind := e.classTable.kindOf(cur)
		switch kind {
		case SegKindField:
			f(SegmentThis, e.classTable.indexOf(cur))
		case SegKindStatic:
			f(SegmentStatic, e.classTable.indexOf(cur))
		}
	} else if e.methodTable.indexOf(cur) >= 0 {
		kind := e.methodTable.kindOf(cur)
		switch kind {
		case SegKindArg:
			f(SegmentArgument, e.methodTable.indexOf(cur))
		case SegKindVar:
			f(SegmentLocal, e.methodTable.indexOf(cur))
		}
	}
}

func (e *CompilationEngine2) pushIdentifier(cur string) {
	e.dealWithIdentifier(cur, e.w.writePush)
}

func (e *CompilationEngine2) popIdentifier(cur string) {
	e.dealWithIdentifier(cur, e.w.writePop)
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

/*
	Symbol Table
*/

type Level int
type kind string

const (
	SymbolTableClassLevel Level = iota
	SymbolTableSubroutineLevel
)

const (
	SegKindStatic kind = "static"
	SegKindField       = "field"
	SegKindArg         = "arg"
	SegKindVar         = "var"
)

type SymbolTable struct {
	Level     Level // class level or subroutine level
	Symbols   map[string]Symbol
	kindCount map[kind]int
}

type Symbol struct {
	symbolName string
	typeName   string
	kind       kind
	seriesNum  int
}

func buildSymbolTable(level Level) *SymbolTable {
	s := &SymbolTable{}
	s.Level = level
	s.Symbols = make(map[string]Symbol, 0)
	s.kindCount = make(map[kind]int, 0)
	return s
}

func (s *SymbolTable) reset() {
	s.Symbols = make(map[string]Symbol, 0)
	s.kindCount = make(map[kind]int, 0)
}

func (s *SymbolTable) define(name string, typeName string, kind kind) {
	kindCurNum, ok := s.kindCount[kind]
	if !ok {
		kindCurNum = 0 // first should start with 0
	}
	symbol := Symbol{
		symbolName: name,
		typeName:   typeName,
		kind:       kind,
		seriesNum:  kindCurNum,
	}
	s.Symbols[symbol.symbolName] = symbol
	s.kindCount[kind] += 1
}

func (s *SymbolTable) varCount(kind kind) int {
	return s.kindCount[kind]
}

func (s *SymbolTable) kindOf(name string) kind {
	return s.Symbols[name].kind
}

func (s *SymbolTable) typeOf(name string) string {
	return s.Symbols[name].typeName
}

func (s *SymbolTable) indexOf(name string) int {
	if _, ok := s.Symbols[name]; !ok {
		return -1
	}
	return s.Symbols[name].seriesNum
}

/*
VMWriter
*/

type Segment string
type Command string

const (
	SegmentConstant Segment = "constant"
	SegmentArgument         = "argument"
	SegmentLocal            = "local"
	SegmentStatic           = "static"
	SegmentThis             = "this"
	SegmentThat             = "that"
	SegmentPointer          = "pointer"
	SegmentTemp             = "temp"
)

const (
	CommandAdd Command = "add"
	CommandSub         = "sub"
	CommandNeg         = "neg"
	CommandEq          = "eq"
	CommandGt          = "gt"
	CommandLt          = "lt"
	CommandAnd         = "and"
	CommandOr          = "or"
	CommandNot         = "not"
)

type VMWriter struct {
	f *os.File
}

func buildVMWriter(f *os.File) *VMWriter {
	return &VMWriter{f: f}
}

func (w *VMWriter) writePush(segment Segment, index int) {
	_, _ = w.f.WriteString(fmt.Sprintf("push %s %d\n", strings.ToLower(string(segment)), index))
}

func (w *VMWriter) writePop(segment Segment, index int) {
	_, _ = w.f.WriteString(fmt.Sprintf("pop %s %d\n", strings.ToLower(string(segment)), index))
}

func (w *VMWriter) writeArithmetic(command Command) {
	_, _ = w.f.WriteString(string(command) + "\n")
}

func (w *VMWriter) writeLabel(label string) {
	// label
	_, _ = w.f.WriteString(fmt.Sprintf("label %s\n", label))
}

func (w *VMWriter) writeGoto(label string) {
	// goto
	_, _ = w.f.WriteString(fmt.Sprintf("goto %s\n", label))
}

func (w *VMWriter) writeIf(label string) {
	// if-goto
	_, _ = w.f.WriteString(fmt.Sprintf("if-goto %s\n", label))
}

func (w *VMWriter) writeCall(name string, nArgs int) {
	// call
	_, _ = w.f.WriteString(fmt.Sprintf("call %s %d\n", name, nArgs))
}

func (w *VMWriter) writeFunction(name string, nVars int) {
	// function command
	_, _ = w.f.WriteString(fmt.Sprintf("function %s %d\n", name, nVars))
}

func (w *VMWriter) writeReturn() {
	// return
	_, _ = w.f.WriteString("return\n")
}

func (w *VMWriter) close() {
	err := w.f.Close()
	if err != nil {
		panic(err)
	}
}

/*
	Helper Functions
*/

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

func getCurrLabelCount() string {
	c := strconv.Itoa(labelCount)
	labelCount += 1
	return c
}
