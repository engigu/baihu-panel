package reposync

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/cmd/clibase"
	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/services/repo"
	"github.com/engigu/baihu-panel/internal/utils"
)

type Config struct {
	SourceType     string
	SourceURL      string
	TargetPath     string
	Branch         string
	Path           string
	SingleFile     bool
	Proxy          string
	ProxyURL       string
	AuthToken      string
	HttpProxy      string
	WhitelistPaths string // Comma or vertical line separated paths to preserve or filter (whitelist)
	Blacklist      string // Script filter blacklist keywords, vertical line separated
	Dependence     string // Script dependence file keywords, vertical line separated
	Extensions     string // Script file extensions, vertical line separated
	TaskID         string
	RepoTaskID     string
	TaskLanguages  string
	TaskTimeout    int
	CommentToTask  string
	PreCommand     string
	PostCommand    string
	RepoName       string
}

func Run(args []string) {
	fs := flag.NewFlagSet("reposync", flag.ExitOnError)
	var cfg Config
	fs.StringVar(&cfg.SourceType, "source-type", "git", "Source type: git or url")
	fs.StringVar(&cfg.SourceURL, "source-url", "", "Source url")
	fs.StringVar(&cfg.TargetPath, "target-path", "", "Target path")
	fs.StringVar(&cfg.Branch, "branch", "", "Branch")
	fs.StringVar(&cfg.Path, "path", "", "Path for sparse checkout")
	fs.BoolVar(&cfg.SingleFile, "single-file", false, "Single file mode")
	fs.StringVar(&cfg.Proxy, "proxy", "none", "Proxy type")
	fs.StringVar(&cfg.ProxyURL, "proxy-url", "", "Custom proxy url")
	fs.StringVar(&cfg.AuthToken, "auth-token", "", "Auth token")
	fs.StringVar(&cfg.HttpProxy, "http-proxy", "", "Http proxy")
	fs.StringVar(&cfg.WhitelistPaths, "whitelist-paths", "", "Separated paths to preserve or filter (whitelist)")
	fs.StringVar(&cfg.Blacklist, "blacklist", "", "Script filter blacklist keywords (| separated)")
	fs.StringVar(&cfg.Dependence, "dependence", "", "Script dependence keywords (| separated)")
	fs.StringVar(&cfg.Extensions, "extensions", "", "Script extensions (| separated)")
	fs.StringVar(&cfg.TaskID, "task-id", "", "Task ID for metadata")
	fs.StringVar(&cfg.TaskLanguages, "task-langs", "", "Configured languages (JSON)")
	fs.StringVar(&cfg.RepoTaskID, "repo-task-id", "", "Original Task ID")
	fs.IntVar(&cfg.TaskTimeout, "task-timeout", 30, "Task timeout (minutes)")
	fs.StringVar(&cfg.CommentToTask, "commenttotask", "false", "Compatible with QL format script comment parsing (true/false)")
	fs.StringVar(&cfg.PreCommand, "pre-command", "", "Default pre-command for discovered tasks")
	fs.StringVar(&cfg.PostCommand, "post-command", "", "Default post-command for discovered tasks")
	fs.StringVar(&cfg.RepoName, "repo-name", "", "Custom repository directory name")

	printHelp := func() {
		fmt.Fprintf(os.Stderr, "\n白虎面板仓库同步工具 (Reposync)\n\n")
		fmt.Fprintf(os.Stderr, "用法:\n")
		fmt.Fprintf(os.Stderr, "  baihu reposync [参数]\n\n")
		fmt.Fprintf(os.Stderr, "参数详情:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  baihu reposync --source-url https://github.com/xxx/repo.git --target-path $SCRIPTS_DIR$/repo1\n\n")
	}

	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		printHelp()
		return
	}

	fs.Usage = printHelp

	if err := fs.Parse(args); err != nil {
		return
	}

	if cfg.SourceURL == "" {
		fmt.Fprintf(os.Stderr, "错误: 必须提供 --source-url 参数\n")
		fs.Usage()
		return
	}

	// 处理 $SCRIPTS_DIR$ 代号替换
	if strings.Contains(cfg.TargetPath, constant.ScriptsDirPlaceholder) {
		scriptsDir := os.Getenv("BH_SCRIPTS_DIR")
		if scriptsDir != "" {
			cfg.TargetPath = filepath.Clean(strings.ReplaceAll(cfg.TargetPath, constant.ScriptsDirPlaceholder, scriptsDir))
		}
	}

	fmt.Println("========================================")
	fmt.Println("  仓库同步任务开始  ")
	fmt.Println("========================================")
	fmt.Printf("[1/3] 解析同步参数: %s\n", strings.Join(args, " "))

	if cfg.SourceType == "git" {
		fmt.Printf("[2/3] 正在通过 Git 同步内容...\n")
		syncGit(cfg)
	} else {
		fmt.Printf("[2/3] 正在通过 URL 下载内容...\n")
		syncURL(cfg)
	}

	// 执行前置指令
	if cfg.PreCommand != "" {
		fmt.Printf("[准备] 执行同步前指令: %s\n", cfg.PreCommand)
		// 计算当前仓库真实的物理路径
		repoDir := getActualRepoDir(cfg)
		fmt.Printf("[准备] 工作目录: %s\n", repoDir)
		fmt.Printf("[准备] 注入环境变量: CURR_REPO_DIR=%s\n", repoDir)

		shell, shellArgs := utils.GetShellCommand(cfg.PreCommand)
		envs := append(os.Environ(), "CURR_REPO_DIR="+repoDir)
		runCmd(append([]string{shell}, shellArgs...), repoDir, envs)
	}

	// 执行脚本过滤（仅限 git 模式，url 加载通常为单文件，暂不处理过滤）
	if cfg.SourceType == "git" {
		fmt.Printf("[3/3] 正在执行脚本过滤与文件清理...\n")
		filterFiles(cfg)

		if cfg.TaskID != "" {
			upsertedIDs, deletedIDs := repo.ParseRepoScriptsAndAddCron(cfg.TaskID, os.Stdout, cfg.CommentToTask == "true")
			if len(upsertedIDs) > 0 || len(deletedIDs) > 0 {
				notifyMainServerToSyncRepoTasks(cfg.TaskID, upsertedIDs, deletedIDs)
			}
		}
	}

	// 执行后置指令
	if cfg.PostCommand != "" {
		fmt.Printf("[收尾] 执行同步后指令: %s\n", cfg.PostCommand)

		// 计算当前仓库真实的物理路径
		repoDir := getActualRepoDir(cfg)
		fmt.Printf("[收尾] 工作目录: %s\n", repoDir)
		fmt.Printf("[收尾] 注入环境变量: CURR_REPO_DIR=%s\n", repoDir)

		shell, shellArgs := utils.GetShellCommand(cfg.PostCommand)
		envs := append(os.Environ(), "CURR_REPO_DIR="+repoDir)
		runCmd(append([]string{shell}, shellArgs...), repoDir, envs)
	}

	fmt.Println("\n========================================")
	fmt.Println("  仓库同步任务完成  ")
	fmt.Println("========================================")
}

func getActualRepoDir(cfg Config) string {
	if cfg.SourceType == "git" {
		repoName := cfg.RepoName
		if repoName == "" {
			repoName = utils.GetRepoIdentifier(cfg.SourceURL, cfg.Branch)
		}
		if repoName == "." {
			return cfg.TargetPath
		}
		return filepath.Join(cfg.TargetPath, repoName)
	}
	return cfg.TargetPath
}

func notifyMainServerToSyncRepoTasks(repoID string, upsertedIDs []string, deletedIDs []string) {
	_, err := clibase.CallInternalAPI("POST", "/internal/tasks/sync-repo-status", map[string]interface{}{
		"repo_id":      repoID,
		"upserted_ids": upsertedIDs,
		"deleted_ids":  deletedIDs,
	})
	if err != nil {
		fmt.Printf(">> [通知] 调度器同步失败: %v\n", err)
		return
	}
	fmt.Println(">> [通知] 已成功将变动任务增量同步至主程序调度器")
}

func syncGit(cfg Config) {
	env := os.Environ()

	if isRawFileURL(cfg.SourceURL) {
		fmt.Println("检测到 raw 文件 URL，自动切换到 URL 下载模式")
		syncURL(cfg)
		return
	}

	if cfg.HttpProxy != "" {
		env = append(env, "http_proxy="+cfg.HttpProxy, "https_proxy="+cfg.HttpProxy)
	}

	repoURL := buildProxyURL(cfg.SourceURL, cfg.Proxy, cfg.ProxyURL)
	if cfg.AuthToken != "" && strings.HasPrefix(repoURL, "https://") {
		repoURL = strings.Replace(repoURL, "https://", "https://"+cfg.AuthToken+"@", 1)
	}

	dest := cfg.TargetPath

	if cfg.Path != "" && cfg.SingleFile {
		syncGitFile(cfg, repoURL, env)
		return
	}

	gitDir := filepath.Join(dest, ".git")
	if isDir(dest) && !pathExists(gitDir) {
		repoName := cfg.RepoName
		if repoName == "" {
			repoName = utils.GetRepoIdentifier(cfg.SourceURL, cfg.Branch)
		}
		if repoName != "." {
			dest = filepath.Join(dest, repoName)
			fmt.Printf("目标路径自动追加仓库名: %s\n", dest)
			gitDir = filepath.Join(dest, ".git")
		} else {
			fmt.Printf("目标路径使用当前目录 (不追加仓库名): %s\n", dest)
		}
	}

	restore := preserve(dest, cfg.WhitelistPaths)
	defer restore()

	if pathExists(gitDir) {
		fmt.Println("检测到已存在仓库，正在更新...")
		runCmd([]string{"git", "fetch", "--all"}, dest, env)

		targetBranch := cfg.Branch
		if targetBranch != "" {
			// 如果切换了分支，或者当前分支偏离，强制切换并对齐远程
			runCmd([]string{"git", "checkout", "-B", targetBranch, "origin/" + targetBranch}, dest, env)
		} else {
			targetBranch = getCurrentBranch(dest, env)
		}

		if targetBranch != "" {
			fmt.Printf("执行强制同步 (reset --hard origin/%s)\n", targetBranch)
			runCmd([]string{"git", "reset", "--hard", "origin/" + targetBranch}, dest, env)
		} else {
			runCmd([]string{"git", "pull", "--rebase"}, dest, env)
		}
	} else {
		fmt.Println("执行 git clone")
		parentDir := filepath.Dir(dest)
		if parentDir != "" {
			os.MkdirAll(parentDir, 0755)
		}

		if pathExists(dest) && !isDirEmpty(dest) {
			// If we still have files after preservation, warn but maybe continue if it's just leftovers that git can handle?
			// Actually git clone requires an empty dir.
			fmt.Printf("警告: 目标目录 '%s' 不为空，尝试清理非保护文件...\n", dest)
			// Optional: delete everything else? User might not want that.
			// For now, keep the error but it's less likely to occur if preservation moved things out.
			fmt.Println("提示: 请清空目标目录或指定一个新目录")
			os.Exit(1)
		}
		// If dest exists but is empty now, git clone might still complain if the directory itself exists?
		// No, git clone works if dir is empty.

		cloneCmd := []string{"git", "clone", "--depth", "1"}
		if cfg.Branch != "" {
			cloneCmd = append(cloneCmd, "-b", cfg.Branch)
		}

		if cfg.Path != "" {
			cloneCmd = append(cloneCmd, "--filter=blob:none", "--no-checkout", repoURL, dest)
			runCmd(cloneCmd, "", env)
			runCmd([]string{"git", "sparse-checkout", "init", "--cone"}, dest, env)
			runCmd([]string{"git", "sparse-checkout", "set", cfg.Path}, dest, env)
			runCmd([]string{"git", "checkout"}, dest, env)
		} else {
			cloneCmd = append(cloneCmd, repoURL, dest)
			runCmd(cloneCmd, "", env)
		}
	}
	fmt.Println("同步完成")
}

func syncURL(cfg Config) {
	downloadURL := buildProxyURL(cfg.SourceURL, cfg.Proxy, cfg.ProxyURL)
	fmt.Printf("下载地址: %s\n", downloadURL)
	dest := cfg.TargetPath

	if isDir(dest) || strings.HasSuffix(dest, string(os.PathSeparator)) || strings.HasSuffix(dest, "/") {
		urlPath := strings.Split(cfg.SourceURL, "?")[0]
		filename := filepath.Base(urlPath)
		if filename == "" {
			filename = "downloaded_file"
		}
		dest = filepath.Join(dest, filename)
		fmt.Printf("目标文件: %s\n", dest)
	}

	restore := preserve(cfg.TargetPath, cfg.WhitelistPaths)
	defer restore()

	downloadFile(downloadURL, dest, cfg.AuthToken)
}

func syncGitFile(cfg Config, repoURL string, env []string) {
	sourceURL := cfg.SourceURL
	filePath := cfg.Path
	dest := cfg.TargetPath

	if isDir(dest) || strings.HasSuffix(dest, string(os.PathSeparator)) || strings.HasSuffix(dest, "/") {
		filename := filepath.Base(filePath)
		dest = filepath.Join(dest, filename)
		fmt.Printf("检测到目标路径为目录 '%s'，自动修正为: '%s'\n", cfg.TargetPath, dest)
	}

	branch := cfg.Branch
	if branch == "" {
		branch = getRemoteDefaultBranch(repoURL, env)
	}

	cleanURL := strings.TrimSuffix(cfg.SourceURL, ".git")
	rawURL := ""

	if strings.Contains(sourceURL, "github.com") {
		base := strings.Replace(strings.TrimSuffix(cfg.SourceURL, ".git"), "github.com", "raw.githubusercontent.com", 1)
		rawURL = fmt.Sprintf("%s/%s/%s", base, branch, filePath)
	} else if strings.Contains(sourceURL, "gitlab.com") {
		rawURL = fmt.Sprintf("%s/-/raw/%s/%s", cleanURL, branch, filePath)
	} else if strings.Contains(sourceURL, "gitee.com") {
		rawURL = fmt.Sprintf("%s/raw/%s/%s", cleanURL, branch, filePath)
	} else {
		rawURL = fmt.Sprintf("%s/raw/%s/%s", cleanURL, branch, filePath)
	}

	rawURL = buildProxyURL(rawURL, cfg.Proxy, cfg.ProxyURL)
	downloadFile(rawURL, dest, cfg.AuthToken)
}

func getRemoteDefaultBranch(repoURL string, env []string) string {
	fmt.Printf("正在检测远程仓库默认分支: %s\n", repoURL)
	cmd := exec.Command("git", "ls-remote", "--symref", repoURL, "HEAD")
	cmd.Env = env
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 && parts[0] == "ref:" && strings.Contains(parts[1], "refs/heads/") {
				branch := strings.TrimPrefix(parts[1], "refs/heads/")
				fmt.Printf("检测到默认分支: %s\n", branch)
				return branch
			}
		}
	}
	fmt.Println("无法检测到默认分支，回退使用 'main'")
	return "main"
}

func buildProxyURL(url string, proxyType string, proxyURL string) string {
	if proxyType == "" || proxyType == "none" {
		return url
	}

	// 如果 URL 已经包含明显的代理前缀 (如用户手动填写的 http://ghproxy.com/...)
	// 则跳过内置代理逻辑
	if strings.Contains(url, "googo.win") || (proxyType == "custom" && strings.HasPrefix(url, proxyURL)) {
		return url
	}

	base := ""
	if proxyType == "ghproxy" {
		base = "https://gh-proxy.com/"
	} else if proxyType == "mirror" {
		base = "https://mirror.ghproxy.com/"
	} else if proxyType == "custom" && proxyURL != "" {
		base = strings.TrimSuffix(proxyURL, "/") + "/"
	}

	if base != "" && strings.HasPrefix(url, "http") && !strings.HasPrefix(url, base) {
		return base + url
	}
	return url
}

func downloadFile(url, dest, authToken string) {
	fmt.Printf("下载地址: %s\n", url)
	fmt.Printf("目标路径: %s\n", dest)

	parentDir := filepath.Dir(dest)
	if parentDir != "" {
		os.MkdirAll(parentDir, 0755)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("下载准备失败: %v\n", err)
		os.Exit(1)
	}
	if authToken != "" {
		req.Header.Set("Authorization", "token "+authToken)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; reposync)")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("下载请求失败: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("下载失败, HTTP 状态码: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	out, err := os.Create(dest)
	if err != nil {
		fmt.Printf("创建文件失败: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		fmt.Printf("写入数据失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("文件大小: %d 字节\n", n)
	fmt.Println("下载完成")
}

func isRawFileURL(url string) bool {
	rawPatterns := []string{
		"raw.githubusercontent.com",
		"/raw/",
		"/-/raw/",
		"/blob/",
	}
	for _, p := range rawPatterns {
		if strings.Contains(url, p) {
			return true
		}
	}
	return false
}

func runCmd(args []string, dir string, env []string) {
	fmt.Printf(">> %s\n", strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = env

	cw := clibase.NewCleanWriter(os.Stdout)
	cmd.Stdout = cw
	cmd.Stderr = cw

	if err := cmd.Run(); err != nil {
		cw.Flush()
		fmt.Printf("命令执行失败: %v\n", err)
		os.Exit(1)
	}
	cw.Flush()
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

func isDirEmpty(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	return err == io.EOF
}

func getCurrentBranch(dir string, env []string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// preserve moves specified paths to a temporary location and returns a function to restore them
func preserve(baseDir string, paths string) func() {
	if paths == "" || !pathExists(baseDir) {
		return func() {}
	}

	preservedList := strings.Split(paths, ",")
	// 优化：将临时目录创建在 baseDir 同一级或内部，确保在同一个文件系统，使得 Rename 是 O(1) 瞬时完成的
	tmpParent, err := os.MkdirTemp(baseDir, ".baihu_sync_preserve_*")
	if err != nil {
		fmt.Printf("警告: 无法在目标目录创建临时目录用于保留文件: %v\n", err)
		return func() {}
	}

	type preservedItem struct {
		relPath string
		tmpPath string
	}
	var items []preservedItem
	processed := make(map[string]bool)

	for _, p := range preservedList {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// Support glob matching
		pattern := filepath.Join(baseDir, p)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			fmt.Printf("警告: 路径模式无效 %s: %v\n", p, err)
			continue
		}

		// If literal path exists but Glob didn't find it (common for direct dir reference), add it manually
		if len(matches) == 0 && pathExists(pattern) {
			matches = []string{pattern}
		}

		for _, fullPath := range matches {
			relPath, err := filepath.Rel(baseDir, fullPath)
			// 同时要排除掉临时目录本身以及上级路径
			if err != nil || strings.HasPrefix(relPath, "..") || relPath == "." || strings.HasPrefix(relPath, ".baihu_sync_preserve") {
				continue
			}

			if processed[relPath] {
				continue
			}
			processed[relPath] = true

			tmpPath := filepath.Join(tmpParent, relPath)
			os.MkdirAll(filepath.Dir(tmpPath), 0755)

			fmt.Printf("正在保护路径: %s\n", relPath)
			if err := os.Rename(fullPath, tmpPath); err == nil {
				items = append(items, preservedItem{relPath: relPath, tmpPath: tmpPath})
			} else {
				// Rename might fail across filesystems, try copy
				if err := utils.CopyPath(fullPath, tmpPath); err == nil {
					os.RemoveAll(fullPath)
					items = append(items, preservedItem{relPath: relPath, tmpPath: tmpPath})
				} else {
					fmt.Printf("警告: 无法保护路径 %s: %v\n", relPath, err)
				}
			}
		}
	}

	return func() {
		// Restore in reverse order to handle nested structures correctly if they were picked up separately
		for i := len(items) - 1; i >= 0; i-- {
			item := items[i]
			destPath := filepath.Join(baseDir, item.relPath)
			os.MkdirAll(filepath.Dir(destPath), 0755)

			if pathExists(destPath) {
				fmt.Printf("目标已存在，覆盖恢复保护路径: %s\n", item.relPath)
				os.RemoveAll(destPath)
			} else {
				fmt.Printf("正在恢复保护路径: %s\n", item.relPath)
			}

			if err := os.Rename(item.tmpPath, destPath); err != nil {
				// Fallback to copy
				utils.CopyPath(item.tmpPath, destPath)
			}
		}
		os.RemoveAll(tmpParent)
	}
}

// filterFiles performs script filtering based on whitelist, blacklist, dependence and extensions.
func filterFiles(cfg Config) {
	// If no filtering is specified, do nothing.
	if cfg.WhitelistPaths == "" && cfg.Blacklist == "" && cfg.Dependence == "" && cfg.Extensions == "" {
		return
	}

	dest := getActualRepoDir(cfg)

	fmt.Printf("开始执行脚本过滤: %s\n", dest)

	whitelist := splitKeywords(cfg.WhitelistPaths)
	blacklist := splitKeywords(cfg.Blacklist)
	dependence := splitKeywords(cfg.Dependence)
	extensions := splitKeywords(cfg.Extensions)

	// We'll collect files to delete to avoid modifying while walking if possible.
	// But os.RemoveAll is fine.

	count := 0
	filepath.Walk(dest, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, _ := filepath.Rel(dest, path)
		rel = filepath.ToSlash(rel)
		filename := info.Name()

		// 1. Check dependence: always keep
		if matchesAny(rel, filename, dependence) {
			return nil
		}

		// 2. Check extensions: delete if not matched and extensions is specified
		if len(extensions) > 0 {
			ext := strings.TrimPrefix(filepath.Ext(filename), ".")
			matchedExt := false
			for _, e := range extensions {
				if strings.EqualFold(ext, strings.TrimPrefix(e, ".")) {
					matchedExt = true
					break
				}
			}
			if !matchedExt {
				fmt.Printf("过滤文件 (后缀不符): %s\n", rel)
				os.Remove(path)
				count++
				return nil
			}
		}

		// 3. Check blacklist: delete if matched
		if matchesAny(rel, filename, blacklist) {
			fmt.Printf("过滤文件 (黑名单): %s\n", rel)
			os.Remove(path)
			count++
			return nil
		}

		// 4. Check whitelist: delete if NOT matched and whitelist is specified
		if len(whitelist) > 0 {
			if !matchesAny(rel, filename, whitelist) {
				fmt.Printf("过滤文件 (不在白名单): %s\n", rel)
				os.Remove(path)
				count++
				return nil
			}
		}

		return nil
	})

	if count > 0 {
		fmt.Printf("过滤完成，共删除 %d 个不符合要求的文件\n", count)
		// Try to clean up empty directories
		cleanEmptyDirs(dest)
	}
}

func splitKeywords(s string) []string {
	if s == "" {
		return nil
	}
	// Try to split by common separators for compatibility
	var parts []string
	if strings.Contains(s, "|") {
		parts = strings.Split(s, "|")
	} else if strings.Contains(s, ",") {
		parts = strings.Split(s, ",")
	} else {
		parts = []string{s}
	}

	var res []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, p)
		}
	}
	return res
}

func matchesAny(rel, filename string, keywords []string) bool {
	if len(keywords) == 0 {
		return false
	}
	for _, k := range keywords {
		// 1. 尝试作为正则整体进行匹配，默认不区分大小写 (?i)
		// 如果关键字不包含正则元字符，则补齐 (?i) 开启忽略大小写
		pattern := k
		if !strings.HasPrefix(pattern, "(?i)") {
			pattern = "(?i)" + pattern
		}

		reg, err := regexp.Compile(pattern)
		if err == nil {
			// 优先匹配文件名（解决 ^jd[^_] 这种锚点在相对路径下失效的问题）
			if reg.MatchString(filename) || reg.MatchString(rel) {
				return true
			}
		} else {
			// 回退逻辑：全小写包含判断
			kLower := strings.ToLower(k)
			if strings.Contains(strings.ToLower(rel), kLower) || strings.Contains(strings.ToLower(filename), kLower) {
				return true
			}
		}
	}
	return false
}

func cleanEmptyDirs(root string) {
	// Post-order traversal to clean up empty dirs
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		if path == root {
			return nil
		}
		if info.Name() == ".git" {
			return filepath.SkipDir
		}
		return nil
	})

	// Actually we need to do this recursively or multiple times.
	// A simpler way:
	entries, _ := os.ReadDir(root)
	for _, entry := range entries {
		if entry.IsDir() {
			if entry.Name() == ".git" {
				continue
			}
			dirPath := filepath.Join(root, entry.Name())
			cleanEmptyDirs(dirPath)
			// Check if now empty
			if isDirEmpty(dirPath) {
				os.Remove(dirPath)
			}
		}
	}

}
