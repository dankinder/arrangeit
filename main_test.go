package main

//type Test struct {
//	Args        []string
//	Input       string
//	ExpOutput   string
//	ExpErrout   string
//	ExpExitCode int
//}
//
//func RunTest(t *testing.T, test Test) {
//	origArgs := os.Args
//	origStdin := stdin
//	origStdout := stdout
//	origStderr := stderr
//	origExitFunc := exitFunc
//	defer func() {
//		os.Args = origArgs
//		stdin = origStdin
//		stdout = origStdout
//		stderr = origStderr
//		exitFunc = origExitFunc
//	}()
//	os.Args = append([]string{os.Args[0]}, test.Args...)
//	stdin = strings.NewReader(test.Input)
//	stdout = &strings.Builder{}
//	stderr = &strings.Builder{}
//	var exitCode int
//	exitFunc = func(code int) {
//		exitCode = code
//	}
//
//	// Unfortunately the flag library may leave these set to a previous value, so we need to zero them out
//	separator = ""
//	useTabSeparator = false
//	ignoreHeaderRow = false
//
//	main()
//	assert.Equal(t, test.ExpOutput, stdout.(*strings.Builder).String())
//	assert.Equal(t, test.ExpErrout, stderr.(*strings.Builder).String())
//	assert.Equal(t, test.ExpExitCode, exitCode)
//}
//
//func TestBasic(t *testing.T) {
//	RunTest(t, Test{Args: []string{"1"}, Input: "a b c\nd e f\n", ExpOutput: "a\nd\n"})
//}
//
//func TestBasicNoNewline(t *testing.T) {
//	RunTest(t, Test{Args: []string{"1"}, Input: "a b c\nd e f", ExpOutput: "a\nd\n"})
//}
//
//func TestBasic2Columns(t *testing.T) {
//	RunTest(t, Test{Args: []string{"1", "3"}, Input: "a b c d\nd e f g", ExpOutput: "a\tc\nd\tf\n"})
//}
//
//func TestExtraColumns(t *testing.T) {
//	RunTest(t, Test{Args: []string{"1", "2", "3"}, Input: "a b\nd e", ExpOutput: "a\tb\nd\te\n"})
//}
//
//func TestTabSpacesMix(t *testing.T) {
//	RunTest(t, Test{Args: []string{"1"}, Input: "a b	c\nd e	f", ExpOutput: "a\nd\n"})
//}
//
//func TestInvalidColumn(t *testing.T) {
//	RunTest(t, Test{Args: []string{"nan"}, Input: "a\t b \n e\tf",
//		ExpErrout:   "ERROR: failed to parse argument \"nan\": strconv.ParseInt: parsing \"nan\": invalid syntax\n",
//		ExpExitCode: 1})
//}
//
//func TestTabSeparatorBasic(t *testing.T) {
//	RunTest(t, Test{Args: []string{"-t", "2"}, Input: "a\t b \n e\tf", ExpOutput: " b \nf\n"})
//}
//
//func TestIgnoreHeaderRow(t *testing.T) {
//	RunTest(t, Test{Args: []string{"-i", "2"}, Input: "a\tb\ne\tf", ExpOutput: "f\n"})
//}
