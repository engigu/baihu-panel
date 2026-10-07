package completion

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/engigu/baihu-panel/internal/constant"
)

// Run 处理 baihu completion 命令
func Run(args []string) {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		printCompletionHelp()
		return
	}

	shell := strings.ToLower(args[0])
	var script string
	var err error

	switch shell {
	case "powershell", "pwsh":
		script, err = renderTemplate(PowerShellTmpl, constant.Commands)
	case "bash":
		script, err = renderTemplate(BashTmpl, constant.Commands)
	case "zsh":
		script, err = renderTemplate(ZshTmpl, constant.Commands)
	default:
		fmt.Fprintf(os.Stderr, "不支持的 Shell 类型: %s。可选类型: powershell, bash, zsh\n\n", shell)
		printCompletionHelp()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "生成补全脚本失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(script)
}

func printCompletionHelp() {
	fmt.Println("用法:")
	fmt.Println("  baihu completion <powershell|bash|zsh>")
	fmt.Println()
	fmt.Println("功能:")
	fmt.Println("  生成 baihu 命令行工具在当前 Shell 下的 Tab 自动补全脚本。")
	fmt.Println()
	fmt.Println("快速启用方法:")
	fmt.Println("  1. PowerShell (Windows):")
	fmt.Println("     # 临时生效 (当前会话):")
	fmt.Println("     baihu completion powershell | Out-String | Invoke-Expression")
	fmt.Println()
	fmt.Println("     # 永久生效 (追加至 PROFILE 文件):")
	fmt.Println("     if (!(Test-Path $PROFILE)) { New-Item -Type File -Path $PROFILE -Force }")
	fmt.Println("     baihu completion powershell | Out-File -Append -Encoding utf8 $PROFILE")
	fmt.Println()
	fmt.Println("  2. Bash (Linux / macOS):")
	fmt.Println("     # 临时生效:")
	fmt.Println("     source <(baihu completion bash)")
	fmt.Println()
	fmt.Println("     # 永久生效:")
	fmt.Println("     baihu completion bash > ~/.baihu_completion.bash")
	fmt.Println("     echo 'source ~/.baihu_completion.bash' >> ~/.bashrc")
	fmt.Println()
	fmt.Println("  3. Zsh (Linux / macOS):")
	fmt.Println("     # 永久生效:")
	fmt.Println("     baihu completion zsh > ~/.baihu_completion.zsh")
	fmt.Println("     echo 'source ~/.baihu_completion.zsh' >> ~/.zshrc")
	fmt.Println()
}

func renderTemplate(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("completion").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ============================================================================
// Shell 模板库 (使用 Go 标准库 text/template 解析，清晰直观且易于重构)
// ============================================================================

const PowerShellTmpl = `# baihu PowerShell Completion Script (Auto-generated)
$baihuCompleter = {
    param($wordToComplete, $commandAst, $cursorPosition)

    $topCommands = @{
{{- range . }}
        '{{ .Name }}' = '{{ .Description }}'
{{- end }}
    }

    $elements = $commandAst.CommandElements
    $argsList = [System.Collections.Generic.List[string]]::new()
    for ($i = 1; $i -lt $elements.Count; $i++) {
        $text = $elements[$i].Extent.Text
        if ($i -eq ($elements.Count - 1) -and $wordToComplete -ne '' -and $text -eq $wordToComplete) {
            break
        }
        $argsList.Add($text)
    }

    # 1. 补全顶级命令 (例如 baihu <Tab> 或 baihu ag<Tab>)
    if ($argsList.Count -eq 0) {
        $topCommands.GetEnumerator() | Where-Object { $_.Key -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_.Key, $_.Key, 'Command', $_.Value)
        }
        return
    }

    $cmd = $argsList[0]

{{- range . }}
    if ($cmd -eq '{{ .Name }}') {
{{- if .SubCommands }}
        # 补全二级子命令 (例如 baihu task <Tab>)
        if ($argsList.Count -eq 1) {
            $subcmds = @{
{{- range $k, $v := .SubCommands }}
                '{{ $k }}' = '{{ $v }}'
{{- end }}
            }
            $subcmds.GetEnumerator() | Where-Object { $_.Key -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_.Key, $_.Key, 'ParameterValue', $_.Value)
            }
            return
        }
{{- if .SubFlags }}
        $sub = $argsList[1]
{{- range $subk, $subflags := .SubFlags }}
        if ($sub -eq '{{ $subk }}') {
            $subFlagsList = @({{ range $i, $f := $subflags }}{{ if $i }}, {{ end }}'{{ $f }}'{{ end }})
            $subFlagsList | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
            }
            return
        }
{{- end }}
{{- end }}
{{- end }}
{{- if .Args }}
        if ($argsList.Count -eq 1) {
            $arguments = @({{ range $i, $arg := .Args }}{{ if $i }}, {{ end }}'{{ $arg }}'{{ end }})
            $arguments | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
            return
        }
{{- end }}
{{- if .Flags }}
        $flags = @({{ range $i, $flag := .Flags }}{{ if $i }}, {{ end }}'{{ $flag }}'{{ end }})
        $flags | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
        }
        return
{{- end }}
    }
{{- end }}
}

@('baihu', 'baihu.exe') | ForEach-Object {
    Register-ArgumentCompleter -Native -CommandName $_ -ScriptBlock $baihuCompleter
}
`

const BashTmpl = `# baihu Bash Completion Script (Auto-generated)
_baihu_completions() {
    local cur prev cmd subcmd
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    cmd="${COMP_WORDS[1]}"
    subcmd="${COMP_WORDS[2]}"

    local commands="{{ range $i, $cmd := . }}{{ if $i }} {{ end }}{{ $cmd.Name }}{{ end }}"

    # 1. 补全顶级命令
    if [ $COMP_CWORD -eq 1 ]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi

    # 2. 补全二级命令与选项
    case "${cmd}" in
{{- range . }}
        {{ .Name }})
{{- if .SubCommands }}
            if [ $COMP_CWORD -eq 2 ]; then
                COMPREPLY=( $(compgen -W "{{ range $k, $v := .SubCommands }}{{ $k }} {{ end }}" -- ${cur}) )
                return 0
            fi
{{- if .SubFlags }}
            case "${subcmd}" in
{{- range $subk, $subflags := .SubFlags }}
                {{ $subk }})
                    COMPREPLY=( $(compgen -W "{{ range $i, $f := $subflags }}{{ if $i }} {{ end }}{{ $f }}{{ end }}" -- ${cur}) )
                    return 0
                    ;;
{{- end }}
            esac
{{- end }}
{{- end }}
{{- if .Args }}
            if [ $COMP_CWORD -eq 2 ]; then
                COMPREPLY=( $(compgen -W "{{ range $i, $arg := .Args }}{{ if $i }} {{ end }}{{ $arg }}{{ end }}" -- ${cur}) )
                return 0
            fi
{{- end }}
{{- if .Flags }}
            COMPREPLY=( $(compgen -W "{{ range $i, $flag := .Flags }}{{ if $i }} {{ end }}{{ $flag }}{{ end }}" -- ${cur}) )
            return 0
{{- end }}
            ;;
{{- end }}
    esac
}
complete -F _baihu_completions baihu
`

const ZshTmpl = `#compdef baihu

# baihu Zsh Completion Script (Auto-generated)
_baihu() {
    local -a commands
    commands=(
{{- range . }}
        '{{ .Name }}:{{ .Description }}'
{{- end }}
    )

    _arguments -C \
        '1: :->command' \
        '2: :->subcommand' \
        '*:: :->args'

    case $state in
        command)
            _describe -t commands 'baihu command' commands
            ;;
        subcommand)
            case $words[2] in
{{- range . }}
{{- if .SubCommands }}
                {{ .Name }})
                    local -a subcmds
                    subcmds=(
{{- range $k, $v := .SubCommands }}
                        '{{ $k }}:{{ $v }}'
{{- end }}
                    )
                    _describe -t subcmds '{{ .Name }} subcommands' subcmds
                    ;;
{{- end }}
{{- if .Args }}
                {{ .Name }})
                    local -a args_list
                    args_list=(
{{- range .Args }}
                        '{{ . }}:{{ . }}'
{{- end }}
                    )
                    _describe -t args_list '{{ .Name }} options' args_list
                    ;;
{{- end }}
{{- if .Flags }}
                {{ .Name }})
                    _values 'options' {{ range $i, $flag := .Flags }}'{{ $flag }}' {{ end }}
                    ;;
{{- end }}
{{- end }}
            esac
            ;;
        args)
            case $words[2] in
{{- range . }}
{{- if .SubFlags }}
                {{ .Name }})
                    case $words[3] in
{{- range $subk, $subflags := .SubFlags }}
                        {{ $subk }})
                            _values 'options' {{ range $i, $flag := $subflags }}'{{ $flag }}' {{ end }}
                            ;;
{{- end }}
                    esac
                    ;;
{{- end }}
{{- if .Flags }}
                {{ .Name }})
                    _values 'options' {{ range $i, $flag := .Flags }}'{{ $flag }}' {{ end }}
                    ;;
{{- end }}
{{- end }}
            esac
            ;;
    esac
}

_baihu "$@"
`
