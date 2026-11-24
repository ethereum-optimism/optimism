# 하드포크 레지스트리 frontmatter 스키마

모든 OP Stack 하드포크에는 `op-stack/protocol/hardforks/<fork>.mdx`에 정확히 하나의 영구적인
레지스트리 페이지가 있습니다. 기계가 읽을 수 있는 사실 정보는 해당 페이지의 frontmatter에 있으며,
`scripts/generate-hardforks.ts`는 이를 [superchain-registry](https://github.com/ethereum-optimism/superchain-registry)
(모노레포의 `superchain-registry/` 서브모듈)와 대조하여 검증하고, `snippets/generated/hardforks/` 아래의
생성된 스니펫(요약 테이블 하나와 포크별 사실 정보 테이블 하나씩)으로 렌더링합니다. 활성화 테이블은
아무도 손으로 수정하지 않습니다. **활성화 행을 변경하는 유일한 방법은 frontmatter를 수정한 뒤
재생성하는 것입니다.**

이 레지스트리는 공유 인프라입니다. Option B 프로그램(slice B4, "Truth from Source")에서 구축되었으며,
Option A 프로그램(slice 6, protocol anchor)에서 사용됩니다. 구현은 정확히 하나만 존재합니다.

## 명령

`docs/public-docs/`에서 실행합니다.

```bash
pnpm gen:hardforks         # validate + regenerate snippets
pnpm gen:hardforks:check   # drift check (fails when snippets are stale)
```

superchain-registry 서브모듈이 초기화되어 있어야 합니다(모노레포 루트에서
`git submodule update --init superchain-registry` 실행). 또는 체크아웃 위치를 가리키는
`--registry <path>`를 전달합니다.

## 스키마

생성기가 YAML 의존성을 필요로 하지 않도록 키는 의도적으로 평면 구조(스칼라 `key: value`와 블록
문자열 목록)로 되어 있습니다. 타임스탬프는 UTC 기준 unix 초 단위입니다.

| 키 | 필수 여부 | 의미 |
| --- | --- | --- |
| `hardfork_name` | 예 | 소문자로 된 포크 이름입니다. 파일 이름과 일치해야 합니다(`karst.mdx`의 경우 `karst`). |
| `hardfork_lifecycle` | 예 | `active`(mainnet superchain 기본값에서 활성화됨), `scheduled`(타임스탬프가 설정되었으며 미래 시점), 또는 `development`(사양 작성 중, 활성화 없음). |
| `hardfork_spec` | 예 | 기준이 되는 사양으로, 현재 경로의 렌더링된 `https://specs.optimism.io/...` URL입니다([저장소 간 링크 정책](../op-stack/contribute/link-policy.mdx) 참조). |
| `hardfork_activation_mainnet` | `development`가 아닌 경우 | `superchain/configs/mainnet/superchain.toml`의 Superchain 전체 mainnet 기본값입니다. `0`은 genesis 시점 활성 / Bedrock 시점을 의미합니다(superchain 기본값 이전의 포크, 예: Regolith). |
| `hardfork_activation_sepolia` | `development`가 아닌 경우 | `superchain/configs/sepolia/superchain.toml`의 Superchain 전체 sepolia 기본값입니다. `0` 규칙은 동일합니다. |
| `hardfork_governance` | 아니요 | 거버넌스 승인 URL입니다(`gov.optimism.io` 스레드 또는 `vote.optimism.io` 제안). |
| `hardfork_notice` | 아니요 | 해당 업그레이드에 대한 운영자 공지의 루트 상대 링크입니다(예: `/notices/archive/upgrade-19`). |
| `hardfork_upgrade_number` | 아니요 | 거버넌스의 "Upgrade N" 번호로, 이 저장소의 공지에서 확인된 경우에만 사용합니다. |
| `hardfork_min_versions` | 아니요 | `"<component> <version>"` 문자열의 블록 목록으로, 포크를 따르기 위한 최소 컴포넌트 버전입니다. |
| `hardfork_min_versions_source` | `hardfork_min_versions`와 함께 | 버전의 출처(업그레이드 공지 또는 릴리스 노트)를 인용하는 Markdown 링크입니다. |

예시:

```yaml
---
title: Karst
description: Hardfork registry entry for the Karst network upgrade.
diataxis: reference
hardfork_name: karst
hardfork_lifecycle: active
hardfork_spec: https://specs.optimism.io/protocol/karst/overview.html
hardfork_activation_mainnet: 1783526401
hardfork_activation_sepolia: 1781712001
hardfork_notice: /notices/archive/upgrade-19
hardfork_upgrade_number: 19
hardfork_min_versions:
  - op-node v1.19.1
  - op-reth v2.3.3
hardfork_min_versions_source: "[Upgrade 19 notice](/notices/archive/upgrade-19)"
---
```

## 검증 규칙

*   superchain-registry의 `[hardforks]` 섹션에 예정된 모든 포크에는 레지스트리 페이지가 있어야 하며,
    페이지의 활성화 값은 레지스트리의 값과 같아야 합니다. superchain-registry가 신뢰할 수 있는 원천(source of truth)입니다.
*   페이지에는 superchain-registry에 없는 0이 아닌 활성화 값을 명시할 수 없습니다.
*   `[hardforks]`의 키 중 포크 시리즈의 L2 하드포크가 아닌 키
    (`pectra_blob_schedule_time`, `keep_karst_upgrade_gas`)는 무시됩니다.
*   렌더링된 테이블에는 타임스탬프(및 해당 UTC 날짜)만 표시됩니다. 하드포크는 L2 블록 타임스탬프를
    기준으로 활성화되며 블록 높이는 체인마다 다르므로, 블록 번호 추정치는 의도적으로 렌더링하지 않습니다.

## 새 하드포크 추가하기

1.  `hardfork_lifecycle: development`와 사양 링크를 포함하여 `op-stack/protocol/hardforks/<fork>.mdx`를
    생성하고, 생성된 스니펫(`/snippets/generated/hardforks/<fork>.mdx`)을 import합니다.
2.  `docs.json` 내비게이션(Protocol 탭 → Reference → Network upgrades 그룹)에 페이지를 추가합니다.
3.  superchain-registry에서 활성화 시간이 예정되면, 해당 타임스탬프와 함께 frontmatter를
    `scheduled`(이후 `active`)로 업데이트합니다.
4.  `pnpm gen:hardforks`를 실행하고 페이지와 재생성된 스니펫을 함께 커밋합니다. 그렇지 않으면
    drift 검사가 실패합니다.
