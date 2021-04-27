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

const initialState = {
  FHEseed: null,
  AESKey: null,
  loggedIn: false,
  isBeginner: true,
  invitationCodeGood: false,
  invitationCodeVerifyLoad: false,
  invitationCodeVerifyError: null,
}

function loginReducer (state = initialState, action) {
  switch (action.type) {
    case 'PROVIDE_PASSWORD':
      return {
        ...state,
        AESKey: action.payload.AESKey,
        FHEseed: action.payload.FHEseed,
      }
    case 'LOGIN':
      return {
        ...state,
        loggedIn: true,
      }
    case 'UPDATE_USER_TYPE':
      return {
        ...state,
        isBeginner: action.payload,
      }
    case 'VERIFY_INVITATION_CODE':
      return {
        ...state,
        invitationCodeGood: false,
        invitationCodeVerifyLoad: true,
        invitationCodeVerifyError: null,
      }
    case 'VERIFY_INVITATION_CODE_SUCCESS':
      return {
        ...state,
        invitationCodeGood: true,
        invitationCodeVerifyLoad: false,
        invitationCodeVerifyError: false,
      }
    case 'VERIFY_INVITATION_CODE_FAILURE':
      return {
        ...state,
        invitationCodeGood: false,
        invitationCodeVerifyLoad: false,
        invitationCodeVerifyError: action.payload,
      }
    default:
      return state;
  }
}

export default loginReducer;