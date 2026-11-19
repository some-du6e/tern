package core

import (
	"path"
	"regexp"
	"strings"
	"unicode"
)

// Command labels, from T3's work-log command-label module
// (packages/client-runtime/src/work-log/commandLabel.ts): what a command
// row says, and the program a live "Running …" row names. T3 parses the
// shell script without running it; this keeps its tokenizer and its rules
// for the common shapes (shell wrappers, env and sudo, setup commands
// before &&), not its PowerShell and heredoc handling.

const maxCommandSegments = 64

var (
	shellPrograms          = set("sh", "bash", "zsh", "dash", "ash", "ksh", "fish")
	shellOptionsWithValue  = set("-o", "-O", "--rcfile", "--init-file")
	shellCommandWrappers   = set("builtin", "command", "exec")
	shellPrecommandModes   = set("nocorrect", "noglob", "time")
	shellSetupPrograms     = set("cd", "source", ".", "unset", "export", "pushd", "popd", "set", "shopt", "alias")
	shellSyntaxWords       = set("if", "then", "else", "elif", "fi", "for", "while", "until", "do", "done", "case", "esac", "select", "function", "repeat", "return", "true", "false", "coproc", "{", "}", "[", "[[", "test", "((", "!")
	wrapperOptionsWithArgs = map[string]map[string]bool{
		"env":  set("-C", "--chdir", "-S", "--split-string", "-u", "--unset"),
		"sudo": set("-C", "--close-from", "-D", "--chdir", "-g", "--group", "-u", "--user"),
	}
	scriptOption = regexp.MustCompile(`^-[a-zA-Z]*c[a-zA-Z]*$`)
	assignment   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\+?=`)
	controlChars = regexp.MustCompile("[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]")
)

func set(xs ...string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// commandDisplayText is a command as its row shows it: without the shell
// wrapper around a lone script, control characters made visible.
func commandDisplayText(command string) string {
	return visibleControls(withoutShellWrapper(command))
}

// visibleControls shows escape bytes and other control characters as
// Unicode control pictures (ESC becomes ␛), as T3 does; tab and newlines
// stay whitespace.
func visibleControls(s string) string {
	return controlChars.ReplaceAllStringFunc(s, func(c string) string {
		if c == "\x7f" {
			return "␡"
		}
		return string(rune(0x2400 + int(c[0])))
	})
}

// withoutShellWrapper is the script of `bash -lc '<script>'`, when that is
// all the command is.
func withoutShellWrapper(command string) string {
	trimmed := strings.TrimSpace(command)
	if _, rest := splitFirstCommand(trimmed); rest != "" {
		return trimmed
	}
	tokens, ok := tokenizeShell(trimmed)
	if !ok || len(tokens) == 0 || !shellPrograms[programBase(tokens[0])] {
		return trimmed
	}
	i := scriptArgument(tokens, 1)
	// Positional arguments can change what the script does: keep those.
	if i < 0 || i != len(tokens)-1 {
		return trimmed
	}
	if s := strings.TrimSpace(tokens[i]); s != "" {
		return s
	}
	return trimmed
}

// commandProgramName is the program a command runs, for "Running git": ""
// when it can't tell, like T3's null.
func commandProgramName(command string) string {
	return programName(command, 0, maxCommandSegments)
}

func programName(command string, depth, segments int) string {
	if depth > 8 {
		return ""
	}
	rest := strings.TrimSpace(command)
	for n := 0; rest != "" && n < segments; n++ {
		first, next := splitFirstCommand(rest)
		name, setup := segmentProgram(first, depth, segments-n)
		if !setup {
			return name
		}
		rest = next
	}
	return ""
}

// segmentProgram is one simple command's program. setup reports a setup
// command (cd, source, export, a bare assignment), after which the next
// command names the program.
func segmentProgram(segment string, depth, segments int) (name string, setup bool) {
	lines := strings.Split(segment, "\n")
	for len(lines) > 0 && (strings.HasPrefix(strings.TrimSpace(lines[0]), "#") || strings.TrimSpace(lines[0]) == "") {
		lines = lines[1:]
	}
	tokens, ok := tokenizeShell(strings.Join(lines, "\n"))
	if !ok || len(tokens) == 0 {
		return "", len(lines) == 0
	}
	i := skipPrefixes(tokens, 0)
	if i == len(tokens) {
		return "", true
	}
	wrapped := false
	for i < len(tokens) {
		word := tokens[i]
		switch base := programBase(word); {
		case shellPrecommandModes[word]:
			i++
		case base == "env" || base == "sudo":
			i, wrapped = wrappedCommand(tokens, i+1, base), true
		case base == "timeout":
			i, wrapped = i+2, true
			for i < len(tokens) && strings.HasPrefix(tokens[i], "-") {
				i++
			}
		case shellCommandWrappers[word]:
			if i+1 < len(tokens) && (strings.HasPrefix(tokens[i+1], "-") || redirection.MatchString(tokens[i+1])) {
				return "", false
			}
			i, wrapped = i+1, true
		case shellSetupPrograms[word]:
			// A setup command under a wrapper runs nothing useful.
			return "", !wrapped
		case shellSyntaxWords[word] || strings.HasPrefix(word, "&") || strings.HasPrefix(word, "(") || strings.HasSuffix(word, "()"):
			return "", false
		case strings.HasPrefix(word, "$"):
			// $HOME/.bun/bin/bun names its program; $PY doesn't.
			if j := strings.LastIndexByte(word, '/'); j >= 0 && j < len(word)-1 && !strings.ContainsRune(word[j:], '$') {
				return word[j+1:], false
			}
			return "", false
		case shellPrograms[base]:
			s := scriptArgument(tokens, i+1)
			switch {
			case s < 0:
				return base, false
			case s >= len(tokens):
				return "", false
			}
			return programName(tokens[s], depth+1, segments), false
		default:
			return base, false
		}
	}
	return "", false
}

// redirection is a redirection's operator: >file, 2>&1, {fd}>file.
var redirection = regexp.MustCompile(`^(?:\d*|\{\w+\})(?:[<>]+&?|&>)`)

// skipPrefixes passes the assignments and redirections before a command's
// program.
func skipPrefixes(tokens []string, i int) int {
	for i < len(tokens) {
		switch t := tokens[i]; {
		case assignment.MatchString(t):
			i++
		case redirection.MatchString(t):
			// An operator on its own takes the next word as its target.
			if loc := redirection.FindStringIndex(t); loc[1] == len(t) {
				i++
			}
			i++
		default:
			return i
		}
	}
	return i
}

// wrappedCommand skips env's or sudo's own options and assignments to the
// command they run.
func wrappedCommand(tokens []string, i int, wrapper string) int {
	for i < len(tokens) {
		t := tokens[i]
		switch {
		case wrapperOptionsWithArgs[wrapper][t]:
			i += 2
		case strings.HasPrefix(t, "-") || (wrapper == "env" && assignment.MatchString(t)):
			i++
		default:
			return i
		}
	}
	return i
}

// programBase is a program's name without its folder or .exe.
func programBase(word string) string {
	word = strings.ReplaceAll(word, `\`, "/")
	return strings.TrimSuffix(path.Base(word), ".exe")
}

// scriptArgument is the index of the script a shell's -c runs, or -1.
func scriptArgument(tokens []string, start int) int {
	for i := start; i < len(tokens); i++ {
		opt := tokens[i]
		if opt == "--" || !strings.HasPrefix(opt, "-") {
			return -1
		}
		if shellOptionsWithValue[opt] {
			i++
			continue
		}
		if opt == "--command" || scriptOption.MatchString(opt) {
			return i + 1
		}
	}
	return -1
}

// splitFirstCommand cuts a script after its first top-level command, at
// &&, ||, ;, | or a newline outside quotes and substitutions.
func splitFirstCommand(s string) (first, rest string) {
	var quote rune
	escaping, depth := false, 0
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		switch {
		case escaping:
			escaping = false
		case c == '\\' && quote != '\'':
			escaping = true
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'' || c == '`':
			quote = c
		case c == '(' || c == '{' && i > 0 && rs[i-1] == '$':
			depth++
		case (c == ')' || c == '}') && depth > 0:
			depth--
		case depth > 0:
		case c == '#' && (i == 0 || unicode.IsSpace(rs[i-1])):
			for i < len(rs) && rs[i] != '\n' {
				i++
			}
			if i < len(rs) {
				if f := strings.TrimSpace(string(rs[:i])); f != "" && !strings.HasPrefix(f, "#") {
					return string(rs[:i]), strings.TrimSpace(string(rs[i+1:]))
				}
			}
		case c == ';' || c == '\n' || c == '|' || c == '&':
			n := 1
			if i+1 < len(rs) && (rs[i+1] == c && c != ';' && c != '\n' || c == '|' && rs[i+1] == '&') {
				n = 2
			}
			if c == '&' && i > 0 && (rs[i-1] == '>' || rs[i-1] == '<') {
				continue
			}
			if f := strings.TrimSpace(string(rs[:i])); f != "" && !strings.HasPrefix(f, "#") {
				return string(rs[:i]), strings.TrimSpace(string(rs[i+n:]))
			}
		}
	}
	return s, ""
}

// tokenizeShell is T3's tokenizeShellCommand: words with their quotes
// removed, keeping substitutions and expansions whole. ok is false for an
// unbalanced quote or substitution.
func tokenizeShell(command string) (tokens []string, ok bool) {
	in := []rune(strings.TrimSpace(command))
	var cur strings.Builder
	var quote rune
	escaping, backticks, started := false, false, false
	subst, param := 0, 0
	for i := 0; i < len(in); i++ {
		c := in[i]
		switch {
		case escaping:
			cur.WriteRune(c)
			escaping, started = false, true
		case c == '\\' && quote != '\'':
			if quote == '"' && i+1 < len(in) && !strings.ContainsRune("\"\\$`\n", in[i+1]) {
				cur.WriteRune(c)
			} else {
				escaping = true
			}
			started = true
		case backticks:
			cur.WriteRune(c)
			backticks = c != '`'
		case quote != 0:
			if c == quote {
				quote = 0
			} else {
				cur.WriteRune(c)
			}
		case c == '`':
			cur.WriteRune(c)
			backticks, started = true, true
		case c == '$' && i+1 < len(in) && in[i+1] == '{':
			cur.WriteString("${")
			param++
			started = true
			i++
		case c == '{' && param > 0:
			cur.WriteRune(c)
			param++
		case c == '}' && param > 0:
			cur.WriteRune(c)
			param--
		case c == '$' && i+1 < len(in) && in[i+1] == '(':
			cur.WriteString("$(")
			subst++
			started = true
			i++
		case c == '(':
			cur.WriteRune(c)
			subst++
			started = true
		case c == ')' && subst > 0:
			cur.WriteRune(c)
			subst--
		case c == '"' || c == '\'':
			quote, started = c, true
		case unicode.IsSpace(c):
			if subst > 0 || param > 0 {
				cur.WriteRune(c)
				continue
			}
			if started {
				tokens = append(tokens, cur.String())
				cur.Reset()
				started = false
			}
		default:
			cur.WriteRune(c)
			started = true
		}
	}
	if quote != 0 || escaping || backticks || subst > 0 || param > 0 {
		return nil, false
	}
	if started {
		tokens = append(tokens, cur.String())
	}
	return tokens, true
}
