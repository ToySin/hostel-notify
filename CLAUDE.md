# CLAUDE.md

## What This Is

선운산유스호스텔(고창군) 예약 페이지의 취소표를 감지해서 Discord로 알림 보내는 개인용 봇.
고창군과 무관한 비공식 도구이며, 공개 페이지만 조회한다.

## Key Context (코드에서 안 보이는 것들)

- 대상 사이트(`gochang.go.kr`)는 운영시간(09:00~22:00 KST) 외에 rooms=0을 반환함 — 이걸 "빈방 없음"으로 처리하면 오탐이 되므로 0건은 스킵해야 함
- 폴링 지터(±40%)는 서버 부하 분산 목적. 고창군 사이트가 소규모라 예의 차원
- 객실 식별자 `RoomSid`는 HTML 내 `writeFunc()` JS 호출에서 추출 — 사이트 구조 바뀌면 정규식(`writeFuncRe`) 깨질 수 있음
- 소스(`~/repository/hostel-notify/`)와 실행환경(`~/apps/hostel-notify/`)이 분리되어 있음. `.env`는 실행환경에만 존재하고 `install.sh`가 최초 생성 후 보존
- macOS에서 caffeinate + 덮개 닫고 상시 구동하는 환경을 전제로 만들어짐

## Testing

테스트(`scraper_test.go`)는 실제 사이트에 요청하는 통합 테스트 — mock 없음, 운영시간 외에는 실패할 수 있음
