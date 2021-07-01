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

import { generateAESKey } from 'cryptoWorker/cryptoWorker';

import serviceAxiosInstance from '../api/serviceAxios';

const updatedPassword = (FHEseed, AESKey) => ({
  type: 'PROVIDE_PASSWORD',
  payload: { FHEseed, AESKey },
})

export const login = () => ({
  type: 'LOGIN',
})

export const updateIsBeginner = (data) => ({
  type: 'UPDATE_USER_TYPE',
  payload: data,
})

const verifyInvitationCodeBegin = () => ({
  type: 'VERIFY_INVITATION_CODE',
})

const verifyInvitationCodeSuccess = () => ({
  type: 'VERIFY_INVITATION_CODE_SUCCESS',
})

const verifyInvitationCodeFailure = (data) => ({
  type: 'VERIFY_INVITATION_CODE_FAILURE',
  payload: data,
})

export const providePassword = (FHEseed) => (dispatch) => {
  generateAESKey(FHEseed).then(AESKey => {
    dispatch(updatedPassword(FHEseed, AESKey));
  })
}

export const verifyInvitationCode = (invitationCode) => (dispatch) => {
  dispatch(verifyInvitationCodeBegin());
  serviceAxiosInstance.post(`invitation.code`, {invitationCode})
  .then(res => {
    if (res.status === 201) {
      dispatch(verifyInvitationCodeSuccess());
    } else {
      dispatch(verifyInvitationCodeFailure(res.status));
    }
  })
}