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
Register-ArgumentCompleter -Native -CommandName baihu -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $commands = @{
{{- range . }}
        '{{ .Name }}' = '{{ .Description }}'
{{- end }}
    }

    $elements = $commandAst.Elements
    if ($elements.Count -eq 2 -or ($elements.Count -eq 3 -and $wordToComplete -ne '')) {
        $commands.GetEnumerator() | Where-Object { $_.Key -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_.Key, $_.Key, 'Command', $_.Value)
        }
        return
    }

    $subcommand = $elements[1].Value
{{- range . }}
{{- if .SubCommands }}
    if ($subcommand -eq '{{ .Name }}') {
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
{{- end }}
{{- if .Args }}
    if ($subcommand -eq '{{ .Name }}') {
        @({{ range $i, $arg := .Args }}{{ if $i }}, {{ end }}'{{ $arg }}'{{ end }}) | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }
{{- end }}
{{- if .Flags }}
    if ($subcommand -eq '{{ .Name }}') {
        $flags = @({{ range $i, $flag := .Flags }}{{ if $i }}, {{ end }}'{{ $flag }}'{{ end }})
        $flags | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
        }
        return
    }
{{- end }}
{{- end }}
}
`

const BashTmpl = `# baihu Bash Completion Script (Auto-generated)
_baihu_completions() {
    local cur prev
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    local commands="{{ range $i, $cmd := . }}{{ if $i }} {{ end }}{{ $cmd.Name }}{{ end }}"

    if [ $COMP_CWORD -eq 1 ]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi

    case "${prev}" in
{{- range . }}
{{- if .SubCommands }}
        {{ .Name }})
            COMPREPLY=( $(compgen -W "{{ range $k, $v := .SubCommands }}{{ $k }} {{ end }}" -- ${cur}) )
            return 0
            ;;
{{- end }}
{{- if .Args }}
        {{ .Name }})
            COMPREPLY=( $(compgen -W "{{ range $i, $arg := .Args }}{{ if $i }} {{ end }}{{ $arg }}{{ end }}" -- ${cur}) )
            return 0
            ;;
{{- end }}
{{- if .Flags }}
        {{ .Name }})
            COMPREPLY=( $(compgen -W "{{ range $i, $flag := .Flags }}{{ if $i }} {{ end }}{{ $flag }}{{ end }}" -- ${cur}) )
            return 0
            ;;
{{- end }}
{{- end }}
    esac

{{- range . }}
{{- if .Flags }}
    if [ "${COMP_WORDS[1]}" = "{{ .Name }}" ]; then
        COMPREPLY=( $(compgen -W "{{ range $i, $flag := .Flags }}{{ if $i }} {{ end }}{{ $flag }}{{ end }}" -- ${cur}) )
        return 0
    fi
{{- end }}
{{- end }}
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
        '*:: :->args'

    case $state in
        command)
            _describe -t commands 'baihu command' commands
            ;;
        args)
            case $words[1] in
{{- range . }}
{{- if .SubCommands }}
                {{ .Name }})
                    local -a subcmds
                    subcmds=(
{{- range $k, $v := .SubCommands }}
                        '{{ $k }}:{{ $v }}'
{{- end }}
                    )
                    _describe -t subcmds '{{ .Name }} command' subcmds
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
    esac
}

_baihu "$@"
`
