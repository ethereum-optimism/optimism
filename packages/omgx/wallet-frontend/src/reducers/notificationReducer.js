/*
  OmgX - A Privacy-Preserving Marketplace
  OmgX uses Fully Homomorphic Encryption to make markets fair. 
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
  notificationText: null,
  notificationButtonText: null,
  notificationButtonAction: null,
  notificationStatus: 'close',
};

function notificationReducer (state = initialState, action) {
  switch (action.type) {
    case 'OPEN_NOTIFICATION':
      return {
        notificationStatus: 'open',
        notificationText: action.payload.notificationText,
        notificationButtonText: action.payload.notificationButtonText,
        notificationButtonAction: action.payload.notificationButtonAction,
      }
    case 'CLOSE_NOTIFICATION':
      return {
        notificationStatus: 'close',
        notificationText: null,
        notificationButtonText: null,
        notificationButtonAction: null,
      }
    default:
      return state;
  }
}

export default notificationReducer;