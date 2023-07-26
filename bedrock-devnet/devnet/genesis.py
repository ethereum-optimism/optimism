import time

# all derived from 'test test test test test test test test test test test junk'
DEV_ACCOUNTS = [
    'f39fd6e51aad88f6f4ce6ab8827279cfffb92266'
    '70997970c51812dc3a010c7d01b50e0d17dc79c8',
    '3c44cdddb6a900fa2b585dd299e03d12fa4293bc',
    '90F79bf6EB2c4f870365E785982E1f101E93b906',
    '15d34AAf54267DB7D7c367839AAf71A00a2C6A65',
    '9965507D1a55bcC2695C58ba16FB37d819B0A4dc',
    '976EA74026E726554dB657fA54763abd0C3a0aa9',
    '14dC79964da2C08b23698B3D3cc7Ca32193d9955',
    '23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f',
    'a0Ee7A142d267C1f36714E4a8F75612F20a79720'
]

GENESIS_TMPL = {
    'config': {
        'chainId': 900,
        "homesteadBlock": 0,
        "eip150Block": 0,
        "eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "eip155Block": 0,
        "eip158Block": 0,
        "byzantiumBlock": 0,
        "constantinopleBlock": 0,
        "petersburgBlock": 0,
        "istanbulBlock": 0,
        "muirGlacierBlock": 0,
        "berlinBlock": 0,
        "londonBlock": 0,
        "arrowGlacierBlock": 0,
        "grayGlacierBlock": 0,
        "shanghaiBlock": None,
        "cancunBlock": None,
        'clique': {
            'period': 3,
            'epoch': 30000
        }
    },
    'nonce': '0x0',
    'timestamp': '{:#x}'.format(int(time.time())),
    'extraData': '0x0000000000000000000000000000000000000000000000000000000000000000ca062b0fd91172d89bcd4bb084ac4e21972cc4670000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000',
    'gasLimit': '0xE4E1C0',
    'difficulty': '0x1',
    'mixHash': '0x0000000000000000000000000000000000000000000000000000000000000000',
    'coinbase': '0x0000000000000000000000000000000000000000',
    'alloc': {
        '{:x}'.format(i).ljust(40, '0'): {
            'balance': '0x1'
        } for i in range(0, 255)
    },
    'number': '0x0',
    'gasUsed': '0x0',
    'parentHash': '0x0000000000000000000000000000000000000000000000000000000000000000',
    'baseFeePergas': '0x3B9ACA00'
}

GENESIS_TMPL['alloc'].update({
    d: {
        'balance': '0x200000000000000000000000000000000000000000000000000000000000000'
    } for d in DEV_ACCOUNTS
})
