// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { MIPS } from "src/cannon/MIPS.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import "src/libraries/DisputeTypes.sol";

contract MIPS_Test is CommonTest {
    MIPS internal mips;
    PreimageOracle internal oracle;

    function setUp() public virtual override {
        super.setUp();
        oracle = new PreimageOracle();
        mips = new MIPS(oracle);
        vm.store(address(mips), 0x0, bytes32(abi.encode(address(oracle))));
        vm.label(address(oracle), "PreimageOracle");
        vm.label(address(mips), "MIPS");
    }

    function test_step_abi_succeeds() external {
        uint32[32] memory registers;
        registers[16] = 0xbfff0000;
        MIPS.State memory state = MIPS.State({
            memRoot: hex"30be14bdf94d7a93989a6263f1e116943dc052d584730cae844bf330dfddce2f",
            preimageKey: bytes32(0),
            preimageOffset: 0,
            pc: 4,
            nextPC: 8,
            lo: 0,
            hi: 0,
            heap: 0,
            exitCode: 0,
            exited: false,
            step: 1,
            registers: registers
        });
        bytes memory proof =
            hex"3c10bfff3610fff0341100013c08ffff3508fffd34090003010950202d420001ae020008ae11000403e000080000000000000000000000000000000000000000ad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5b4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d3021ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85e58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a193440eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d887c22bd8750d34016ac3c66b5ff102dacdd73f6b014e710b51e8022af9a1968ffd70157e48063fc33c97a050f7f640233bf646cc98d9524c6b92bcf3ab56f839867cc5f7f196b93bae1e27e6320742445d290f2263827498b54fec539f756afcefad4e508c098b9a7e1d8feb19955fb02ba9675585078710969d3440f5054e0f9dc3e7fe016e050eff260334f18a5d4fe391d82092319f5964f2e2eb7c1c3a5f8b13a49e282f609c317a833fb8d976d11517c571d1221a265d25af778ecf8923490c6ceeb450aecdc82e28293031d10c7d73bf85e57bf041a97360aa2c5d99cc1df82d9c4b87413eae2ef048f94b4d3554cea73d92b0f7af96e0271c691e2bb5c67add7c6caf302256adedf7ab114da0acfe870d449a3a489f781d659e8beccda7bce9f4e8618b6bd2f4132ce798cdc7a60e7e1460a7299e3c6342a579626d22733e50f526ec2fa19a22b31e8ed50f23cd1fdf94c9154ed3a7609a2f1ff981fe1d3b5c807b281e4683cc6d6315cf95b9ade8641defcb32372f1c126e398ef7a5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0b46a28b6f55540f89444f63de0378e3d121be09e06cc9ded1c20e65876d36aa0c65e9645644786b620e2dd2ad648ddfcbf4a7e5b1a3a4ecfe7f64667a3f0b7e2f4418588ed35a2458cffeb39b93d26f18d2ab13bdce6aee58e7b99359ec2dfd95a9c16dc00d6ef18b7933a6f8dc65ccb55667138776f7dea101070dc8796e3774df84f40ae0c8229d0d6069e5c8f39a7c299677a09d367fc7b05e3bc380ee652cdc72595f74c7b1043d0e1ffbab734648c838dfb0527d971b602bc216c9619ef0abf5ac974a1ed57f4050aa510dd9c74f508277b39d7973bb2dfccc5eeb0618db8cd74046ff337f0a7bf2c8e03e10f642c1886798d71806ab1e888d9e5ee87d00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000";

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertNotEq(postState, bytes32(0));
    }

    function test_add_succeeds() external {
        uint32 insn = encodespec(17, 18, 8, 0x20); // add t0, s1, s2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[17] = 12;
        state.registers[18] = 20;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] + state.registers[18]; // t0
        expect.registers[17] = state.registers[17];
        expect.registers[18] = state.registers[18];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_addu_succeeds() external {
        uint32 insn = encodespec(17, 18, 8, 0x21); // addu t0, s1, s2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[17] = 12;
        state.registers[18] = 20;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] + state.registers[18]; // t0
        expect.registers[17] = state.registers[17];
        expect.registers[18] = state.registers[18];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_addi_succeeds() external {
        uint16 imm = 40;
        uint32 insn = encodeitype(0x8, 17, 8, imm); // addi t0, s1, 40
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 1; // t0
        state.registers[17] = 4; // s1
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] + imm;
        expect.registers[17] = state.registers[17];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_addiSign_succeeds() external {
        uint16 imm = 0xfffe; // -2
        uint32 insn = encodeitype(0x8, 17, 8, imm); // addi t0, s1, 40
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 1; // t0
        state.registers[17] = 2; // s1
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 0;
        expect.registers[17] = state.registers[17];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_addui_succeeds() external {
        uint16 imm = 40;
        uint32 insn = encodeitype(0x9, 17, 8, imm); // addui t0, s1, 40
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 1; // t0
        state.registers[17] = 4; // s1
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] + imm;
        expect.registers[17] = state.registers[17];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_sub_succeeds() external {
        uint32 insn = encodespec(17, 18, 8, 0x22); // sub t0, s1, s2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[17] = 20;
        state.registers[18] = 12;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] - state.registers[18]; // t0
        expect.registers[17] = state.registers[17];
        expect.registers[18] = state.registers[18];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_subu_succeeds() external {
        uint32 insn = encodespec(17, 18, 8, 0x23); // subu t0, s1, s2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[17] = 20;
        state.registers[18] = 12;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] - state.registers[18]; // t0
        expect.registers[17] = state.registers[17];
        expect.registers[18] = state.registers[18];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_and_succeeds() external {
        uint32 insn = encodespec(17, 18, 8, 0x24); // and t0, s1, s2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[17] = 1200;
        state.registers[18] = 490;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] & state.registers[18]; // t0
        expect.registers[17] = state.registers[17];
        expect.registers[18] = state.registers[18];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_andi_succeeds() external {
        uint16 imm = 40;
        uint32 insn = encodeitype(0xc, 17, 8, imm); // andi t0, s1, 40
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 1; // t0
        state.registers[17] = 4; // s1
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] & imm;
        expect.registers[17] = state.registers[17];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_or_succeeds() external {
        uint32 insn = encodespec(17, 18, 8, 0x25); // or t0, s1, s2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[17] = 1200;
        state.registers[18] = 490;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] | state.registers[18]; // t0
        expect.registers[17] = state.registers[17];
        expect.registers[18] = state.registers[18];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_ori_succeeds() external {
        uint16 imm = 40;
        uint32 insn = encodeitype(0xd, 17, 8, imm); // ori t0, s1, 40
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 1; // t0
        state.registers[17] = 4; // s1
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] | imm;
        expect.registers[17] = state.registers[17];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_xor_succeeds() external {
        uint32 insn = encodespec(17, 18, 8, 0x26); // xor t0, s1, s2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[17] = 1200;
        state.registers[18] = 490;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] ^ state.registers[18]; // t0
        expect.registers[17] = state.registers[17];
        expect.registers[18] = state.registers[18];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_xori_succeeds() external {
        uint16 imm = 40;
        uint32 insn = encodeitype(0xe, 17, 8, imm); // xori t0, s1, 40
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 1; // t0
        state.registers[17] = 4; // s1
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] ^ imm;
        expect.registers[17] = state.registers[17];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_nor_succeeds() external {
        uint32 insn = encodespec(17, 18, 8, 0x27); // nor t0, s1, s2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[17] = 1200;
        state.registers[18] = 490;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = ~(state.registers[17] | state.registers[18]); // t0
        expect.registers[17] = state.registers[17];
        expect.registers[18] = state.registers[18];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_slt_succeeds() external {
        uint32 insn = encodespec(17, 18, 8, 0x2a); // slt t0, s1, s2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[17] = 0xFF_FF_FF_FE; // -2
        state.registers[18] = 5;

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 1; // t0
        expect.registers[17] = state.registers[17];
        expect.registers[18] = state.registers[18];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");

        // swap and check again
        uint32 tmp = state.registers[17];
        state.registers[17] = state.registers[18];
        state.registers[18] = tmp;
        expect.registers[17] = state.registers[17];
        expect.registers[18] = state.registers[18];
        expect.registers[8] = 0; // t0
        postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_sltu_succeeds() external {
        uint32 insn = encodespec(17, 18, 8, 0x2b); // sltu t0, s1, s2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[17] = 1200;
        state.registers[18] = 490;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[17] < state.registers[18] ? 1 : 0; // t0
        expect.registers[17] = state.registers[17];
        expect.registers[18] = state.registers[18];

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_lb_succeeds() external {
        uint32 t1 = 0x100;
        uint32 insn = encodeitype(0x20, 0x9, 0x8, 0x4); // lb $t0, 4($t1)
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, 0x12_00_00_00);
        state.registers[8] = 0; // t0
        state.registers[9] = t1;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 0x12; // t0
        expect.registers[9] = t1;

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_lh_succeeds() external {
        uint32 t1 = 0x100;
        uint32 val = 0x12_23_00_00;
        uint32 insn = encodeitype(0x21, 0x9, 0x8, 0x4); // lh $t0, 4($t1)

        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, val);
        state.registers[8] = 0; // t0
        state.registers[9] = t1;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 0x12_23; // t0
        expect.registers[9] = t1;

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_lw_succeeds() external {
        uint32 t1 = 0x100;
        uint32 val = 0x12_23_45_67;
        uint32 insn = encodeitype(0x23, 0x9, 0x8, 0x4); // lw $t0, 4($t1)

        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, val);
        state.registers[8] = 0; // t0
        state.registers[9] = t1;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = val; // t0
        expect.registers[9] = t1;

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_lbu_succeeds() external {
        uint32 t1 = 0x100;
        uint32 insn = encodeitype(0x24, 0x9, 0x8, 0x4); // lbu $t0, 4($t1)
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, 0x12_23_00_00);
        state.registers[8] = 0; // t0
        state.registers[9] = t1;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 0x12; // t0
        expect.registers[9] = t1;

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_lhu_succeeds() external {
        uint32 t1 = 0x100;
        uint32 insn = encodeitype(0x25, 0x9, 0x8, 0x4); // lhu $t0, 4($t1)
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, 0x12_23_00_00);
        state.registers[8] = 0; // t0
        state.registers[9] = t1;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 0x12_23; // t0
        expect.registers[9] = t1;

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_lwl_succeeds() external {
        uint32 t1 = 0x100;
        uint32 insn = encodeitype(0x22, 0x9, 0x8, 0x4); // lwl $t0, 4($t1)
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, 0x12_34_56_78);
        state.registers[8] = 0xaa_bb_cc_dd; // t0
        state.registers[9] = t1;

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 0x12_34_56_78; // t0
        expect.registers[9] = t1;

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");

        // test unaligned address
        insn = encodeitype(0x22, 0x9, 0x8, 0x5); // lwl $t0, 5($t1)
        (state.memRoot, proof) = ffi.getCannonMemoryProof(0, insn, t1 + 4, 0x12_34_56_78);
        expect.memRoot = state.memRoot;
        expect.registers[8] = 0x34_56_78_dd; // t0
        postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_lwr_succeeds() external {
        uint32 t1 = 0x100;
        uint32 insn = encodeitype(0x26, 0x9, 0x8, 0x4); // lwr $t0, 4($t1)
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, 0x12_34_56_78);
        state.registers[8] = 0xaa_bb_cc_dd; // t0
        state.registers[9] = t1;

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 0xaa_bb_cc_12; // t0
        expect.registers[9] = t1;

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");

        // test unaligned address
        insn = encodeitype(0x26, 0x9, 0x8, 0x5); // lwr $t0, 5($t1)
        (state.memRoot, proof) = ffi.getCannonMemoryProof(0, insn, t1 + 4, 0x12_34_56_78);
        expect.memRoot = state.memRoot;
        expect.registers[8] = 0xaa_bb_12_34; // t0
        postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_sb_succeeds() external {
        uint32 t1 = 0x100;
        uint32 insn = encodeitype(0x28, 0x9, 0x8, 0x4); // sb $t0, 4($t1)
        // note. cannon memory is zero-initalized. mem[t+4] = 0 is a no-op
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, 0);
        state.registers[8] = 0xaa_bb_cc_dd; // t0
        state.registers[9] = t1;

        MIPS.State memory expect;
        (expect.memRoot,) = ffi.getCannonMemoryProof(0, insn, t1 + 4, 0xdd_00_00_00);
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[8];
        expect.registers[9] = state.registers[9];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_sh_succeeds() external {
        uint32 t1 = 0x100;
        uint32 insn = encodeitype(0x29, 0x9, 0x8, 0x4); // sh $t0, 4($t1)
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, 0);
        state.registers[8] = 0xaa_bb_cc_dd; // t0
        state.registers[9] = t1;

        MIPS.State memory expect;
        (expect.memRoot,) = ffi.getCannonMemoryProof(0, insn, t1 + 4, 0xcc_dd_00_00);
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[8];
        expect.registers[9] = state.registers[9];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_swl_succeeds() external {
        uint32 t1 = 0x100;
        uint32 insn = encodeitype(0x2a, 0x9, 0x8, 0x4); // swl $t0, 4($t1)
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, 0);
        state.registers[8] = 0xaa_bb_cc_dd; // t0
        state.registers[9] = t1;

        MIPS.State memory expect;
        (expect.memRoot,) = ffi.getCannonMemoryProof(0, insn, t1 + 4, 0xaa_bb_cc_dd);
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[8];
        expect.registers[9] = state.registers[9];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_sw_succeeds() external {
        uint32 t1 = 0x100;
        uint32 insn = encodeitype(0x2b, 0x9, 0x8, 0x4); // sw $t0, 4($t1)
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, 0);
        state.registers[8] = 0xaa_bb_cc_dd; // t0
        state.registers[9] = t1;

        MIPS.State memory expect;
        (expect.memRoot,) = ffi.getCannonMemoryProof(0, insn, t1 + 4, 0xaa_bb_cc_dd);
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[8];
        expect.registers[9] = state.registers[9];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_swr_succeeds() external {
        uint32 t1 = 0x100;
        uint32 insn = encodeitype(0x2e, 0x9, 0x8, 0x5); // swr $t0, 5($t1)
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, 0);
        state.registers[8] = 0xaa_bb_cc_dd; // t0
        state.registers[9] = t1;

        MIPS.State memory expect;
        (expect.memRoot,) = ffi.getCannonMemoryProof(0, insn, t1 + 4, 0xcc_dd_00_00);
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[8];
        expect.registers[9] = state.registers[9];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_ll_succeeds() external {
        uint32 t1 = 0x100;
        uint32 val = 0x12_23_45_67;
        uint32 insn = encodeitype(0x30, 0x9, 0x8, 0x4); // ll $t0, 4($t1)

        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, val);
        state.registers[8] = 0; // t0
        state.registers[9] = t1;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = val; // t0
        expect.registers[9] = t1;

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_sc_succeeds() external {
        uint32 t1 = 0x100;
        uint32 insn = encodeitype(0x38, 0x9, 0x8, 0x4); // sc $t0, 4($t1)
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, t1 + 4, 0);
        state.registers[8] = 0xaa_bb_cc_dd; // t0
        state.registers[9] = t1;

        MIPS.State memory expect;
        (expect.memRoot,) = ffi.getCannonMemoryProof(0, insn, t1 + 4, 0xaa_bb_cc_dd);
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 0x1;
        expect.registers[9] = state.registers[9];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_movn_succeeds() external {
        // test mips mov instruction
        uint32 insn = encodespec(0x9, 0xa, 0x8, 0xb); // movn $t0, $t1, $t2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 0xa; // t0
        state.registers[9] = 0xb; // t1
        state.registers[10] = 0x1; // t2

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[9];
        expect.registers[9] = state.registers[9];
        expect.registers[10] = state.registers[10];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");

        state.registers[10] = 0x0; // t2
        expect.registers[10] = 0x0; // t2
        expect.registers[8] = state.registers[8];
        postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_movz_succeeds() external {
        // test mips mov instruction
        uint32 insn = encodespec(0x9, 0xa, 0x8, 0xa); // movz $t0, $t1, $t2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 0xa; // t0
        state.registers[9] = 0xb; // t1
        state.registers[10] = 0x0; // t2

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[9];
        expect.registers[9] = state.registers[9];
        expect.registers[10] = state.registers[10];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");

        state.registers[10] = 0x1; // t2
        expect.registers[10] = 0x1; // t2
        expect.registers[8] = state.registers[8];
        postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_mflo_succeeds() external {
        uint32 insn = encodespec(0x0, 0x0, 0x8, 0x12); // mflo $t0
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.lo = 0xdeadbeef;

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.lo = state.lo;
        expect.registers[8] = state.lo;

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_mfhi_succeeds() external {
        uint32 insn = encodespec(0x0, 0x0, 0x8, 0x10); // mfhi $t0
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.hi = 0xdeadbeef;

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.hi = state.hi;
        expect.registers[8] = state.hi;

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_mthi_succeeds() external {
        uint32 insn = encodespec(0x8, 0x0, 0x0, 0x11); // mthi $t0
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 0xdeadbeef; // t0

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.hi = state.registers[8];
        expect.registers[8] = state.registers[8];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_mtlo_succeeds() external {
        uint32 insn = encodespec(0x8, 0x0, 0x0, 0x13); // mtlo $t0
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 0xdeadbeef; // t0

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.lo = state.registers[8];
        expect.registers[8] = state.registers[8];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_mul_succeeds() external {
        uint32 insn = encodespec2(0x9, 0xa, 0x8, 0x2); // mul t0, t1, t2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 5; // t1
        state.registers[10] = 2; // t2

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[9] * state.registers[10]; // t0
        expect.registers[9] = 5;
        expect.registers[10] = 2;

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_mult_succeeds() external {
        uint32 insn = encodespec(0x9, 0xa, 0x0, 0x18); // mult t1, t2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 0x0F_FF_00_00; // t1
        state.registers[10] = 100; // t2

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[9] = state.registers[9];
        expect.registers[10] = state.registers[10];
        expect.lo = 0x3F_9C_00_00;
        expect.hi = 0x6;

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_multu_succeeds() external {
        uint32 insn = encodespec(0x9, 0xa, 0x0, 0x19); // multu t1, t2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 0x0F_FF_00_00; // t1
        state.registers[10] = 100; // t2

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[9] = state.registers[9];
        expect.registers[10] = state.registers[10];
        expect.lo = 0x3F_9C_00_00;
        expect.hi = 0x6;

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_div_succeeds() external {
        uint32 insn = encodespec(0x9, 0xa, 0x0, 0x1a); // div t1, t2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 5; // t1
        state.registers[10] = 2; // t2

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[9] = state.registers[9];
        expect.registers[10] = state.registers[10];
        expect.lo = 2;
        expect.hi = 1;

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_divu_succeeds() external {
        uint32 insn = encodespec(0x9, 0xa, 0x0, 0x1b); // divu t1, t2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 5; // t1
        state.registers[10] = 2; // t2

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[9] = state.registers[9];
        expect.registers[10] = state.registers[10];
        expect.lo = 2;
        expect.hi = 1;

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_beq_succeeds() external {
        uint16 boff = 0x10;
        uint32 insn = encodeitype(0x4, 0x9, 0x8, boff); // beq $t0, $t1, 16
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 0xdeadbeef; // t0
        state.registers[9] = 0xdeadbeef; // t1

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + (uint32(boff) << 2);
        expect.step = state.step + 1;
        expect.registers[8] = 0xdeadbeef;
        expect.registers[9] = 0xdeadbeef;

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");

        // branch not taken
        state.registers[8] = 0xaa;
        expect.registers[8] = 0xaa;
        expect.nextPC = state.nextPC + 4;
        postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_bne_succeeds() external {
        uint16 boff = 0x10;
        uint32 insn = encodeitype(0x5, 0x9, 0x8, boff); // bne $t0, $t1, 16
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 0xdeadbeef; // t0
        state.registers[9] = 0xaa; // t1

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + (uint32(boff) << 2);
        expect.step = state.step + 1;
        expect.registers[8] = 0xdeadbeef;
        expect.registers[9] = 0xaa;

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_blez_succeeds() external {
        uint16 boff = 0x10;
        uint32 insn = encodeitype(0x6, 0x8, 0x0, boff); // blez $t0, 16
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 0; // t0

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + (uint32(boff) << 2);
        expect.step = state.step + 1;
        expect.registers[8] = 0;

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_bgtz_succeeds() external {
        uint16 boff = 0xa0;
        uint32 insn = encodeitype(0x7, 0x8, 0x0, boff); // bgtz $t0, 0xa0
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 1; // t0

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + (uint32(boff) << 2);
        expect.step = state.step + 1;
        expect.registers[8] = 1;

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_bltz_succeeds() external {
        uint16 boff = 0x10;
        uint32 insn = encodeitype(0x1, 0x8, 0x0, boff); // bltz $t0, 16
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 0xF0_00_00_00; // t0

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + (uint32(boff) << 2);
        expect.step = state.step + 1;
        expect.registers[8] = 0xF0_00_00_00;

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_bgez_succeeds() external {
        uint16 boff = 0x10;
        uint32 insn = encodeitype(0x1, 0x8, 0x1, boff); // bgez $t0, 16
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 0x00_00_00_01; // t0

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + (uint32(boff) << 2);
        expect.step = state.step + 1;
        expect.registers[8] = 0x00_00_00_01;

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_jump_succeeds() external {
        uint32 label = 0x02_00_00_02; // set the 26th bit to assert no sign extension
        uint32 insn = uint32(0x08_00_00_00) | label; // j label
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = label << 2;
        expect.step = state.step + 1;

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_jump_nonzeroRegion_succeeds() external {
        uint32 pcRegion1 = 0x10000000;
        uint32 label = 0x2;
        uint32 insn = uint32(0x08_00_00_00) | label; // j label
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(pcRegion1, insn, 0x4, 0);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = (state.nextPC & 0xF0_00_00_00) | (uint32(label) << 2);
        expect.step = state.step + 1;

        bytes memory witness = encodeState(state);
        bytes32 postState = mips.step(witness, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_jal_succeeds() external {
        uint32 label = 0x02_00_00_02; // set the 26th bit to assert no sign extension
        uint32 insn = uint32(0x0c_00_00_00) | label; // jal label
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = label << 2;
        expect.step = state.step + 1;
        expect.registers[31] = state.pc + 8; // ra

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_jal_nonzeroRegion_succeeds() external {
        uint32 pcRegion1 = 0x10000000;
        uint32 label = 0x2;
        uint32 insn = uint32(0x0c_00_00_00) | label; // jal label
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(pcRegion1, insn, 0x4, 0);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = (state.nextPC & 0xF0_00_00_00) | (uint32(label) << 2);
        expect.step = state.step + 1;
        expect.registers[31] = state.pc + 8; // ra

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_jr_succeeds() external {
        uint16 tgt = 0x34;
        uint32 insn = encodespec(0x8, 0, 0, 0x8); // jr t0
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = tgt;

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = tgt;
        expect.step = state.step + 1;
        expect.registers[8] = tgt;

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_jalr_succeeds() external {
        uint16 tgt = 0x34;
        uint32 insn = encodespec(0x8, 0, 0x9, 0x9); // jalr t1, t0
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = tgt; // t0

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = tgt;
        expect.step = state.step + 1;
        expect.registers[8] = tgt;
        expect.registers[9] = state.pc + 8; // t1

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_sll_succeeds() external {
        uint8 shiftamt = 4;
        uint32 insn = encodespec(0x0, 0x9, 0x8, uint16(shiftamt) << 6); // sll t0, t1, 3
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 0x20; // t1

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[9] << shiftamt;
        expect.registers[9] = state.registers[9];

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_srl_succeeds() external {
        uint8 shiftamt = 4;
        uint32 insn = encodespec(0x0, 0x9, 0x8, uint16(shiftamt) << 6 | 2); // srl t0, t1, 3
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 0x20; // t1

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[9] >> shiftamt;
        expect.registers[9] = state.registers[9];

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_sra_succeeds() external {
        uint8 shiftamt = 4;
        uint32 insn = encodespec(0x0, 0x9, 0x8, uint16(shiftamt) << 6 | 3); // sra t0, t1, 3
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 0x80_00_00_20; // t1

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 0xF8_00_00_02; // 4 shifts while preserving sign bit
        expect.registers[9] = state.registers[9];

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_sllv_succeeds() external {
        uint32 insn = encodespec(0xa, 0x9, 0x8, 4); // sllv t0, t1, t2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 0x20; // t1
        state.registers[10] = 4; // t2

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[9] << state.registers[10]; // t0
        expect.registers[9] = state.registers[9];
        expect.registers[10] = state.registers[10];

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_srlv_succeeds() external {
        uint32 insn = encodespec(0xa, 0x9, 0x8, 6); // srlv t0, t1, t2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 0x20_00; // t1
        state.registers[10] = 4; // t2

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[9] >> state.registers[10]; // t0
        expect.registers[9] = state.registers[9];
        expect.registers[10] = state.registers[10];

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_srav_succeeds() external {
        uint32 insn = encodespec(0xa, 0x9, 0x8, 7); // srav t0, t1, t2
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 0x20_00; // t1
        state.registers[10] = 4; // t2

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = state.registers[9] >> state.registers[10]; // t0
        expect.registers[9] = state.registers[9];
        expect.registers[10] = state.registers[10];

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_lui_succeeds() external {
        uint32 insn = encodeitype(0xf, 0x0, 0x8, 0x4); // lui $t0, 0x04
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 0x00_04_00_00; // t0

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_clo_succeeds() external {
        uint32 insn = encodespec2(0x9, 0x0, 0x8, 0x21); // clo t0, t1
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 0xFF_00_00_00; // t1

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 8; // t0
        expect.registers[9] = state.registers[9];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_clz_succeeds() external {
        uint32 insn = encodespec2(0x9, 0x0, 0x8, 0x20); // clz t0, t1
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[9] = 0x00_00_F0_00; // t1

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[8] = 16; // t0
        expect.registers[9] = state.registers[9];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_preimage_read_succeeds() external {
        uint32 pc = 0x0;
        uint32 insn = 0x0000000c; // syscall
        uint32 a1 = 0x4;
        uint32 a1_val = 0x0000abba;
        (bytes32 memRoot, bytes memory proof) = ffi.getCannonMemoryProof(pc, insn, a1, a1_val);

        uint32[32] memory registers;
        registers[2] = 4003; // read syscall
        registers[4] = 5; // fd
        registers[5] = a1; // addr
        registers[6] = 4; // count

        MIPS.State memory state = MIPS.State({
            memRoot: memRoot,
            preimageKey: bytes32(uint256(1) << 248 | 0x01),
            preimageOffset: 8, // start reading past the pre-image length prefix
            pc: pc,
            nextPC: pc + 4,
            lo: 0,
            hi: 0,
            heap: 0,
            exitCode: 0,
            exited: false,
            step: 1,
            registers: registers
        });
        bytes memory encodedState = encodeState(state);

        // prime the pre-image oracle
        bytes32 word = bytes32(uint256(0xdeadbeef) << 224);
        uint8 size = 4;
        uint8 partOffset = 8;
        oracle.loadLocalData(uint256(state.preimageKey), 0, word, size, partOffset);

        MIPS.State memory expect = state;
        expect.preimageOffset += 4;
        expect.pc = state.nextPC;
        expect.nextPC += 4;
        expect.step += 1;
        expect.registers[2] = 4; // return
        expect.registers[7] = 0; // errno
        // recompute merkle root of written pre-image
        (expect.memRoot,) = ffi.getCannonMemoryProof(pc, insn, a1, 0xdeadbeef);

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_preimage_write_succeeds() external {
        uint32 pc = 0x0;
        uint32 insn = 0x0000000c; // syscall
        uint32 a1 = 0x4;
        uint32 a1_val = 0x0000abba;
        (bytes32 memRoot, bytes memory proof) = ffi.getCannonMemoryProof(pc, insn, a1, a1_val);

        uint32[32] memory registers;
        registers[2] = 4004; // write syscall
        registers[4] = 6; // fd
        registers[5] = a1; // addr
        registers[6] = 4; // count

        MIPS.State memory state = MIPS.State({
            memRoot: memRoot,
            preimageKey: bytes32(0),
            preimageOffset: 1,
            pc: pc,
            nextPC: 4,
            lo: 0,
            hi: 0,
            heap: 0,
            exitCode: 0,
            exited: false,
            step: 1,
            registers: registers
        });
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect = state;
        expect.preimageOffset = 0; // preimage write resets offset
        expect.pc = state.nextPC;
        expect.nextPC += 4;
        expect.step += 1;
        expect.preimageKey = bytes32(uint256(0xabba));
        expect.registers[2] = 4; // return
        expect.registers[7] = 0; // errno

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_mmap_succeeds() external {
        uint32 insn = 0x0000000c; // syscall
        (bytes32 memRoot, bytes memory proof) = ffi.getCannonMemoryProof(0, insn);

        MIPS.State memory state;
        state.memRoot = memRoot;
        state.nextPC = 4;
        state.registers[2] = 4090; // mmap syscall
        state.registers[4] = 0x0; // a0
        state.registers[5] = 4095; // a1
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        // assert page allocation is aligned to 4k
        expect.step = state.step + 1;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.heap = state.heap + 4096;
        expect.registers[2] = 0; // return old heap
        expect.registers[4] = 0x0; // a0
        expect.registers[5] = 4095; // a1

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_brk_succeeds() external {
        uint32 insn = 0x0000000c; // syscall
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[2] = 4045; // brk syscall
        state.registers[4] = 0xdead;
        bytes memory encodedState = encodeState(state);

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.step = state.step + 1;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.registers[2] = 0x40000000;
        expect.registers[4] = state.registers[4]; // registers unchanged

        bytes32 postState = mips.step(encodedState, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_clone_succeeds() external {
        uint32 insn = 0x0000000c; // syscall
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[2] = 4120; // clone syscall

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.step = state.step + 1;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.registers[2] = 1;

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_exit_succeeds() external {
        uint32 insn = 0x0000000c; // syscall
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[2] = 4246; // exit_group syscall
        state.registers[4] = 0x5; // a0

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.pc;
        expect.nextPC = state.nextPC;
        expect.step = state.step + 1;
        expect.registers[2] = state.registers[2]; // unchanged
        expect.registers[4] = state.registers[4]; // unchanged
        expect.exited = true;
        expect.exitCode = uint8(state.registers[4]);

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_fcntl_succeeds() external {
        uint32 insn = 0x0000000c; // syscall
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[2] = 4055; // fnctl syscall
        state.registers[4] = 0x0; // a0
        state.registers[5] = 0x3; // a1

        MIPS.State memory expect;
        expect.memRoot = state.memRoot;
        expect.pc = state.nextPC;
        expect.nextPC = state.nextPC + 4;
        expect.step = state.step + 1;
        expect.registers[2] = 0;
        expect.registers[5] = state.registers[5];

        bytes32 postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");

        // assert O_WRONLY
        state.registers[4] = 0x1; // a0
        expect.registers[4] = state.registers[4];
        expect.registers[2] = 1;
        postState = mips.step(encodeState(state), proof, 0);
        assertEq(postState, outputState(expect), "unexpected post state");
    }

    function test_prestate_exited_succeeds() external {
        uint32 insn = 0x0000000c; // syscall
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.exited = true;

        bytes memory enc = encodeState(state);
        bytes32 postState = mips.step(enc, proof, 0);
        assertEq(postState, outputState(state), "unexpected post state");
    }

    function test_illegal_instruction_fails() external {
        uint32 illegal_insn = 0xFF_FF_FF_FF;
        // the illegal instruction is partially decoded as containing a memory operand
        // so we stuff random data to the expected address
        uint32 addr = 0xFF_FF_FF_FC; // 4-byte aligned ff..ff
        (bytes32 memRoot, bytes memory proof) = ffi.getCannonMemoryProof(0, illegal_insn, addr, 0);

        MIPS.State memory state;
        state.memRoot = memRoot;
        bytes memory encodedState = encodeState(state);
        vm.expectRevert("invalid instruction");
        mips.step(encodedState, proof, 0);
    }

    function test_invalid_root_fails() external {
        uint32 insn = 0x0000000c; // syscall
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[2] = 4246; // exit_group syscall
        state.registers[4] = 0x5; // a0

        // invalidate proof
        for (uint256 i = 0; i < proof.length; i++) {
            proof[i] = 0x0;
        }
        vm.expectRevert(hex"000000000000000000000000000000000000000000000000000000000badf00d");
        mips.step(encodeState(state), proof, 0);
    }

    function test_jump_inDelaySlot_fails() external {
        uint16 label = 0x2;
        uint32 insn = uint32(0x08_00_00_00) | label; // j label
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.nextPC = 0xa;

        vm.expectRevert("jump in delay slot");
        mips.step(encodeState(state), proof, 0);
    }

    function test_branch_inDelaySlot_fails() external {
        uint16 boff = 0x10;
        uint32 insn = encodeitype(0x4, 0x9, 0x8, boff); // beq $t0, $t1, 16
        (MIPS.State memory state, bytes memory proof) = constructMIPSState(0, insn, 0x4, 0);
        state.registers[8] = 0xdeadbeef; // t0
        state.registers[9] = 0xdeadbeef; // t1
        state.nextPC = 0xa;

        vm.expectRevert("branch in delay slot");
        mips.step(encodeState(state), proof, 0);
    }

    function encodeState(MIPS.State memory state) internal pure returns (bytes memory) {
        bytes memory registers;
        for (uint256 i = 0; i < state.registers.length; i++) {
            registers = bytes.concat(registers, abi.encodePacked(state.registers[i]));
        }
        return abi.encodePacked(
            state.memRoot,
            state.preimageKey,
            state.preimageOffset,
            state.pc,
            state.nextPC,
            state.lo,
            state.hi,
            state.heap,
            state.exitCode,
            state.exited,
            state.step,
            registers
        );
    }

    /// @dev MIPS VM status codes:
    ///      0. Exited with success (Valid)
    ///      1. Exited with success (Invalid)
    ///      2. Exited with failure (Panic)
    ///      3. Unfinished
    function vmStatus(MIPS.State memory state) internal pure returns (VMStatus out_) {
        if (!state.exited) {
            return VMStatuses.UNFINISHED;
        } else if (state.exitCode == 0) {
            return VMStatuses.VALID;
        } else if (state.exitCode == 1) {
            return VMStatuses.INVALID;
        } else {
            return VMStatuses.PANIC;
        }
    }

    function outputState(MIPS.State memory state) internal pure returns (bytes32 out_) {
        bytes memory enc = encodeState(state);
        VMStatus status = vmStatus(state);
        assembly {
            out_ := keccak256(add(enc, 0x20), 226)
            out_ := or(and(not(shl(248, 0xFF)), out_), shl(248, status))
        }
    }

    function constructMIPSState(
        uint32 pc,
        uint32 insn,
        uint32 addr,
        uint32 val
    )
        internal
        returns (MIPS.State memory state, bytes memory proof)
    {
        (state.memRoot, proof) = ffi.getCannonMemoryProof(pc, insn, addr, val);
        state.pc = pc;
        state.nextPC = pc + 4;
    }

    function encodeitype(uint8 opcode, uint8 rs, uint8 rt, uint16 imm) internal pure returns (uint32 insn) {
        insn = uint32(opcode) << 26 | uint32(rs) << 21 | uint32(rt) << 16 | imm;
    }

    function encodespec(uint8 rs, uint8 rt, uint8 rd, uint16 funct) internal pure returns (uint32 insn) {
        insn = uint32(rs) << 21 | uint32(rt) << 16 | uint32(rd) << 11 | uint32(funct);
    }

    function encodespec2(uint8 rs, uint8 rt, uint8 rd, uint8 funct) internal pure returns (uint32 insn) {
        insn = uint32(28) << 26 | uint32(rs) << 21 | uint32(rt) << 16 | uint32(rd) << 11 | uint32(funct);
    }
}
