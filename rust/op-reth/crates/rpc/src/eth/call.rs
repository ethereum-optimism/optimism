use crate::{
    OpEthApi, OpEthApiError,
    eth::{OpRpcProvider, RpcNodeCore},
};
use reth_evm::{EvmEnvFor, env::BlockEnvironment};
use reth_optimism_evm::ConfigurePostExecEvm;
use reth_primitives_traits::{HeaderTy, SealedHeader};
use reth_rpc_eth_api::{
    FromEvmError, RpcConvert,
    helpers::{Call, EthCall, estimate::EstimateCall},
};
use revm::context::Block;

impl<N, Rpc> EthCall for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    N::Provider: OpRpcProvider,
    N::Evm: ConfigurePostExecEvm,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError, Evm = N::Evm>,
{
}

impl<N, Rpc> EstimateCall for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    N::Provider: OpRpcProvider,
    N::Evm: ConfigurePostExecEvm,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError, Evm = N::Evm>,
{
}

impl<N, Rpc> Call for OpEthApi<N, Rpc>
where
    N: RpcNodeCore,
    N::Provider: OpRpcProvider,
    N::Evm: ConfigurePostExecEvm,
    OpEthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = OpEthApiError, Evm = N::Evm>,
{
    #[inline]
    fn call_gas_limit(&self) -> u64 {
        self.inner.eth_api.gas_cap()
    }

    #[inline]
    fn max_simulate_blocks(&self) -> u64 {
        self.inner.eth_api.max_simulate_blocks()
    }

    #[inline]
    fn compute_state_root_for_eth_simulate(&self) -> bool {
        self.inner.eth_api.compute_state_root_for_eth_simulate()
    }

    #[inline]
    fn evm_memory_limit(&self) -> u64 {
        self.inner.eth_api.evm_memory_limit()
    }

    fn apply_simulation_evm_env_overrides(
        &self,
        parent: &SealedHeader<HeaderTy<Self::Primitives>>,
        evm_env: &mut EvmEnvFor<Self::Evm>,
    ) -> Result<(), Self::Error> {
        let next_timestamp = evm_env.block_env.timestamp().saturating_to();
        evm_env.block_env.inner_mut().basefee =
            self.base_fee_quote_at_timestamp(parent.header(), next_timestamp)?;
        Ok(())
    }
}
