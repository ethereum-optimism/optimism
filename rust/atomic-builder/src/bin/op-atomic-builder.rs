//! One-bundle, private stdio worker. The host owns pinned state and signing keys.
use alloy_consensus::{Transaction, TxEnvelope, transaction::SignerRecoverable};
use alloy_eips::{eip2718::Decodable2718, eip2930::AccessList};
use op_atomic_builder::router::{Builder, Chain};
use op_revm::{DefaultOp, OpSpecId, OpTransaction};
use revm::{
    Context,
    context::{CfgEnv, TxEnv},
    database::CacheDB,
    database_interface::{DBErrorMarker, DatabaseRef},
    primitives::{Address, B256, Bytes, U256},
    state::{AccountInfo, Bytecode},
};
use serde::{Deserialize, Serialize, de::DeserializeOwned};
use serde_json::{Value, json};
use std::{
    cell::RefCell,
    collections::BTreeMap,
    io::{self, BufRead, Write},
    rc::Rc,
};
const MAX_LINE: u64 = 16 * 1024 * 1024;
#[derive(Debug, Clone, thiserror::Error)]
#[error("{0}")]
struct Error(String);
impl DBErrorMarker for Error {}
impl From<io::Error> for Error {
    fn from(e: io::Error) -> Self {
        Self(e.to_string())
    }
}
impl From<serde_json::Error> for Error {
    fn from(e: serde_json::Error) -> Self {
        Self(e.to_string())
    }
}
struct Wire<R, W> {
    input: R,
    output: W,
}
impl<R: BufRead, W: Write> Wire<R, W> {
    fn read<T: DeserializeOwned>(&mut self) -> Result<T, Error> {
        let mut line = String::new();
        io::Read::take(&mut self.input, MAX_LINE + 1).read_line(&mut line)?;
        if line.len() as u64 > MAX_LINE || !line.ends_with('\n') {
            return Err(Error("truncated or oversized worker message".into()));
        }
        Ok(serde_json::from_str(&line)?)
    }
    fn write(&mut self, value: &Value) -> Result<(), Error> {
        serde_json::to_writer(&mut self.output, value)?;
        self.output.write_all(b"\n")?;
        self.output.flush()?;
        Ok(())
    }
    fn request<T: DeserializeOwned>(&mut self, value: Value) -> Result<T, Error> {
        self.write(&value)?;
        let reply: Value = self.read()?;
        if let Some(error) = reply.get("error") {
            return Err(Error(error.to_string()));
        }
        Ok(serde_json::from_value(
            reply.get("result").cloned().ok_or_else(|| Error("missing host result".into()))?,
        )?)
    }
}
struct Remote<R, W> {
    chain: u64,
    wire: Rc<RefCell<Wire<R, W>>>,
}
impl<R, W> Clone for Remote<R, W> {
    fn clone(&self) -> Self {
        Self { chain: self.chain, wire: self.wire.clone() }
    }
}
#[derive(Deserialize)]
struct Account {
    balance: U256,
    nonce: u64,
    code: Bytes,
}
impl<R: BufRead, W: Write> DatabaseRef for Remote<R, W> {
    type Error = Error;
    fn basic_ref(&self, address: Address) -> Result<Option<AccountInfo>, Error> {
        let a: Option<Account> = self
            .wire
            .borrow_mut()
            .request(json!({"method":"account", "chain":self.chain, "address":address}))?;
        Ok(a.map(|a| {
            let code = Bytecode::new_raw(a.code);
            AccountInfo {
                balance: a.balance,
                nonce: a.nonce,
                code_hash: code.hash_slow(),
                code: Some(code),
                ..Default::default()
            }
        }))
    }
    fn code_by_hash_ref(&self, _: B256) -> Result<Bytecode, Error> {
        Err(Error("code must be loaded with its account".into()))
    }
    fn storage_ref(&self, address: Address, index: U256) -> Result<U256, Error> {
        self.wire.borrow_mut().request(
            json!({"method":"storage", "chain":self.chain, "address":address, "index":index}),
        )
    }
    fn block_hash_ref(&self, number: u64) -> Result<B256, Error> {
        self.wire
            .borrow_mut()
            .request(json!({"method":"block_hash", "chain":self.chain, "number":number}))
    }
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct ChainInput {
    chain: u64,
    router: Address,
    sender: Address,
    application_gas: u64,
    prefix_logs: u32,
    prefix_gas: u64,
    spec: OpSpecId,
    number: u64,
    timestamp: u64,
    gas_limit: u64,
    base_fee: u64,
    coinbase: Address,
    prev_randao: B256,
}
#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct BuildInput {
    version: u8,
    root: u64,
    nonce: U256,
    target: Address,
    data: Bytes,
    max_calls: u16,
    max_bytes: usize,
    chains: Vec<ChainInput>,
}
#[derive(Serialize)]
struct Included {
    chain: u64,
    raw: Bytes,
    gas_used: u64,
    success: bool,
    logs: Vec<revm::primitives::Log>,
}
fn signed(raw: Bytes, chain: u64, accesses: &AccessList) -> Result<OpTransaction<TxEnv>, Error> {
    let mut cursor = raw.as_ref();
    let tx = TxEnvelope::decode_2718(&mut cursor).map_err(|e| Error(e.to_string()))?;
    if !cursor.is_empty() ||
        !matches!(tx, TxEnvelope::Eip1559(_)) ||
        tx.chain_id() != Some(chain) ||
        tx.access_list() != Some(accesses)
    {
        return Err(Error("expected exact chain-bound EIP-1559 envelope/access list".into()));
    }
    let caller = tx.recover_signer().map_err(|e| Error(e.to_string()))?;
    let base = TxEnv::builder()
        .caller(caller)
        .kind(tx.kind())
        .value(tx.value())
        .data(tx.input().clone())
        .nonce(tx.nonce())
        .gas_limit(tx.gas_limit())
        .gas_price(tx.max_fee_per_gas())
        .gas_priority_fee(tx.max_priority_fee_per_gas())
        .chain_id(Some(chain))
        .access_list(accesses.clone())
        .build()
        .map_err(|e| Error(e.to_string()))?;
    Ok(OpTransaction { base, enveloped_tx: Some(raw), deposit: Default::default() })
}
fn run<R: BufRead, W: Write>(wire: Rc<RefCell<Wire<R, W>>>) -> Result<(), Error> {
    let input: BuildInput = wire.borrow_mut().read()?;
    if input.version != 1 ||
        input.chains.is_empty() ||
        input.chains.len() > 32 ||
        input.max_bytes > 1024 * 1024
    {
        return Err(Error("unsupported worker protocol or bounds".into()));
    }
    let mut chains = BTreeMap::new();
    let mut remaining = BTreeMap::new();
    for c in input.chains {
        if c.spec != OpSpecId::LAGOON ||
            c.number == 0 ||
            c.prefix_gas > c.gas_limit ||
            chains.contains_key(&c.chain)
        {
            return Err(Error("expected unique Lagoon chain and valid candidate prefix".into()));
        }
        remaining.insert(c.chain, c.gas_limit - c.prefix_gas);
        let db = CacheDB::new(Remote { chain: c.chain, wire: wire.clone() });
        let context = Context::op()
            .with_db(db)
            .with_cfg(CfgEnv::new_with_spec(c.spec).with_chain_id(c.chain))
            .modify_block_chained(|b| {
                b.number = U256::from(c.number);
                b.timestamp = U256::from(c.timestamp);
                b.gas_limit = c.gas_limit;
                b.basefee = c.base_fee;
                b.beneficiary = c.coinbase;
                b.prevrandao = Some(c.prev_randao);
                b.set_blob_excess_gas_and_price(0, 5007716);
            });
        chains.insert(
            c.chain,
            Chain {
                context,
                router: c.router,
                sender: c.sender,
                prefix_logs: c.prefix_logs,
                application_gas: c.application_gas,
            },
        );
    }
    let mut builder = Builder {
        chains,
        max_calls: input.max_calls,
        max_bytes: input.max_bytes,
        envelope: |chain, data: Bytes, accesses: AccessList| {
            let raw: Bytes = wire
                .borrow_mut()
                .request(
                    json!({"method":"envelope", "chain":chain, "data":data, "accesses":accesses}),
                )
                .map_err(|e| e.to_string())?;
            let tx = signed(raw, chain, &accesses).map_err(|e| e.to_string())?;
            if tx.base.gas_limit > remaining[&chain] {
                return Err("transaction exceeds remaining block gas".into());
            }
            Ok(tx)
        },
    };
    let bundle = builder
        .build(input.root, input.nonce, input.target, input.data)
        .map_err(|e| Error(e.to_string()))?;
    let mut included = Vec::new();
    for (chain, result) in bundle.chains {
        included.push(Included {
            chain,
            raw: result.transaction.enveloped_tx.unwrap(),
            gas_used: result.execution.result.tx_gas_used(),
            success: result.execution.result.is_success(),
            logs: result.execution.result.logs().to_vec(),
        });
    }
    wire.borrow_mut()
        .write(&json!({"method":"complete", "reverted":bundle.reverted, "included":included}))
}
fn main() {
    let wire =
        Rc::new(RefCell::new(Wire { input: io::stdin().lock(), output: io::stdout().lock() }));
    if let Err(error) = run(wire.clone()) {
        let _ = wire.borrow_mut().write(&json!({"method":"error", "error":error.to_string()}));
        std::process::exit(1);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::{SignableTransaction, TxEip1559};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_signer::SignerSync;
    use alloy_signer_local::PrivateKeySigner;
    use revm::primitives::{TxKind, address};

    #[test]
    fn exact_signed_envelope_is_bound_to_chain_and_access_list() {
        let signer: PrivateKeySigner = B256::with_last_byte(1).to_string().parse().unwrap();
        let tx = TxEip1559 {
            chain_id: 901,
            nonce: 7,
            gas_limit: 100_000,
            max_fee_per_gas: 100,
            max_priority_fee_per_gas: 3,
            to: TxKind::Call(address!("1000000000000000000000000000000000000000")),
            ..Default::default()
        };
        let signature = signer.sign_hash_sync(&tx.signature_hash()).unwrap();
        let envelope: TxEnvelope = tx.into_signed(signature).into();
        let raw: Bytes = envelope.encoded_2718().into();
        let parsed = signed(raw.clone(), 901, &AccessList::default()).unwrap();
        assert_eq!(parsed.base.caller, signer.address());
        assert_eq!(parsed.base.nonce, 7);
        assert_eq!(parsed.base.gas_price, 100);
        assert_eq!(parsed.enveloped_tx, Some(raw.clone()));
        assert!(signed(raw.clone(), 902, &AccessList::default()).is_err());
        let wrong = AccessList(vec![alloy_eips::eip2930::AccessListItem {
            address: Address::ZERO,
            storage_keys: vec![],
        }]);
        assert!(signed(raw.clone(), 901, &wrong).is_err());
        let mut trailing = raw.to_vec();
        trailing.push(0);
        assert!(signed(trailing.into(), 901, &AccessList::default()).is_err());
    }
    #[test]
    fn truncated_and_oversized_messages_fail() {
        for bytes in [b"{}".to_vec(), vec![b' '; MAX_LINE as usize + 1]] {
            let mut wire = Wire { input: io::Cursor::new(bytes), output: Vec::new() };
            assert!(wire.read::<Value>().is_err());
        }
    }
}
