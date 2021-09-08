import json
import yaml
import pymysql
import boto3
import string
import random
import time
import requests
import redis

def watcher_getL1Transactions(event, context):

  # Parse incoming event
  body = json.loads(event["body"])
  address = body.get("address")
  fromRange = int(body.get("fromRange"))
  toRange = int(body.get("toRange"))

  # Read YML
  with open("env.yml", 'r') as ymlfile:
    config = yaml.load(ymlfile, Loader=yaml.FullLoader)

  # Get MySQL host and port
  endpoint = config.get('RDS_ENDPOINT')
  user = config.get('RDS_MYSQL_NAME')
  dbpassword = config.get('RDS_MYSQL_PASSWORD')
  dbname = config.get('RDS_DBNAME')

  con = pymysql.connect(host=endpoint, user=user, db=dbname,
                        passwd=dbpassword, connect_timeout=5)

  transactionData = []
  with con.cursor() as cur:
    try:
      # Order
      #  0              1                   2                      3      4     5                6             7         8                   9                           10
      #  l1Bridge.hash, l1Bridge.blockHash, l1Bridge.blockNumber, `from`, `to`, contractAddress, contractName, activity, crossDomainMessage, crossDomainMessageFinalize, crossDomainMessageSendTime,
      #  11                                       12                               13      14             15           16      17    18
      #  crossDomainMessageEstimateFinalizedTime, crossDomainMessageFinalizedTime, l2Hash, l2BlockNumber, l2BlockHash, l2From, l2To, l1Bridge.fastDeposit,
      #  19             20         21            22             23              24              25      26
      #  depositSender, depositTo, depositToken, depositAmount, depositReceive, depositFeeRate, status, timestamp
      cur.execute("""SELECT
        l1Bridge.hash, l1Bridge.blockHash, l1Bridge.blockNumber, `from`, `to`, contractAddress, contractName, activity, crossDomainMessage, crossDomainMessageFinalize, crossDomainMessageSendTime,
        crossDomainMessageEstimateFinalizedTime, crossDomainMessageFinalizedTime, l2Hash, l2BlockNumber, l2BlockHash, l2From, l2To, l1Bridge.fastDeposit,
        depositSender, depositTo, depositToken, depositAmount, depositReceive, depositFeeRate, status, timestamp
        FROM l1Bridge
        LEFT JOIN depositL2
        on l1Bridge.blockNumber = depositL2.blockNumber AND l1Bridge.hash = depositL2.hash
        WHERE `from`=%s ORDER BY CAST(l1Bridge.blockNumber as unsigned) DESC LIMIT %s OFFSET %s""", (address, toRange - fromRange, fromRange))
      transactionsDataRaw = cur.fetchall()
      for transactionDataRaw in transactionsDataRaw:
        # No cross domain message
        if transactionDataRaw[8] == False:
          crossDomainMessageSendTime, crossDomainMessageEstimateFinalizedTime, fastDeposit = None, None, None
        else:
          crossDomainMessageSendTime = transactionDataRaw[10]
          fastDeposit = transactionDataRaw[18]
          crossDomainMessageEstimateFinalizedTime = transactionDataRaw[11]
        # depositL2
        if transactionDataRaw[7] == "ClientDepositL1" or transactionDataRaw[7] == "ETHDepositInitiated" or transactionDataRaw[7] == "ERC20DepositInitiated":
          depositL2 = True
        else: depositL2 = False

        transactionData.append({
          "hash": transactionDataRaw[0],
          "blockNumber": int(transactionDataRaw[2]),
          "from": transactionDataRaw[3],
          "to": transactionDataRaw[4],
          "timeStamp": transactionDataRaw[26],
          "contractName": transactionDataRaw[6],
          "contractAddress": transactionDataRaw[5],
          "activity": transactionDataRaw[7],
          "depositL2": depositL2,
          "crossDomainMessage": {
            "crossDomainMessage": transactionDataRaw[8],
            "crossDomainMessageFinalize": transactionDataRaw[9],
            "crossDomainMessageSendTime": crossDomainMessageSendTime,
            "crossDomainMessageEstimateFinalizedTime": crossDomainMessageEstimateFinalizedTime,
            "fast": fastDeposit,
            "l2Hash": transactionDataRaw[13],
            "l2BlockNumber": transactionDataRaw[14],
            "l2BlockHash": transactionDataRaw[15],
            "l2From": transactionDataRaw[16],
            "l2To": transactionDataRaw[17]
          },
          "action": {
            "sender": transactionDataRaw[19],
            "to": transactionDataRaw[20],
            "token": transactionDataRaw[21],
            "amount": transactionDataRaw[22],
            "receive": transactionDataRaw[23],
            "feeRate": transactionDataRaw[24],
            "fast": fastDeposit,
            "status": transactionDataRaw[25]
          }
        })

    except Exception as e:
      print(e)
      transactionData = []

  con.close()

  response = {
    "statusCode": 201,
    "headers": {
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Credentials": True,
      "Strict-Transport-Security": "max-age=63072000; includeSubdomains; preload",
      "X-Content-Type-Options": "nosniff",
      "X-Frame-Options": "DENY",
      "X-XSS-Protection": "1; mode=block",
      "Referrer-Policy": "same-origin",
      "Permissions-Policy": "*",
    },
    "body": json.dumps(transactionData),
  }
  return response
