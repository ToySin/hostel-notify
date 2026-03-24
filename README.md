# 선운산유스호스텔 예약 알림봇

고창군 선운산유스호스텔 예약 페이지를 모니터링하여, 취소표(새로 예약 가능해진 객실)가 나오면 Discord로 즉시 알림을 보내는 봇.

## 요구 사항

- Go 1.22+
- Discord Bot Token

## Disclaimer

이 프로젝트는 고창군과 어떠한 제휴도 없는 비공식 도구입니다.
인터넷에서 공개적으로 접근 가능한 정보만을 조회하며,
개인적인 알림 목적으로 제작되었습니다.

이 소프트웨어의 사용으로 발생하는 모든 문제에 대해
개발자는 책임을 지지 않습니다.

This project is an unofficial tool with no affiliation to Gochang-gun.
Use at your own risk.

## 빠른 시작

```bash
# 클론
git clone {repo} && cd hostel-notify

# 설치 (빌드 + ~/apps/hostel-notify/ 에 배포)
./install.sh

# .env 편집 (Discord 봇 토큰 설정)
vi ~/apps/hostel-notify/.env

# 실행
cd ~/apps/hostel-notify && ./run.sh
```

## Discord 봇 생성

1. https://discord.com/developers/applications → **New Application**
2. **Bot** 탭 → **Reset Token** → 토큰 복사
3. **Bot** 탭 → **MESSAGE CONTENT INTENT** 켜기
4. **OAuth2 → URL Generator** → scopes: `bot` → permissions: `Send Messages`, `Read Message History`
5. 생성된 URL로 원하는 서버에 봇 초대

## 디렉토리 구조

```
~/repository/hostel-notify/     ← 소스코드 (git repo)
├── install.sh               # 빌드 + 배포
├── run.sh                   # 실행 스크립트 (install 시 복사됨)
└── *.go                     # 소스 코드

~/apps/hostel-notify/           ← 실행 환경
├── hostel-notify             # 바이너리
├── run.sh                   # 실행 스크립트
├── .env                     # 설정 (최초 1회 생성, 이후 보존)
└── hostel.log               # 로그
```

## 봇 명령어

| 명령어 | 설명 |
|--------|------|
| `!watch 2026-04-05` | 4/5 1박2일 감시 시작 (현재 상태 출력 후 변동시 알림) |
| `!watch 2026-04-05 2` | 4/5 2박3일 감시 시작 |
| `!unwatch 2026-04-05` | 감시 해제 |
| `!list` | 감시 목록 조회 |
| `!check 2026-04-05` | 즉시 1회 조회 (감시 없이) |
| `!help` | 도움말 |

## 동작 방식

```
Discord에서 !watch 2026-04-05 입력
  ↓
현재 상태 즉시 조회 → 결과 출력 (예약가능/예약완료 목록)
  ↓
1분(±40% 지터) 간격으로 폴링
  ↓
변경 감지 시 Discord 알림 (새로 예약가능 / 예약마감)
  ↓
해당 날짜가 지나면 자동 정리
```

## 설정

환경변수 (`.env` 파일):

```bash
# Discord 봇 토큰 (필수)
DISCORD_TOKEN=your-token-here

# 허용 채널 ID (콤마 구분, 비어있으면 기본값)
DISCORD_CHANNEL_IDS=1485933351494750378
```

## 관리

```bash
# 로그 확인
tail -f ~/apps/hostel-notify/hostel.log

# 중지
pkill -f hostel-notify

# 재배포 (코드 수정 후)
cd ~/repository/hostel-notify && git pull && ./install.sh
cd ~/apps/hostel-notify && ./run.sh
```

## macOS 상시 구동 팁

```bash
# 잠자기 방지 (백그라운드 실행)
caffeinate -id &
```

- 전원 어댑터 연결 상태에서 덮개 닫아도 OK
- Discord 연결 끊김 시 discordgo가 자동 재연결
- 크래시 시 run.sh가 10초 후 자동 재시작
