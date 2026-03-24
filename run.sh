#!/bin/bash

# 선운산유스호스텔 알림봇 실행 스크립트
# ~/apps/hostel-notify/ 에서 실행
# 사용법: ./run.sh

APP_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ ! -f "$APP_DIR/hostel-notify" ]; then
    echo "❌ hostel-notify 바이너리가 없습니다. install.sh를 먼저 실행하세요."
    exit 1
fi

# .env 로드
if [ -f "$APP_DIR/.env" ]; then
    set -a
    source "$APP_DIR/.env"
    set +a
fi

if [ -z "$DISCORD_TOKEN" ]; then
    echo "❌ DISCORD_TOKEN이 설정되지 않았습니다. .env 파일을 확인하세요."
    exit 1
fi

# 기존 프로세스 종료
if pgrep -f "hostel-notify" >/dev/null 2>&1; then
    echo "🔄 기존 프로세스 종료..."
    pkill -f "hostel-notify" 2>/dev/null
    sleep 2
fi

# 백그라운드 실행 (크래시 시 자동 재시작)
echo "🚀 백그라운드 실행..."
nohup bash -c '
cd "'"$APP_DIR"'"
set -a; source .env 2>/dev/null; set +a
while true; do
    echo "$(date) [run.sh] hostel-notify 시작"
    ./hostel-notify
    echo "$(date) [run.sh] 프로세스 종료됨, 10초 후 재시작..."
    sleep 10
done
' > "$APP_DIR/hostel.log" 2>&1 &

echo ""
echo "✅ 실행 완료! (PID: $!)"
echo ""
echo "  로그 확인: tail -f $APP_DIR/hostel.log"
echo "  중지:      pkill -f hostel-notify"
