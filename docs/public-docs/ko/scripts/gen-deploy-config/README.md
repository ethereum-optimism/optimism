# gen-deploy-config — docs.optimism.io용 deploy-config 스키마 생성기

`op-chain-ops/genesis`의 `DeployConfig` 구조체 트리로부터 OP Stack `DeployConfig`(평면(flat)
JSON 배포 설정)에 대한 MDX 스키마 레퍼런스를 `docs/public-docs/snippets/generated/` 아래에
생성합니다. 문서 페이지(`chain-operators/reference/rollup-deployment-configuration.mdx`)는
구조체를 손으로 옮겨 적는 대신 이 스니펫을 import하므로, 게시된 스키마가 소스보다
조용히 뒤처지는 일이 발생하지 않습니다.

이 도구는 [`../gen-flags`](../gen-flags/README.md)의 설정 스키마 버전에 해당하는 도구이며,
같은 규칙을 재사용합니다. 즉, 수정 금지(do-not-edit) 헤더, 최종 확정된 릴리스 태그를 명시하는
출처(provenance) 줄, 그리고 생성된 스니펫별 `{tag, sha256}`을 기록하는 `manifest.json`을
사용합니다. 두 도구를 하나의 공통 코어로 통합하는 작업은 후속 과제로 기록되어 있습니다.
각 도구를 하나의 검토 가능한 변경으로 반영할 수 있도록 현재는 분리되어 있습니다.

## 사용법

모노레포 루트에서 실행합니다.

```bash
go run ./docs/public-docs/scripts/gen-deploy-config
```

커밋된 스니펫을 다시 작성하지 않고 `manifest.json`에 기록된 SHA-256과 일치하는지
검증하려면 다음을 실행합니다(손으로 수정한 부분 감지용이며, 불일치가 있으면 0이 아닌
값으로 종료합니다).

```bash
go run ./docs/public-docs/scripts/gen-deploy-config -check
```

이 도구는 Go 소스를 파싱(`go/ast`)할 뿐 패키지를 빌드하지 않습니다. 따라서
`just build-superchain-go` 단계가 필요 없으며, 어떤 태그의 소스 체크아웃만으로도 실행할 수 있습니다.

## 동작 방식

- `main.go`는 genesis 패키지 소스(기본값은 `op-chain-ops/genesis`이며, `-source`로
  재정의 가능)를 파싱하여 `DeployConfig` 구조체를 찾고, 내장된(embedded) 설정 구조체
  (`L2CoreDeployConfig`, `FaultProofDeployConfig`, `UpgradeScheduleDeployConfig`, …)를
  선언 순서대로 평면화합니다. 이는 `encoding/json`이 내장 필드를 하나의 평면 JSON 객체로
  평면화하는 방식과 동일합니다.
- 각 내장 구조체는 `groupTitles` 테이블에서 가져온 제목을 가진 섹션이 되며, 구조체의
  Go doc 주석이 섹션 소개로 사용됩니다. 각 필드는 JSON 키(`json:"..."` 구조체 태그에서 가져옴),
  `typeLabels` 테이블에서 가져온 읽기 쉬운 타입, 그리고 Go doc 주석으로 구성된 테이블 행이 됩니다.
  `json:"-"` 태그가 지정된 필드는 건너뜁니다.
- 생성기는 매핑할 수 없는 항목이 있으면 명시적으로 실패합니다. 예를 들어 `groupTitles`에 없는
  내장 구조체, `typeLabels`에 없는 필드 타입, `json` 태그가 없는 필드, 다른 패키지의 구조체 내장,
  중복된 JSON 키가 이에 해당합니다. 이를 통해 업스트림 스키마 변경 시 조용한 불일치나 Go 타입
  이름의 노출 대신 의도적인 문서 결정을 내리도록 강제합니다.
- `manifest.json`은 스니펫이 생성된 최종 확정 릴리스 태그와 스니펫의 SHA-256을 기록하며,
  출처 줄도 같은 태그를 명시합니다. 이 세 가지는 재생성 시에만 함께 업데이트됩니다.

## 태그가 op-deployer 태그인 이유

`op-chain-ops`는 릴리스되는 바이너리가 아니라 라이브러리이므로 자체 릴리스 태그가 없습니다.
파이프라인을 통해 `DeployConfig`를 도출하고 출력하는 릴리스 도구(그리고 문서에서 기본으로
제시하는 배포 경로)는 **op-deployer**입니다. 따라서 스니펫은 최신의 최종 확정된(rc 및 alpha가 아닌)
`op-deployer/v*` 태그 기준의 스키마를 문서화하며, 재생성도 해당 릴리스를 기준으로 이루어집니다.

## 새 릴리스에 맞춰 재생성하기

스니펫은 *릴리스된* 소프트웨어를 문서화하므로, 릴리스되지 않은 코드로부터 재생성해서는 안 됩니다.
새로운 최종 확정 `op-deployer/v*` 태그가 게시되면 다음을 수행합니다.

1. `manifest.json`의 `tag`를 새 릴리스 태그로 업데이트합니다.
2. 해당 태그의 스키마 소스로부터 생성합니다. 태그를 체크아웃하거나, 도구가 파싱하는
   패키지만 추출한 뒤 `-source`로 지정합니다.

   ```bash
   git archive op-deployer/vX.Y.Z op-chain-ops/genesis | tar -x -C /tmp/opd-tag
   go run ./docs/public-docs/scripts/gen-deploy-config -source /tmp/opd-tag/op-chain-ops/genesis
   ```

   (태그와 현재 체크아웃 간에 `op-chain-ops/genesis`가 동일하다면
   — `git diff op-deployer/vX.Y.Z HEAD -- op-chain-ops/genesis`로 확인 —
   `-source` 없이 실행해도 동일한 결과를 얻습니다.)
3. 재생성된 스니펫과 `manifest.json`을 같은 변경으로 커밋합니다.

생성된 스니펫은 절대 손으로 수정하지 마십시오(`-check`는 손으로 수정한 부분이 있으면 실패합니다).
`-check`는 커밋된 스니펫을 현재 트리로부터의 재생성 결과가 아니라 manifest 해시와 비교하여
검증한다는 점에 유의하십시오. `develop`의 스키마 소스는 릴리스 사이에 태그보다 앞서 나갈 수
있으므로, 트리와의 일치 여부는 참고용으로만 보고됩니다.

재생성은 아직 자동화되지 않았습니다. 각 최종 확정 `op-deployer/v*` 태그에 의해 트리거되는
모노레포 CI 작업이 생성기를 실행하고 문서 pull request를 열 예정입니다
(ethereum-optimism/solutions#1518, Phase 2). 그 전까지는 릴리스가 게시되면 위 단계에 따라
로컬에서 재생성하십시오. `-check`로는 언제든지 커밋된 스니펫을 검증할 수 있습니다.

## 소유권

이 파이프라인의 모든 항목은 `docs/public-docs/` 아래에 있으며, 이 디렉터리의 생성기 코드는
CODEOWNERS 규칙(`/docs/public-docs/scripts/  @ethereum-optimism/solutions`)에 따라 solutions가
소유합니다. Solutions 저장소의
`projects/docs-improver/plans/option-b-truth-from-source.md`에 있는 소유권 모델에 따르면 다음과 같습니다.

| 산출물 | 공식 작성자 | 검토자 | 오래된 레퍼런스 분류 |
| --- | --- | --- | --- |
| 생성기 코드 + CI 작업(이 디렉터리, 재생성 워크플로) | Matthew Cruz (@sbvegan), 문서 소유자 — 제안됨, 확인 대기 중 | @ethereum-optimism/solutions (`/docs/public-docs/scripts/` CODEOWNERS 규칙을 통해) | 공식 작성자 |
| 생성된 스니펫(`snippets/generated/deploy-config-schema.mdx`) | 파이프라인 — 아무도 손으로 수정하지 않으며, 구조상 `-check`는 손으로 수정한 경우 실패함 | @ethereum-optimism/solutions가 재생성 PR을 검토 | 깔끔하게 재생성할 수 없는 스니펫은 Solutions 보드에 `accuracy` 레이블이 지정된 이슈로 등록 |
| 스키마 사실 정보(`op-chain-ops/genesis` 구조체 태그 + doc 주석) | 컴포넌트 엔지니어 | 컴포넌트 팀 | 컴포넌트 팀. 문서 스키마는 다음 최종 확정 op-deployer 릴리스에서 이를 따라감 |

알려진 잔여 공백(설계상 수용됨):

- 스키마는 `develop`이 아니라 manifest의 릴리스 태그를 문서화합니다. 릴리스 이후 병합된
  구조체 변경은 다음 최종 확정 태그가 게시되고 CI 작업(또는 유지 관리자)이 재생성할 때까지
  의도적으로 반영되지 않습니다.
- 설명은 소스의 Go doc 주석을 사용합니다. doc 주석이 없는 필드(개발 전용 L1/L2 genesis 블록 필드,
  `channelTimeoutGranite`)는 em 대시로 표시됩니다. 이를 개선하려면 스니펫을 손으로 수정하는 것이
  아니라, 설계에 따라 업스트림의 doc 주석을 개선해야 합니다.
- 검증 규칙(0이 아니어야 하는 값, 조건부 요구 사항)은 구조체의 `Check` 메서드에 있으며 추출되지
  않습니다. 운영상 중요한 규칙은 레퍼런스 페이지에 직접 작성된 안내에서 다룹니다.
- 코드가 아닌 정책에 해당하는 요구 사항(표준 구성 요구 사항, 권장 값)은 레퍼런스 페이지에
  직접 작성된 상태로 유지됩니다.
