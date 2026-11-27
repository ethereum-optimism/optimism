//! Print op-revm fee results for the corpus from the Go fee-diff-dump command.

use alloy_primitives::{U256, hex};
use op_revm::{L1BlockInfo, OpSpecId, encoded_tx_da_footprint};
use serde::Deserialize;
use std::{
    error::Error,
    io::{self, BufRead, Write},
};

#[derive(Deserialize)]
struct FeeCase {
    tx: String,
    gas: u64,
    base_fee: u64,
    blob_base_fee: u64,
    base_scalar: u32,
    blob_scalar: u32,
    overhead: u64,
    operator_scalar: u32,
    operator_const: u64,
    da_scalar: u16,
}

fn main() -> Result<(), Box<dyn Error>> {
    let mut out = io::BufWriter::new(io::stdout().lock());
    for line in io::stdin().lock().lines() {
        let case: FeeCase = serde_json::from_str(&line?)?;
        let tx = hex::decode(case.tx)?;
        let mut l1 = L1BlockInfo {
            l1_base_fee: U256::from(case.base_fee),
            l1_fee_overhead: Some(U256::from(case.overhead)),
            l1_base_fee_scalar: U256::from(case.base_scalar),
            l1_blob_base_fee: Some(U256::from(case.blob_base_fee)),
            l1_blob_base_fee_scalar: Some(U256::from(case.blob_scalar)),
            operator_fee_scalar: Some(U256::from(case.operator_scalar)),
            operator_fee_constant: Some(U256::from(case.operator_const)),
            ..Default::default()
        };

        let bedrock = l1.calculate_tx_l1_cost(&tx, OpSpecId::BEDROCK);
        l1.clear_tx_l1_cost();
        let regolith = l1.calculate_tx_l1_cost(&tx, OpSpecId::REGOLITH);
        l1.clear_tx_l1_cost();
        let ecotone = l1.calculate_tx_l1_cost(&tx, OpSpecId::ECOTONE);
        l1.clear_tx_l1_cost();
        let fjord = l1.calculate_tx_l1_cost(&tx, OpSpecId::FJORD);
        let isthmus = l1.operator_fee_charge(&tx, U256::from(case.gas), OpSpecId::ISTHMUS);
        let jovian = l1.operator_fee_charge(&tx, U256::from(case.gas), OpSpecId::JOVIAN);
        let footprint = encoded_tx_da_footprint(&tx, u64::from(case.da_scalar));
        writeln!(out, "{bedrock} {regolith} {ecotone} {fjord} {isthmus} {jovian} {footprint}")?;
    }
    out.flush()?;
    Ok(())
}
