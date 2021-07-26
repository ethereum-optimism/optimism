
# Run a single cvl e.g.:
#  make -B spec/certora/CErc20/borrowAndRepayFresh.cvl

# TODO:
#  - mintAndRedeemFresh.cvl in progress and is failing due to issues with tool proving how the exchange rate can change
#    hoping for better division modelling - currently fails to prove (a + 1) / b >= a / b
#  - CErc20Delegator/*.cvl cannot yet be run with the tool
#  - cDAI proofs are WIP, require using the delegate and the new revert message assertions

.PHONY: certora-clean

CERTORA_BIN = $(abspath script/certora)
CERTORA_RUN = $(CERTORA_BIN)/run.py
CERTORA_CLI = $(CERTORA_BIN)/cli.jar
CERTORA_EMV = $(CERTORA_BIN)/emv.jar

export CERTORA = $(CERTORA_BIN)
export CERTORA_DISABLE_POPUP = 1

spec/certora/Math/%.cvl:
	$(CERTORA_RUN) \
	 spec/certora/contracts/MathCertora.sol \
	--verify \
	 MathCertora:$@

spec/certora/Comp/search.cvl:
	$(CERTORA_RUN) \
	spec/certora/contracts/CompCertora.sol \
	--settings -b=4,-graphDrawLimit=0,-assumeUnwindCond,-depth=100 \
	--solc_args "'--evm-version istanbul'" \
	--verify \
	 CompCertora:$@

spec/certora/Comp/transfer.cvl:
	$(CERTORA_RUN) \
	spec/certora/contracts/CompCertora.sol \
	--settings -graphDrawLimit=0,-assumeUnwindCond,-depth=100 \
	--solc_args "'--evm-version istanbul'" \
	--verify \
	 CompCertora:$@

spec/certora/Governor/%.cvl:
	$(CERTORA_RUN) \
	 spec/certora/contracts/GovernorAlphaCertora.sol \
	 spec/certora/contracts/TimelockCertora.sol \
	 spec/certora/contracts/CompCertora.sol \
	 --settings -assumeUnwindCond,-enableWildcardInlining=false \
	 --solc_args "'--evm-version istanbul'" \
	 --link \
	 GovernorAlphaCertora:timelock=TimelockCertora \
	 GovernorAlphaCertora:comp=CompCertora \
	--verify \
	 GovernorAlphaCertora:$@

spec/certora/Comptroller/%.cvl:
	$(CERTORA_RUN) \
	 spec/certora/contracts/ComptrollerCertora.sol \
	 spec/certora/contracts/PriceOracleModel.sol \
	--link \
	 ComptrollerCertora:oracle=PriceOracleModel \
	--verify \
	 ComptrollerCertora:$@

spec/certora/cDAI/%.cvl:
	$(CERTORA_RUN) \
	 spec/certora/contracts/CDaiDelegateCertora.sol \
	 spec/certora/contracts/UnderlyingModelNonStandard.sol \
	 spec/certora/contracts/mcd/dai.sol:Dai \
	 spec/certora/contracts/mcd/pot.sol:Pot \
	 spec/certora/contracts/mcd/vat.sol:Vat \
	 spec/certora/contracts/mcd/join.sol:DaiJoin \
	 tests/Contracts/BoolComptroller.sol \
	--link \
	 CDaiDelegateCertora:comptroller=BoolComptroller \
	 CDaiDelegateCertora:underlying=Dai \
	 CDaiDelegateCertora:potAddress=Pot \
	 CDaiDelegateCertora:vatAddress=Vat \
	 CDaiDelegateCertora:daiJoinAddress=DaiJoin \
	--verify \
	 CDaiDelegateCertora:$@ \
	--settings -cache=certora-run-cdai

spec/certora/CErc20/%.cvl:
	$(CERTORA_RUN) \
	 spec/certora/contracts/CErc20ImmutableCertora.sol \
	 spec/certora/contracts/CTokenCollateral.sol \
	 spec/certora/contracts/ComptrollerCertora.sol \
	 spec/certora/contracts/InterestRateModelModel.sol \
	 spec/certora/contracts/UnderlyingModelNonStandard.sol \
	--link \
	 CErc20ImmutableCertora:otherToken=CTokenCollateral \
	 CErc20ImmutableCertora:comptroller=ComptrollerCertora \
	 CErc20ImmutableCertora:underlying=UnderlyingModelNonStandard \
	 CErc20ImmutableCertora:interestRateModel=InterestRateModelModel \
	 CTokenCollateral:comptroller=ComptrollerCertora \
	 CTokenCollateral:underlying=UnderlyingModelNonStandard \
	--verify \
	 CErc20ImmutableCertora:$@ \
	--settings -cache=certora-run-cerc20-immutable

spec/certora/CErc20Delegator/%.cvl:
	$(CERTORA_RUN) \
	 spec/certora/contracts/CErc20DelegatorCertora.sol \
	 spec/certora/contracts/CErc20DelegateCertora.sol \
	 spec/certora/contracts/CTokenCollateral.sol \
	 spec/certora/contracts/ComptrollerCertora.sol \
	 spec/certora/contracts/InterestRateModelModel.sol \
	 spec/certora/contracts/UnderlyingModelNonStandard.sol \
	--link \
	 CErc20DelegatorCertora:implementation=CErc20DelegateCertora \
	 CErc20DelegatorCertora:otherToken=CTokenCollateral \
	 CErc20DelegatorCertora:comptroller=ComptrollerCertora \
	 CErc20DelegatorCertora:underlying=UnderlyingModelNonStandard \
	 CErc20DelegatorCertora:interestRateModel=InterestRateModelModel \
	 CTokenCollateral:comptroller=ComptrollerCertora \
	 CTokenCollateral:underlying=UnderlyingModelNonStandard \
	--verify \
	 CErc20DelegatorCertora:$@ \
	--settings -assumeUnwindCond \
	--settings -cache=certora-run-cerc20-delegator

spec/certora/Maximillion/%.cvl:
	$(CERTORA_RUN) \
	 spec/certora/contracts/MaximillionCertora.sol \
	 spec/certora/contracts/CEtherCertora.sol \
	--link \
	 MaximillionCertora:cEther=CEtherCertora \
	--verify \
	 MaximillionCertora:$@

spec/certora/Timelock/%.cvl:
	$(CERTORA_RUN) \
	 spec/certora/contracts/TimelockCertora.sol \
	--verify \
	 TimelockCertora:$@

certora-clean:
	rm -rf .certora_build.json .certora_config certora_verify.json emv-*
