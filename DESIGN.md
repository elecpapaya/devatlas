
# 지역별 소프트웨어 채용·산업 분석 시스템 설계 문서 (v1)

## 1. 목적 (Goals)

본 시스템의 목적은 다음 질문에 데이터 기반으로 답변하는 것이다.

1. 어느 지역이 소프트웨어 사업을 잘하고 있는가?
2. 어느 지역이 소프트웨어 개발자를 많이 채용하고 있는가?
3. 현재 특정 지역에서 어떤 기업이 채용을 진행 중인지 구직자가 지도에서 확인할 수 있는가?

이를 위해:
- 채용 데이터를 주기적으로 수집
- 지역 단위로 집계·시계열화
- 결과를 정적 웹 페이지(GitHub Pages)로 배포한다.

본 시스템은 분석·공유 중심이며, 원천 데이터 전체를 공개하지 않는다.

---

## 2. 핵심 설계 원칙

### 2.1 수집과 공개의 분리
- 로컬 환경: 원천 수집 데이터, 상세 로그, 품질 관리
- GitHub Pages: 시각화 가능한 최소 집계 데이터 + 최신 상태만 공개

### 2.2 스냅샷 기반 시계열
- 매 실행(run)은 하나의 시점 스냅샷
- 시계열 히스토리는 차트로 표현 가능한 집계 데이터만 포함

### 2.3 덮어쓰기 배포
- GitHub에는 항상 최신 번들 1개만 유지
- 히스토리는 최신 번들 내부에 포함 (요약 형태)

### 2.4 정적 사이트 우선
- 서버 없음
- API 없음
- GitHub Pages + JSON 데이터 fetch 방식

### 2.5 단일 소스 우선
- 초기(v1) 수집은 하나의 구인 사이트에 집중
- 복수 소스 혼합은 포맷/콘텐츠 불일치 리스크로 인해 보류
- 향후 확장 시에도 소스별 스키마/정규화 규칙을 먼저 정의

### 2.6 MVP 우선
- 꼭 필요하지 않은 기능은 후순위로 미루고 MVP 기능에 집중
- 초기 구현은 동작 확인과 데이터 흐름 검증을 목표로 한다

---

## 3. 전체 아키텍처 개요

[로컬 스케줄러]
  └─ 수집 → 정제 → 매칭 → 집계 → 번들 생성
       └─ 최신 데이터만 GitHub push
            └─ GitHub Pages (지도 + 차트)

---

## 4. 실행 주기 및 오케스트레이션

### 4.1 실행 주기
- 1일 1회 (cron / Windows Task Scheduler)
- 실행 시점(run_at)을 기준으로 데이터 스냅샷 생성

### 4.2 Run 개념
각 실행(run)은 다음을 가진다:
- run_at: 실행 기준 시각
- window: 이전 run 이후 ~ 현재 run
- 모든 집계는 해당 run 기준으로 계산됨

---

## 5. 데이터 처리 파이프라인 (로컬)

### 5.1 수집 (Collector)
- 단일 채용 데이터 소스(API/허용된 웹 페이지)
- 신규 또는 갱신된 채용 공고 수집
- v1 권장 소스: 사람인 API
- 대안: 원티드(Wanted) OpenAPI

### 5.1.1 사람인 API 수집 필드/매핑 (v1)
- API: `https://oapi.saramin.co.kr/job-search` (JSON)
- 응답 루트: `jobs.job[]`
- `fields` 옵션으로 확장되는 항목: `posting-date`, `expiration-date`, `count`

| 내부 필드 | Saramin JSON 경로 | 비고 |
| --- | --- | --- |
| source | - | `"saramin"` 고정 |
| source_job_id | `jobs.job[].id` | 필수 |
| source_url | `jobs.job[].url` | 필수 |
| active | `jobs.job[].active` | 1=진행중, 0=마감 |
| company_name | `jobs.job[].company.detail.name` | 필수 |
| company_url | `jobs.job[].company.detail.href` | 공개된 경우만 |
| title | `jobs.job[].position.title` | 필수 |
| industry_code | `jobs.job[].position.industry.code` | 선택 |
| industry_name | `jobs.job[].position.industry.name` | 선택 |
| location_code | `jobs.job[].position.location.code` | 다중 코드(콤마 구분) 가능 |
| location_name | `jobs.job[].position.location.name` | `시/도` 추출에 사용 |
| job_type_code | `jobs.job[].position.job-type.code` | 선택 |
| job_type_name | `jobs.job[].position.job-type.name` | 선택 |
| job_mid_code | `jobs.job[].position.job-mid-code.code` | 개발자 여부 판단 입력 |
| job_mid_name | `jobs.job[].position.job-mid-code.name` | 개발자 여부 판단 입력 |
| job_code | `jobs.job[].position.job-code.code` | 개발자 여부 판단 입력 |
| job_name | `jobs.job[].position.job-code.name` | 개발자 여부 판단 입력 |
| experience_code | `jobs.job[].position.experience-level.code` | 선택 |
| experience_min | `jobs.job[].position.experience-level.min` | 선택 |
| experience_max | `jobs.job[].position.experience-level.max` | 선택 |
| education_code | `jobs.job[].position.required-education-level.code` | 선택 |
| education_name | `jobs.job[].position.required-education-level.name` | 선택 |
| keyword | `jobs.job[].keyword` | 개발자 여부 판단 보조 |
| salary_code | `jobs.job[].salary.code` | 선택 |
| salary_name | `jobs.job[].salary.name` | 선택 |
| posting_ts | `jobs.job[].posting-timestamp` | Unix seconds |
| modification_ts | `jobs.job[].modification-timestamp` | Unix seconds |
| opening_ts | `jobs.job[].opening-timestamp` | Unix seconds |
| expiration_ts | `jobs.job[].expiration-timestamp` | Unix seconds |
| expiration_date | `jobs.job[].expiration-date` | `fields=expiration-date` 필요 |
| close_type_code | `jobs.job[].close-type.code` | 선택 |
| close_type_name | `jobs.job[].close-type.name` | 선택 |
| read_cnt | `jobs.job[].read-cnt` | `fields=count` 필요 |
| apply_cnt | `jobs.job[].apply-cnt` | `fields=count` 필요 |

처리 규칙 (v1):
- `location_code`가 다중일 경우 분리 저장, 집계는 지역 단위로 각각 반영
- `location_name`에서 시/도 문자열을 추출해 지역 매핑
- 개발자 여부는 `job_mid_code`/`job_code`를 1차 기준으로 사용하고, 부족 시 `keyword`로 보완

### 5.1.2 사람인 API 요청 파라미터(v1) 범위/쿼리 전략
요청 헤더:
- `Accept: application/json`

요청 파라미터 (v1에서 사용하는 범위):

| 파라미터 | 사용 | 비고 |
| --- | --- | --- |
| access-key | 필수 | 발급 키 |
| job_cd | 기본 | `5.2.1`의 개발자 직무 코드 목록을 다중 지정 |
| job_mid_cd | 선택 | 넓은 수집 모드에서 `2(IT개발·데이터)` 사용 |
| loc_cd | 선택 | 지역 제한 시 사용, 미설정 시 전 지역 |
| sr | 선택 | `directhire`로 헤드헌팅/파견 공고 제외 |
| fields | 기본 | `posting-date,expiration-date,count` |
| updated_min / updated_max | 기본 | 증분 수집 기준 (run window) |
| published_min / published_max | 초기 | 초기 백필 기간 설정 |
| start | 기본 | 0 기반 페이지 인덱스 |
| count | 기본 | 페이지 크기, 기본 10, 최대 110 |
| sort | 기본 | `ud`(최근수정순) 또는 `pd`(게시일 역순) |

쿼리 전략 (v1):
- 초기 백필: `published_min`/`published_max`로 기간을 나눠 수집
- 일일 증분: `updated_min=prev_run_at`, `updated_max=run_at`
- 필터링 우선순위: `job_cd` 목록 기반 수집 → 필요 시 `job_mid_cd=2` 확장
- 페이지네이션: `start=0`부터 `count=110`으로 반복, `jobs.total` 기준 종료

### 5.1.3 백필 정책 (가벼운 운영 기준)
- 초기 백필: 최근 180일만 대상으로 30일 단위로 분할 수집 (총 6회)
- 누락 복구: 매일 증분 수집 후 최근 7일 범위 1회 재수집
- 규칙 변경: 변경일 기준 최근 30일만 재처리

### 5.1.4 run 기준 타임라인 (증분/재수집)
```text
시간 ---->

초기 백필 (총 6회):
[run_at-180d ........ run_at-150d] [run_at-150d ........ run_at-120d]
[run_at-120d ........ run_at-90d ] [run_at-90d  ........ run_at-60d ]
[run_at-60d  ........ run_at-30d ] [run_at-30d  ........ run_at     ]

일일 증분:
[prev_run_at ......... run_at]

일일 재수집(누락 복구):
[run_at-7d ........... run_at]
```

### 5.2 정규화 (Normalizer)
- 회사명
- 채용 직무(소프트웨어 개발자 여부)
- 근무 지역(주소/지역명)
- 공고 링크

### 5.2.1 개발자 직무 코드 목록 (사람인, v1 초안)
- 기준: 사람인 직무 코드표 `IT개발·데이터(mcode=2)`
- 역할/도메인 중심 1차 포함 리스트이며, 운영 중 보완
- 기술 스택(언어/프레임워크) 코드는 `keyword` 기반으로 보완

#### 앱/웹/모바일
| 코드 | 직무 키워드명 |
| --- | --- |
| 84 | 백엔드/서버개발 |
| 92 | 프론트엔드 |
| 2232 | 풀스택 |
| 86 | 앱개발 |
| 195 | Android |
| 234 | iOS |
| 87 | 웹개발 |
| 113 | 반응형웹 |
| 124 | 웹표준·웹접근성 |
| 2249 | 클라이언트 |
| 103 | 검색엔진 |
| 135 | 크롤링 |
| 142 | API |

#### 데이터/AI
| 코드 | 직무 키워드명 |
| --- | --- |
| 82 | 데이터분석가 |
| 83 | 데이터엔지니어 |
| 2248 | 데이터 사이언티스트 |
| 2246 | BI 엔지니어 |
| 106 | 데이터마이닝 |
| 107 | 데이터시각화 |
| 116 | 빅데이터 |
| 108 | 딥러닝 |
| 109 | 머신러닝 |
| 181 | AI(인공지능) |
| 160 | NLP(자연어처리) |
| 161 | NLU(자연어이해) |
| 133 | 컴퓨터비전 |
| 123 | 영상처리 |
| 162 | OCR |
| 171 | STT |
| 172 | TTS |
| 131 | 챗봇 |
| 148 | DW |
| 150 | ETL |

#### 인프라/클라우드/임베디드/DB
| 코드 | 직무 키워드명 |
| --- | --- |
| 100 | SE(시스템엔지니어) |
| 101 | SI개발 |
| 104 | 네트워크 |
| 127 | 인프라 |
| 136 | 클라우드 |
| 146 | DevOps |
| 156 | IoT |
| 128 | 임베디드 |
| 320 | 임베디드리눅스 |
| 139 | 펌웨어 |
| 180 | 아키텍쳐 |
| 95 | DBA |
| 145 | DBMS |
| 164 | RDBMS |

#### 보안/품질
| 코드 | 직무 키워드명 |
| --- | --- |
| 90 | 정보보안 |
| 85 | 보안컨설팅 |
| 2239 | 보안관제 |
| 111 | 모의해킹 |
| 132 | 취약점진단 |
| 99 | QA/테스터 |
| 2229 | SQA |

#### 게임
| 코드 | 직무 키워드명 |
| --- | --- |
| 80 | 게임개발 |

### 5.3 회사 단위 집계
- 동일 회사의 다수 공고를 회사 단위로 묶음
- 최근 N일 내 관측 기준으로 채용 진행 여부 판단

### 5.4 재무 정보 결합 (선택)
- 상장사의 경우 DART 기준 영업이익 여부
- 지역별 흑자 기업 채용 지표 생성

### 5.5 지오코딩
- 근무지 주소 → 좌표
- 실패 시 지역 centroid 사용
- 동일 주소는 캐시 재사용

---

## 6. 공개 데이터 설계 (GitHub Pages)

### 6.1 공개 범위
- 원문 공고 ❌
- 상세 기업 정보 ❌
- 집계 시계열 + 최신 채용 기업 위치 ⭕

---

## 7. 데이터 파일 스펙 (v1)

### 7.1 지역별 시계열 집계 번들
파일: data/latest_bundle.json (매일 덮어쓰기)

포함 데이터:
- 최근 180~365일
- 지역 단위: 시/도 (v1)

스키마 예시:
```json
{
  "meta": {
    "run_at": "2026-01-19T00:10:00+09:00",
    "region_level": "sido",
    "days": 365
  },
  "series": [
    {
      "date": "2026-01-01",
      "region": "서울",
      "dev_posts": 120,
      "hiring_companies": 45,
      "profitable_companies": 18
    }
  ]
}
```

---

### 7.2 현재 채용 중 기업 지도 데이터
파일: data/latest_companies.json (매일 덮어쓰기)

스키마 예시:
```json
{
  "meta": {
    "run_at": "2026-01-19T00:10:00+09:00",
    "region_level": "sido"
  },
  "companies": [
    {
      "name": "A사",
      "lat": 37.5665,
      "lng": 126.9780,
      "region": "서울",
      "url": "https://company-or-jobsite-link",
      "asof": "2026-01-19"
    }
  ]
}
```

---

### 7.3 지역 집계 결과 (v1 최소)
파일: data/region_counts.json (선택적, 매일 덮어쓰기)

스키마 예시:
```json
{
  "meta": {
    "run_at": "2026-01-19T00:10:00+09:00",
    "window_start": "2026-01-18T00:10:00+09:00",
    "window_end": "2026-01-19T00:10:00+09:00",
    "missing_regions": 12
  },
  "regions": [
    {
      "region": "서울",
      "job_count": 120,
      "company_count": 45
    }
  ]
}
```

## 8. 현재 채용 중 정의
- last_seen_date >= run_at - N일
- N 기본값: 14~30일
- 공고 마감 여부 대신 최근 관측 기준 사용

---

## 9. GitHub Pages UI 구성

### 9.1 주요 화면
1. 트렌드 분석
   - 지역별 개발자 채용 추이
   - 흑자 기업 채용 비율
2. 채용 기업 지도
   - 기업 마커 표시
   - 클릭 시 외부 채용 페이지 이동

---

## 10. 배포 전략

### 10.1 로컬 → GitHub
- 로컬에서 데이터 생성
- Pages 전용 레포에 JSON 파일 덮어쓰기
- 변경 있을 때만 commit/push

### 10.2 GitHub 역할
- 데이터 저장 ❌
- 수집 ❌
- 시각화 + 공유 ⭕

---

## 11. 확장 계획 (v2 이후)
- 지역 단위: 시군구
- 시간 단위: 주 단위 리샘플링
- 직무/스택 필터
- 흑자 지속성 기반 지수
- 지역 소프트웨어 산업 지수 산출

---

## 12. 요약
본 시스템은:
- 무거운 데이터는 로컬
- 가벼운 분석 결과만 공개
- 정적 페이지로 충분한 인사이트 제공
을 목표로 한다.
