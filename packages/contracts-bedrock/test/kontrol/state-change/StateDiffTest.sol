pragma solidity ^0.8.13;

import { StateDiffCheatcode } from "./StateDiffCheatcode.sol";
import { KontrolUtils } from "./KontrolUtils.sol";
import { Types } from "src/libraries/Types.sol";
import { OptimismPortalInterface as OptimismPortal, SuperchainConfigInterface as SuperchainConfig} from "./interfaces/KontrolInterfaces.sol";

contract StateDiffTest is StateDiffCheatcode, KontrolUtils {

    OptimismPortal optimismPortal;
    SuperchainConfig superchainConfig;

    function setUp() public {
        recreateDeployment();
        optimismPortal = OptimismPortal(payable(OptimismPortalProxyAddress));
        superchainConfig = SuperchainConfig(SuperchainConfigProxyAddress);
    }

        function test_kontrol_in_foundry(bytes[] memory _withdrawalProof,
                                         Types.OutputRootProof memory _outputRootProof,
                                         uint256 _l2OutputIndex,
                                         Types.WithdrawalTransaction memory _tx) external {
        /* _defaultTx = Types.WithdrawalTransaction({ */
        /*     nonce: 0, */
        /*     sender: alice, */
        /*     target: bob, */
        /*     value: 100, */
        /*     gasLimit: 100_000, */
        /*     data: hex"" */
        /*     }); */

            assert(optimismPortal.paused() == false);

            /* Pause Optimism Portal */
            vm.prank(optimismPortal.GUARDIAN());
            superchainConfig.pause("identifier");

            /* Portal is now paused */
            assert(optimismPortal.paused() == true);

            /* No one can call proveWithdrawalTransaction */
            vm.expectRevert("OptimismPortal: paused");
            optimismPortal.proveWithdrawalTransaction(
                                                      _tx,
                                                      _l2OutputIndex,
                                                      _outputRootProof,
                                                      _withdrawalProof
            );
    }


    function test_proveWithdrawalTransaction_paused(
                                /* WithdrawalTransaction args */
								/* uint256 _tx0, */
								address _tx1,
								address _tx2,
								/* uint256 _tx3, */
								/* uint256 _tx4, */
								/* bytes   memory _tx5, */
                                uint256 _l2OutputIndex,
                                /* OutputRootProof args */
                                bytes32 _outputRootProof0,
                                bytes32 _outputRootProof1,
                                bytes32 _outputRootProof2,
                                bytes32 _outputRootProof3
                                /* bytes[] calldata _withdrawalProof */
    ) external {
        uint256 _tx0 = kevm.freshUInt(32);
        uint256 _tx3 = kevm.freshUInt(32);
        uint256 _tx4 = kevm.freshUInt(32);
        bytes memory _tx5 = abi.encode(kevm.freshUInt(32));

        bytes[] memory _withdrawalProof = freshWithdrawalProof();
        /* bytes[] memory _withdrawalProof = new bytes[](1); */
        /* _withdrawalProof[0] = abi.encode(kevm.freshUInt(32)); */

        Types.WithdrawalTransaction memory _tx = createWithdrawalTransaction (
            _tx0,
            _tx1,
            _tx2,
            _tx3,
            _tx4,
            _tx5
            );
        Types.OutputRootProof memory _outputRootProof = Types.OutputRootProof(
            _outputRootProof0,
            _outputRootProof1,
            _outputRootProof2,
            _outputRootProof3
        );

        /* After deployment, Optimism portal is enabled */
        assert(optimismPortal.paused() == false);

        /* Pause Optimism Portal */
        vm.prank(optimismPortal.GUARDIAN());
        superchainConfig.pause("identifier");

        /* Portal is now paused */
        assert(optimismPortal.paused() == true);

        /* No one can call proveWithdrawalTransaction */
        /* vm.prank(address(uint160(kevm.freshUInt(20)))); */
        vm.expectRevert("OptimismPortal: paused");
        optimismPortal.proveWithdrawalTransaction(
                                                  _tx,
                                                  _l2OutputIndex,
                                                  _outputRootProof,
                                                  _withdrawalProof
        );
    }

}
