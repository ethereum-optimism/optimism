use alloy_primitives::{B64, Bytes};
use alloy_rlp::{Decodable, Encodable, Header};
use alloy_rpc_types_engine::PayloadId;
use ed25519_dalek::{Signature, Signer, SigningKey, Verifier, VerifyingKey};
use serde::{Deserialize, Serialize};
use thiserror::Error;

/// An authorization token that grants a builder permission to publish flashblocks for a specific payload.
///
/// The `authorizer_sig` is made over the `payload_id`, `timestamp`, and `builder_vk`. This is
/// useful because it allows the authorizer to control which builders can publish flashblocks in
/// real time, without relying on consumers to verify the builder's public key against a
/// pre-defined list.
#[derive(Copy, Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
pub struct Authorization {
    /// The unique identifier of the payload this authorization applies to
    pub payload_id: PayloadId,
    /// Unix timestamp when this authorization was created
    pub timestamp: u64,
    /// The public key of the builder who is authorized to sign messages
    pub builder_vk: VerifyingKey,
    /// The authorizer's signature over the payload_id, timestamp, and builder_vk
    pub authorizer_sig: Signature,
}

#[derive(Debug, Error, PartialEq)]
pub enum AuthorizationError {
    #[error("invalid authorizer signature")]
    InvalidAuthorizerSig,
}

impl Authorization {
    /// Creates a new authorization token for a builder to publish messages for a specific payload.
    ///
    /// This function creates a cryptographic authorization by signing a message containing the
    /// payload ID, timestamp, and builder's public key using the authorizer's signing key.
    ///
    /// # Arguments
    ///
    /// * `payload_id` - The unique identifier of the payload this authorization applies to
    /// * `timestamp` - Unix timestamp associated with this `payload_id`
    /// * `authorizer_sk` - The authorizer's signing key used to create the signature
    /// * `actor_vk` - The verifying key of the actor being authorized
    ///
    /// # Returns
    ///
    /// A new `Authorization` instance with the generated signature
    pub fn new(
        payload_id: PayloadId,
        timestamp: u64,
        authorizer_sk: &SigningKey,
        actor_vk: VerifyingKey,
    ) -> Self {
        let mut msg = payload_id.0.to_vec();
        msg.extend_from_slice(&timestamp.to_le_bytes());
        msg.extend_from_slice(actor_vk.as_bytes());
        let hash = blake3::hash(&msg);
        let sig = authorizer_sk.sign(hash.as_bytes());

        Self {
            payload_id,
            timestamp,
            builder_vk: actor_vk,
            authorizer_sig: sig,
        }
    }

    /// Verifies the authorization signature against the provided authorizer's verifying key.
    ///
    /// This function reconstructs the signed message from the authorization data and verifies
    /// that the signature was created by the holder of the authorizer's private key.
    ///
    /// # Arguments
    ///
    /// * `authorizer_sk` - The verifying key of the authorizer to verify against
    ///
    /// # Returns
    ///
    /// * `Ok(())` if the signature is valid
    /// * `Err(FlashblocksP2PError::InvalidAuthorizerSig)` if the signature is invalid
    pub fn verify(&self, authorizer_sk: VerifyingKey) -> Result<(), AuthorizationError> {
        let mut msg = self.payload_id.0.to_vec();
        msg.extend_from_slice(&self.timestamp.to_le_bytes());
        msg.extend_from_slice(self.builder_vk.as_bytes());
        let hash = blake3::hash(&msg);
        authorizer_sk
            .verify(hash.as_bytes(), &self.authorizer_sig)
            .map_err(|_| AuthorizationError::InvalidAuthorizerSig)
    }
}

impl Encodable for Authorization {
    fn encode(&self, out: &mut dyn alloy_rlp::BufMut) {
        // pre-serialize the key & sig once so we can reuse the bytes & lengths
        let pub_bytes = Bytes::copy_from_slice(self.builder_vk.as_bytes()); // 33 bytes
        let sig_bytes = Bytes::copy_from_slice(&self.authorizer_sig.to_bytes()); // 64 bytes

        let payload_len = self.payload_id.0.length()
            + self.timestamp.length()
            + pub_bytes.length()
            + sig_bytes.length();

        Header {
            list: true,
            payload_length: payload_len,
        }
        .encode(out);

        // 1. payload_id (inner B64 already Encodable)
        self.payload_id.0.encode(out);
        // 2. timestamp
        self.timestamp.encode(out);
        // 3. builder_pub
        pub_bytes.encode(out);
        // 4. authorizer_sig
        sig_bytes.encode(out);
    }

    fn length(&self) -> usize {
        let pub_bytes = Bytes::copy_from_slice(self.builder_vk.as_bytes());
        let sig_bytes = Bytes::copy_from_slice(&self.authorizer_sig.to_bytes());

        let payload_len = self.payload_id.0.length()
            + self.timestamp.length()
            + pub_bytes.length()
            + sig_bytes.length();

        Header {
            list: true,
            payload_length: payload_len,
        }
        .length()
            + payload_len
    }
}

impl Decodable for Authorization {
    fn decode(buf: &mut &[u8]) -> Result<Self, alloy_rlp::Error> {
        let header = Header::decode(buf)?;
        if !header.list {
            return Err(alloy_rlp::Error::UnexpectedString);
        }
        let mut body = &buf[..header.payload_length];

        // 1. payload_id
        let payload_id = alloy_rpc_types_engine::PayloadId(B64::decode(&mut body)?);

        // 2. timestamp
        let timestamp = u64::decode(&mut body)?;

        // 3. builder_pub
        let pub_bytes = Bytes::decode(&mut body)?;
        let builder_pub = VerifyingKey::try_from(pub_bytes.as_ref())
            .map_err(|_| alloy_rlp::Error::Custom("bad builder_pub"))?;

        // 4. authorizer_sig
        let sig_bytes = Bytes::decode(&mut body)?;
        let authorizer_sig = Signature::try_from(sig_bytes.as_ref())
            .map_err(|_| alloy_rlp::Error::Custom("bad signature"))?;

        // advance caller’s slice cursor
        *buf = &buf[header.payload_length..];

        Ok(Self {
            payload_id,
            timestamp,
            builder_vk: builder_pub,
            authorizer_sig,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rlp::{Decodable, Encodable, encode};

    fn key_pair(seed: u8) -> (SigningKey, VerifyingKey) {
        let bytes = [seed; 32];
        let sk = SigningKey::from_bytes(&bytes);
        let vk = sk.verifying_key();
        (sk, vk)
    }

    #[test]
    fn authorization_rlp_roundtrip_and_verify() {
        let (authorizer_sk, authorizer_vk) = key_pair(1);
        let (_, builder_vk) = key_pair(2);

        let auth = Authorization::new(
            PayloadId::default(),
            1_700_000_123,
            &authorizer_sk,
            builder_vk,
        );

        let encoded = encode(auth);
        assert_eq!(encoded.len(), auth.length(), "length impl correct");

        let mut slice = encoded.as_ref();
        let decoded = Authorization::decode(&mut slice).expect("decoding succeeds");
        assert!(slice.is_empty(), "decoder consumed all bytes");
        assert_eq!(decoded, auth, "round-trip preserves value");

        // Signature is valid
        decoded.verify(authorizer_vk).expect("signature verifies");
    }

    #[test]
    fn authorization_signature_tamper_is_detected() {
        let (authorizer_sk, authorizer_vk) = key_pair(1);
        let (_, builder_vk) = key_pair(2);

        let mut auth = Authorization::new(PayloadId::default(), 42, &authorizer_sk, builder_vk);

        let mut sig_bytes = auth.authorizer_sig.to_bytes();
        sig_bytes[0] ^= 1;
        auth.authorizer_sig = Signature::try_from(sig_bytes.as_ref()).unwrap();

        assert!(auth.verify(authorizer_vk).is_err());
    }
}
