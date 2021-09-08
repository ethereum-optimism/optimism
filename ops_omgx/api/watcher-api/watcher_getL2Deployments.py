import json
import yaml
import pymysql
import boto3
import string
import random
import time
import requests
import redis

def watcher_getL2Deployments(event, context):

  # Parse incoming event
  body = json.loads(event["body"])
  address = body.get("address")

  # Read YML
  with open("env.yml", 'r') as ymlfile:
    config = yaml.load(ymlfile)

  # Get MySQL host and port
  endpoint = config.get('RDS_ENDPOINT')
  user = config.get('RDS_MYSQL_NAME')
  dbpassword = config.get('RDS_MYSQL_PASSWORD')
  dbname = config.get('RDS_DBNAME')

  con = pymysql.connect(endpoint, user=user, db=dbname,
                        passwd=dbpassword, connect_timeout=5)

  transactionData = []
  with con:
    try:
      cur = con.cursor()
      cur.execute("""SELECT hash, blockNumber, `from`, timestamp, contractAddress FROM receipt WHERE `from`=%s AND `to` is null ORDER BY CAST(blockNumber as unsigned) DESC""", (address))
      transactionsDataRaw = cur.fetchall()
      for transactionDataRaw in transactionsDataRaw:
        transactionData.append({
          "hash": transactionDataRaw[0],
          "blockNumber": int(transactionDataRaw[1]),
          "from": transactionDataRaw[2],
          "timeStamp": transactionDataRaw[3],
          "contractAddress": transactionDataRaw[4]
        })
    except Exception as e:
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
