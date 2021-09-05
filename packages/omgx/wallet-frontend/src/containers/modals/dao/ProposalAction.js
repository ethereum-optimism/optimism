/*
Copyright 2019-present OmiseGO Pte Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License. */
import React, { useState } from 'react';

import * as styles from './daoModal.module.scss';

import Input from  'components/input/Input';

import { utils } from 'ethers';

const ProposalAction = ({
    index,
    contracts,
    setContracts,
    actionList,
    setActionList,
    setCallData,
}) => {

    const [contract, setContract] = useState('');
    const [action, setAction] = useState('');
    const [options, setOptions] = useState('');

    const [params, setParams] = useState([]);
    const [paramNames, setParamNames] = useState([]);
    const [paramTypes, setParamTypes] = useState([]);
    const [interfaceIndex, setInterfaceIndex] = useState(0);

    // const {delegate, timelock} = networkService()
    const { delegate, timelock } = { delegate: { address: 'asd' }, timelock: { address: 'd' } }
    const interfaces = [delegate, timelock];

    const onContractChange = (e) => {

        let newContracts = contracts
        newContracts[index] = e.target.value
        setContract(e.target.value)
        setContracts(newContracts)

        // resetting action on change of contract;
        let newActions = actionList
        newActions[index] = ""
        setAction("")
        setActionList(newActions)

        let newOptions = ['asdf', 'test'];

        interfaces.forEach((item, index) => {
            let functions = item.interface.functions;
            setInterfaceIndex(index);
            for (let fragment in functions) {
                let fn = item.interface.getFunction(fragment);
                if (fn.stateMutability === "nonpayable") {
                    newOptions.push(fn.name);
                }
            }
        })

        setOptions(newOptions);
    };


    if (typeof delegate === "undefined" || typeof timelock === "undefined") {
        return "Loading...";
    }

    const onActionChange = (e) => {
        setAction(e.target.value);
        let newActionList = actionList;
        if (e.target.value !== "") {
            let fn = interfaces[interfaceIndex].interface.getFunction(e.target.value);
            newActionList[index] = fn.format();
            setParams([]);
            setParamNames([]);
            setParamTypes([]);
            for (let i = 0; i < fn.inputs.length; i++) {
                setParams((params) => [...params, ""]);
                setParamNames((paramNames) => [...paramNames, fn.inputs[i].name]);
                setParamTypes((paramTypes) => [...paramTypes, fn.inputs[i].type]);
            }
        } else {
            newActionList[index] = e.target.value;
        }
        setActionList(newActionList)
    }

    const onUpdateParams = (e, pIndex) => {
        let newParams = params;
        newParams[pIndex] = e.target.value;
        setParams(newParams);
        setCallData(utils.defaultAbiCoder.encode(paramTypes, params));
    }

    return <div className={styles.actionContainer}>
        <h4># {index + 1}</h4>
        <div className={styles.actionContent}>
            <select
                style={{
                    height: '30px',
                    borderRadius: '8px'
                }}
                value={contract}
                onChange={(e) => { onContractChange(e) }}
            >
                <option value="select">Select a Contract</option>
                <option value="boba">Boba Fees</option>
                <option value={delegate.address}>Governor Bravo Delegate</option>
                <option value={timelock.address}>Timelock</option>
            </select>

            {options.length === 0 ? null : (
                <select
                    style={{
                        height: '30px',
                        borderRadius: '8px'
                    }}
                    value={action} onChange={(e) => onActionChange(e)}>
                    <option value="">Select a Function</option>
                    {options.map((fn, i) => (
                        <option key={i} value={fn}>
                            {fn}
                        </option>
                    ))}
                </select>
            )}

            {options.length > 0 && action ? params.map((p, pIndex) => {
                const { name } = p;

                return <Input
                    key={pIndex}
                    type='text'
                    placeHolder={`${name} ${paramTypes[index]}`}
                    onChange={(e) => onUpdateParams(e, pIndex)}
                />
            }) : null}
        </div>
    </div>
}


export default React.memo(ProposalAction);

