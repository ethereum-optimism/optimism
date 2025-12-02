//! Kona quantity serialization and deserialization helpers.

use alloc::string::ToString;
use core::str::FromStr;
use private::ConvertRuint;
use serde::{self, Deserialize, Deserializer, Serialize, Serializer, de};
use serde_json::Value;

/// Serializes a primitive number as a "quantity" hex string.
pub fn serialize<T, S>(value: &T, serializer: S) -> Result<S::Ok, S::Error>
where
    T: ConvertRuint,
    S: Serializer,
{
    value.into_ruint().serialize(serializer)
}

/// Deserializes a primitive number from a "quantity" hex string or raw number.
pub fn deserialize<'de, T, D>(deserializer: D) -> Result<T, D::Error>
where
    T: ConvertRuint,
    D: Deserializer<'de>,
{
    use serde::de::Error;
    match Value::deserialize(deserializer)? {
        Value::String(s) => T::Ruint::from_str(&s)
            .map_err(|_| D::Error::custom("failed to deserialize str"))
            .map(T::from_ruint),
        Value::Number(num) => T::Ruint::from_str(&num.to_string())
            .map_err(|_| de::Error::custom("failed to deserialize number"))
            .map(T::from_ruint),
        _ => Err(de::Error::custom("only string and number types are supported")),
    }
}

/// Private implementation details of the [`quantity`](self) module.
#[allow(unnameable_types)]
mod private {
    use core::str::FromStr;

    #[doc(hidden)]
    pub trait ConvertRuint: Copy + Sized {
        type Ruint: Copy
            + serde::Serialize
            + serde::de::DeserializeOwned
            + TryFrom<Self>
            + TryInto<Self>
            + FromStr;

        #[inline]
        fn into_ruint(self) -> Self::Ruint {
            // We have to use `Try*` traits because `From` is not implemented by ruint types.
            // They shouldn't ever error.
            self.try_into().ok().unwrap()
        }

        #[inline]
        fn from_ruint(ruint: Self::Ruint) -> Self {
            // We have to use `Try*` traits because `From` is not implemented by ruint types.
            // They shouldn't ever error.
            ruint.try_into().ok().unwrap()
        }
    }

    macro_rules! impl_from_ruint {
        ($($primitive:ty = $ruint:ty),* $(,)?) => {
            $(
                impl ConvertRuint for $primitive {
                    type Ruint = $ruint;
                }
            )*
        };
    }

    impl_from_ruint! {
        bool = alloy_primitives::ruint::aliases::U1,
        u8   = alloy_primitives::U8,
        u16  = alloy_primitives::U16,
        u32  = alloy_primitives::U32,
        u64  = alloy_primitives::U64,
        u128 = alloy_primitives::U128,
    }
}
