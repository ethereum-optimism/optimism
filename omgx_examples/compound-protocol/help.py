import os


files = [f for f in os.listdir('./contracts/')]
for f in files:
    if ".sol" in f:
        print(f'var {f[:-4]} = artifacts.require("{f[:-4]}");')

for f in files:
    if ".sol" in f:
        print(f'deployer.deploy({f[:-4]});')