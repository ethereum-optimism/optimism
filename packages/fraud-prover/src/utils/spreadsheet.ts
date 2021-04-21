import { GoogleSpreadsheet } from 'google-spreadsheet'

export default class SpreadSheet {
  public doc
  public sheet

  constructor(id) {
    this.doc = new GoogleSpreadsheet(id)
    this.sheet = null
  }

  async init(email, privateKey) {
    await this.doc.useServiceAccountAuth({
      client_email: email,
      private_key: privateKey,
    })

    await this.doc.loadInfo()
    this.sheet = this.doc.sheetsByIndex[0]
  }

  async addRow(row) {
    return this.sheet.addRow(row)
  }
}
