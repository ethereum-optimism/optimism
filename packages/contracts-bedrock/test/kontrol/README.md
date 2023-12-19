# Kontrol Verification

This folder contains Kontrol symbolic property tests.

## Directory structure

The directory structure is as follows

```tree
test/kontrol
├── Counter.sol
├── Counter.t.sol
├── kontrol
│   ├── pausability-lemmas.k
│   └── scripts
│       ├── json
│       │   ├── clean_json.py
│       │   └── reverse_key_values.py
│       ├── make-summary-deployment.sh
│       ├── run-kontrol-local.sh
│       └── run-kontrol.sh
├── KontrolDeployment.sol
└── proofs
    ├── interfaces
    │   └── KontrolInterfaces.sol
    ├── OptimismPortal.k.sol
    └── utils
        ├── DeploymentSummaryCode.sol
        ├── DeploymentSummary.sol
        └── KontrolUtils.sol
```

### Root folder

- [`KontrolDeployment.sol`](./KontrolDeployment.sol): Reduced deployment to generate the summary contract
- [`proofs`](./proofs): Where the proofs of the project live
- [`kontrol`](./kontrol): Lemmas and utilities for the project set up

### [`proofs`](./proofs) folder

- [`OptimismPortal.k.sol`](./proofs/OptimismPortal.k.sol): Symbolic property tests
- [`interfaces`](./proofs/interfaces): Files with the signature of the functions involved in the verification effort
- [`utils`](./proofs/utils): Dependencies for `OptimismPortal.k.sol`

### [`kontrol`](./kontrol) folder

- [`pausability-lemmas.k`](./kontrol/pausability-lemmas.k): File containing the necessary lemmas for this project
- [`scripts`](./kontrol/scripts): Contains
    - [`make-summary-deployment.sh`](./kontrol/scripts/make-summary-deployment.sh): Executes [`KontrolDeployment.sol`](./KontrolDeployment.sol), curates the result and writes the summary deployment contract
    - [`run-kontrol.sh`](./kontrol/scrpts/run-kontrol.sh): CI execution script
    - [`run-kontrol-local.sh`](./kontrol/scrpts/run-kontrol-local.sh): Local execution script
    - [`json`](./kontrol/scripts/json): Data cleaning scripts for the output of [`KontrolDeployment.sol`](./KontrolDeployment.sol)

## Local verification exeuction

The verification execution consists of two steps and there's one script to run per step. These commands should be run from the [`contracts-bedrock`](../../) directory.

1. Generate a deployment summary contract from [`KontrolDeployment.sol`](./KontrolDeployment.sol)
```bash
  bash test/kontrol/kontrol/scripts/make-summary-deployment.sh
```
2. Execute the tests in [`OptimismPortal.k.sol`](./proofs/OptimismPortal.k.sol)
```bash
  ./test/kontrol/kontrol/scripts/run-kontrol-local.sh
```

## Kontrol Foundry profiles

This project uses two different [Foundry profiles](../../foundry.toml), `kdeploy` and `kprove`.

- `kdeploy`: This profile is used to generate the deployment summary solidity contract, which is used by the `kprove` profile to load the post-setUp state directly into kontrol. We don't need the output artifacts from this step, so we save them to the `kout-deployment` directory, which is not used elsewhere. We also point the script path to the `scripts-kontrol` directory, which does not exist, to avoid compiling scripts we don't need which reduces execution time.

- `kprove`: This profile is used after running `bash test/kontrol/kontrol/script/make-summary-deployment`, which uses the `kdeploy` profile. The proofs are executed using the `kprove` profile. The `src` directory points to a test folder because we only want to compile what is in the `test/kontrol/proofs` folder since it contains all the deployed code and the proofs. We similarly point the script path to a non-existent directory for the same reason as above.

Note that the compilation of the necessary `src/L1` files is done with the `kdeploy` profile, and the results are saved into `test/kontrol/proofs/utils`. So, when running the `kprove` profile, the compiled `src/L1` files are in the automatically generated file `test/kontrol/proofs/utils/DeploymentSummaryCode.sol`.

## References

[Kontrol docs](https://docs.runtimeverification.com/kontrol/overview/readme)
