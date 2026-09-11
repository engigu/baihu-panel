#!/usr/bin/env bash
set -eo pipefail

# ============================================================
# 日志样式与格式化输出
# ============================================================
C_RESET="\033[0m"
C_BOLD="\033[1m"
C_CYAN="\033[1;36m"
C_GREEN="\033[1;32m"
C_YELLOW="\033[1;33m"
C_BLUE="\033[1;34m"

log_info() {
    printf "${C_CYAN}[GoBuild]${C_RESET} %s\n" "$*"
}

log_step() {
    printf "\n${C_BLUE}==>${C_RESET} ${C_BOLD}%s${C_RESET}\n" "$*"
}

log_success() {
    printf "${C_GREEN}✔ %s${C_RESET}\n" "$*"
}

log_warn() {
    printf "${C_YELLOW}⚠ %s${C_RESET}\n" "$*"
}

get_file_size() {
    ls -lh "$1" 2>/dev/null | awk '{print $5}' || echo ""
}

# ============================================================
# 基础路径与版本解析
# ============================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${ROOT_DIR}"

# 解析版本与构建时间（优先级：外部环境变量 > build-info目录 > git命令 > 默认值）
if [ -z "${VERSION}" ] && [ -f "/build-info/version.txt" ]; then
    VERSION="$(cat /build-info/version.txt)"
fi
if [ -z "${BUILD_TIME}" ] && [ -f "/build-info/build_time.txt" ]; then
    BUILD_TIME="$(cat /build-info/build_time.txt)"
fi

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
BUILD_TIME="${BUILD_TIME:-$(TZ='Asia/Shanghai' date '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date '+%Y-%m-%d %H:%M:%S')}"

LDFLAGS="-s -w -X 'github.com/engigu/baihu-panel/internal/constant.Version=${VERSION}' -X 'github.com/engigu/baihu-panel/internal/constant.BuildTime=${BUILD_TIME}'"
AGENT_LDFLAGS="-s -w -X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}'"

# ============================================================
# 1. 编译主服务端二进制 (baihu)
# ============================================================
build_server() {
    local target_os="${1:-${TARGETOS:-$(go env GOOS)}}"
    local target_arch="${2:-${TARGETARCH:-$(go env GOARCH)}}"
    local output_path="${3:-./baihu}"
    local tags="${4:-${BUILD_TAGS:-}}"
    local arm_version="${5:-${GOARM:-}}"

    log_step "开始编译主服务端程序: ${target_os}/${target_arch}"
    log_info "目标架构 : ${target_os}/${target_arch}"
    log_info "输出路径 : ${output_path}"
    [ -n "${tags}" ] && log_info "编译标签 : -tags '${tags}'"
    [ -n "${arm_version}" ] && log_info "ARM 版本 : GOARM=${arm_version}"

    mkdir -p "$(dirname "${output_path}")"

    log_info "检查依赖模块缓存 (go mod download)..."
    go mod download

    local build_cmd=(go build)
    if [ -n "${tags}" ]; then
        build_cmd+=(-tags "${tags}")
    fi
    if [ "${target_os}" = "android" ]; then
        build_cmd+=(-trimpath)
    fi
    build_cmd+=(-ldflags="${LDFLAGS}" -o "${output_path}" .)

    local start_time
    start_time=$(date +%s)

    if [ -n "${arm_version}" ]; then
        CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" GOARM="${arm_version}" "${build_cmd[@]}"
    else
        CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" "${build_cmd[@]}"
    fi

    local duration=$(( $(date +%s) - start_time ))
    local fsize
    fsize=$(get_file_size "${output_path}")

    log_success "服务端编译完成 -> ${output_path} (大小: ${fsize:-未知}, 耗时: ${duration}s)"
}

# ============================================================
# 2. 编译 Windows Tray GUI 程序 (./cmd/tray)
# ============================================================
build_tray() {
    local output_path="${1:-./bin/baihu-tray.exe}"

    log_step "开始编译 Windows 托盘 GUI 辅助程序"
    log_info "目标架构 : windows/amd64"
    log_info "输出路径 : ${output_path}"

    mkdir -p "$(dirname "${output_path}")"

    log_info "检查依赖模块缓存 (go mod download)..."
    go mod download

    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
        go build -ldflags="-s -w -H=windowsgui" -o "${output_path}" ./cmd/tray

    local fsize
    fsize=$(get_file_size "${output_path}")
    log_success "Windows 托盘程序编译完成 -> ${output_path} (大小: ${fsize:-未知})"
}

# ============================================================
# 3. 编译全平台客户端 Agent 并打包
# ============================================================
build_agents() {
    local output_dir="${1:-./data/agent}"
    local include_windows="${2:-false}"

    mkdir -p "${output_dir}"
    output_dir="$(cd "${output_dir}" && pwd)"

    log_step "开始编译全平台客户端 Agent"
    log_info "归档输出目录 : ${output_dir}"
    log_info "包含 Windows : ${include_windows}"

    echo "${VERSION}" > "${output_dir}/version.txt"
    log_info "已写入版本标识 : ${output_dir}/version.txt (${VERSION})"

    cd "${ROOT_DIR}/agent"
    log_info "检查 Agent 依赖模块缓存 (go mod download)..."
    go mod download

    _compile_single_agent() {
        local os=$1; local arch=$2; local suffix=$3; local is_zip=$4
        local tmpdir="build-${os}-${arch}"
        mkdir -p "${tmpdir}"
        cp config.example.ini "${tmpdir}/"

        local flags=()
        if [ "${os}" = "android" ]; then
            flags+=(-trimpath)
        fi

        log_info "正在编译 Agent [${os}/${arch}]..."
        CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" \
            go build "${flags[@]}" -ldflags="${AGENT_LDFLAGS}" -o "${tmpdir}/baihu-agent${suffix}" .

        local target_archive
        if [ "${is_zip}" = "true" ]; then
            target_archive="${output_dir}/baihu-agent-${os}-${arch}.zip"
            log_info "正在打包 zip 归档: baihu-agent-${os}-${arch}.zip..."
            (cd "${tmpdir}" && zip -q -r "${target_archive}" "baihu-agent${suffix}" config.example.ini)
        else
            target_archive="${output_dir}/baihu-agent-${os}-${arch}.tar.gz"
            log_info "正在打包 tar.gz 归档: baihu-agent-${os}-${arch}.tar.gz..."
            tar -czf "${target_archive}" -C "${tmpdir}" "baihu-agent${suffix}" config.example.ini
        fi
        rm -rf "${tmpdir}"

        local fsize
        fsize=$(get_file_size "${target_archive}")
        log_success "Agent 包生成完毕 -> $(basename "${target_archive}") (大小: ${fsize:-未知})"
    }

    _compile_single_agent linux amd64 "" false
    _compile_single_agent linux arm64 "" false
    _compile_single_agent android arm64 "" false
    _compile_single_agent darwin amd64 "" false
    _compile_single_agent darwin arm64 "" false

    if [ "${include_windows}" = "true" ]; then
        _compile_single_agent windows amd64 ".exe" true
    fi

    cd "${ROOT_DIR}"
    log_success "所有 Agent 编译与打包流程执行完毕！"
}

# ============================================================
# 4. CI 一键装配构建（专供 .github/workflows/deploy.yml）
# ============================================================
build_ci_artifacts() {
    local base_output="${1:-dist-assets}"

    log_step "启动 CI 极速制品装配流程"
    log_info "基础输出目录 : ${base_output}"

    mkdir -p "${base_output}"
    base_output="$(cd "${base_output}" && pwd)"

    log_info "预先预热依赖缓存..."
    go mod download

    # 依次编译服务端双架构二进制
    build_server linux amd64 "${base_output}/bin/linux-amd64/baihu"
    build_server linux arm64 "${base_output}/bin/linux-arm64/baihu"

    # 编译全平台客户端 Agent
    build_agents "${base_output}/agent" false

    log_step "CI 制品装配完毕，产物清单概览:"
    find "${base_output}" -type f -exec ls -lh {} + | awk '{print "   " $9 " (" $5 ")"}'
    log_success "CI 所有必需产物已全部准备就绪！"
}

# ============================================================
# 5. Release 全平台发布构建（专供 .github/workflows/release.yml）
# ============================================================
build_release_artifacts() {
    local base_output="${1:-.}"

    log_step "启动 Release 全平台发布包构建流程"
    log_info "基础输出目录 : ${base_output}"

    mkdir -p "${base_output}"
    base_output="$(cd "${base_output}" && pwd)"

    # 确保静态资源已准备好用于内嵌
    if [ -d "web/dist" ]; then
        log_info "同步前端构建物至 internal/static/dist 以供静态内嵌..."
        rm -rf internal/static/dist
        cp -r web/dist internal/static/dist
        log_success "前端静态资源内嵌目录准备就绪"
    else
        log_warn "未检测到 web/dist 目录，将使用现有 static 配置"
    fi

    log_info "预先同步依赖模块..."
    go mod download

    _compile_and_pack_server() {
        local os=$1; local arch=$2; local arm_val=$3; local ext=$4; local is_zip=$5
        local bin_name="baihu-${os}-${arch}${ext}"

        log_step "编译带 WebUI 内嵌的服务端: ${bin_name}"
        local build_cmd=(go build -tags web)
        if [ "${os}" = "android" ] || [ -n "${arm_val}" ]; then
            build_cmd+=(-trimpath)
        fi
        build_cmd+=(-ldflags="${LDFLAGS}" -o "${bin_name}" main.go)

        if [ -n "${arm_val}" ]; then
            CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" GOARM="${arm_val}" "${build_cmd[@]}"
        else
            CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" "${build_cmd[@]}"
        fi

        local archive_path
        if [ "${is_zip}" = "true" ]; then
            archive_path="${base_output}/baihu-${os}-${arch}.zip"
            log_info "压缩归档 -> $(basename "${archive_path}")..."
            zip -q -r "${archive_path}" "${bin_name}"
        else
            archive_path="${base_output}/baihu-${os}-${arch}.tar.gz"
            log_info "压缩归档 -> $(basename "${archive_path}")..."
            tar -czf "${archive_path}" "${bin_name}"
        fi
        rm -f "${bin_name}"

        local fsize
        fsize=$(get_file_size "${archive_path}")
        log_success "发布包生成完毕 -> $(basename "${archive_path}") (大小: ${fsize:-未知})"
    }

    _compile_and_pack_server linux amd64 "" "" false
    _compile_and_pack_server linux arm64 "" "" false
    _compile_and_pack_server android arm64 "" "" false
    _compile_and_pack_server linux arm "7" "" false

    # Windows 主程序与托盘程序
    log_step "编译 Windows 主服务二进制与托盘辅助程序"
    mkdir -p "${base_output}/bin"
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
        go build -tags web -ldflags="${LDFLAGS}" -o "${base_output}/bin/baihu.exe" main.go

    build_tray "${base_output}/bin/baihu-tray.exe"

    cp "${base_output}/bin/baihu.exe" "baihu-windows-amd64.exe"
    zip -q -r "${base_output}/baihu-windows-amd64.zip" "baihu-windows-amd64.exe"
    rm -f "baihu-windows-amd64.exe"
    local win_size
    win_size=$(get_file_size "${base_output}/baihu-windows-amd64.zip")
    log_success "Windows 发布包生成完毕 -> baihu-windows-amd64.zip (大小: ${win_size:-未知})"

    # 编译全平台 Agent（包括 Windows）
    build_agents "${base_output}/data/agent" true

    log_step "Release 发布包构建完毕！"
}

# ============================================================
# 命令行入口调度与环境横幅
# ============================================================
CMD="${1:-server}"
shift || true

echo "============================================================"
echo "           白虎面板 (Baihu Panel) Go 构建系统"
echo "============================================================"
log_info "当前构建指令 : ${CMD}"
log_info "项目根目录   : ${ROOT_DIR}"
log_info "构建版本号   : ${VERSION}"
log_info "构建时间戳   : ${BUILD_TIME}"
log_info "Go 编译器    : $(go version 2>/dev/null || echo '未检测到 Go 环境')"
echo "============================================================"

case "${CMD}" in
    server)
        build_server "$@"
        ;;
    tray)
        build_tray "$@"
        ;;
    agents)
        build_agents "$@"
        ;;
    ci-artifacts)
        build_ci_artifacts "$@"
        ;;
    release-artifacts)
        build_release_artifacts "$@"
        ;;
    help|--help|-h)
        echo "使用方法: $0 {server [os] [arch] [output] [tags]|tray [output]|agents [output_dir] [include_windows]|ci-artifacts [output_dir]|release-artifacts [output_dir]}"
        exit 0
        ;;
    *)
        log_warn "未知指令: ${CMD}"
        echo "使用方法: $0 {server [os] [arch] [output] [tags]|tray [output]|agents [output_dir] [include_windows]|ci-artifacts [output_dir]|release-artifacts [output_dir]}"
        exit 1
        ;;
esac
