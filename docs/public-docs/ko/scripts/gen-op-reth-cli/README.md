# gen-op-reth-cli — docs.optimism.io용 op-reth CLI 레퍼런스 생성기

고정된(pinned) op-reth 릴리스 바이너리의 `--help` 출력으로부터 op-reth CLI 레퍼런스 트리
(`docs/public-docs/node-operators/op-reth/cli/op-reth.mdx` 및 `.../cli/op-reth/` 아래의 모든 항목)와
docs.json 내비게이션 조각(fragment)을 다시 생성합니다. 이를 통해 게시된 명령 목록이 릴리스보다
조용히 뒤처지는 일이 발생하지 않습니다.

이 페이지들은 재생성 파이프라인 없이, 사용이 종료된 Vocs 사이트의 일회성 스냅샷으로 모노레포에
들어왔으며(optimism#19778), 이후 내용이 어긋났습니다(optimism#21845에서 오래된 `db settings set`
페이지 두 개를 수동으로 삭제했습니다). 이 생성기가 바로 그 파이프라인입니다. 이 생성기는 업스트림
reth의 `docs/cli/update.sh` + `docs/cli/help.rs`(스냅샷의 출력 형식과 정확히 일치하는 도구)를
이식한 것으로, Markdown 제목 대신 `diataxis: reference`가 포함된 frontmatter를 가진 Mintlify MDX를
출력하고, Vocs 사이드바 대신 docs.json 내비게이션 조각을 출력합니다.

## 사용법

최종 확정된 op-reth 릴리스 태그에서 바이너리를 빌드한 다음, 어느 위치에서든 실행합니다
(경로는 이 디렉터리를 기준으로 해석됩니다).

```bash
# in a checkout of the release tag
cargo build --bin op-reth --manifest-path rust/op-reth/bin/Cargo.toml

node docs/public-docs/scripts/gen-op-reth-cli/main.mjs \
  --bin rust/target/debug/op-reth \
  --tag op-reth/vX.Y.Z
```

아무것도 다시 작성하지 않고 커밋된 트리를 바이너리와 비교하여 검증하려면 다음을 실행합니다
(불일치가 있으면 0이 아닌 값으로 종료합니다).

```bash
node docs/public-docs/scripts/gen-op-reth-cli/main.mjs --bin <op-reth> --check
```

## 동작 방식

- 바이너리의 `--help` 트리를 재귀적으로 순회합니다(하위 명령은 각 help 출력의 `Commands:` 섹션에서
  파싱하며, `help` 자체는 건너뜁니다). 안정적인 형식을 위해
  `NO_COLOR=1 COLUMNS=100 LINES=10000`을 사용합니다.
- 명령마다 하나의 MDX 페이지를 작성합니다. 페이지는 따옴표로 감싼 `title`, `diataxis: reference`
  frontmatter, 명령의 한 줄 설명, 그리고 `txt` 코드 블록 안의 원문 그대로의 help 텍스트로
  구성됩니다. 환경에 따라 달라지는 출력(홈 경로, 버전 및 target triple, CPU 개수 기반 기본값)은
  업스트림 `help.rs`와 동일한 치환 규칙으로 제거되며, 그 결과 `<CACHE_DIR>` /
  `<VERSION>` / `<OS>` 자리 표시자가 생성됩니다.
- 재생성된 명령 트리를 `docs.json`의 `op-reth CLI reference` 그룹(Node Operators > Reference 아래의
  접힌 그룹)에 이어 붙이며, 직접 작성된 `cli/overview` 페이지는 그룹의 첫 번째 항목으로 유지합니다.
  중첩된 하위 명령은 전체 명령 이름을 가진 중첩 그룹이 되며, `--help` 순서를 따릅니다. 직접 작성된
  개요 페이지는 절대 수정하지 않습니다.
- `--help`에서 사라진 명령의 페이지는 삭제하고 해당 URL을 출력합니다. **삭제된 모든 페이지에는 같은
  PR에서 `docs.json`에 redirect를 추가해야 합니다**(`REDIRECTS_GUIDE.md` 참조). redirect lint가
  이를 강제합니다.
- `manifest.json`은 트리가 생성된 최종 확정 릴리스 태그, 바이너리가 보고한 버전 줄, 생성된 페이지
  전체에 대한 SHA-256을 기록합니다. 이 항목들은 재생성 시에만 함께 다시 작성됩니다.

### `--check`가 검증하는 내용

트리는 *릴리스된* 바이너리를 문서화합니다. 트리는 manifest 태그 시점에 생성되며 `develop`은 흔히
그 태그보다 앞서 나가므로, 임의의 커밋에서 빌드한 바이너리와 바이트 단위로 비교하는 것은 태그
시점(또는 그와 동일한 상태)에서만 의미가 있습니다. `--check`는 주어진 바이너리로 전체 트리를
메모리에서 다시 계산하고, 커밋된 페이지, docs.json 내비게이션 조각, manifest 해시 중 어느 것과라도
차이가 있으면 실패합니다. 따라서 생성된 페이지를 손으로 수정하면 다음 검사에서 실패하게 됩니다.

## 새 릴리스에 맞춰 재생성하기

새로운 최종 확정(rc가 아닌) `op-reth/vX.Y.Z` 태그가 게시되면 다음을 수행합니다.

1. 태그를 체크아웃하고 기본 기능 세트로 바이너리를 빌드합니다
   (`cargo build --bin op-reth --manifest-path rust/op-reth/bin/Cargo.toml`).
   릴리스되지 않은 코드로부터 재생성하지 마십시오. 태그 이후 CLI가 변경되었다면, 페이지는 어떤
   릴리스된 바이너리에도 없는 동작을 문서화하게 됩니다.
2. `--tag`를 새 태그로 설정하여 생성기를 실행합니다(태그 체크아웃에서, 커밋할 문서 트리를 대상으로).
3. 실행 결과 삭제된 페이지가 있다면, 같은 변경에서 삭제된 URL마다 하나의 redirect를 `docs.json`에
   추가하고, 가장 가까운 남아 있는 명령 페이지를 가리키도록 합니다.
4. 문서 lint(`pnpm lint:nav`, `pnpm lint:redirects`, `pnpm lint:reference`,
   `node scripts/lint-link-policy.mjs --baseline scripts/lint-link-policy.baseline.json`)를 실행하고,
   페이지, 내비게이션 조각, redirect, `manifest.json`을 함께 커밋합니다.

같은 바이너리로 생성기를 연속 두 번 실행해도 아무 변화가 없습니다.

## CI에서의 재생성

재생성은 아직 자동화되지 않았습니다. 각 최종 확정 `op-reth/v*` 태그에 의해 트리거되는 모노레포 CI
작업이 바이너리를 빌드하고, 재생성한 뒤, 삭제된 페이지에 대한 redirect 항목과 함께 pull request를
열 예정입니다(ethereum-optimism/solutions#1518, Phase 2). 그 전까지는 릴리스가 게시되면 위 단계에
따라 로컬에서 재생성하십시오. manifest 태그에서 빌드한 바이너리로 `--check`를 실행하면 언제든지
커밋된 트리를 검증할 수 있습니다.

## 소유권

이 파이프라인의 모든 항목은 `docs/public-docs/` 아래에 있으며, 이 디렉터리의 생성기 코드는
CODEOWNERS 규칙(`/docs/public-docs/scripts/  @ethereum-optimism/solutions`)에 따라 solutions가
소유합니다. Solutions 저장소의
`projects/docs-improver/plans/option-b-truth-from-source.md`에 있는 소유권 모델에 따르면 다음과 같습니다.

| 산출물 | 공식 작성자 | 검토자 | 오래된 레퍼런스 분류 |
| --- | --- | --- | --- |
| 생성기 코드 + CI 작업(이 디렉터리, 재생성 워크플로) | Matthew Cruz (@sbvegan), 문서 소유자 — 제안됨, 확인 대기 중 | @ethereum-optimism/solutions (`/docs/public-docs/scripts/` CODEOWNERS 규칙을 통해) | 공식 작성자 |
| 생성된 페이지(`node-operators/op-reth/cli/op-reth*`)와 내비게이션 조각 | 파이프라인 — 아무도 손으로 수정하지 않으며, 구조상 `--check`는 손으로 수정한 경우 실패함 | @ethereum-optimism/solutions가 재생성 PR을 검토 | 깔끔하게 재생성할 수 없는 트리는 Solutions 보드에 `accuracy` 레이블이 지정된 이슈로 등록 |
| CLI 사실 정보(`rust/op-reth` 및 고정된 업스트림 reth crate) | 컴포넌트 엔지니어 | 컴포넌트 팀 | 컴포넌트 팀. 문서 트리는 다음 최종 확정 릴리스에서 이를 따라감 |

알려진 잔여 공백(설계상 수용됨):

- 트리는 `develop`이 아니라 manifest의 릴리스 태그를 문서화합니다. 릴리스 이후 병합된 CLI 변경은
  다음 최종 확정 태그가 게시되고 CI 작업(또는 유지 관리자)이 재생성할 때까지 의도적으로 반영되지
  않습니다.
- `--check`는 바이너리가 필요하므로 순수한 콘텐츠 lint로 실행할 수 없습니다. 그러나 redirect 및
  내비게이션 lint가 모든 PR에서 생성된 페이지의 URL과 내비게이션 항목을 계속 보호합니다.
- 직접 작성된 `cli/overview.mdx` 설명 페이지는 명령을 이름으로 참조합니다. 최상위 명령의 이름이
  변경되는 경우, 재생성 PR에서 개요 페이지를 손으로 업데이트해야 합니다(내비게이션 및 링크 lint가
  끊어진 링크를 감지합니다).
