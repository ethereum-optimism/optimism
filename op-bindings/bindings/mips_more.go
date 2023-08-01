// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const MIPSStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"src/cannon/MIPS.sol:MIPS\",\"label\":\"oracle\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_contract(IPreimageOracle)1001\"}],\"types\":{\"t_contract(IPreimageOracle)1001\":{\"encoding\":\"inplace\",\"label\":\"contract IPreimageOracle\",\"numberOfBytes\":\"20\"}}}"

var MIPSStorageLayout = new(solc.StorageLayout)

var MIPSDeployedBin = "0x608060405234801561001057600080fd5b50600436106100415760003560e01c8063155633fe146100465780637dc0d1d01461006b578063f8e0cb96146100b0575b600080fd5b610051634000000081565b60405163ffffffff90911681526020015b60405180910390f35b60005461008b9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610062565b6100c36100be366004611baa565b6100d1565b604051908152602001610062565b60006100db611ad7565b608081146100e857600080fd5b604051610600146100f857600080fd5b6064861461010557600080fd5b610166841461011357600080fd5b8535608052602086013560a052604086013560e090811c60c09081526044880135821c82526048880135821c61010052604c880135821c610120526050880135821c61014052605488013590911c61016052605887013560f890811c610180526059880135901c6101a052605a870135901c6101c0526102006101e0819052606287019060005b60208110156101be57823560e01c825260049092019160209091019060010161019a565b505050806101200151156101dc576101d4610612565b91505061060a565b6101408101805160010167ffffffffffffffff169052606081015160009061020490826106ba565b9050603f601a82901c16600281148061022357508063ffffffff166003145b15610270576102668163ffffffff1660021461024057601f610243565b60005b60ff166002610259856303ffffff16601a610776565b63ffffffff16901b6107e9565b935050505061060a565b6101608301516000908190601f601086901c81169190601587901c166020811061029c5761029c611c16565b602002015192508063ffffffff851615806102bd57508463ffffffff16601c145b156102f4578661016001518263ffffffff16602081106102df576102df611c16565b6020020151925050601f600b86901c166103b0565b60208563ffffffff161015610356578463ffffffff16600c148061031e57508463ffffffff16600d145b8061032f57508463ffffffff16600e145b15610340578561ffff1692506103b0565b61034f8661ffff166010610776565b92506103b0565b60288563ffffffff1610158061037257508463ffffffff166022145b8061038357508463ffffffff166026145b156103b0578661016001518263ffffffff16602081106103a5576103a5611c16565b602002015192508190505b60048563ffffffff16101580156103cd575060088563ffffffff16105b806103de57508463ffffffff166001145b156103fd576103ef858784876108e3565b97505050505050505061060a565b63ffffffff60006020878316106104625761041d8861ffff166010610776565b9095019463fffffffc86166104338160016106ba565b915060288863ffffffff161015801561045357508763ffffffff16603014155b1561046057809250600093505b505b600061047089888885610af3565b63ffffffff9081169150603f8a16908916158015610495575060088163ffffffff1610155b80156104a75750601c8163ffffffff16105b15610583578063ffffffff16600814806104c757508063ffffffff166009145b156104fe576104ec8163ffffffff166008146104e357856104e6565b60005b896107e9565b9b50505050505050505050505061060a565b8063ffffffff16600a0361051e576104ec858963ffffffff8a1615611196565b8063ffffffff16600b0361053f576104ec858963ffffffff8a161515611196565b8063ffffffff16600c03610555576104ec61127c565b60108163ffffffff16101580156105725750601c8163ffffffff16105b15610583576104ec81898988611790565b8863ffffffff16603814801561059e575063ffffffff861615155b156105d35760018b61016001518763ffffffff16602081106105c2576105c2611c16565b63ffffffff90921660209290920201525b8363ffffffff1663ffffffff146105f0576105f08460018461198a565b6105fc85836001611196565b9b5050505050505050505050505b949350505050565b60408051608051815260a051602082015260dc519181019190915260fc51604482015261011c51604882015261013c51604c82015261015c51605082015261017c51605482015261019f5160588201526101bf5160598201526101d851605a8201526000906102009060628101835b60208110156106a557601c8401518252602090930192600490910190600101610681565b506000815281810382a0819003902092915050565b6000806106c683611a2e565b905060038416156106d657600080fd5b6020810190358460051c8160005b601b81101561073c5760208501943583821c600116801561070c576001811461072157610732565b60008481526020839052604090209350610732565b600082815260208590526040902093505b50506001016106e4565b50608051915081811461075757630badf00d60005260206000fd5b5050601f94909416601c0360031b9390931c63ffffffff169392505050565b600063ffffffff8381167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80850183169190911c821615159160016020869003821681901b830191861691821b92911b01826107d35760006107d5565b815b90861663ffffffff16179250505092915050565b60006107f3611ad7565b60809050806060015160040163ffffffff16816080015163ffffffff161461087c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601260248201527f6a756d7020696e2064656c617920736c6f74000000000000000000000000000060448201526064015b60405180910390fd5b60608101805160808301805163ffffffff9081169093528583169052908516156108d257806008018261016001518663ffffffff16602081106108c1576108c1611c16565b63ffffffff90921660209290920201525b6108da610612565b95945050505050565b60006108ed611ad7565b608090506000816060015160040163ffffffff16826080015163ffffffff1614610973576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f6272616e636820696e2064656c617920736c6f740000000000000000000000006044820152606401610873565b8663ffffffff166004148061098e57508663ffffffff166005145b15610a0a5760008261016001518663ffffffff16602081106109b2576109b2611c16565b602002015190508063ffffffff168563ffffffff161480156109da57508763ffffffff166004145b80610a0257508063ffffffff168563ffffffff1614158015610a0257508763ffffffff166005145b915050610a87565b8663ffffffff16600603610a275760008460030b13159050610a87565b8663ffffffff16600703610a435760008460030b139050610a87565b8663ffffffff16600103610a8757601f601087901c166000819003610a6c5760008560030b1291505b8063ffffffff16600103610a855760008560030b121591505b505b606082018051608084015163ffffffff169091528115610acd576002610ab28861ffff166010610776565b63ffffffff90811690911b8201600401166080840152610adf565b60808301805160040163ffffffff1690525b610ae7610612565b98975050505050505050565b6000603f601a86901c81169086166020821015610eb75760088263ffffffff1610158015610b275750600f8263ffffffff16105b15610bc7578163ffffffff16600803610b4257506020610bc2565b8163ffffffff16600903610b5857506021610bc2565b8163ffffffff16600a03610b6e5750602a610bc2565b8163ffffffff16600b03610b845750602b610bc2565b8163ffffffff16600c03610b9a57506024610bc2565b8163ffffffff16600d03610bb057506025610bc2565b8163ffffffff16600e03610bc2575060265b600091505b8163ffffffff16600003610e0b57601f600688901c16602063ffffffff83161015610ce55760088263ffffffff1610610c055786935050505061060a565b8163ffffffff16600003610c285763ffffffff86811691161b925061060a915050565b8163ffffffff16600203610c4b5763ffffffff86811691161c925061060a915050565b8163ffffffff16600303610c75576102668163ffffffff168763ffffffff16901c82602003610776565b8163ffffffff16600403610c98575050505063ffffffff8216601f84161b61060a565b8163ffffffff16600603610cbb575050505063ffffffff8216601f84161c61060a565b8163ffffffff16600703610ce5576102668763ffffffff168763ffffffff16901c88602003610776565b8163ffffffff1660201480610d0057508163ffffffff166021145b15610d1257858701935050505061060a565b8163ffffffff1660221480610d2d57508163ffffffff166023145b15610d3f57858703935050505061060a565b8163ffffffff16602403610d5a57858716935050505061060a565b8163ffffffff16602503610d7557858717935050505061060a565b8163ffffffff16602603610d9057858718935050505061060a565b8163ffffffff16602703610dab57505050508282171961060a565b8163ffffffff16602a03610ddd578560030b8760030b12610dcd576000610dd0565b60015b60ff16935050505061060a565b8163ffffffff16602b03610e05578563ffffffff168763ffffffff1610610dcd576000610dd0565b50611134565b8163ffffffff16600f03610e2d5760108563ffffffff16901b9250505061060a565b8163ffffffff16601c03610eb2578063ffffffff16600203610e545750505082820261060a565b8063ffffffff1660201480610e6f57508063ffffffff166021145b15610eb2578063ffffffff16602003610e86579419945b60005b6380000000871615610ea8576401fffffffe600197881b169601610e89565b925061060a915050565b611134565b60288263ffffffff16101561101a578163ffffffff16602003610f0357610efa8660031660080260180363ffffffff168563ffffffff16901c60ff166008610776565b9250505061060a565b8163ffffffff16602103610f3857610efa8660021660080260100363ffffffff168563ffffffff16901c61ffff166010610776565b8163ffffffff16602203610f685750505063ffffffff60086003851602811681811b198416918316901b1761060a565b8163ffffffff16602303610f8057839250505061060a565b8163ffffffff16602403610fb3578560031660080260180363ffffffff168463ffffffff16901c60ff169250505061060a565b8163ffffffff16602503610fe7578560021660080260100363ffffffff168463ffffffff16901c61ffff169250505061060a565b8163ffffffff16602603610eb25750505063ffffffff60086003851602601803811681811c198416918316901c1761060a565b8163ffffffff166028036110515750505060ff63ffffffff60086003861602601803811682811b9091188316918416901b1761060a565b8163ffffffff166029036110895750505061ffff63ffffffff60086002861602601003811682811b9091188316918416901b1761060a565b8163ffffffff16602a036110b95750505063ffffffff60086003851602811681811c198316918416901c1761060a565b8163ffffffff16602b036110d157849250505061060a565b8163ffffffff16602e036111045750505063ffffffff60086003851602601803811681811b198316918416901b1761060a565b8163ffffffff1660300361111c57839250505061060a565b8163ffffffff1660380361113457849250505061060a565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601360248201527f696e76616c696420696e737472756374696f6e000000000000000000000000006044820152606401610873565b60006111a0611ad7565b506080602063ffffffff861610611213576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f76616c69642072656769737465720000000000000000000000000000000000006044820152606401610873565b63ffffffff8516158015906112255750825b1561125957838161016001518663ffffffff166020811061124857611248611c16565b63ffffffff90921660209290920201525b60808101805163ffffffff808216606085015260049091011690526108da610612565b6000611286611ad7565b506101e051604081015160808083015160a084015160c09094015191936000928392919063ffffffff8616610ffa036113005781610fff8116156112cf57610fff811661100003015b8363ffffffff166000036112f65760e08801805163ffffffff8382011690915295506112fa565b8395505b5061174f565b8563ffffffff16610fcd0361131b576340000000945061174f565b8563ffffffff1661101803611333576001945061174f565b8563ffffffff166110960361136857600161012088015260ff831661010088015261135c610612565b97505050505050505090565b8563ffffffff16610fa3036115b25763ffffffff83161561174f577ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb63ffffffff84160161156c5760006113c38363fffffffc1660016106ba565b60208901519091508060001a6001036114305761142d81600090815233602052604090207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790565b90505b6000805460408b81015190517fe03110e10000000000000000000000000000000000000000000000000000000081526004810185905263ffffffff9091166024820152829173ffffffffffffffffffffffffffffffffffffffff169063e03110e1906044016040805180830381865afa1580156114b1573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906114d59190611c45565b915091506003861680600403828110156114ed578092505b50818610156114fa578591505b8260088302610100031c9250826008828460040303021b9250600180600883600403021b036001806008858560040303021b039150811981169050838119871617955050506115518663fffffffc1660018661198a565b60408b018051820163ffffffff16905297506115ad92505050565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd63ffffffff8416016115a15780945061174f565b63ffffffff9450600993505b61174f565b8563ffffffff16610fa4036116a35763ffffffff8316600114806115dc575063ffffffff83166002145b806115ed575063ffffffff83166004145b156115fa5780945061174f565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa63ffffffff8416016115a157600061163a8363fffffffc1660016106ba565b60208901519091506003841660040383811015611655578093505b83900360089081029290921c7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600193850293841b0116911b1760208801526000604088015293508361174f565b8563ffffffff16610fd70361174f578163ffffffff166003036117435763ffffffff831615806116d9575063ffffffff83166005145b806116ea575063ffffffff83166003145b156116f8576000945061174f565b63ffffffff831660011480611713575063ffffffff83166002145b80611724575063ffffffff83166006145b80611735575063ffffffff83166004145b156115a1576001945061174f565b63ffffffff9450601693505b6101608701805163ffffffff808816604090920191909152905185821660e09091015260808801805180831660608b0152600401909116905261135c610612565b600061179a611ad7565b506080600063ffffffff87166010036117b8575060c0810151611921565b8663ffffffff166011036117d75763ffffffff861660c0830152611921565b8663ffffffff166012036117f0575060a0810151611921565b8663ffffffff1660130361180f5763ffffffff861660a0830152611921565b8663ffffffff166018036118435763ffffffff600387810b9087900b02602081901c821660c08501521660a0830152611921565b8663ffffffff166019036118745763ffffffff86811681871602602081901c821660c08501521660a0830152611921565b8663ffffffff16601a036118ca578460030b8660030b8161189757611897611c69565b0763ffffffff1660c0830152600385810b9087900b816118b9576118b9611c69565b0563ffffffff1660a0830152611921565b8663ffffffff16601b03611921578463ffffffff168663ffffffff16816118f3576118f3611c69565b0663ffffffff90811660c08401528581169087168161191457611914611c69565b0463ffffffff1660a08301525b63ffffffff84161561195c57808261016001518563ffffffff166020811061194b5761194b611c16565b63ffffffff90921660209290920201525b60808201805163ffffffff8082166060860152600490910116905261197f610612565b979650505050505050565b600061199583611a2e565b905060038416156119a557600080fd5b6020810190601f8516601c0360031b83811b913563ffffffff90911b1916178460051c60005b601b811015611a235760208401933582821c60011680156119f35760018114611a0857611a19565b60008581526020839052604090209450611a19565b600082815260208690526040902094505b50506001016119cb565b505060805250505050565b60ff81166103800261016681019036906104e601811015611ad1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f636865636b207468617420746865726520697320656e6f7567682063616c6c6460448201527f61746100000000000000000000000000000000000000000000000000000000006064820152608401610873565b50919050565b6040805161018081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101919091526101608101611b3d611b42565b905290565b6040518061040001604052806020906020820280368337509192915050565b60008083601f840112611b7357600080fd5b50813567ffffffffffffffff811115611b8b57600080fd5b602083019150836020828501011115611ba357600080fd5b9250929050565b60008060008060408587031215611bc057600080fd5b843567ffffffffffffffff80821115611bd857600080fd5b611be488838901611b61565b90965094506020870135915080821115611bfd57600080fd5b50611c0a87828801611b61565b95989497509550505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60008060408385031215611c5857600080fd5b505080516020909101519092909150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fdfea164736f6c634300080f000a"

var MIPSDeployedSourceMap = "1131:37174:105:-:0;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;1711:45;;1746:10;1711:45;;;;;188:10:253;176:23;;;158:42;;146:2;131:18;1711:45:105;;;;;;;;2137:29;;;;;;;;;;;;412:42:253;400:55;;;382:74;;370:2;355:18;2137:29:105;211:251:253;24692:6295:105;;;;;;:::i;:::-;;:::i;:::-;;;1687:25:253;;;1675:2;1660:18;24692:6295:105;1541:177:253;24692:6295:105;24770:7;24813:18;;:::i;:::-;24960:4;24953:5;24950:15;24940:113;;25033:1;25031;25024:11;24940:113;25089:4;25083:11;25096;25080:28;25070:116;;25166:1;25164;25157:11;25070:116;25234:3;25216:16;25213:25;25203:129;;25312:1;25310;25303:11;25203:129;25376:3;25362:12;25359:21;25349:124;;25453:1;25451;25444:11;25349:124;25733:24;;26078:4;25779:20;26149:2;25837:21;;25733:24;25895:18;25779:20;25837:21;;;25733:24;25710:21;25706:52;;;25895:18;25779:20;;;25837:21;;;25733:24;25706:52;;25779:20;;25837:21;;;25733:24;25706:52;;25895:18;25779:20;25837:21;;;25733:24;25706:52;;25895:18;25779:20;25837:21;;;25733:24;25706:52;;25895:18;25779:20;25837:21;;;25733:24;25706:52;;;25895:18;25779:20;25837:21;;;25733:24;25710:21;25706:52;;;25895:18;25779:20;25837:21;;;25733:24;25706:52;;25895:18;25779:20;25837:21;;;25733:24;25706:52;;25895:18;25779:20;26776:10;25895:18;26766:21;;;25837;;;;26879:1;26864:113;26889:2;26886:1;26883:9;26864:113;;;25733:24;;25710:21;25706:52;25779:20;;26957:1;25837:21;;;;25721:2;25895:18;;;;26907:1;26900:9;26864:113;;;26868:14;;;27055:5;:12;;;27051:71;;;27094:13;:11;:13::i;:::-;27087:20;;;;;27051:71;27136:10;;;:15;;27150:1;27136:15;;;;;27221:8;;;;-1:-1:-1;;27213:20:105;;-1:-1:-1;27213:7:105;:20::i;:::-;27199:34;-1:-1:-1;27263:10:105;27271:2;27263:10;;;;27340:1;27330:11;;;:26;;;27345:6;:11;;27355:1;27345:11;27330:26;27326:348;;;27595:64;27606:6;:11;;27616:1;27606:11;:20;;27624:2;27606:20;;;27620:1;27606:20;27595:64;;27657:1;27628:25;27631:4;27638:10;27631:17;27650:2;27628;:25::i;:::-;:30;;;;27595:10;:64::i;:::-;27588:71;;;;;;;27326:348;27923:15;;;;27718:9;;;;27855:4;27849:2;27841:10;;;27840:19;;;27923:15;27948:2;27940:10;;;27939:19;27923:36;;;;;;;:::i;:::-;;;;;;-1:-1:-1;27988:5:105;28012:11;;;;;:29;;;28027:6;:14;;28037:4;28027:14;28012:29;28008:832;;;28104:5;:15;;;28120:5;28104:22;;;;;;;;;:::i;:::-;;;;;;-1:-1:-1;;28167:4:105;28161:2;28153:10;;;28152:19;28008:832;;;28205:4;28196:6;:13;;;28192:648;;;28326:6;:13;;28336:3;28326:13;:30;;;;28343:6;:13;;28353:3;28343:13;28326:30;:47;;;;28360:6;:13;;28370:3;28360:13;28326:47;28322:253;;;28436:4;28443:6;28436:13;28431:18;;28192:648;;28322:253;28535:21;28538:4;28545:6;28538:13;28553:2;28535;:21::i;:::-;28530:26;;28192:648;;;28609:4;28599:6;:14;;;;:32;;;;28617:6;:14;;28627:4;28617:14;28599:32;:50;;;;28635:6;:14;;28645:4;28635:14;28599:50;28595:245;;;28719:5;:15;;;28735:5;28719:22;;;;;;;;;:::i;:::-;;;;;28714:27;;28820:5;28812:13;;28595:245;28869:1;28859:6;:11;;;;:25;;;;;28883:1;28874:6;:10;;;28859:25;28858:42;;;;28889:6;:11;;28899:1;28889:11;28858:42;28854:125;;;28927:37;28940:6;28948:4;28954:5;28961:2;28927:12;:37::i;:::-;28920:44;;;;;;;;;;;28854:125;29012:13;28993:16;29164:4;29154:14;;;;29150:444;;29233:19;29236:4;29241:6;29236:11;29249:2;29233;:19::i;:::-;29227:25;;;;29289:10;29284:15;;29323:16;29284:15;29337:1;29323:7;:16::i;:::-;29317:22;;29371:4;29361:6;:14;;;;:32;;;;;29379:6;:14;;29389:4;29379:14;;29361:32;29357:223;;;29458:4;29446:16;;29560:1;29552:9;;29357:223;29170:424;29150:444;29627:10;29640:26;29648:4;29654:2;29658;29662:3;29640:7;:26::i;:::-;29669:10;29640:39;;;;-1:-1:-1;29765:4:105;29758:11;;;29797;;;:24;;;;;29820:1;29812:4;:9;;;;29797:24;:39;;;;;29832:4;29825;:11;;;29797:39;29793:787;;;29860:4;:9;;29868:1;29860:9;:22;;;;29873:4;:9;;29881:1;29873:9;29860:22;29856:124;;;29924:37;29935:4;:9;;29943:1;29935:9;:21;;29951:5;29935:21;;;29947:1;29935:21;29958:2;29924:10;:37::i;:::-;29917:44;;;;;;;;;;;;;;;29856:124;30002:4;:11;;30010:3;30002:11;29998:101;;30052:28;30061:5;30068:2;30072:7;;;;30052:8;:28::i;29998:101::-;30120:4;:11;;30128:3;30120:11;30116:101;;30170:28;30179:5;30186:2;30190:7;;;;;30170:8;:28::i;30116:101::-;30287:4;:11;;30295:3;30287:11;30283:80;;30329:15;:13;:15::i;30283:80::-;30466:4;30458;:12;;;;:27;;;;;30481:4;30474;:11;;;30458:27;30454:112;;;30516:31;30527:4;30533:2;30537;30541:5;30516:10;:31::i;30454:112::-;30640:6;:14;;30650:4;30640:14;:28;;;;-1:-1:-1;30658:10:105;;;;;30640:28;30636:93;;;30713:1;30688:5;:15;;;30704:5;30688:22;;;;;;;;;:::i;:::-;:26;;;;:22;;;;;;:26;30636:93;30775:9;:26;;30788:13;30775:26;30771:92;;30821:27;30830:9;30841:1;30844:3;30821:8;:27::i;:::-;30944:26;30953:5;30960:3;30965:4;30944:8;:26::i;:::-;30937:33;;;;;;;;;;;;;24692:6295;;;;;;;:::o;2707:1770::-;3254:4;3248:11;;3170:4;2973:31;2962:43;;3033:13;2973:31;3372:2;3072:13;;2962:43;2979:24;2973:31;3072:13;;;2962:43;;;;2979:24;2973:31;3072:13;;;2962:43;2979:24;2973:31;3072:13;;;2962:43;2979:24;2973:31;3072:13;;;2962:43;2979:24;2973:31;3072:13;;;2962:43;2979:24;2973:31;3072:13;;;2962:43;2979:24;2973:31;3072:13;;;2962:43;2979:24;2973:31;3072:13;;;2962:43;2979:24;2973:31;3072:13;;;2962:43;2748:12;;3977:13;;3072;;;2748:12;4070:112;4095:2;4092:1;4089:9;4070:112;;;2989:13;2979:24;;2973:31;2962:43;;2993:2;3033:13;;;;4166:1;3072:13;;;;4113:1;4106:9;4070:112;;;4074:14;4245:1;4241:2;4234:13;4340:5;4336:2;4332:14;4325:5;4320:27;4446:14;;;4429:32;;;2707:1770;-1:-1:-1;;2707:1770:105:o;20539:1935::-;20612:11;20723:14;20740:24;20752:11;20740;:24::i;:::-;20723:41;;20872:1;20865:5;20861:13;20858:69;;;20907:1;20904;20897:12;20858:69;21056:2;21044:15;;;20997:20;21486:5;21483:1;21479:13;21521:4;21557:1;21542:411;21567:2;21564:1;21561:9;21542:411;;;21690:2;21678:15;;;21627:20;21725:12;;;21739:1;21721:20;21762:86;;;;21854:1;21849:86;;;;21714:221;;21762:86;21220:1;21213:12;;;21253:2;21246:13;;;21298:2;21285:16;;21795:31;;21762:86;;21849;21220:1;21213:12;;;21253:2;21246:13;;;21298:2;21285:16;;21882:31;;21714:221;-1:-1:-1;;21585:1:105;21578:9;21542:411;;;21546:14;22063:4;22057:11;22042:26;;22149:7;22143:4;22140:17;22130:124;;22191:10;22188:1;22181:21;22233:2;22230:1;22223:13;22130:124;-1:-1:-1;;22381:2:105;22370:14;;;;22358:10;22354:31;22351:1;22347:39;22415:16;;;;22433:10;22411:33;;20539:1935;-1:-1:-1;;;20539:1935:105:o;2265:334::-;2326:6;2385:18;;;;2394:8;;;;2385:18;;;;;;2384:25;;;;;2401:1;2448:2;:9;;;2442:16;;;;;2441:22;;2440:32;;;;;;;2502:9;;2501:15;2384:25;2559:21;;2579:1;2559:21;;;2570:6;2559:21;2544:11;;;;;:37;;-1:-1:-1;;;2265:334:105;;;;:::o;17679:821::-;17748:12;17835:18;;:::i;:::-;17903:4;17894:13;;17955:5;:8;;;17964:1;17955:10;17939:26;;:5;:12;;;:26;;;17935:93;;17985:28;;;;;2114:2:253;17985:28:105;;;2096:21:253;2153:2;2133:18;;;2126:30;2192:20;2172:18;;;2165:48;2230:18;;17985:28:105;;;;;;;;17935:93;18117:8;;;;;18150:12;;;;;18139:23;;;;;;;18176:20;;;;;18117:8;18308:13;;;18304:90;;18369:6;18378:1;18369:10;18341:5;:15;;;18357:8;18341:25;;;;;;;;;:::i;:::-;:38;;;;:25;;;;;;:38;18304:90;18470:13;:11;:13::i;:::-;18463:20;17679:821;-1:-1:-1;;;;;17679:821:105:o;12542:2024::-;12639:12;12725:18;;:::i;:::-;12793:4;12784:13;;12825:17;12885:5;:8;;;12894:1;12885:10;12869:26;;:5;:12;;;:26;;;12865:95;;12915:30;;;;;2461:2:253;12915:30:105;;;2443:21:253;2500:2;2480:18;;;2473:30;2539:22;2519:18;;;2512:50;2579:18;;12915:30:105;2259:344:253;12865:95:105;13030:7;:12;;13041:1;13030:12;:28;;;;13046:7;:12;;13057:1;13046:12;13030:28;13026:947;;;13078:9;13090:5;:15;;;13106:6;13090:23;;;;;;;;;:::i;:::-;;;;;13078:35;;13154:2;13147:9;;:3;:9;;;:25;;;;;13160:7;:12;;13171:1;13160:12;13147:25;13146:58;;;;13185:2;13178:9;;:3;:9;;;;:25;;;;;13191:7;:12;;13202:1;13191:12;13178:25;13131:73;;13060:159;13026:947;;;13316:7;:12;;13327:1;13316:12;13312:661;;13377:1;13369:3;13363:15;;;;13348:30;;13312:661;;;13481:7;:12;;13492:1;13481:12;13477:496;;13541:1;13534:3;13528:14;;;13513:29;;13477:496;;;13662:7;:12;;13673:1;13662:12;13658:315;;13750:4;13744:2;13735:11;;;13734:20;13720:10;13777:8;;;13773:84;;13837:1;13830:3;13824:14;;;13809:29;;13773:84;13878:3;:8;;13885:1;13878:8;13874:85;;13939:1;13931:3;13925:15;;;;13910:30;;13874:85;13676:297;13658:315;14049:8;;;;;14127:12;;;;14116:23;;;;;14283:178;;;;14374:1;14348:22;14351:5;14359:6;14351:14;14367:2;14348;:22::i;:::-;:27;;;;;;;14334:42;;14343:1;14334:42;14319:57;:12;;;:57;14283:178;;;14430:12;;;;;14445:1;14430:16;14415:31;;;;14283:178;14536:13;:11;:13::i;:::-;14529:20;12542:2024;-1:-1:-1;;;;;;;;12542:2024:105:o;31033:7270::-;31120:6;31178:10;31186:2;31178:10;;;;;;31229:11;;31341:4;31332:13;;31328:6915;;;31472:1;31462:6;:11;;;;:27;;;;;31486:3;31477:6;:12;;;31462:27;31458:568;;;31517:6;:11;;31527:1;31517:11;31513:455;;-1:-1:-1;31539:4:105;31513:455;;;31591:6;:11;;31601:1;31591:11;31587:381;;-1:-1:-1;31613:4:105;31587:381;;;31661:6;:13;;31671:3;31661:13;31657:311;;-1:-1:-1;31685:4:105;31657:311;;;31730:6;:13;;31740:3;31730:13;31726:242;;-1:-1:-1;31754:4:105;31726:242;;;31800:6;:13;;31810:3;31800:13;31796:172;;-1:-1:-1;31824:4:105;31796:172;;;31869:6;:13;;31879:3;31869:13;31865:103;;-1:-1:-1;31893:4:105;31865:103;;;31937:6;:13;;31947:3;31937:13;31933:35;;-1:-1:-1;31961:4:105;31933:35;32006:1;31997:10;;31458:568;32087:6;:11;;32097:1;32087:11;32083:3550;;32151:4;32146:1;32138:9;;;32137:18;32188:4;32138:9;32181:11;;;32177:1319;;;32280:4;32272;:12;;;32268:1206;;32323:2;32316:9;;;;;;;32268:1206;32437:4;:12;;32445:4;32437:12;32433:1041;;32488:11;;;;;;;;-1:-1:-1;32481:18:105;;-1:-1:-1;;32481:18:105;32433:1041;32612:4;:12;;32620:4;32612:12;32608:866;;32663:11;;;;;;;;-1:-1:-1;32656:18:105;;-1:-1:-1;;32656:18:105;32608:866;32790:4;:12;;32798:4;32790:12;32786:688;;32841:27;32850:5;32844:11;;:2;:11;;;;32862:5;32857:2;:10;32841:2;:27::i;32786:688::-;32990:4;:12;;32998:4;32990:12;32986:488;;-1:-1:-1;;;;33041:17:105;;;33053:4;33048:9;;33041:17;33034:24;;32986:488;33181:4;:12;;33189:4;33181:12;33177:297;;-1:-1:-1;;;;33232:17:105;;;33244:4;33239:9;;33232:17;33225:24;;33177:297;33375:4;:12;;33383:4;33375:12;33371:103;;33426:21;33435:2;33429:8;;:2;:8;;;;33444:2;33439;:7;33426:2;:21::i;33371:103::-;33656:4;:12;;33664:4;33656:12;:28;;;;33672:4;:12;;33680:4;33672:12;33656:28;33652:1149;;;33724:2;33719;:7;33712:14;;;;;;;33652:1149;33814:4;:12;;33822:4;33814:12;:28;;;;33830:4;:12;;33838:4;33830:12;33814:28;33810:991;;;33882:2;33877;:7;33870:14;;;;;;;33810:991;33964:4;:12;;33972:4;33964:12;33960:841;;34016:2;34011;:7;34004:14;;;;;;;33960:841;34097:4;:12;;34105:4;34097:12;34093:708;;34150:2;34145;:7;34137:16;;;;;;;34093:708;34233:4;:12;;34241:4;34233:12;34229:572;;34286:2;34281;:7;34273:16;;;;;;;34229:572;34369:4;:12;;34377:4;34369:12;34365:436;;-1:-1:-1;;;;34418:7:105;;;34416:10;34409:17;;34365:436;34529:4;:12;;34537:4;34529:12;34525:276;;34594:2;34576:21;;34582:2;34576:21;;;:29;;34604:1;34576:29;;;34600:1;34576:29;34569:36;;;;;;;;;34525:276;34718:4;:12;;34726:4;34718:12;34714:87;;34768:2;34765:5;;:2;:5;;;:13;;34777:1;34765:13;;34714:87;32100:2719;31328:6915;;32083:3550;34890:6;:13;;34900:3;34890:13;34886:747;;34940:2;34934;:8;;;;34927:15;;;;;;34886:747;35015:6;:14;;35025:4;35015:14;35011:622;;35084:4;:9;;35092:1;35084:9;35080:100;;-1:-1:-1;;;35135:21:105;;;35121:36;;35080:100;35232:4;:12;;35240:4;35232:12;:28;;;;35248:4;:12;;35256:4;35248:12;35232:28;35228:387;;;35292:4;:12;;35300:4;35292:12;35288:83;;35341:3;;;35288:83;35396:8;35434:125;35444:10;35441:13;;:18;35434:125;;35524:8;35491:3;35524:8;;;;;35491:3;35434:125;;;35591:1;-1:-1:-1;35584:8:105;;-1:-1:-1;;35584:8:105;35228:387;31328:6915;;;35678:4;35669:6;:13;;;35665:2578;;;35728:6;:14;;35738:4;35728:14;35724:1208;;35773:42;35791:2;35796:1;35791:6;35801:1;35790:12;35785:2;:17;35777:26;;:3;:26;;;;35807:4;35776:35;35813:1;35773:2;:42::i;:::-;35766:49;;;;;;35724:1208;35882:6;:14;;35892:4;35882:14;35878:1054;;35927:45;35945:2;35950:1;35945:6;35955:1;35944:12;35939:2;:17;35931:26;;:3;:26;;;;35961:6;35930:37;35969:2;35927;:45::i;35878:1054::-;36040:6;:14;;36050:4;36040:14;36036:896;;-1:-1:-1;;;36091:21:105;36110:1;36105;36100:6;;36099:12;36091:21;;36148:36;;;36219:5;36214:10;;36091:21;;;;;36213:18;36206:25;;36036:896;36298:6;:14;;36308:4;36298:14;36294:638;;36343:3;36336:10;;;;;;36294:638;36414:6;:14;;36424:4;36414:14;36410:522;;36474:2;36479:1;36474:6;36484:1;36473:12;36468:2;:17;36460:26;;:3;:26;;;;36490:4;36459:35;36452:42;;;;;;36410:522;36562:6;:14;;36572:4;36562:14;36558:374;;36622:2;36627:1;36622:6;36632:1;36621:12;36616:2;:17;36608:26;;:3;:26;;;;36638:6;36607:37;36600:44;;;;;;36558:374;36712:6;:14;;36722:4;36712:14;36708:224;;-1:-1:-1;;;36763:26:105;36787:1;36782;36777:6;;36776:12;36771:2;:17;36763:26;;36825:41;;;36901:5;36896:10;;36763:26;;;;;36895:18;36888:25;;35665:2578;36986:6;:14;;36996:4;36986:14;36982:1261;;-1:-1:-1;;;37039:4:105;37033:34;37065:1;37060;37055:6;;37054:12;37049:2;:17;37033:34;;37119:27;;;37099:48;;;37173:10;;37034:9;;;37033:34;;37172:18;37165:25;;36982:1261;37245:6;:14;;37255:4;37245:14;37241:1002;;-1:-1:-1;;;37298:6:105;37292:36;37326:1;37321;37316:6;;37315:12;37310:2;:17;37292:36;;37380:29;;;37360:50;;;37436:10;;37293:11;;;37292:36;;37435:18;37428:25;;37241:1002;37509:6;:14;;37519:4;37509:14;37505:738;;-1:-1:-1;;;37556:20:105;37574:1;37569;37564:6;;37563:12;37556:20;;37608:36;;;37676:5;37670:11;;37556:20;;;;;37669:19;37662:26;;37505:738;37743:6;:14;;37753:4;37743:14;37739:504;;37784:2;37777:9;;;;;;37739:504;37842:6;:14;;37852:4;37842:14;37838:405;;-1:-1:-1;;;37889:25:105;37912:1;37907;37902:6;;37901:12;37896:2;:17;37889:25;;37946:41;;;38019:5;38013:11;;37889:25;;;;;38012:19;38005:26;;37838:405;38086:6;:14;;38096:4;38086:14;38082:161;;38127:3;38120:10;;;;;;38082:161;38185:6;:14;;38195:4;38185:14;38181:62;;38226:2;38219:9;;;;;;38181:62;38257:29;;;;;2810:2:253;38257:29:105;;;2792:21:253;2849:2;2829:18;;;2822:30;2888:21;2868:18;;;2861:49;2927:18;;38257:29:105;2608:343:253;18781:782:105;18867:12;18954:18;;:::i;:::-;-1:-1:-1;19022:4:105;19129:2;19117:14;;;;19109:41;;;;;;;3158:2:253;19109:41:105;;;3140:21:253;3197:2;3177:18;;;3170:30;3236:16;3216:18;;;3209:44;3270:18;;19109:41:105;2956:338:253;19109:41:105;19246:14;;;;;;;:30;;;19264:12;19246:30;19242:102;;;19325:4;19296:5;:15;;;19312:9;19296:26;;;;;;;;;:::i;:::-;:33;;;;:26;;;;;;:33;19242:102;19399:12;;;;;19388:23;;;;:8;;;:23;19455:1;19440:16;;;19425:31;;;19533:13;:11;:13::i;4518:7638::-;4561:12;4647:18;;:::i;:::-;-1:-1:-1;4825:15:105;;:18;;;;4715:4;4985:18;;;;5029;;;;5073;;;;;4715:4;;4805:17;;;;4985:18;5029;5163;;;5177:4;5163:18;5159:6687;;5213:2;5240:4;5237:7;;:12;5233:120;;5329:4;5326:7;;5318:4;:16;5312:22;5233:120;5374:2;:7;;5380:1;5374:7;5370:161;;5410:10;;;;;5442:16;;;;;;;;5410:10;-1:-1:-1;5370:161:105;;;5510:2;5505:7;;5370:161;5183:362;5159:6687;;;5647:10;:18;;5661:4;5647:18;5643:6203;;1746:10;5685:14;;5643:6203;;;5783:10;:18;;5797:4;5783:18;5779:6067;;5826:1;5821:6;;5779:6067;;;5951:10;:18;;5965:4;5951:18;5947:5899;;6004:4;5989:12;;;:19;6026:26;;;:14;;;:26;6077:13;:11;:13::i;:::-;6070:20;;;;;;;;;4518:7638;:::o;5947:5899::-;6216:10;:18;;6230:4;6216:18;6212:5634;;6367:14;;;6363:2662;6212:5634;6363:2662;6537:22;;;;;6533:2492;;6662:10;6675:27;6683:2;6688:10;6683:15;6700:1;6675:7;:27::i;:::-;6786:17;;;;6662:40;;-1:-1:-1;6786:17:105;6764:19;6936:14;6955:1;6930:26;6926:131;;6998:36;7022:11;1277:21:106;1426:15;;;1467:8;1461:4;1454:22;1595:4;1582:18;;1602:19;1578:44;1624:11;1575:61;;1222:430;6998:36:105;6984:50;;6926:131;7079:11;7110:6;;7143:20;;;;;7110:54;;;;;;;;3472:25:253;;;3545:10;3533:23;;;3513:18;;;3506:51;7079:11:105;;7110:6;;;:19;;3445:18:253;;7110:54:105;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;:::i;:::-;7078:86;;;;7391:1;7387:2;7383:10;7488:9;7485:1;7481:17;7570:6;7563:5;7560:17;7557:40;;;7590:5;7580:15;;7557:40;;7673:6;7669:2;7666:14;7663:34;;;7693:2;7683:12;;7663:34;7799:3;7794:1;7786:6;7782:14;7777:3;7773:24;7769:34;7762:41;;7899:3;7895:1;7883:9;7874:6;7871:1;7867:14;7863:30;7859:38;7855:48;7848:55;;8023:1;8019;8015;8003:9;8000:1;7996:17;7992:25;7988:33;7984:41;8150:1;8146;8142;8133:6;8121:9;8118:1;8114:17;8110:30;8106:38;8102:46;8098:54;8080:72;;8250:10;8246:15;8240:4;8236:26;8228:34;;8366:3;8358:4;8354:9;8349:3;8345:19;8342:28;8335:35;;;;8512:33;8521:2;8526:10;8521:15;8538:1;8541:3;8512:8;:33::i;:::-;8567:20;;;:38;;;;;;;;;-1:-1:-1;6533:2492:105;;-1:-1:-1;;;6533:2492:105;;8724:18;;;;;8720:305;;8894:2;8889:7;;6212:5634;;8720:305;8964:10;8959:15;;2054:3;8996:10;;8720:305;6212:5634;;;9154:10;:18;;9168:4;9154:18;9150:2696;;9308:15;;;1825:1;9308:15;;:34;;-1:-1:-1;9327:15:105;;;1860:1;9327:15;9308:34;:57;;;-1:-1:-1;9346:19:105;;;1937:1;9346:19;9308:57;9304:1609;;;9394:2;9389:7;;9150:2696;;9304:1609;9520:23;;;;;9516:1397;;9567:10;9580:27;9588:2;9593:10;9588:15;9605:1;9580:7;:27::i;:::-;9683:17;;;;9567:40;;-1:-1:-1;9926:1:105;9918:10;;10020:1;10016:17;10095:13;;;10092:32;;;10117:5;10111:11;;10092:32;10403:14;;;10209:1;10399:22;;;10395:32;;;;10292:26;10316:1;10201:10;;;10296:18;;;10292:26;10391:43;10197:20;;10499:12;10627:17;;;:23;10695:1;10672:20;;;:24;10205:2;-1:-1:-1;10205:2:105;6212:5634;;9150:2696;11115:10;:18;;11129:4;11115:18;11111:735;;11209:2;:7;;11215:1;11209:7;11205:627;;11282:14;;;;;:40;;-1:-1:-1;11300:22:105;;;1979:1;11300:22;11282:40;:62;;;-1:-1:-1;11326:18:105;;;1898:1;11326:18;11282:62;11278:404;;;11377:1;11372:6;;11205:627;;11278:404;11423:15;;;1825:1;11423:15;;:34;;-1:-1:-1;11442:15:105;;;1860:1;11442:15;11423:34;:61;;;-1:-1:-1;11461:23:105;;;2022:1;11461:23;11423:61;:84;;;-1:-1:-1;11488:19:105;;;1937:1;11488:19;11423:84;11419:263;;;11540:1;11535:6;;6212:5634;;11205:627;11733:10;11728:15;;2088:4;11765:11;;11205:627;11921:15;;;;;:23;;;;:18;;;;:23;;;;11958:15;;:23;;;:18;;;;:23;-1:-1:-1;12047:12:105;;;;12036:23;;;:8;;;:23;12103:1;12088:16;12073:31;;;;;12126:13;:11;:13::i;14907:2480::-;15001:12;15087:18;;:::i;:::-;-1:-1:-1;15155:4:105;15187:10;15295:13;;;15304:4;15295:13;15291:1705;;-1:-1:-1;15334:8:105;;;;15291:1705;;;15453:5;:13;;15462:4;15453:13;15449:1547;;15486:14;;;:8;;;:14;15449:1547;;;15616:5;:13;;15625:4;15616:13;15612:1384;;-1:-1:-1;15655:8:105;;;;15612:1384;;;15774:5;:13;;15783:4;15774:13;15770:1226;;15807:14;;;:8;;;:14;15770:1226;;;15948:5;:13;;15957:4;15948:13;15944:1052;;16075:9;16021:17;16001;;;16021;;;;16001:37;16082:2;16075:9;;;;;16057:8;;;:28;16103:22;:8;;;:22;15944:1052;;;16262:5;:13;;16271:4;16262:13;16258:738;;16329:11;16315;;;16329;;;16315:25;16384:2;16377:9;;;;;16359:8;;;:28;16405:22;:8;;;:22;16258:738;;;16586:5;:13;;16595:4;16586:13;16582:414;;16656:3;16637:23;;16643:3;16637:23;;;;;;;:::i;:::-;;16619:42;;:8;;;:42;16697:23;;;;;;;;;;;;;:::i;:::-;;16679:42;;:8;;;:42;16582:414;;;16890:5;:13;;16899:4;16890:13;16886:110;;16940:3;16934:9;;:3;:9;;;;;;;:::i;:::-;;16923:20;;;;:8;;;:20;16972:9;;;;;;;;;;;:::i;:::-;;16961:20;;:8;;;:20;16886:110;17089:14;;;;17085:85;;17152:3;17123:5;:15;;;17139:9;17123:26;;;;;;;;;:::i;:::-;:32;;;;:26;;;;;;:32;17085:85;17224:12;;;;;17213:23;;;;:8;;;:23;17280:1;17265:16;;;17250:31;;;17357:13;:11;:13::i;:::-;17350:20;14907:2480;-1:-1:-1;;;;;;;14907:2480:105:o;22810:1758::-;22986:14;23003:24;23015:11;23003;:24::i;:::-;22986:41;;23135:1;23128:5;23124:13;23121:69;;;23170:1;23167;23160:12;23121:69;23325:2;23519:15;;;23344:2;23333:14;;23321:10;23317:31;23314:1;23310:39;23475:16;;;23260:20;;23460:10;23449:22;;;23445:27;23435:38;23432:60;23961:5;23958:1;23954:13;24032:1;24017:411;24042:2;24039:1;24036:9;24017:411;;;24165:2;24153:15;;;24102:20;24200:12;;;24214:1;24196:20;24237:86;;;;24329:1;24324:86;;;;24189:221;;24237:86;21220:1;21213:12;;;21253:2;21246:13;;;21298:2;21285:16;;24270:31;;24237:86;;24324;21220:1;21213:12;;;21253:2;21246:13;;;21298:2;21285:16;;24357:31;;24189:221;-1:-1:-1;;24060:1:105;24053:9;24017:411;;;-1:-1:-1;;24527:4:105;24520:18;-1:-1:-1;;;;22810:1758:105:o;19767:558::-;20089:20;;;20113:7;20089:32;20082:3;:40;;;20179:14;;20222:17;;20216:24;;;20208:72;;;;;;;4209:2:253;20208:72:105;;;4191:21:253;4248:2;4228:18;;;4221:30;4287:34;4267:18;;;4260:62;4358:5;4338:18;;;4331:33;4381:19;;20208:72:105;4007:399:253;20208:72:105;20294:14;19767:558;;;:::o;-1:-1:-1:-;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;:::i;:::-;;;;:::o;:::-;;;;;;;;;;;;;;;;;;;;;;;;:::o;467:347:253:-;518:8;528:6;582:3;575:4;567:6;563:17;559:27;549:55;;600:1;597;590:12;549:55;-1:-1:-1;623:20:253;;666:18;655:30;;652:50;;;698:1;695;688:12;652:50;735:4;727:6;723:17;711:29;;787:3;780:4;771:6;763;759:19;755:30;752:39;749:59;;;804:1;801;794:12;749:59;467:347;;;;;:::o;819:717::-;909:6;917;925;933;986:2;974:9;965:7;961:23;957:32;954:52;;;1002:1;999;992:12;954:52;1042:9;1029:23;1071:18;1112:2;1104:6;1101:14;1098:34;;;1128:1;1125;1118:12;1098:34;1167:58;1217:7;1208:6;1197:9;1193:22;1167:58;:::i;:::-;1244:8;;-1:-1:-1;1141:84:253;-1:-1:-1;1332:2:253;1317:18;;1304:32;;-1:-1:-1;1348:16:253;;;1345:36;;;1377:1;1374;1367:12;1345:36;;1416:60;1468:7;1457:8;1446:9;1442:24;1416:60;:::i;:::-;819:717;;;;-1:-1:-1;1495:8:253;-1:-1:-1;;;;819:717:253:o;1723:184::-;1775:77;1772:1;1765:88;1872:4;1869:1;1862:15;1896:4;1893:1;1886:15;3568:245;3647:6;3655;3708:2;3696:9;3687:7;3683:23;3679:32;3676:52;;;3724:1;3721;3714:12;3676:52;-1:-1:-1;;3747:16:253;;3803:2;3788:18;;;3782:25;3747:16;;3782:25;;-1:-1:-1;3568:245:253:o;3818:184::-;3870:77;3867:1;3860:88;3967:4;3964:1;3957:15;3991:4;3988:1;3981:15"

func init() {
	if err := json.Unmarshal([]byte(MIPSStorageLayoutJSON), MIPSStorageLayout); err != nil {
		panic(err)
	}

	layouts["MIPS"] = MIPSStorageLayout
	deployedBytecodes["MIPS"] = MIPSDeployedBin
}
