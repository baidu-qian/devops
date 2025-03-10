#!/bin/bash
source ~/.bash_profile
cd $(dirname "$0")

LOG_FILE="../logs/shake_$(date +%Y%m%d).log"
WATCHDOG_PID_FILE="watchdog.pid"

bb_start() {
  pid=$(ps -ef | grep "redis-shake ../config/shake.toml" | grep -v grep | awk '{print $2}')
  if [ -n "$pid" ]; then
    echo "redis-shake is already running. PID: $pid"
    return
  fi
  mkdir -p ../logs
  nohup ./redis-shake ../config/shake.toml > "$LOG_FILE" 2>&1 &
  echo "redis-shake started."
}

bb_stop() {
  pid=$(ps -ef | grep redis-shake | grep -v grep | awk '{print $2}')
  if [ -n "$pid" ]; then
    kill $pid
    sleep 1
    if ps -p $pid > /dev/null; then
      kill -9 $pid
    fi
    echo "redis-shake stopped."
  else
    echo "redis-shake is not running."
  fi

  # 停止 watchdog
  if [ -f "$WATCHDOG_PID_FILE" ]; then
    watchdog_pid=$(cat "$WATCHDOG_PID_FILE")
    if ps -p $watchdog_pid > /dev/null; then
      kill $watchdog_pid
      echo "Watchdog process stopped."
    fi
    rm -f "$WATCHDOG_PID_FILE"
  fi
}

bb_restart() {
  bb_stop
  sleep 2
  bb_start
}

bb_status() {
  pid=$(ps -ef | grep redis-shake | grep -v grep | awk '{print $2}')
  if [ -n "$pid" ]; then
    echo "redis-shake is running. PID: $pid"
  else
    echo "redis-shake is not running."
  fi

  # 检查 watchdog 状态
  if [ -f "$WATCHDOG_PID_FILE" ]; then
    watchdog_pid=$(cat "$WATCHDOG_PID_FILE")
    if ps -p $watchdog_pid > /dev/null; then
      echo "Watchdog is running. PID: $watchdog_pid"
    else
      echo "Watchdog is not running."
    fi
  else
    echo "Watchdog is not running."
  fi
}

bb_watchdog() {
  echo "Starting watchdog to monitor redis-shake..."
  
  # 确保只启动一个 watchdog
  if [ -f "$WATCHDOG_PID_FILE" ]; then
    echo "Watchdog is already running."
    exit 0
  fi

  # 启动 watchdog
  while true; do
    pid=$(ps -ef | grep "redis-shake ../config/shake.toml" | grep -v grep | awk '{print $2}')
    if [ -z "$pid" ]; then
      echo "$(date): redis-shake is not running. Restarting..." >> "$LOG_FILE"
      bb_start
    fi
    sleep 180  # 每180秒检查一次
  done &
  
  # 记录 watchdog 的 PID
  echo $! > "$WATCHDOG_PID_FILE"
  echo "Watchdog started with PID $(cat $WATCHDOG_PID_FILE)."
}

case $1 in
# 无watchdog启动
nodogstart)
  bb_start
  ;;
start)
  bb_start
  bb_watchdog
  ;;
stop)
  bb_stop
  ;;
restart)
  bb_restart
  ;;
status)
  bb_status
  ;;
watchdog)
  bb_watchdog
  ;;
*)
  echo "Usage: { start | stop | restart | status | watchdog }"
  exit 1
  ;;
esac
