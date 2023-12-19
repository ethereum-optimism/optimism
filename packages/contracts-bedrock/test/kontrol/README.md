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

## References

[Kontrol docs](https://docs.runtimeverification.com/kontrol/overview/readme)
