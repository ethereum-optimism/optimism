// Promisify the it.next(cb) function
export const itNext = (it): Promise<{ key: Buffer, value: Buffer }> => {
  return new Promise((resolve, reject) => {
    it.next((err, key, value) => {
      if (err) {
        reject(err)
      }
      resolve({ key, value })
    })
  })
}

// Promisify the it.end(cb) function
export const itEnd = (it) => {
  return new Promise((resolve, reject) => {
    it.end((err) => {
      if (err) {
        reject(err)
      }
      resolve()
    })
  })
}
