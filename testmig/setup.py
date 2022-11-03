from distutils.core import setup
from setuptools import find_packages

setup(
    name='testmig',
    version='1.0',
    description='Bedrock migration tester tool',
    install_requires=[
        'click==8.1.3',
        'docker==6.0.0',
        'web3==5.31.1',
        'requests==2.28.1',
        'tqdm==4.64.1'
    ],
    packages=find_packages(include=['testmig', 'testmig.*']),
    entry_points={
        'console_scripts': ['testmig=testmig.cli:group']
    }
)
