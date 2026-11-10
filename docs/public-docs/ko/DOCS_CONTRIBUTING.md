# Optimism Docs에 기여하기

기여해 주셔서 감사합니다! ❤️

## 목차
- [Optimism Docs에 기여하기](#contributing-to-optimism-docs)
  - [목차](#table-of-contents)
  - [개요](#overview)
  - [시작하기](#getting-started)
    - [사전 요구 사항](#prerequisites)
    - [개발 환경 설정](#development-setup)
  - [기여 절차](#contributing-process)
    - [파일 구조](#file-architecture)
    - [콘텐츠 가이드라인](#content-guidelines)
    - [공유 스니펫](#shared-snippets)
    - [로컬 테스트](#local-testing)
    - [Docs lint](#docs-lints)
    - [릴리스되지 않은 변경 사항 문서화](#documenting-unreleased-changes)
  - [Pull Request 절차](#pull-request-process)
    - [제출 전 확인 사항](#before-submitting)
    - [제출 가이드라인](#submission-guidelines)
    - [리뷰 절차](#review-process)
  - [행동 강령](#code-of-conduct)
  - [기타 기여 방법](#additional-ways-to-contribute)

## 개요

Optimism 문서는 오픈 소스이며, GitHub의 [`ethereum-optimism/optimism`](https://github.com/ethereum-optimism/optimism) 모노레포 내 [`docs/public-docs`](https://github.com/ethereum-optimism/optimism/tree/develop/docs/public-docs) 경로에서 호스팅됩니다. 문서는 [docs.optimism.io](https://docs.optimism.io)에서 렌더링됩니다. 다음 방법 중 하나로 기여할 수 있습니다.
- `optimism` 저장소를 포크하여 로컬에서 작업
- 작은 수정의 경우 각 문서 페이지의 "Suggest edits" 버튼 사용

현재 모든 기여, pull request, issue는 영어로 작성해야 합니다. 

## 시작하기

### 사전 요구 사항
- Git 및 GitHub에 대한 기본 지식
- Markdown에 대한 이해
- 기술 문서 작성 원칙에 대한 이해
- Node.js 및 npm 설치

### 개발 환경 설정
1. [pnpm](https://pnpm.io/installation)을 설치합니다
2. `pnpm i`를 실행하여 의존성을 설치합니다
3. `pnpm dev`를 실행하여 개발 서버를 시작합니다
4. [localhost:3000](http://localhost:3000)에 접속합니다

이제 콘텐츠를 수정할 수 있으며, 새 파일을 저장할 때마다 웹사이트가 실시간으로 업데이트되는 것을 확인할 수 있습니다. 🤓

## 기여 절차

### 파일 구조

[mintlify 문서](https://www.mintlify.com/docs/organize/navigation)를 참고하십시오.

**경고**: `public` 폴더에는 SEO 목적의 `robots.txt` 및 `sitemap.xml` 파일이 포함되어 있습니다. 이 파일들은 Documentation 팀만 관리합니다.

### 콘텐츠 가이드라인
문서는 [mintlify](https://www.mintlify.com/docs)를 기반으로 운영됩니다.

자세한 서식 지침은 종합 [스타일 가이드](https://docs.optimism.io/op-stack/contribute/style-guide) ([소스](op-stack/contribute/style-guide.mdx))를 참고하십시오.

새 콘텐츠를 추가하기 전에 [콘텐츠 가이드](https://docs.optimism.io/op-stack/contribute/content-guide) ([소스](op-stack/contribute/content-guide.mdx))를 확인하십시오. 이 가이드는 docs.optimism.io에 포함될 콘텐츠의 범위, 각 콘텐츠 유형의 정식 위치, 그리고 서드파티 콘텐츠를 표시하는 방법을 정의합니다.

### 공유 스니펫

모든 사실은 한 번만 기술하고 다른 곳에서는 이를 참조해야 합니다. 동일한
산문 블록(경고, 면책 조항, 서문 등)이 두 개 이상의 페이지에 나타나는 경우,
복사하는 대신 [Mintlify 스니펫](https://www.mintlify.com/docs/create/reusable-snippets)으로
추출하십시오. `snippets/`는 Mintlify의 예약 폴더이며, 이 폴더의 파일은
독립적인 페이지로 렌더링되지 않습니다.

**배치 규칙**

- 기계가 생성한 스니펫은 `snippets/generated/` 아래에 위치하며
  수정 금지(do-not-edit) 출처 헤더를 포함합니다. 이 스니펫은 생성 파이프라인이 소유하며,
  직접 수정하면 drift 검사에 실패합니다.
- 직접 관리하는 공유 산문은 `snippets/` 루트에 kebab-case 파일로 위치하며,
  파일 맨 앞에 목적, 사용법, 사용 페이지를 명시하는 `{/* ... */}` 주석을 둡니다
  (선례: `snippets/third-party-content.mdx`,
  `snippets/op-geth-eol.mdx`). import를 추가하거나 제거할 때는 헤더 주석의
  사용 페이지 목록을 최신 상태로 유지하십시오.

**Import 형식**

- 일반 포함(plain include): 페이지가 스니펫을 default export로 import하고
  컴포넌트로 렌더링합니다.

  ```mdx
  import OpGethEol from "/snippets/op-geth-eol.mdx"

  <OpGethEol />
  ```

- 매개변수화된 포함(parameterized include): 스니펫이 props를 받는 화살표 함수 컴포넌트를
  export하고, 이름으로 import합니다(선례: `snippets/normative-spec.mdx`).

  ```mdx
  import { NormativeSpec } from "/snippets/normative-spec.mdx"

  <NormativeSpec what="..." title="..." href="..." />
  ```

**추출 규칙**

원문 그대로이거나 거의 동일한 블록만 추출하십시오. 특정 독자를 위한 표현은
스니펫 밖, 페이지에 그대로 둡니다. 사본들의 내용이 서로 달라졌다면, 먼저
신뢰할 수 있는 원본(source of truth)을 기준으로 문구를 맞춘 뒤 추출하십시오.
독자에 따라 다르게 읽혀야 하는 산문은 매개변수화하지 마십시오.

### 로컬 테스트

로컬 변경 사항은 이 [문서](https://www.mintlify.com/docs/installation)를 따르십시오.

### Docs lint

`docs/public-docs/`를 수정하는 모든 변경에는 세 가지 결정론적 검사가
적용됩니다. 처음 두 가지는 `docs.json`(내비게이션 + 리디렉션)을 검사하고,
세 번째는 생성된 컴포넌트 레퍼런스를 검사합니다. 이 검사들은 모든 docs
pull request에서 CI로 실행됩니다(CI job은
[ethereum-optimism/solutions#1518](https://github.com/ethereum-optimism/solutions/issues/1518)에서
추적 중이며, 반영되기 전까지는 push 전에 로컬에서 실행하는 것이 관문 역할을 합니다).

- **Nav validator** (`scripts/lint/validate-nav.ts`): 디스크에 있는 모든 `.mdx` 파일은
  `docs.json` 내비게이션에서 접근 가능하거나, 사유 문자열과 함께
  `scripts/lint/nav-allowlist.json`에 명시적으로 허용 목록으로 등록되어 있어야 합니다. 중복 nav
  항목이 없어야 하며, 파일이 없는 nav 항목도 없어야 합니다.
- **Redirect lint** (`scripts/lint/validate-redirects.ts`): 페이지를 삭제하거나
  이동하는 경우 **같은 PR에서** 리디렉션을 추가해야 합니다(
  [리디렉션 가이드](REDIRECTS_GUIDE.md) 참고). 연쇄 리디렉션, 존재하지 않는 대상으로의
  리디렉션, 중복된 리디렉션 소스, 실제 페이지를 가리는 리디렉션 소스, 존재하지 않는
  경로로의 내부 링크가 없어야 합니다.
- **Reference lint** (`scripts/lint/validate-reference.ts`):
  [컴포넌트 레퍼런스](https://docs.optimism.io/op-stack/contribute/content-guide#component-reference)
  규칙을 강제합니다. 최상위 `reference/` 디렉터리 아래의 모든 페이지는 생성된
  `DO NOT EDIT` 헤더로 시작해야 합니다(직접 작성한 콘텐츠는 가이드에 속합니다). 모든
  `<Unreleased>` 콜아웃은 지원 대상 컴포넌트와, 있는 경우 유효한 버전을 명시해야 합니다
  (이미 릴리스 태그가 지정된 콜아웃은 여기서는 경고이지만, lint를 `--strict`로 실행하는
  주간 CI 점검에서는 실패로 처리되므로 해당 콜아웃을 제거하게 됩니다). 생성기 매니페스트의
  모든 태그는 실제 git 태그여야 합니다.

push 전에 로컬에서 실행하십시오.

```bash
# from docs/public-docs (uses the tsx devDependency)
pnpm lint:nav
pnpm lint:redirects
pnpm lint:reference

# or from the monorepo root with bun (zero-dependency)
bun docs/public-docs/scripts/lint/validate-nav.ts
bun docs/public-docs/scripts/lint/validate-redirects.ts
bun docs/public-docs/scripts/lint/validate-reference.ts
```

검사 도입 이전부터 존재하던 위반 사항은
`scripts/lint/nav-allowlist.json` 및 `scripts/lint/redirect-lint-baseline.json`에 예외로 등록되어 있습니다.
이 파일들은 줄어들기만 합니다. PR에서 예외로 등록된 위반 사항을 수정했다면, 같은 PR에서 해당
항목을 제거하십시오(오래된 항목이 남아 있으면 검사에 실패합니다). 새로운 위반을 숨기기 위해
baseline 항목을 추가해서는 안 됩니다. 대신 누락된 리디렉션이나 nav 항목을 추가하십시오.
허용 목록은 의도적으로 목록에서 제외한 페이지에만 사용하며, 그 사유를 기록해야 합니다.

### 릴리스되지 않은 변경 사항 문서화

사이트는 `develop`에서 배포되고, 컴포넌트는 릴리스 태그에서 출시됩니다. 컴포넌트
PR이 동작을 변경하는 경우, 같은 PR에서 직접 작성한 가이드를 업데이트하고
변경된 문장을 `<Unreleased component="…" />`로 감싸십시오(릴리스가 이미 정해져 있다면
`version="…"`를 추가하십시오. `snippets/unreleased.mdx` 참고).
생성된 레퍼런스 페이지는 수정하지 마십시오. 이 페이지들은 다음 확정 태그에서 다시 생성되며,
`develop` 버전은 병합될 때마다 다시 생성됩니다. 전체 규칙은
[컴포넌트 레퍼런스 › 릴리스되지 않은 변경 사항 문서화](https://docs.optimism.io/op-stack/contribute/content-guide#documenting-unreleased-changes)를 참고하십시오.

## Pull Request 절차

### 제출 전 확인 사항
- 보고된 문제를 모두 수정합니다
- 콘텐츠의 정확성을 검증합니다
- 모든 링크와 참조를 테스트합니다
- `develop` 브랜치를 대상으로 합니다

### 제출 가이드라인
1. [새 pull request](https://github.com/ethereum-optimism/optimism/compare)를 생성합니다
2. 적절한 PR 유형을 선택하거나 빈 템플릿을 사용합니다
3. 명확한 제목과 정확한 설명을 작성합니다
4. 라벨을 추가합니다

> **중요**: PR 콘텐츠가 제품 릴리스와 동시에 공개되어야 하는 경우 `flag:merge-pending-release` 라벨을 추가하십시오.

> **팁**: 작업이 아직 진행 중이라면 "[Create draft pull request](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request)"를 사용하십시오.

### 리뷰 절차
1. Documentation 팀 구성원에게 할당
2. 정확성에 대한 기술 리뷰
3. 품질 및 범위 적합성 검토
4. 최소 1명의 리뷰어 승인 필요
5. 리뷰어는 승인하거나, 변경을 요청하거나, 코멘트와 함께 pull request를 닫습니다
6. 병합 후 [docs.optimism.io](https://docs.optimism.io)에 자동 배포

## 행동 강령
- 서로 존중하고 포용적으로 행동합니다
- 프로젝트 가이드라인을 따릅니다
- 건설적인 피드백을 제공합니다
- 전문적인 소통을 유지합니다
- 부적절한 행동을 신고합니다

## 기타 기여 방법
직접 코드를 기여하지 않더라도 다음과 같은 방법으로 지원할 수 있습니다.
- ⭐ 프로젝트에 Star 누르기
- 🐦 소셜 미디어에 공유하기
- 📝 자신의 프로젝트에서 언급하기
- 🗣️ 커뮤니티에 널리 알리기

Optimism Docs에 기여해 주셔서 감사합니다! 🎉
