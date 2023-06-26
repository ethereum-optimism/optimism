import sys
import dagviz
import networkx as nx
from eth_abi import decode

# The parent of the root claim is uint32 max.
ROOT_PARENT = 4294967295

# Get the abi-encoded input
b = sys.argv[1].removeprefix('0x')

# Decode the input
t = decode(['(uint32,bool,bytes32,uint128,uint128)[]'], bytes.fromhex(b))[0]

# Create the graph
G = nx.DiGraph()
for c in t:
    claim = c[2].hex()
    key = f"Position: {bin(c[3])[2:]} | Claim: 0x{claim[:4]}..{claim[60:64]}"
    G.add_node(key)
    if int(c[0]) != ROOT_PARENT:
        pclaim = t[c[0]][2].hex()
        G.add_edge(f"Position: {bin(t[c[0]][3])[2:]} | Claim: 0x{pclaim[:4]}..{pclaim[60:64]}", key)
r = dagviz.render_svg(G)

f = open('dispute_game.svg', 'w')
f.write(r)
f.close()
