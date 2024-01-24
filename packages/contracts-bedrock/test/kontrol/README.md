# Kontrol Verification

This folder contains Kontrol symbolic property tests.

## Directory structure

The directory is structured as follows

```tree
test/kontrol
├── deployment
│   ├── DeploymentSummary.t.sol
│   └── KontrolDeployment.sol
├── pausability-lemmas.k
├── proofs
│   ├── interfaces
│   │   └── KontrolInterfaces.sol
│   ├── L1CrossDomainMessenger.k.sol
│   ├── OptimismPortal.k.sol
│   └── utils
│       ├── DeploymentSummaryCode.sol
│       ├── DeploymentSummary.sol
│       └── KontrolUtils.sol
├── README.md
└── scripts
    ├── json
    │   ├── clean_json.py
    │   └── reverse_key_values.py
    ├── make-summary-deployment.sh
    └── run-kontrol.sh
```

### Root folder

- [`pausability-lemmas.k`](./pausability-lemmas.k): File containing the necessary lemmas for this project
- [`deployment`](./deployment): Custom deploy sequence for Kontrol proofs and tests for its summarization
- [`proofs`](./proofs): Where the proofs of the project live
- [`scripts`](./scripts): Where the scripts of the projects live

### [`deployment`](./deployment) folder

- [`KontrolDeployment.sol`](./deployment/KontrolDeployment.sol): Simplified deployment sequence for Kontrol proofs
- [`DeploymentSummary.t.sol`](./deployment/DeploymentSummary.t.sol): Tests for the summarization of custom deployment

### [`proofs`](./proofs) folder

- [`L1CrossDomainMessenger.k.sol`](./proofs/L1CrossDomainMessenger.k.sol): Symbolic property tests for [`L1CrossDomainMessenger`](../../src/L1/L1CrossDomainMessenger.sol)
- [`OptimismPortal.k.sol`](./proofs/OptimismPortal.k.sol): Symbolic property tests for [`OptimismPortal`](../../src/L1/OptimismPortal.sol)
- [`interfaces`](./proofs/interfaces): Files with the signature of the functions involved in the verification effort
- [`utils`](./proofs/utils): Proof dependencies, including the summary contracts

### [`scripts`](./scripts) folder

- [`make-summary-deployment.sh`](./scripts/make-summary-deployment.sh): Executes [`KontrolDeployment.sol`](./KontrolDeployment.sol), curates the result and writes the summary deployment contract
- [`run-kontrol.sh`](./scrpts/run-kontrol.sh): Proof execution script
- [`json`](./scripts/json): Data cleaning scripts for the output of [`KontrolDeployment.sol`](./KontrolDeployment.sol)

## Verification execution

The verification execution consists of two steps, although the first step may be omitted to use the committed version. There's one script to run per step. These scripts should be run from the [`contracts-bedrock`](../../) directory.

1. Generate a deployment summary contract from [`KontrolDeployment.sol`](./KontrolDeployment.sol)
```bash
  ./test/kontrol/scripts/make-summary-deployment.sh
```
This step is optional. The default summary can be found [here](./proofs/utils/DeploymentSummary.sol), which is the summarization of the [`KontrolDeployment.sol`](./KontrolDeployment.sol) script.

2. Execute the tests in [`OptimismPortal.k.sol`](./proofs/OptimismPortal.k.sol)
```bash
  ./test/kontrol/scripts/run-kontrol.sh $option
```
See below for further documentation on `run-kontrol.sh`.

### `run-kontrol.sh` script
The `run-kontrol.sh` script handles all modes of proof execution. The modes, corresponding to the available arguments of the script, are the following:
- `container`: Run the proofs in the same docker image used in CI. The intended use case is CI debugging. This is the default execution mode, meaning that if no arguments are provided, the proofs will be executed in this mode.
- `local`: Run the proofs with your local Kontrol install, enforcing the version to be the same as the one used in CI. The intended use case is running the proofs without risking discrepancies because of different Kontrol versions.
- `dev`: Run the proofs with your local Kontrol install, without enforcing any version in particular. The intended use case is proof development and related matters.

For a similar description of the options run `run-kontrol.sh --help`.

## Kontrol Foundry profiles

This project uses two different [Foundry profiles](../../foundry.toml), `kdeploy` and `kprove`.

- `kdeploy`: This profile is used to generate a summary contract from the execution of the [`KontrolDeployment.sol`](./KontrolDeployment.sol) script. In particular, the `kdeploy` profile is used by the [`make-summary-deployment.sh`](./scripts/make-summary-deployment.sh) script to generate the [deployment summary contract](./proofs/utils/DeploymentSummary.sol). The summary contract is then used with the `kprove` profile to load the post-setUp state directly into Kontrol. We don't need the output artifacts from this step, so we save them to the `kout-deployment` directory, which is not used anywhere else. We also point the script path to the `scripts-kontrol` directory, which does not exist, to avoid compiling scripts we don't need, which reduces execution time.

- `kprove`: This profile is used by the [`run-kontrol.sh`](./scrpts/run-kontrol.sh) script, which needs to be run after executing [`./test/kontrol/script/make-summary-deployment`](./scripts/make-summary-deployment.sh) (this last script uses the `kdeploy` profile). The proofs are executed using the `kprove` profile. The `src` directory points to a test folder because we only want to compile what is in the `test/kontrol/proofs` folder since it contains all the deployed bytecode and the proofs. We similarly point the script path to a non-existent directory for the same reason as above. The `out` folder for this profile is `kout-proofs`.

Note that the compilation of the necessary `src/L1` files is done with the `kdeploy` profile, and the results are saved into [`test/kontrol/proofs/utils/DeploymentSummary.sol`](./proofs/utils/DeploymentSummary.sol). So, when running the `kprove` profile, the deployed bytecode of the `src/L1` files are recorded in the automatically generated file `test/kontrol/proofs/utils/DeploymentSummaryCode.sol`.

## References

[Kontrol docs](https://docs.runtimeverification.com/kontrol/overview/readme)
