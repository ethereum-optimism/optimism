/*
  Varna - A Privacy-Preserving Marketplace
  Varna uses Fully Homomorphic Encryption to make markets fair. 
  Copyright (C) 2021 Enya Inc. Palo Alto, CA

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

import networkService from 'services/networkService';
import { createAction } from './createAction'

/***********************************************/
/*****           DAO Action                *****/
/***********************************************/

export function fetchDaoBalance() {
    return createAction('BALANCE/DAO/GET', () => networkService.getDaoBalance())
}

export function fetchDaoVotes() {
    return createAction('VOTES/DAO/GET', () => networkService.getDaoVotes())
}

export function transferDao({ recipient, amount }) {
    return createAction('TRANSFER/DAO/CREATE', () => networkService.transferDao({ recipient, amount }))
}

export function delegateVotes({ recipient }) {
    return createAction('DELEGATE/VOTES/CREATE', () => networkService.delegateVotes({ recipient }))
}

export function getProposalThreshold() {
    return createAction('PROPOSAL/THRESHOLD/GET', () => networkService.getProposalThreshold())
}

export function fetchDaoProposals() {
    return createAction('PROPOSALS/GET', () => networkService.fetchProposals())
}

export function createDaoProposal(payload) {
    return createAction('PROPOSAL/CREATE', () => networkService.createProposal(payload))
}

export function castProposalVote(payload) {
    return createAction('PROPOSAL/CAST/VOTE', () => networkService.castProposalVote(payload))
}
