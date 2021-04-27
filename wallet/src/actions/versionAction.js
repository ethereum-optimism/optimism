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

// import { SERVICE_API_URL, VERSION } from 'Settings';

export const checkVersion = () => (dispatch) => {
  // fetch(SERVICE_API_URL + 'version', {
  //   method: "GET",
  //   headers: {
  //     'Accept': 'application/json',
  //     'Content-Type': 'application/json',
  //   },
  // }).then(res => {
  //   if (res.status === 201) {
  //     return res.json()
  //   } else {
  //     return ""
  //   }
  // }).then(data => {
  //   if (data !== "") {
  //     if (data.version !== VERSION) {
  //       caches.keys().then(async function(names) {
  //         await Promise.all(names.map(name => caches.delete(name)))
  //       });
  //     }
  //   }
  // })
}