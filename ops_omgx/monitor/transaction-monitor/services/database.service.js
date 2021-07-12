#!/usr/bin/env node

const mysql = require('mysql');
const util = require('util');

const OptimismEnv = require('./utilities/optimismEnv');

class DatabaseService extends OptimismEnv{
  constructor() {
    super(...arguments);
    this.con = null;
    this.query = null;
  }

  async initDatabaseService() {
    this.con = mysql.createConnection({
      host: this.MySQLHostURL,
      port: this.MySQLPort,
      user: this.MySQLUsername,
      password: this.MySQLPassword,
    });
    this.query = util.promisify(this.con.query).bind(this.con);
  }

  async initMySQL() {
    this.logger.info('Initializing the database...');
    await this.query(`CREATE DATABASE IF NOT EXISTS ${this.MySQLDatabaseName}`);
    await this.query(`USE ${this.MySQLDatabaseName}`);
    await this.query(`CREATE TABLE IF NOT EXISTS block
      ( 
        hash VARCHAR(255) NOT NULL,
        parentHash VARCHAR(255) NOT NULL,
        blockNumber INT NOT NULL,
        timestamp INT,
        nonce VARCHAR(255),
        gasLimit INT,
        gasUsed INT,
        PRIMARY KEY ( hash )
      )`
    );
    await this.query(`CREATE TABLE IF NOT EXISTS transaction
      ( 
        hash VARCHAR(255) NOT NULL,
        blockHash VARCHAR(255) NOT NULL,
        blockNumber INT NOT NULL,
        \`from\` VARCHAR(255),
        \`to\` VARCHAR(255),
        value VARCHAR(255),
        nonce VARCHAR(255),
        gasLimit INT,
        gasPrice INT,
        timestamp INT,
        PRIMARY KEY ( hash )
      )`
    );
    await this.query(`CREATE TABLE IF NOT EXISTS receipt
      ( 
        hash VARCHAR(255) NOT NULL,
        blockHash VARCHAR(255) NOT NULL,
        blockNumber INT NOT NULL,
        \`from\` VARCHAR(255),
        \`to\` VARCHAR(255),
        gasUsed INT,
        cumulativeGasUsed INT,
        crossDomainMessage BOOL,
        crossDomainMessageFinalize BOOL,
        crossDomainMessageSendTime INT,
        crossDomainMessageEstimateFinalizedTime INT,
        timestamp INT,
        crossDomainMessageFinalizedTime INT,
        PRIMARY KEY ( hash )
      )`
    );
    this.logger.info('Initialized the database.');
  }

  async insertBlockData(blockData) {
    await this.query(`USE ${this.MySQLDatabaseName}`);
    await this.query(`INSERT IGNORE INTO block
      SET hash='${blockData.hash.toString()}',
      parentHash='${blockData.parentHash.toString()}',
      blockNumber='${blockData.number.toString()}',
      timestamp='${blockData.timestamp.toString()}',
      nonce='${blockData.nonce.toString()}',
      gasLimit='${blockData.gasLimit.toString()}',
      gasUsed='${blockData.gasUsed.toString()}'
    `);
  }

  async insertTransactionData(transactionData) {
    await this.query(`USE ${this.MySQLDatabaseName}`);
    await this.query(`INSERT IGNORE INTO transaction
      SET hash='${transactionData.hash.toString()}',
      blockHash='${transactionData.blockHash.toString()}',
      blockNumber='${transactionData.blockNumber.toString()}',
      \`from\`=${transactionData.from ? "'" + transactionData.from + "'" : null},
      \`to\`=${transactionData.to ? "'" + transactionData.to + "'" : null},
      value='${transactionData.value.toString()}',
      nonce='${transactionData.nonce.toString()}',
      gasLimit='${transactionData.gasLimit.toString()}',
      gasPrice='${transactionData.gasPrice.toString()}',
      timestamp='${transactionData.timestamp.toString()}'
    `);
  }

  async insertReceiptData(receiptData) {
    await this.query(`USE ${this.MySQLDatabaseName}`);
    await this.query(`INSERT IGNORE INTO receipt
      SET hash='${receiptData.transactionHash.toString()}',
      blockHash='${receiptData.blockHash.toString()}',
      blockNumber='${receiptData.blockNumber.toString()}',
      \`from\`=${receiptData.from ? "'" + receiptData.from + "'" : null},
      \`to\`=${receiptData.to ? "'" + receiptData.to + "'" : null},
      gasUsed='${receiptData.gasUsed.toString()}',
      cumulativeGasUsed='${receiptData.cumulativeGasUsed.toString()}',
      crossDomainMessage=${receiptData.crossDomainMessage},
      crossDomainMessageFinalize=${receiptData.crossDomainMessageFinalize},
      crossDomainMessageSendTime=${receiptData.crossDomainMessageSendTime},
      crossDomainMessageEstimateFinalizedTime=${receiptData.crossDomainMessage ? receiptData.crossDomainMessageEstimateFinalizedTime : null},
      crossDomainMessageFinalizedTime = ${receiptData.crossDomainMessageFinalizedTime ? receiptData.crossDomainMessageFinalizedTime : null},
      timestamp='${receiptData.timestamp.toString()}'
    `);
  }

  async getCrossDomainData() {
    await this.query(`USE ${this.MySQLDatabaseName}`);
    return await this.query(`SELECT hash, blockNumber FROM receipt
      WHERE crossDomainMessage=${true}
      AND crossDomainMessageFinalize=${false}
      AND UNIX_TIMESTAMP() > crossDomainMessageEstimateFinalizedTime
    `);
  }

  async updateCrossDomainData(receiptData) {
    await this.query(`USE ${this.MySQLDatabaseName}`);
    return await this.query(`UPDATE receipt
      SET crossDomainMessageFinalize=${receiptData.crossDomainMessageFinalize},
      crossDomainMessageFinalizedTime=${receiptData.crossDomainMessageFinalizedTime}
      WHERE hash='${receiptData.transactionHash.toString()}'
      AND blockHash='${receiptData.blockHash.toString()}'
    `);
  }

  async getNewestBlock(){
    await this.query(`USE OMGXRinkeby`);
    return await this.query(`SELECT MAX(blockNumber) from block`);
  }
}

module.exports = DatabaseService;