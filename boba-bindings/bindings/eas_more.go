// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/bobanetwork/v3-anchorage/boba-bindings/solc"
)

const EASStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"contracts/EAS/EAS.sol:EAS\",\"label\":\"_nonces\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_mapping(t_address,t_uint256)\"},{\"astId\":1001,\"contract\":\"contracts/EAS/EAS.sol:EAS\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"1\",\"type\":\"t_array(t_uint256)49_storage\"},{\"astId\":1002,\"contract\":\"contracts/EAS/EAS.sol:EAS\",\"label\":\"_db\",\"offset\":0,\"slot\":\"50\",\"type\":\"t_mapping(t_bytes32,t_struct(Attestation)1006_storage)\"},{\"astId\":1003,\"contract\":\"contracts/EAS/EAS.sol:EAS\",\"label\":\"_timestamps\",\"offset\":0,\"slot\":\"51\",\"type\":\"t_mapping(t_bytes32,t_uint64)\"},{\"astId\":1004,\"contract\":\"contracts/EAS/EAS.sol:EAS\",\"label\":\"_revocationsOffchain\",\"offset\":0,\"slot\":\"52\",\"type\":\"t_mapping(t_address,t_mapping(t_bytes32,t_uint64))\"},{\"astId\":1005,\"contract\":\"contracts/EAS/EAS.sol:EAS\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"53\",\"type\":\"t_array(t_uint256)47_storage\"}],\"types\":{\"t_address\":{\"encoding\":\"inplace\",\"label\":\"address\",\"numberOfBytes\":\"20\"},\"t_array(t_uint256)47_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[47]\",\"numberOfBytes\":\"1504\",\"base\":\"t_uint256\"},\"t_array(t_uint256)49_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[49]\",\"numberOfBytes\":\"1568\",\"base\":\"t_uint256\"},\"t_bool\":{\"encoding\":\"inplace\",\"label\":\"bool\",\"numberOfBytes\":\"1\"},\"t_bytes32\":{\"encoding\":\"inplace\",\"label\":\"bytes32\",\"numberOfBytes\":\"32\"},\"t_bytes_storage\":{\"encoding\":\"bytes\",\"label\":\"bytes\",\"numberOfBytes\":\"32\"},\"t_mapping(t_address,t_mapping(t_bytes32,t_uint64))\":{\"encoding\":\"mapping\",\"label\":\"mapping(address =\u003e mapping(bytes32 =\u003e uint64))\",\"numberOfBytes\":\"32\",\"key\":\"t_address\",\"value\":\"t_mapping(t_bytes32,t_uint64)\"},\"t_mapping(t_address,t_uint256)\":{\"encoding\":\"mapping\",\"label\":\"mapping(address =\u003e uint256)\",\"numberOfBytes\":\"32\",\"key\":\"t_address\",\"value\":\"t_uint256\"},\"t_mapping(t_bytes32,t_struct(Attestation)1006_storage)\":{\"encoding\":\"mapping\",\"label\":\"mapping(bytes32 =\u003e struct Attestation)\",\"numberOfBytes\":\"32\",\"key\":\"t_bytes32\",\"value\":\"t_struct(Attestation)1006_storage\"},\"t_mapping(t_bytes32,t_uint64)\":{\"encoding\":\"mapping\",\"label\":\"mapping(bytes32 =\u003e uint64)\",\"numberOfBytes\":\"32\",\"key\":\"t_bytes32\",\"value\":\"t_uint64\"},\"t_struct(Attestation)1006_storage\":{\"encoding\":\"inplace\",\"label\":\"struct Attestation\",\"numberOfBytes\":\"224\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"},\"t_uint64\":{\"encoding\":\"inplace\",\"label\":\"uint64\",\"numberOfBytes\":\"8\"}}}"

var EASStorageLayout = new(solc.StorageLayout)

var EASDeployedBin = "0x6080604052600436106101805760003560e01c8063b469318d116100d6578063e45d03f91161007f578063ed24911d11610059578063ed24911d1461049e578063f10b5cc8146104b3578063f17325e7146104e257600080fd5b8063e45d03f914610458578063e57a6b1b1461046b578063e71ff3651461047e57600080fd5b8063d45c4435116100b0578063d45c4435146103cf578063e13458fc14610406578063e30bb5631461041957600080fd5b8063b469318d14610322578063b83010d31461037c578063cf190f34146103af57600080fd5b8063469262671161013857806354fd4d501161011257806354fd4d50146102cd578063831e05a1146102e2578063a3112a64146102f557600080fd5b806346926267146102855780634cb7e9e51461029a5780634d003070146102ad57600080fd5b806317d7de7c1161016957806317d7de7c146102005780632d0335ab1461022257806344adc90e1461026557600080fd5b806312b11a171461018557806313893f61146101c7575b600080fd5b34801561019157600080fd5b507fdbfdf8dc2b135c26253e00d5b6cbe6f20457e003fd526d97cea183883570de615b6040519081526020015b60405180910390f35b3480156101d357600080fd5b506101e76101e23660046133ee565b6104f5565b60405167ffffffffffffffff90911681526020016101be565b34801561020c57600080fd5b5061021561053a565b6040516101be919061349e565b34801561022e57600080fd5b506101b461023d3660046134ea565b73ffffffffffffffffffffffffffffffffffffffff1660009081526020819052604090205490565b6102786102733660046133ee565b61056a565b6040516101be9190613507565b61029861029336600461354b565b6106a1565b005b6102986102a83660046133ee565b610725565b3480156102b957600080fd5b506101e76102c8366004613563565b61080d565b3480156102d957600080fd5b5061021561081a565b6102786102f03660046133ee565b6108bd565b34801561030157600080fd5b50610315610310366004613563565b610b0e565b6040516101be9190613663565b34801561032e57600080fd5b506101e761033d366004613676565b73ffffffffffffffffffffffffffffffffffffffff919091166000908152603460209081526040808320938352929052205467ffffffffffffffff1690565b34801561038857600080fd5b507fa98d02348410c9c76735e0d0bb1396f4015ac2bb9615f9c2611d19d7a8a996506101b4565b3480156103bb57600080fd5b506101e76103ca366004613563565b610cd1565b3480156103db57600080fd5b506101e76103ea366004613563565b60009081526033602052604090205467ffffffffffffffff1690565b6101b46104143660046136a2565b610cdf565b34801561042557600080fd5b50610448610434366004613563565b600090815260326020526040902054151590565b60405190151581526020016101be565b6102986104663660046133ee565b610de2565b6102986104793660046136dd565b610f5d565b34801561048a57600080fd5b506101e76104993660046133ee565b611002565b3480156104aa57600080fd5b506101b461103a565b3480156104bf57600080fd5b5060405173420000000000000000000000000000000000002081526020016101be565b6101b46104f03660046136ef565b611044565b60004282825b8181101561052e57610526338787848181106105195761051961372a565b9050602002013585611102565b6001016104fb565b50909150505b92915050565b60606105657f0000000000000000000000000000000000000000000000000000000000000000611201565b905090565b606060008267ffffffffffffffff81111561058757610587613759565b6040519080825280602002602001820160405280156105ba57816020015b60608152602001906001900390816105a55790505b509050600034815b8581101561068c577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff86018114368888848181106106025761060261372a565b90506020028101906106149190613788565b9050600061063b823561062a60208501856137c6565b61063391613a3f565b33888761138f565b805190915061064a9086613ae2565b945080602001518785815181106106635761066361372a565b6020026020010181905250806020015151860195505050506106858160010190565b90506105c2565b506106978383611aa1565b9695505050505050565b604080516001808252818301909252600091816020015b60408051808201909152600080825260208201528152602001906001900390816106b85790505090506106f336839003830160208401613b44565b816000815181106107065761070661372a565b602090810291909101015261072082358233346001611b6e565b505050565b3460005b82811015610807577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff83018114368585848181106107695761076961372a565b905060200281019061077b9190613788565b90506107e8813561078f6020840184613b60565b808060200260200160405190810160405280939291908181526020016000905b828210156107db576107cc60408302860136819003810190613b44565b815260200190600101906107af565b5050505050338786611b6e565b6107f29085613ae2565b935050506108008160010190565b9050610729565b50505050565b60004261053483826121a8565b60606108457f000000000000000000000000000000000000000000000000000000000000000061226a565b61086e7f000000000000000000000000000000000000000000000000000000000000000061226a565b6108977f000000000000000000000000000000000000000000000000000000000000000061226a565b6040516020016108a993929190613bc8565b604051602081830303815290604052905090565b606060008267ffffffffffffffff8111156108da576108da613759565b60405190808252806020026020018201604052801561090d57816020015b60608152602001906001900390816108f85790505b509050600034815b8581101561068c577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff86018114368888848181106109555761095561372a565b90506020028101906109679190613c3e565b905036600061097960208401846137c6565b909250905080158061099957506109936040840184613c72565b82141590505b156109d0576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60005b81811015610a9257610a8a604051806080016040528086600001358152602001858585818110610a0557610a0561372a565b9050602002810190610a179190613cd9565b610a2090613d0d565b8152602001610a326040880188613c72565b85818110610a4257610a4261372a565b905060600201803603810190610a589190613d84565b8152602001610a6d60808801606089016134ea565b73ffffffffffffffffffffffffffffffffffffffff1690526123a7565b6001016109d3565b506000610abb8435610aa48486613a3f565b610ab460808801606089016134ea565b8a8961138f565b8051909150610aca9088613ae2565b96508060200151898781518110610ae357610ae361372a565b6020026020010181905250806020015151880197505050505050610b078160010190565b9050610915565b604080516101408101825260008082526020820181905291810182905260608082018390526080820183905260a0820183905260c0820183905260e0820183905261010082019290925261012081019190915260008281526032602090815260409182902082516101408101845281548152600182015492810192909252600281015467ffffffffffffffff808216948401949094526801000000000000000081048416606084015270010000000000000000000000000000000090049092166080820152600382015460a0820152600482015473ffffffffffffffffffffffffffffffffffffffff90811660c0830152600583015490811660e083015274010000000000000000000000000000000000000000900460ff16151561010082015260068201805491929161012084019190610c4890613da0565b80601f0160208091040260200160405190810160405280929190818152602001828054610c7490613da0565b8015610cc15780601f10610c9657610100808354040283529160200191610cc1565b820191906000526020600020905b815481529060010190602001808311610ca457829003601f168201915b5050505050815250509050919050565b600042610534338483611102565b6000610cf2610ced83613ded565b6123a7565b604080516001808252818301909252600091816020015b6040805160c081018252600080825260208083018290529282018190526060808301829052608083015260a082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff909201910181610d09579050509050610d776020840184613cd9565b610d8090613d0d565b81600081518110610d9357610d9361372a565b6020908102919091010152610dbc833582610db460c0870160a088016134ea565b34600161138f565b60200151600081518110610dd257610dd261372a565b6020026020010151915050919050565b3460005b82811015610807577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff830181146000858584818110610e2757610e2761372a565b9050602002810190610e399190613c3e565b610e4290613ed2565b60208101518051919250901580610e5f5750816040015151815114155b15610e96576040517f947d5a8400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60005b8151811015610f2757610f1f604051806080016040528085600001518152602001848481518110610ecc57610ecc61372a565b6020026020010151815260200185604001518481518110610eef57610eef61372a565b60200260200101518152602001856060015173ffffffffffffffffffffffffffffffffffffffff16815250612536565b600101610e99565b50610f3d82600001518284606001518887611b6e565b610f479086613ae2565b9450505050610f568160010190565b9050610de6565b610f74610f6f36839003830183613fb1565b612536565b604080516001808252818301909252600091816020015b6040805180820190915260008082526020820152815260200190600190039081610f8b579050509050610fc636839003830160208401613b44565b81600081518110610fd957610fd961372a565b6020908102919091010152610720823582610ffa60e0860160c087016134ea565b346001611b6e565b60004282825b8181101561052e576110328686838181106110255761102561372a565b90506020020135846121a8565b600101611008565b60006105656125c4565b604080516001808252818301909252600091829190816020015b6040805160c081018252600080825260208083018290529282018190526060808301829052608083015260a082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff90920191018161105e5790505090506110cc6020840184613cd9565b6110d590613d0d565b816000815181106110e8576110e861372a565b6020908102919091010152610dbc8335823334600161138f565b73ffffffffffffffffffffffffffffffffffffffff83166000908152603460209081526040808320858452918290529091205467ffffffffffffffff1615611176576040517fec9d6eeb00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008381526020829052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff861690811790915590519091859173ffffffffffffffffffffffffffffffffffffffff8816917f92a1f7a41a7c585a8b09e25b195e225b1d43248daca46b0faf9e0792777a222991a450505050565b604080516020808252818301909252606091600091906020820181803683370190505090506000805b60208110156112cc5760008582602081106112475761124761372a565b1a60f81b90507fff00000000000000000000000000000000000000000000000000000000000000811660000361127d57506112cc565b808484815181106112905761129061372a565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350506001918201910161122a565b5060008167ffffffffffffffff8111156112e8576112e8613759565b6040519080825280601f01601f191660200182016040528015611312576020820181803683370190505b50905060005b82811015611386578381815181106113325761133261372a565b602001015160f81c60f81b82828151811061134f5761134f61372a565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350600101611318565b50949350505050565b60408051808201909152600081526060602082015284516040805180820190915260008152606060208201528167ffffffffffffffff8111156113d4576113d4613759565b6040519080825280602002602001820160405280156113fd578160200160208202803683370190505b5060208201526040517fa2ea7c6e000000000000000000000000000000000000000000000000000000008152600481018990526000907342000000000000000000000000000000000000209063a2ea7c6e90602401600060405180830381865afa15801561146f573d6000803e3d6000fd5b505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01682016040526114b5919081019061400d565b80519091506114f0576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008367ffffffffffffffff81111561150b5761150b613759565b6040519080825280602002602001820160405280156115aa57816020015b60408051610140810182526000808252602080830182905292820181905260608083018290526080830182905260a0830182905260c0830182905260e0830182905261010083019190915261012082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9092019101816115295790505b50905060008467ffffffffffffffff8111156115c8576115c8613759565b6040519080825280602002602001820160405280156115f1578160200160208202803683370190505b50905060005b85811015611a805760008b82815181106116135761161361372a565b60200260200101519050600067ffffffffffffffff16816020015167ffffffffffffffff161415801561165e57504267ffffffffffffffff16816020015167ffffffffffffffff1611155b15611695576040517f08e8b93700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b84604001511580156116a8575080604001515b156116df576040517f157bd4c300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006040518061014001604052806000801b81526020018f81526020016117034290565b67ffffffffffffffff168152602001836020015167ffffffffffffffff168152602001600067ffffffffffffffff16815260200183606001518152602001836000015173ffffffffffffffffffffffffffffffffffffffff1681526020018d73ffffffffffffffffffffffffffffffffffffffff16815260200183604001511515815260200183608001518152509050600080600090505b6117a583826126f8565b600081815260326020526040902054909250156117c45760010161179b565b81835260008281526032602090815260409182902085518155908501516001820155908401516002820180546060870151608088015167ffffffffffffffff908116700100000000000000000000000000000000027fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff92821668010000000000000000027fffffffffffffffffffffffffffffffff000000000000000000000000000000009094169190951617919091171691909117905560a0840151600382015560c084015160048201805473ffffffffffffffffffffffffffffffffffffffff9283167fffffffffffffffffffffffff000000000000000000000000000000000000000090911617905560e0850151600583018054610100880151151574010000000000000000000000000000000000000000027fffffffffffffffffffffff000000000000000000000000000000000000000000909116929093169190911791909117905561012084015184919060068201906119449082614133565b50505060608401511561199b57606084015160009081526032602052604090205461199b576040517fc5723b5100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b828786815181106119ae576119ae61372a565b60200260200101819052508360a001518686815181106119d0576119d061372a565b60200260200101818152505081896020015186815181106119f3576119f361372a565b6020026020010181815250508f8e73ffffffffffffffffffffffffffffffffffffffff16856000015173ffffffffffffffffffffffffffffffffffffffff167f8bf46bf4cfd674fa735a3d63ec1c9ad4153f033c290341f3a588b75685141b3585604051611a6391815260200190565b60405180910390a450505050611a798160010190565b90506115f7565b50611a9083838360008c8c612757565b845250919998505050505050505050565b606060008267ffffffffffffffff811115611abe57611abe613759565b604051908082528060200260200182016040528015611ae7578160200160208202803683370190505b5090506000805b855181101561052e576000868281518110611b0b57611b0b61372a565b6020026020010151905060005b8151811015611b6457818181518110611b3357611b3361372a565b6020026020010151858581518110611b4d57611b4d61372a565b602090810291909101015260019384019301611b18565b5050600101611aee565b6040517fa2ea7c6e0000000000000000000000000000000000000000000000000000000081526004810186905260009081907342000000000000000000000000000000000000209063a2ea7c6e90602401600060405180830381865afa158015611bdc573d6000803e3d6000fd5b505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0168201604052611c22919081019061400d565b8051909150611c5d576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b855160008167ffffffffffffffff811115611c7a57611c7a613759565b604051908082528060200260200182016040528015611d1957816020015b60408051610140810182526000808252602080830182905292820181905260608083018290526080830182905260a0830182905260c0830182905260e0830182905261010083019190915261012082015282527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff909201910181611c985790505b50905060008267ffffffffffffffff811115611d3757611d37613759565b604051908082528060200260200182016040528015611d60578160200160208202803683370190505b50905060005b8381101561218a5760008a8281518110611d8257611d8261372a565b6020908102919091018101518051600090815260329092526040909120805491925090611ddb576040517fc5723b5100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8c816001015414611e18576040517fbf37b20e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600581015473ffffffffffffffffffffffffffffffffffffffff8c8116911614611e6e576040517f4ca8886700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600581015474010000000000000000000000000000000000000000900460ff16611ec4576040517f157bd4c300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6002810154700100000000000000000000000000000000900467ffffffffffffffff1615611f1e576040517f905e710700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b426002820180547fffffffffffffffff0000000000000000ffffffffffffffffffffffffffffffff811670010000000000000000000000000000000067ffffffffffffffff948516810291821793849055604080516101408101825287548152600188015460208201529386169286169290921791830191909152680100000000000000008304841660608301529091049091166080820152600382015460a0820152600482015473ffffffffffffffffffffffffffffffffffffffff90811660c0830152600583015490811660e083015274010000000000000000000000000000000000000000900460ff161515610100820152600682018054839161012084019161202a90613da0565b80601f016020809104026020016040519081016040528092919081815260200182805461205690613da0565b80156120a35780601f10612078576101008083540402835291602001916120a3565b820191906000526020600020905b81548152906001019060200180831161208657829003601f168201915b5050505050815250508584815181106120be576120be61372a565b602002602001018190525081602001518484815181106120e0576120e061372a565b60200260200101818152505080600101548b73ffffffffffffffffffffffffffffffffffffffff168260040160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167ff930a6e2523c9cc298691873087a740550b8fc85a0680830414c148ed927f615856000015160405161217891815260200190565b60405180910390a45050600101611d66565b5061219a84838360018b8b612757565b9a9950505050505050505050565b60008281526033602052604090205467ffffffffffffffff16156121f8576040517f2e26794600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008281526033602052604080822080547fffffffffffffffffffffffffffffffffffffffffffffffff00000000000000001667ffffffffffffffff85169081179091559051909184917f5aafceeb1c7ad58e4a84898bdee37c02c0fc46e7d24e6b60e8209449f183459f9190a35050565b6060816000036122ad57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b81156122d757806122c18161424d565b91506122d09050600a836142b4565b91506122b1565b60008167ffffffffffffffff8111156122f2576122f2613759565b6040519080825280601f01601f19166020018201604052801561231c576020820181803683370190505b5090505b841561239f57612331600183613ae2565b915061233e600a866142c8565b6123499060306142dc565b60f81b81838151811061235e5761235e61372a565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350612398600a866142b4565b9450612320565b949350505050565b60208082015160408084015160608086015173ffffffffffffffffffffffffffffffffffffffff1660009081528086528381208054600181019091558751865187890151878901519589015160808a01518051908c01209851999a9799949895976124ad97612492977fdbfdf8dc2b135c26253e00d5b6cbe6f20457e003fd526d97cea183883570de619791939290918c9101978852602088019690965273ffffffffffffffffffffffffffffffffffffffff94909416604087015267ffffffffffffffff9290921660608601521515608085015260a084015260c083015260e08201526101000190565b60405160208183030381529060405280519060200120612b31565b9050846060015173ffffffffffffffffffffffffffffffffffffffff166124e282856000015186602001518760400151612b44565b73ffffffffffffffffffffffffffffffffffffffff161461252f576040517f8baa579f00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b5050505050565b60208181015160408084015160608086015173ffffffffffffffffffffffffffffffffffffffff1660009081528086528381208054600181019091558751865186517fa98d02348410c9c76735e0d0bb1396f4015ac2bb9615f9c2611d19d7a8a99650998101999099529588015291860193909352608085018190529293909291906124ad9060a001612492565b60003073ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001614801561262a57507f000000000000000000000000000000000000000000000000000000000000000046145b1561265457507f000000000000000000000000000000000000000000000000000000000000000090565b50604080517f00000000000000000000000000000000000000000000000000000000000000006020808301919091527f0000000000000000000000000000000000000000000000000000000000000000828401527f000000000000000000000000000000000000000000000000000000000000000060608301524660808301523060a0808401919091528351808403909101815260c0909201909252805191012090565b60208083015160c084015160e0850151604080870151606088015161010089015160a08a01516101208b0151945160009961273999989796918c91016142ef565b60405160208183030381529060405280519060200120905092915050565b845160009060018190036127af576127a7888860008151811061277c5761277c61372a565b6020026020010151886000815181106127975761279761372a565b6020026020010151888888612b6c565b915050610697565b602088015173ffffffffffffffffffffffffffffffffffffffff81166128415760005b82811015612835578781815181106127ec576127ec61372a565b602002602001015160001461282d576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6001016127d2565b50600092505050610697565b6000805b8381101561296b5760008982815181106128615761286161372a565b60200260200101519050806000141580156128e857508373ffffffffffffffffffffffffffffffffffffffff1663ce46e0466040518163ffffffff1660e01b8152600401602060405180830381865afa1580156128c2573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906128e691906143cd565b155b1561291f576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b87811115612959576040517f1101129400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b96879003969190910190600101612845565b508615612a46576040517f88e5b2d900000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8316906388e5b2d99083906129c8908d908d906004016143ea565b60206040518083038185885af11580156129e6573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190612a0b91906143cd565b612a41576040517fbf2f3a8b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b612b15565b6040517f91db0b7e00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8316906391db0b7e908390612a9c908d908d906004016143ea565b60206040518083038185885af1158015612aba573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190612adf91906143cd565b612b15576040517fe8bee83900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8415612b2457612b2486612e82565b9998505050505050505050565b6000610534612b3e6125c4565b83612e95565b6000806000612b5587878787612ed7565b91509150612b6281612fef565b5095945050505050565b602086015160009073ffffffffffffffffffffffffffffffffffffffff8116612bd1578515612bc7576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000915050610697565b8515801590612c4c57508073ffffffffffffffffffffffffffffffffffffffff1663ce46e0466040518163ffffffff1660e01b8152600401602060405180830381865afa158015612c26573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190612c4a91906143cd565b155b15612c83576040517f1574f9f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b83861115612cbd576040517f1101129400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b85840393508415612d9a576040517fe49617e100000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff82169063e49617e1908890612d1c908b90600401613663565b60206040518083038185885af1158015612d3a573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190612d5f91906143cd565b612d95576040517fccf3bb2700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b612e67565b6040517fe60c350500000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff82169063e60c3505908890612dee908b90600401613663565b60206040518083038185885af1158015612e0c573d6000803e3d6000fd5b50505050506040513d601f19601f82011682018060405250810190612e3191906143cd565b612e67576040517fbd8ba84d00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8215612e7657612e7684612e82565b50939695505050505050565b8015612e9257612e923382613248565b50565b6040517f190100000000000000000000000000000000000000000000000000000000000060208201526022810183905260428101829052600090606201612739565b6000807f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0831115612f0e5750600090506003612fe6565b8460ff16601b14158015612f2657508460ff16601c14155b15612f375750600090506004612fe6565b6040805160008082526020820180845289905260ff881692820192909252606081018690526080810185905260019060a0016020604051602081039080840390855afa158015612f8b573d6000803e3d6000fd5b50506040517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0015191505073ffffffffffffffffffffffffffffffffffffffff8116612fdf57600060019250925050612fe6565b9150600090505b94509492505050565b6000816004811115613003576130036144a3565b0361300b5750565b600181600481111561301f5761301f6144a3565b0361308b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601860248201527f45434453413a20696e76616c6964207369676e6174757265000000000000000060448201526064015b60405180910390fd5b600281600481111561309f5761309f6144a3565b03613106576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f45434453413a20696e76616c6964207369676e6174757265206c656e677468006044820152606401613082565b600381600481111561311a5761311a6144a3565b036131a7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45434453413a20696e76616c6964207369676e6174757265202773272076616c60448201527f75650000000000000000000000000000000000000000000000000000000000006064820152608401613082565b60048160048111156131bb576131bb6144a3565b03612e92576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45434453413a20696e76616c6964207369676e6174757265202776272076616c60448201527f75650000000000000000000000000000000000000000000000000000000000006064820152608401613082565b804710156132b2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f416464726573733a20696e73756666696369656e742062616c616e63650000006044820152606401613082565b60008273ffffffffffffffffffffffffffffffffffffffff168260405160006040518083038185875af1925050503d806000811461330c576040519150601f19603f3d011682016040523d82523d6000602084013e613311565b606091505b5050905080610720576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f416464726573733a20756e61626c6520746f2073656e642076616c75652c207260448201527f6563697069656e74206d617920686176652072657665727465640000000000006064820152608401613082565b60008083601f8401126133b457600080fd5b50813567ffffffffffffffff8111156133cc57600080fd5b6020830191508360208260051b85010111156133e757600080fd5b9250929050565b6000806020838503121561340157600080fd5b823567ffffffffffffffff81111561341857600080fd5b613424858286016133a2565b90969095509350505050565b60005b8381101561344b578181015183820152602001613433565b50506000910152565b6000815180845261346c816020860160208601613430565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006134b16020830184613454565b9392505050565b73ffffffffffffffffffffffffffffffffffffffff81168114612e9257600080fd5b80356134e5816134b8565b919050565b6000602082840312156134fc57600080fd5b81356134b1816134b8565b6020808252825182820181905260009190848201906040850190845b8181101561353f57835183529284019291840191600101613523565b50909695505050505050565b60006060828403121561355d57600080fd5b50919050565b60006020828403121561357557600080fd5b5035919050565b6000610140825184526020830151602085015260408301516135aa604086018267ffffffffffffffff169052565b5060608301516135c6606086018267ffffffffffffffff169052565b5060808301516135e2608086018267ffffffffffffffff169052565b5060a083015160a085015260c083015161361460c086018273ffffffffffffffffffffffffffffffffffffffff169052565b5060e083015161363c60e086018273ffffffffffffffffffffffffffffffffffffffff169052565b50610100838101511515908501526101208084015181860183905261069783870182613454565b6020815260006134b1602083018461357c565b6000806040838503121561368957600080fd5b8235613694816134b8565b946020939093013593505050565b6000602082840312156136b457600080fd5b813567ffffffffffffffff8111156136cb57600080fd5b820160c081850312156134b157600080fd5b600060e0828403121561355d57600080fd5b60006020828403121561370157600080fd5b813567ffffffffffffffff81111561371857600080fd5b8201604081850312156134b157600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc18336030181126137bc57600080fd5b9190910192915050565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe18436030181126137fb57600080fd5b83018035915067ffffffffffffffff82111561381657600080fd5b6020019150600581901b36038213156133e757600080fd5b60405160c0810167ffffffffffffffff8111828210171561385157613851613759565b60405290565b6040516080810167ffffffffffffffff8111828210171561385157613851613759565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016810167ffffffffffffffff811182821017156138c1576138c1613759565b604052919050565b600067ffffffffffffffff8211156138e3576138e3613759565b5060051b60200190565b8015158114612e9257600080fd5b80356134e5816138ed565b600067ffffffffffffffff82111561392057613920613759565b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01660200190565b600082601f83011261395d57600080fd5b813561397061396b82613906565b61387a565b81815284602083860101111561398557600080fd5b816020850160208301376000918101602001919091529392505050565b600060c082840312156139b457600080fd5b6139bc61382e565b905081356139c9816134b8565b8152602082013567ffffffffffffffff80821682146139e757600080fd5b8160208401526139f9604085016138fb565b6040840152606084013560608401526080840135915080821115613a1c57600080fd5b50613a298482850161394c565b60808301525060a082013560a082015292915050565b6000613a4d61396b846138c9565b80848252602080830192508560051b850136811115613a6b57600080fd5b855b81811015613aa757803567ffffffffffffffff811115613a8d5760008081fd5b613a9936828a016139a2565b865250938201938201613a6d565b50919695505050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b8181038181111561053457610534613ab3565b600060408284031215613b0757600080fd5b6040516040810181811067ffffffffffffffff82111715613b2a57613b2a613759565b604052823581526020928301359281019290925250919050565b600060408284031215613b5657600080fd5b6134b18383613af5565b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112613b9557600080fd5b83018035915067ffffffffffffffff821115613bb057600080fd5b6020019150600681901b36038213156133e757600080fd5b60008451613bda818460208901613430565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551613c16816001850160208a01613430565b60019201918201528351613c31816002840160208801613430565b0160020195945050505050565b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff818336030181126137bc57600080fd5b60008083357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe1843603018112613ca757600080fd5b83018035915067ffffffffffffffff821115613cc257600080fd5b60200191506060810236038213156133e757600080fd5b600082357fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff418336030181126137bc57600080fd5b600061053436836139a2565b600060608284031215613d2b57600080fd5b6040516060810181811067ffffffffffffffff82111715613d4e57613d4e613759565b604052905080823560ff81168114613d6557600080fd5b8082525060208301356020820152604083013560408201525092915050565b600060608284031215613d9657600080fd5b6134b18383613d19565b600181811c90821680613db457607f821691505b60208210810361355d577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b600060c08236031215613dff57600080fd5b613e07613857565b82358152602083013567ffffffffffffffff811115613e2557600080fd5b613e31368286016139a2565b602083015250613e443660408501613d19565b604082015260a0830135613e57816134b8565b606082015292915050565b600082601f830112613e7357600080fd5b81356020613e8361396b836138c9565b82815260609283028501820192828201919087851115613ea257600080fd5b8387015b85811015613ec557613eb88982613d19565b8452928401928101613ea6565b5090979650505050505050565b600060808236031215613ee457600080fd5b613eec613857565b8235815260208084013567ffffffffffffffff80821115613f0c57600080fd5b9085019036601f830112613f1f57600080fd5b8135613f2d61396b826138c9565b81815260069190911b83018401908481019036831115613f4c57600080fd5b938501935b82851015613f7557613f633686613af5565b82528582019150604085019450613f51565b80868801525050506040860135925080831115613f9157600080fd5b5050613f9f36828601613e62565b604083015250613e57606084016134da565b600060e08284031215613fc357600080fd5b613fcb613857565b82358152613fdc8460208501613af5565b6020820152613fee8460608501613d19565b604082015260c0830135614001816134b8565b60608201529392505050565b6000602080838503121561402057600080fd5b825167ffffffffffffffff8082111561403857600080fd5b908401906080828703121561404c57600080fd5b614054613857565b8251815283830151614065816134b8565b818501526040830151614077816138ed565b604082015260608301518281111561408e57600080fd5b80840193505086601f8401126140a357600080fd5b825191506140b361396b83613906565b82815287858486010111156140c757600080fd5b6140d683868301878701613430565b60608201529695505050505050565b601f82111561072057600081815260208120601f850160051c8101602086101561410c5750805b601f850160051c820191505b8181101561412b57828155600101614118565b505050505050565b815167ffffffffffffffff81111561414d5761414d613759565b6141618161415b8454613da0565b846140e5565b602080601f8311600181146141b4576000841561417e5750858301515b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600386901b1c1916600185901b17855561412b565b6000858152602081207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08616915b82811015614201578886015182559484019460019091019084016141e2565b508582101561423d57878501517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600388901b60f8161c191681555b5050505050600190811b01905550565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361427e5761427e613ab3565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b6000826142c3576142c3614285565b500490565b6000826142d7576142d7614285565b500690565b8082018082111561053457610534613ab3565b89815260007fffffffffffffffffffffffffffffffffffffffff000000000000000000000000808b60601b166020840152808a60601b166034840152507fffffffffffffffff000000000000000000000000000000000000000000000000808960c01b166048840152808860c01b1660508401525085151560f81b60588301528460598301528351614388816079850160208801613430565b80830190507fffffffff000000000000000000000000000000000000000000000000000000008460e01b166079820152607d81019150509a9950505050505050505050565b6000602082840312156143df57600080fd5b81516134b1816138ed565b6000604082016040835280855180835260608501915060608160051b8601019250602080880160005b8381101561445f577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa088870301855261444d86835161357c565b95509382019390820190600101614413565b50508584038187015286518085528782019482019350915060005b828110156144965784518452938101939281019260010161447a565b5091979650505050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fdfea164736f6c6343000813000a"

func init() {
	if err := json.Unmarshal([]byte(EASStorageLayoutJSON), EASStorageLayout); err != nil {
		panic(err)
	}

	layouts["EAS"] = EASStorageLayout
	deployedBytecodes["EAS"] = EASDeployedBin
}
