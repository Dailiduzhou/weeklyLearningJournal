#!/usr/bin/env bash
#
# mvc_rust 开发脚本：一条命令把数据库和应用都拉起来，并把演示页地址打出来。
#
#   ./run.sh                # 起 MySQL（3307 上已有就复用）+ 建表种子 + cargo run
#   ./run.sh full           # 全容器：docker compose --profile full up --build
#   ./run.sh db             # 只起 MySQL 并建好表，然后退出
#   ./run.sh stop           # 停掉容器（保留数据卷）；./run.sh stop --volumes 连数据一起删
#   ./run.sh reset          # 删掉数据卷重新初始化（数据库清空重建）
#   ./run.sh status         # 看端口 / 容器 / 应用 / 表里有哪些用户
#   ./run.sh logs [服务]    # 跟踪容器日志，服务默认 app
#   ./run.sh sql            # 打开 MySQL 交互式客户端
#   ./run.sh help           # 就是这份说明
#
# 可用环境变量覆盖：
#   DB_HOST DB_PORT DB_USER DB_PASSWORD DB_NAME APP_PORT
#   ADMIN_USER ADMIN_PASSWORD ADMIN_NAME
#   CARGO_PROFILE=release    # cargo run 换成 --profile release
#   WAIT_DB_TIMEOUT WAIT_APP_TIMEOUT
#
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR"

DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3307}"          # 和 src/main.rs 里的默认连接串一致
DB_USER="${DB_USER:-root}"
DB_PASSWORD="${DB_PASSWORD:-123456}"
DB_NAME="${DB_NAME:-ginserver}"
APP_PORT="${APP_PORT:-8080}"

ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"
ADMIN_NAME="${ADMIN_NAME:-管理员}"

CARGO_PROFILE="${CARGO_PROFILE:-}"
WAIT_DB_TIMEOUT="${WAIT_DB_TIMEOUT:-120}"
WAIT_APP_TIMEOUT="${WAIT_APP_TIMEOUT:-180}"

COMPOSE=(docker compose)
DB_VIA=""        # docker | host | none —— 数据库怎么访问
MYSQL_BIN=""
MYSQL_OPTS_FILE=""   # 临时的 --defaults-extra-file，避免密码出现在命令行/环境里的告警
APP_PID=""

if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
  C_INFO=$'\033[36m'; C_OK=$'\033[32m'; C_WARN=$'\033[33m'; C_ERR=$'\033[31m'; C_OFF=$'\033[0m'
else
  C_INFO=""; C_OK=""; C_WARN=""; C_ERR=""; C_OFF=""
fi

log()  { printf '%s[mvc_rust]%s %s\n' "$C_INFO" "$C_OFF" "$*"; }
ok()   { printf '%s[mvc_rust]%s %s\n' "$C_OK" "$C_OFF" "$*"; }
warn() { printf '%s[mvc_rust] 注意:%s %s\n' "$C_WARN" "$C_OFF" "$*" >&2; }
die()  { printf '%s[mvc_rust] 错误:%s %s\n' "$C_ERR" "$C_OFF" "$*" >&2; exit 1; }

usage() {
  # 打印文件头部的注释块：从第 2 行开始，遇到第一个非注释行就停
  awk 'NR > 1 { if ($0 !~ /^#/) exit; sub(/^# ?/, ""); print }' "${BASH_SOURCE[0]}"
}

need() { command -v "$1" >/dev/null 2>&1 || die "找不到命令 $1：$2"; }
docker_ok() { command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; }

# 纯 bash 探端口，不依赖 nc
port_open() { (exec 3<>"/dev/tcp/$1/$2") >/dev/null 2>&1; }

host_mysql_bin() {
  local c
  # 优先 mariadb：Arch 上的 /usr/bin/mysql 只是 mariadb 的软链，会多打一行 Deprecated 提示
  for c in mariadb mysql; do
    if command -v "$c" >/dev/null 2>&1; then printf '%s' "$c"; return 0; fi
  done
  return 1
}

# 一次性写好连接参数（权限 600），比 -p密码 干净：不会有 “password on the command line” / ssl 告警
mysql_opts_file() {
  if [[ -z "$MYSQL_OPTS_FILE" ]]; then
    MYSQL_OPTS_FILE="$(mktemp)"
    chmod 600 "$MYSQL_OPTS_FILE"
    cat > "$MYSQL_OPTS_FILE" <<EOF
[client]
user=$DB_USER
password=$DB_PASSWORD
host=$DB_HOST
port=$DB_PORT
EOF
  fi
  printf '%s' "$MYSQL_OPTS_FILE"
}

cleanup() {
  stop_app
  if [[ -n "$MYSQL_OPTS_FILE" ]]; then rm -f "$MYSQL_OPTS_FILE"; fi
}

# 判断该用容器里的客户端还是宿主机的客户端
detect_db_via() {
  if docker_ok && "${COMPOSE[@]}" ps -q mysql 2>/dev/null | grep -q .; then
    DB_VIA=docker; return 0
  fi
  if MYSQL_BIN="$(host_mysql_bin)"; then
    DB_VIA=host; return 0
  fi
  DB_VIA=none; return 1
}

# 从 stdin 读 SQL 执行；"$@" 透传给 mysql 客户端（比如 -N 去掉表头）
sql() {
  case "$DB_VIA" in
    docker)
      "${COMPOSE[@]}" exec -T -e MYSQL_PWD="$DB_PASSWORD" mysql \
        mysql -u"$DB_USER" "$@" ;;
    host)
      "$MYSQL_BIN" --defaults-extra-file="$(mysql_opts_file)" "$@" ;;
    *)
      detect_db_via || die "既没有运行中的 mysql 容器，也没装 mysql/mariadb 客户端" ;;
  esac
}
wait_for_db() {
  local waited=0
  log "等待 MySQL 就绪（$DB_HOST:$DB_PORT）…"
  while (( waited < WAIT_DB_TIMEOUT )); do
    if echo 'SELECT 1;' | sql -N >/dev/null 2>&1; then
      ok "MySQL 已就绪（第 $((waited + 1)) 秒）"
      return 0
    fi
    sleep 2
    waited=$((waited + 2))
  done
  die "MySQL 在 ${WAIT_DB_TIMEOUT}s 内没就绪，用 ./run.sh logs mysql 看容器日志"
}

start_db() {
  if port_open "$DB_HOST" "$DB_PORT"; then
    ok "检测到 $DB_HOST:$DB_PORT 上已有 MySQL，直接复用（不启动容器）"
    detect_db_via || die "复用了外部 MySQL，但宿主机没有 mysql/mariadb 客户端，装一个或用 ./run.sh stop 让它走容器"
    return 0
  fi
  need docker "请先安装 Docker，或者自行在本机 $DB_PORT 端口跑一个 MySQL"
  docker_ok || die "Docker 守护进程没在运行：sudo systemctl start docker（或 sudo systemctl enable --now docker.service）"
  log "启动 MySQL 容器（${COMPOSE[*]} up -d mysql）…"
  "${COMPOSE[@]}" up -d mysql
  DB_VIA=docker
}

ensure_schema() {
  log "建库建表 + 补种子数据（scripts/db/init.sql，幂等，重复执行没事）"
  sql < scripts/db/init.sql
  ok "表结构就绪"
}

wait_for_app() {
  local waited=0
  log "等待应用启动（http://127.0.0.1:$APP_PORT）…"
  while (( waited < WAIT_APP_TIMEOUT )); do
    if [[ "$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$APP_PORT/api/users/login" || true)" == "200" ]]; then
      ok "应用已就绪（第 $((waited + 1)) 秒）"
      return 0
    fi
    sleep 1
    waited=$((waited + 1))
  done
  die "应用 ${WAIT_APP_TIMEOUT}s 内没起来，看看上面的编译/运行日志"
}

# 用真实 HTTP 请求验证种子账号能不能登录（顺带把整条链路走通一遍）
check_admin_login() {
  local body_file code
  body_file="$(mktemp)"
  code="$(curl -sS -o "$body_file" -w '%{http_code}' \
    -X POST "http://127.0.0.1:$APP_PORT/api/users/login" \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    --data-urlencode "username=$ADMIN_USER" \
    --data-urlencode "password=$ADMIN_PASSWORD" 2>/dev/null || true)"
  if [[ "$code" == "200" ]]; then
    ok "种子账号登录验证通过（$ADMIN_USER / $ADMIN_PASSWORD）→ $(tr -d '\n' < "$body_file")"
  else
    warn "种子账号登录返回 $code：$(tr -d '\n' < "$body_file")"
    warn "如果这是旧的 mysql_data 数据卷，里面的密码可能不是 $ADMIN_PASSWORD；./run.sh reset 可以清库重建"
  fi
  rm -f "$body_file"
}

print_urls() {
  cat <<EOF
  登录演示页(POST 演示)  http://127.0.0.1:$APP_PORT/api/users/login
  注册页                 http://127.0.0.1:$APP_PORT/api/users
  我的信息(需会话)       http://127.0.0.1:$APP_PORT/api/users/me
  修改资料 / 修改密码    /api/users/profiles   /api/users/password
  管理员接口(需 admin)   http://127.0.0.1:$APP_PORT/api/admin/users
  内置账号               $ADMIN_USER / $ADMIN_PASSWORD
EOF
}

stop_app() {
  if [[ -n "$APP_PID" ]] && kill -0 "$APP_PID" 2>/dev/null; then
    log "停止应用（cargo PID $APP_PID）"
    kill "$APP_PID" 2>/dev/null || true
    wait "$APP_PID" 2>/dev/null || true
  fi
  # cargo run 是父进程，真正在监听 8080 的是它的子进程 target/*/mvc_rust；
  # 只杀 cargo 的话服务器会变成孤儿进程继续占着端口，所以再按名字收一遍。
  local leftover
  leftover="$(pgrep -f "$PROJECT_DIR/target/(debug|release)/mvc_rust$|target/(debug|release)/mvc_rust$" 2>/dev/null || true)"
  if [[ -n "$leftover" ]]; then
    log "清理残留的服务进程：$leftover"
    # shellcheck disable=SC2086
    kill $leftover 2>/dev/null || true
    sleep 1
    leftover="$(pgrep -f 'target/(debug|release)/mvc_rust$' 2>/dev/null || true)"
    # shellcheck disable=SC2086
    [[ -n "$leftover" ]] && kill -9 $leftover 2>/dev/null || true
  fi
  APP_PID=""
}

# ---------- 子命令 ----------

cmd_dev() {
  need cargo "请先装 Rust 工具链（rustup）"
  start_db
  wait_for_db
  ensure_schema

  local cargo_args=(run)
  if [[ -n "$CARGO_PROFILE" ]]; then cargo_args+=(--profile "$CARGO_PROFILE"); fi

  log "启动应用：cargo ${cargo_args[*]}（Ctrl-C 退出，数据库容器会留着）"
  cargo "${cargo_args[@]}" &
  APP_PID=$!
  trap 'cleanup' EXIT INT TERM

  wait_for_app
  check_admin_login
  echo
  ok "全部就绪，可以用下面的地址了："
  print_urls
  echo
  log "下面是应用实时日志（Ctrl-C 结束）"
  echo "----------------------------------------------------------------"
  wait "$APP_PID" || true
  echo "----------------------------------------------------------------"
  log "应用已退出，数据库容器还在跑；要一起停掉用：./run.sh stop"
}

cmd_full() {
  need docker "请先安装 Docker"
  docker_ok || die "Docker 守护进程没在运行：sudo systemctl start docker"
  log "全容器模式：${COMPOSE[*]} --profile full up --build（首次构建要编译 actix 全家桶，会慢；Ctrl-C 停止）"
  echo
  echo "  构建完成后："
  print_urls
  echo
  "${COMPOSE[@]}" --profile full up --build
  echo
  log "容器已停止；重新起来用 ./run.sh full，清理用 ./run.sh stop"
}

cmd_db() {
  start_db
  wait_for_db
  ensure_schema
  echo
  ok "数据库就绪，接下来在宿主机跑应用：cargo run"
  echo "  连接串  mysql://$DB_USER:<password>@$DB_HOST:$DB_PORT/$DB_NAME"
  echo "  快速连接  ./run.sh sql"
}

cmd_stop() {
  local mode="${1:-}"
  docker_ok || die "Docker 守护进程没在运行"
  if [[ "$mode" == "--volumes" || "$mode" == "-v" ]]; then
    log "停止容器并删除数据卷（数据库会被清空）"
    "${COMPOSE[@]}" --profile full down --volumes --remove-orphans
  else
    log "停止容器（保留数据卷，数据还在）"
    "${COMPOSE[@]}" --profile full down --remove-orphans
  fi
  # 顺手收掉可能在后台跑着的 cargo run
  pkill -f 'target/(debug|release)/mvc_rust$' 2>/dev/null || true
  ok "已停止"
}

cmd_reset() {
  docker_ok || die "Docker 守护进程没在运行"
  log "重置数据库：停止容器 + 删数据卷"
  "${COMPOSE[@]}" --profile full down --volumes --remove-orphans
  start_db
  wait_for_db
  ensure_schema
  ok "数据库已重建（表是空的，种子账号已补回）"
}

cmd_status() {
  echo "== 端口 =="
  if port_open 127.0.0.1 "$DB_PORT"; then ok "MySQL $DB_PORT 在监听"; else warn "MySQL $DB_PORT 没监听"; fi
  if port_open 127.0.0.1 "$APP_PORT"; then ok "应用 $APP_PORT 在监听"; else warn "应用 $APP_PORT 没监听"; fi

  echo
  echo "== 容器 =="
  if docker_ok; then
    "${COMPOSE[@]}" --profile full ps || true
  else
    warn "Docker 守护进程没在运行"
  fi

  echo
  echo "== 应用健康 =="
  local code
  code="$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$APP_PORT/api/users/login" || true)"
  if [[ "$code" == "200" ]]; then ok "登录页返回 200"; else warn "登录页返回 ${code:-无响应}"; fi

  echo
  echo "== 数据库 =="
  if port_open 127.0.0.1 "$DB_PORT"; then
    if detect_db_via; then
      echo "访问方式：$DB_VIA"
      # 用 USE 选库：sql() 不带默认库，这样首次初始化时才能先 CREATE DATABASE
      printf 'USE `%s`;\nSELECT id, username, name, created_at FROM users;\n' "$DB_NAME" | sql -t \
        || warn "查询失败，表可能还没建，先跑 ./run.sh db"
    else
      warn "没有可用的 mysql 客户端，没法查表"
    fi
  else
    warn "数据库没起来，跳过"
  fi
}

cmd_logs() {
  docker_ok || die "Docker 守护进程没在运行"
  "${COMPOSE[@]}" --profile full logs -f "${1:-app}"
}

cmd_sql() {
  detect_db_via || die "既没有运行中的 mysql 容器，也没装 mysql/mariadb 客户端"
  log "进入 MySQL 客户端（输入 \\q 退出，默认库 $DB_NAME）"
  if [[ "$DB_VIA" == docker ]]; then
    "${COMPOSE[@]}" exec -e MYSQL_PWD="$DB_PASSWORD" mysql mysql -u"$DB_USER" "$DB_NAME"
  else
    "$MYSQL_BIN" --defaults-extra-file="$(mysql_opts_file)" "$DB_NAME"
  fi
}

# ---------- 入口 ----------
trap 'cleanup' EXIT

CMD="${1:-dev}"
if (( $# > 0 )); then shift; fi

case "$CMD" in
  dev|"")     cmd_dev ;;
  full|up)    cmd_full ;;
  db|mysql)   cmd_db ;;
  stop|down)  cmd_stop "${1:-}" ;;
  reset)      cmd_reset ;;
  status|ps)  cmd_status ;;
  logs)       cmd_logs "${1:-app}" ;;
  sql|shell)  cmd_sql ;;
  help|-h|--help) usage ;;
  *)          usage; echo; die "未知命令：$CMD" ;;
esac
