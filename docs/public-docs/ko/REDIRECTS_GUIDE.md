# Mintlify 리디렉션 가이드

Mintlify 문서(Nextra에서 마이그레이션됨)의 리디렉션 관리를 위한 종합 가이드입니다.

---

## 목차

- [개요](#overview)
- [리디렉션 추가 방법](#how-to-add-redirects)
- [예시](#examples)
- [제한 사항](#limitations)

---

## 개요

리디렉션은 문서 구조를 재편(IA 리팩터링)할 때 다음을 보장하기 위해 필수적입니다.

- 기존 링크가 깨지지 않음
- 검색 엔진 순위 유지
- 사용자 북마크가 계속 작동함
- 외부 링크가 유효하게 유지됨

**위치**: 모든 Mintlify 리디렉션은 `docs/public-docs/docs.json`에서 설정합니다

---

## 리디렉션 추가 방법

### 1단계: `docs.json` 열기

Mintlify 문서 루트로 이동하여 `docs/public-docs/docs.json`을 엽니다.

### 2단계: `redirects` 배열 찾기

`redirects` 배열 / 키를 찾습니다.

```json
{
  "$schema": "https://mintlify.com/docs.json",
  "theme": "mint",
  "name": "Optimism Documentation",
  ...
  "redirects": [
    // Redirects go here
  ],
  ...
}
```

### 3단계: 리디렉션 추가

배열에 새 객체를 추가합니다.

```json
{
  "redirects": [
    {
      "source": "/old-page-path",
      "destination": "/new-page-path"
    }
  ]
}
```

### 4단계: 로컬에서 테스트

```bash
# Run the dev server
mint dev

# Test the redirect
# Navigate to http://localhost:3000/old-page-path
# Should redirect to /new-page-path
```

---

## 예시

### 예시 1: 단순 페이지 리디렉션

**시나리오**: `getting-started.mdx`의 이름을 `quickstart.mdx`로 변경한 경우

```json
{
  "source": "/getting-started",
  "destination": "/quickstart"
}
```

### 예시 2: 섹션 재구성

**시나리오**: 모든 튜토리얼을 `/docs/`에서 `/tutorials/`로 이동한 경우

```json
{
  "source": "/docs/deploy-contract",
  "destination": "/tutorials/deploy-contract"
},
{
  "source": "/docs/setup-wallet",
  "destination": "/tutorials/setup-wallet"
},
{
  "source": "/docs/bridge-tokens",
  "destination": "/tutorials/bridge-tokens"
}
```

**참고**: 페이지마다 리디렉션이 하나씩 필요합니다(와일드카드 미지원!)

### 예시 3: 대규모 재구성

**시나리오**: `app-developers/tools/supersim.mdx`를 `interop/tools/supersim.mdx`로 이동한 경우

```json
{
  "source": "/app-developers/tools/supersim",
  "destination": "/interop/tools/supersim"
}
```

### 예시 4: 삭제된 페이지

**시나리오**: 페이지를 삭제하고 관련 페이지로 리디렉션하려는 경우

```json
{
  "source": "/deprecated-feature",
  "destination": "/new-feature-overview"
}
```

### 예시 5: 외부 리디렉션

**시나리오**: 콘텐츠를 외부 문서로 이동한 경우

```json
{
  "source": "/old-specs",
  "destination": "https://specs.optimism.io"
}
```

---

## 제한 사항

### 와일드카드 미지원

**Nextra** (지원됨):

```
/docs/* /tutorials/:splat 301
```

**Mintlify** (지원되지 않음):

```json
{
  "source": "/docs/*",
  "destination": "/tutorials/*"
}
```

**우회 방법**: 각 페이지마다 개별 리디렉션을 생성합니다.

```json
{
  "redirects": [
    { "source": "/docs/page1", "destination": "/tutorials/page1" },
    { "source": "/docs/page2", "destination": "/tutorials/page2" },
    { "source": "/docs/page3", "destination": "/tutorials/page3" }
  ]
}
```
