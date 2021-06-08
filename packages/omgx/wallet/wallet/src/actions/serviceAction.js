const VERSION = "1.0.5";
const SERVICE_API_URL = "https://api-service.rinkeby.omgx.network/";

export const checkVersion = () => {
  fetch(SERVICE_API_URL + 'get.wallet.version', {
    method: "GET",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
  }).then(res => {
    if (res.status === 201) {
      return res.json()
    } else {
      return ""
    }
  }).then(data => {
    if (data !== "") {
      if (data.version !== VERSION) {
        caches.keys().then(async function(names) {
          await Promise.all(names.map(name => caches.delete(name)))
        });
      }
    }
  })
}