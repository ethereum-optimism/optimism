use std::str::FromStr;

use alloy_consensus::SignableTransaction;
use alloy_primitives::{Address, B256, Signature, U256};
use k256::sha2::Sha256;
use op_alloy_consensus::OpTypedTransaction;
use reth_optimism_primitives::OpTransactionSigned;
use reth_primitives::Recovered;
use secp256k1::{Message, PublicKey, SECP256K1, Secp256k1, SecretKey, rand::rngs::OsRng};
use sha3::{Digest, Keccak256};

/// Simple struct to sign txs/messages.
/// Mainly used to sign payout txs from the builder and to create test data.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Signer {
    pub address: Address,
    pub pubkey: PublicKey,
    pub secret: SecretKey,
}

impl Signer {
    pub fn try_from_secret(secret: B256) -> Result<Self, secp256k1::Error> {
        let secret = SecretKey::from_slice(secret.as_ref())?;
        let pubkey = secret.public_key(SECP256K1);
        let address = public_key_to_address(&pubkey);

        Ok(Self {
            address,
            pubkey,
            secret,
        })
    }

    pub fn sign_message(&self, message: B256) -> Result<Signature, secp256k1::Error> {
        let s = SECP256K1
            .sign_ecdsa_recoverable(&Message::from_digest_slice(&message[..])?, &self.secret);
        let (rec_id, data) = s.serialize_compact();

        let signature = Signature::new(
            U256::try_from_be_slice(&data[..32]).expect("The slice has at most 32 bytes"),
            U256::try_from_be_slice(&data[32..64]).expect("The slice has at most 32 bytes"),
            i32::from(rec_id) != 0,
        );
        Ok(signature)
    }

    pub fn sign_tx(
        &self,
        tx: OpTypedTransaction,
    ) -> Result<Recovered<OpTransactionSigned>, secp256k1::Error> {
        let signature_hash = match &tx {
            OpTypedTransaction::Legacy(tx) => tx.signature_hash(),
            OpTypedTransaction::Eip2930(tx) => tx.signature_hash(),
            OpTypedTransaction::Eip1559(tx) => tx.signature_hash(),
            OpTypedTransaction::Eip7702(tx) => tx.signature_hash(),
            OpTypedTransaction::Deposit(_) => B256::ZERO,
        };
        let signature = self.sign_message(signature_hash)?;
        let signed = OpTransactionSigned::new_unhashed(tx, signature);
        Ok(Recovered::new_unchecked(signed, self.address))
    }

    pub fn random() -> Self {
        Self::try_from_secret(B256::random()).expect("failed to create random signer")
    }
}

impl FromStr for Signer {
    type Err = eyre::Error;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        Self::try_from_secret(B256::from_str(s)?)
            .map_err(|e| eyre::eyre!("invalid secret key {:?}", e.to_string()))
    }
}

pub fn generate_signer() -> Signer {
    let secp = Secp256k1::new();

    // Generate cryptographically secure random private key
    let private_key = SecretKey::new(&mut OsRng);

    // Derive public key
    let public_key = PublicKey::from_secret_key(&secp, &private_key);

    // Derive Ethereum address
    let address = public_key_to_address(&public_key);

    Signer {
        address,
        pubkey: public_key,
        secret: private_key,
    }
}

/// Converts a public key to an Ethereum address
pub fn public_key_to_address(public_key: &PublicKey) -> Address {
    // Get uncompressed public key (65 bytes: 0x04 + 64 bytes)
    let pubkey_bytes = public_key.serialize_uncompressed();

    // Skip the 0x04 prefix and hash the remaining 64 bytes
    let hash = Keccak256::digest(&pubkey_bytes[1..65]);

    // Take last 20 bytes as address
    Address::from_slice(&hash[12..32])
}

// Generate a key deterministically from a seed for debug and testing
// Do not use in production
pub fn generate_key_from_seed(seed: &str) -> Signer {
    // Hash the seed
    let mut hasher = Sha256::new();
    hasher.update(seed.as_bytes());
    let hash = hasher.finalize();

    // Create signing key
    let secp = Secp256k1::new();
    let private_key = SecretKey::from_slice(&hash).expect("Failed to create private key");
    let public_key = PublicKey::from_secret_key(&secp, &private_key);
    let address = public_key_to_address(&public_key);

    Signer {
        address,
        pubkey: public_key,
        secret: private_key,
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use alloy_consensus::{TxEip1559, transaction::SignerRecoverable};
    use alloy_primitives::{TxKind as TransactionKind, address, fixed_bytes};
    #[test]
    fn test_sign_transaction() {
        let secret =
            fixed_bytes!("7a3233fcd52c19f9ffce062fd620a8888930b086fba48cfea8fc14aac98a4dce");
        let address = address!("B2B9609c200CA9b7708c2a130b911dabf8B49B20");
        let signer = Signer::try_from_secret(secret).expect("signer creation");
        assert_eq!(signer.address, address);

        let tx = OpTypedTransaction::Eip1559(TxEip1559 {
            chain_id: 1,
            nonce: 2,
            gas_limit: 21000,
            max_fee_per_gas: 1000,
            max_priority_fee_per_gas: 20000,
            to: TransactionKind::Call(address),
            value: U256::from(3000u128),
            ..Default::default()
        });

        let signed_tx = signer.sign_tx(tx).expect("sign tx");
        assert_eq!(signed_tx.signer(), address);

        let signed = signed_tx.into_inner();
        assert_eq!(signed.recover_signer().ok(), Some(address));
    }

    #[test]
    fn test_public_key_format() {
        let secp = Secp256k1::new();
        let private_key = SecretKey::new(&mut OsRng);
        let public_key = PublicKey::from_secret_key(&secp, &private_key);

        let pubkey_bytes = public_key.serialize_uncompressed();

        // Verify the public key format
        assert_eq!(
            pubkey_bytes.len(),
            65,
            "Uncompressed public key should be 65 bytes"
        );
        assert_eq!(
            pubkey_bytes[0], 0x04,
            "Uncompressed public key should start with 0x04"
        );

        // Verify report data would be 64 bytes
        let report_data = &pubkey_bytes[1..65];
        assert_eq!(
            report_data.len(),
            64,
            "Report data should be exactly 64 bytes"
        );
    }

    #[test]
    fn test_deterministic_address_derivation() {
        // Test with a known private key to ensure deterministic results
        let secp = Secp256k1::new();
        let private_key = SecretKey::from_slice(&[0x42; 32]).unwrap();
        let public_key = PublicKey::from_secret_key(&secp, &private_key);

        let address1 = public_key_to_address(&public_key);
        let address2 = public_key_to_address(&public_key);

        assert_eq!(
            address1, address2,
            "Address derivation should be deterministic"
        );
    }
}
