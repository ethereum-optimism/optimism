import json
import yaml
import pymysql
import boto3
import string
import random
import time
import requests
import redis

def watcher_getTransactions(event, context):

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
      cur.execute("""SELECT hash, blockNumber, `from`, `to`, timestamp, crossDomainMessage, crossDomainMessageFinalize, crossDomainMessageSendTime, crossDomainMessageEstimateFinalizedTime, fastRelay
        FROM receipt WHERE `from`=%s ORDER BY CAST(blockNumber as unsigned) DESC LIMIT %s OFFSET %s""", (address, toRange - fromRange, fromRange))
      transactionsDataRaw = cur.fetchall()

      for transactionDataRaw in transactionsDataRaw:
        if transactionDataRaw[5] == False:
          crossDomainMessageSendTime, crossDomainMessageEstimateFinalizedTime, fastRelay = None, None, None
        else:
          crossDomainMessageSendTime, fastRelay = transactionDataRaw[7], transactionDataRaw[9]
          if fastRelay == True:
            # Estimate time is 3 minutes
            crossDomainMessageEstimateFinalizedTime = crossDomainMessageSendTime + 60 * 3
          else:
            # Estimate time is 7 days
            crossDomainMessageEstimateFinalizedTime = crossDomainMessageSendTime + 60 * 60 * 24 * 7

        transactionData.append({
          "hash": transactionDataRaw[0],
          "blockNumber": int(transactionDataRaw[1]),
          "from": transactionDataRaw[2],
          "to": transactionDataRaw[3],
          "timeStamp": transactionDataRaw[4],
          "crossDomainMessage": transactionDataRaw[5],
          "crossDomainMessageFinailze": transactionDataRaw[6],
          "crossDomainMessageSendTime": crossDomainMessageSendTime,
          "crossDomainMessageEstimateFinalizedTime": crossDomainMessageEstimateFinalizedTime,
          "fastRelay": fastRelay
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