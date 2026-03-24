#!/bin/bash

# 선운산유스호스텔 알림봇 설치 스크립트
# 빌드 + ~/apps/hostel-notify/ 에 배포

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_DIR="$HOME/apps/hostel-notify"

# 1. Go 확인
if ! command -v go &>/dev/null; then
    echo "❌ Go가 설치되어 있지 않습니다. brew install go"
    exit 1
fi

# 2. 빌드
echo "📦 빌드 중..."
cd "$SCRIPT_DIR"
go build -o hostel-notify . || exit 1
echo "✅ 빌드 완료"

# 3. 앱 디렉토리 세팅
mkdir -p "$APP_DIR"
cp hostel-notify "$APP_DIR/hostel-notify"
cp run.sh "$APP_DIR/run.sh"
chmod +x "$APP_DIR/run.sh"

# 4. .env 파일 (최초 1회만)
if [ ! -f "$APP_DIR/.env" ]; then
    cat > "$APP_DIR/.env" <<'EOF'
# Discord 봇 토큰 (필수)
DISCORD_TOKEN=

# 허용 채널 ID (콤마 구분, 비어있으면 기본값 사용)
# DISCORD_CHANNEL_IDS=1485933351494750378
EOF
    echo "📋 .env 생성 → $APP_DIR/.env (DISCORD_TOKEN 설정 필요)"
else
    echo "📋 .env 이미 존재 — 건너뜀"
fi

# 5. 정리
rm -f hostel-notify

echo ""
echo "✅ 설치 완료!"
echo ""
echo "  앱 디렉토리: $APP_DIR"
echo "  실행: cd $APP_DIR && ./run.sh"
echo ""
echo "  ⚠️  처음이면 .env에서 DISCORD_TOKEN을 설정하세요"
