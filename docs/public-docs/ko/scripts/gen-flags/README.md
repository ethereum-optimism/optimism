# gen-flags — docs.optimism.io용 플래그 레퍼런스 생성기

OP Stack Go 서비스가 실제로 등록하는 urfave/cli 플래그 정의로부터
`docs/public-docs/snippets/generated/` 아래에 MDX 플래그 레퍼런스 테이블을 생성합니다.
문서 페이지는 `--help` 출력을 손으로 옮겨 적는 대신 이 스니펫을 import하므로,
게시된 플래그 목록이 릴리스보다 조용히 뒤처지는 일이 발생하지 않습니다.

대상 컴포넌트: op-batcher, op-node, op-proposer, op-challenger,
op-conductor — 플래그 레퍼런스 페이지가 있는 모든 자체(first-party) Go 서비스입니다.

## 사용법

모노레포 루트에서 실행합니다.

```bash
go run ./docs/public-docs/scripts/gen-flags
```

특정 컴포넌트로 실행 범위를 제한하려면 다음을 실행합니다(한 컴포넌트를 해당 릴리스 태그에서
재생성할 때 사용합니다).

```bash
go run ./docs/public-docs/scripts/gen-flags -only op-node
```

커밋된 스니펫을 다시 작성하지 않고 검증하려면 다음을 실행합니다(문제가 있으면 0이 아닌 값으로
종료합니다).

```bash
go run ./docs/public-docs/scripts/gen-flags -check
```

모노레포 Go 빌드는 `op-core/superchain/bundle/superchain-configs.zip`(gitignore 대상)을
내장합니다. 새로 체크아웃한 경우 먼저 이를 빌드하십시오.

```bash
just build-superchain-go
```

## 동작 방식

- `main.go`는 각 컴포넌트의 조합된 플래그 슬라이스(예: op-batcher의 경우 `init()`에서
  추가되는 op-service 플래그 계열을 포함한 `op-batcher/flags.Flags`)를 import하고,
  컴포넌트마다 하나의 스니펫 `snippets/generated/<component>-flags.mdx`를 렌더링합니다.
- 플래그 이름, 사용법, 기본값, 환경 변수는 urfave/cli 자체의 `DocGenerationFlag` 접근자에서
  가져오므로 출력이 `--help`의 의미와 일치합니다. 숨겨진 플래그(지원 중단된 op-node 플래그 포함)는
  건너뜁니다.
- `manifest.json`은 컴포넌트별로 각 스니펫이 생성된 최종 확정 릴리스 태그(예: `op-batcher/v1.16.11`)와
  생성 시점에 기록된 스니펫 바이트의 SHA-256을 기록합니다. 스니펫의 출처(provenance) 줄도 같은
  태그를 명시합니다. 이 세 가지는 재생성 시에만 함께 업데이트됩니다.

### `-check`가 검증하는 내용

스니펫은 *릴리스된* 소프트웨어를 문서화합니다. 각 스니펫은 manifest 태그 시점의 소스로부터
생성되며, 컴포넌트들은 독립적으로 릴리스됩니다. 따라서 `develop`의 플래그 정의 소스는 어떤
컴포넌트의 태그보다는 앞서 나가면서 다른 컴포넌트의 태그와는 일치하는 경우가 흔합니다.
그러므로 커밋된 스니펫을 임의의 트리로부터 재생성한 결과와 바이트 단위로 비교하는 것은
해당 컴포넌트의 태그 시점(또는 그와 동일한 상태)에서만 의미가 있습니다.

`-check`는 모든 커밋에서 검증 가능한 사항을 검증합니다. 즉, 커밋된 각 스니펫이
`manifest.json`에 기록된 SHA-256과 일치하는지 확인하며, 스니펫을 손으로 수정한 경우
(또는 스니펫과 manifest가 불일치하는 경우) 실패합니다. 또한 컴포넌트별로 현재 트리가 여전히
동일한 바이트를 생성하는지도 보고합니다. 이 부분의 불일치는 실패가 아니라 참고 정보("소스가
태그보다 앞서 나갔음")이며, 다음 최종 확정 릴리스가 게시되고 스니펫이 재생성되면 해소됩니다.

## 새 릴리스에 맞춰 재생성하기

컴포넌트의 새로운 최종 확정(rc 및 pre-release가 아닌) 릴리스 태그가 게시되면 다음을 수행합니다.

1. 해당 태그를 체크아웃하여 재생성합니다. 다른 커밋에서 작업하는 경우에는 먼저 플래그를 정의하는
   패키지가 태그와 동일한지 확인합니다(`git diff <tag> HEAD -- <paths>`). 대상은 컴포넌트의
   `flags/` 패키지(op-challenger의 경우 `config/` 기본값 포함)와 컴포넌트가 추가하는
   op-service 플래그 계열(컴포넌트에 따라 `op-service/rpc`, `op-service/log`,
   `op-service/metrics`, `op-service/oppprof`, `op-service/txmgr`,
   `op-service/flags`, `op-alt-da`)입니다.
2. `manifest.json`에서 해당 컴포넌트의 `tag`를 새 릴리스 태그로 업데이트합니다.
3. 해당 컴포넌트로 제한하여 생성기를 실행합니다.
   `go run ./docs/public-docs/scripts/gen-flags -only <component>`
   (태그 체크아웃에서 실행하며, 커밋할 문서 루트와 다른 경우 `-docs-dir`로 그 위치를
   지정합니다). 스니펫, 출처 줄, manifest의 `sha256`이 함께 다시 작성됩니다. manifest와
   스니펫을 같은 변경으로 커밋하십시오.

생성된 스니펫은 절대 손으로 수정하지 마십시오(`-check`는 손으로 수정한 부분이 있으면 실패합니다).
또한 릴리스되지 않은 코드로부터 재생성하지 마십시오. manifest 태그 이후 플래그가 변경되었다면,
재생성된 테이블은 어떤 릴리스된 바이너리에도 없는 동작을 문서화하게 됩니다.

재생성은 아직 자동화되지 않았습니다. 각 최종 확정 릴리스 태그에 의해 트리거되는 모노레포 CI
작업이 생성기를 실행하고 문서 pull request를 열 예정입니다(ethereum-optimism/solutions#1518,
Phase 2). 그 전까지는 릴리스가 게시되면 위 단계에 따라 로컬에서 재생성하십시오. `-check`로는
언제든지 커밋된 스니펫을 검증할 수 있습니다.

## 컴포넌트 추가하기

1. `main.go`의 `components` 테이블에 컴포넌트의 플래그 슬라이스를 추가하고, 필수 플래그 이름을
   그대로 반영합니다(컴포넌트들은 필수 슬라이스를 export하지 않으며, 런타임 정합성 검사가 이름
   변경을 감지합니다).
2. 최신 최종 확정 릴리스 태그를 `manifest.json`에 추가합니다.
3. (해당 태그에서, 또는 태그와 동일함이 검증된 커밋에서) 생성기를 실행하고, 컴포넌트의
   레퍼런스 페이지에서 새 스니펫을 import합니다.
4. CI 재생성 작업이 새 컴포넌트의 릴리스 태그 패턴을 감시하도록 확장합니다.

## 소유권

이 파이프라인의 모든 항목은 `docs/public-docs/` 아래에 있으며, 이 디렉터리의 생성기 코드는
CODEOWNERS 규칙(`/docs/public-docs/scripts/  @ethereum-optimism/solutions`)에 따라 solutions가
소유합니다. Solutions 저장소의
`projects/docs-improver/plans/option-b-truth-from-source.md`에 있는 소유권 모델에 따르면 다음과 같습니다.

| 산출물 | 공식 작성자 | 검토자 | 오래된 레퍼런스 분류 |
| --- | --- | --- | --- |
| 생성기 코드 + CI 작업(이 디렉터리, 재생성 워크플로) | Matthew Cruz (@sbvegan), 문서 소유자 — 제안됨, 확인 대기 중 | @ethereum-optimism/solutions (`/docs/public-docs/scripts/` CODEOWNERS 규칙을 통해) | 공식 작성자 |
| 생성된 스니펫(`snippets/generated/`) | 파이프라인 — 아무도 손으로 수정하지 않으며, 구조상 `-check`는 손으로 수정한 경우 실패함 | @ethereum-optimism/solutions가 재생성 PR을 검토 | 깔끔하게 재생성할 수 없는 스니펫은 Solutions 보드에 `accuracy` 레이블이 지정된 이슈로 등록 |
| 플래그 사실 정보(각 컴포넌트의 `flags/` 패키지, op-service 플래그 계열) | 컴포넌트 엔지니어 | 컴포넌트 팀 | 컴포넌트 팀. 문서 테이블은 다음 최종 확정 릴리스에서 이를 따라감 |

알려진 잔여 공백(설계상 수용됨):

- 테이블은 `develop`이 아니라 manifest의 릴리스 태그를 문서화합니다. 릴리스 이후 병합된 플래그
  변경은 다음 최종 확정 태그가 게시되고 CI 작업(또는 유지 관리자)이 재생성할 때까지 의도적으로
  반영되지 않습니다.
- 컴포넌트가 필수 플래그 목록을 export하지 않으므로 `main.go`에 해당 목록을 그대로 복제해 두었습니다.
  이름이 변경되면 생성기의 정합성 검사가 실패하지만, 복제 목록을 업데이트하지 않고 완전히 새로운
  필수 플래그를 추가하면 해당 플래그는 선택 사항으로 표시됩니다.
  (op-challenger의 복제 목록에는 `l2-eth-rpc`도 포함되어 있습니다. 이 플래그는 필수 슬라이스
  외부에서 `CheckRequired`가 무조건 강제합니다.)
- `-check`는 태그와의 재생성 일치 여부가 아니라 manifest 해시를 기준으로 스니펫 무결성을
  검증합니다. 실수로 태그가 아닌 소스에서 재생성한 경우에도 `-check`를 통과할 수 있습니다.
  재생성 PR에 대한 검토 절차와 위의 태그 일치 확인 절차가 이를 방지하는 안전장치입니다.
- op-batcher의 `--compressor` 옵션 목록(`compressor.KindKeys`)은 map 순회로 생성되므로, 프로세스가
  시작될 때마다 순서가 바뀝니다. 이는 여기뿐만 아니라 `op-batcher --help` 자체에서도 마찬가지입니다.
  생성기는 출력이 결정적(deterministic)이 되도록 이 목록 하나를 정렬합니다(`main.go`의
  `canonicalizeUsage`). 근본적인 해결책은 소스에서 `KindKeys`를 정렬하는 것이며
  (`op-batcher/compressor/compressors.go`의 한 줄), 후속 작업으로 제안되어 있습니다.
- `--network` 사용법 문자열(op-node, op-challenger, op-conductor)은 superchain-registry에서
  번들된 체인을 나열하며, 이 목록은 컴포넌트 릴리스와 무관하게 레지스트리 서브모듈이 업데이트될
  때마다 바뀝니다. `canonicalizeUsage`는 나열된 목록을 레지스트리를 가리키는 고정된 안내로 대체하여
  스니펫이 바이트 단위로 안정적으로 유지되도록 합니다. 번들된 정확한 목록은 항상
  `<component> --help`에서 확인할 수 있습니다.
