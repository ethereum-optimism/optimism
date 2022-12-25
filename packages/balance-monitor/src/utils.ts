// import 'ethers'
import { ethers } from 'ethers'
import fetch from 'node-fetch'

// new function to log an account an it's balance
export const describeFinding = (
  account: string,
  actual: ethers.BigNumber,
  threshold: ethers.BigNumber
) => {
  return `Balance of account ${account} is (${ethers.utils.formatEther(
    actual
  )} eth) below threshold (${ethers.utils.formatEther(threshold)} eth)`
}

// Create an alert in ops-genie. The alias will be used an unique identifier for the alert.
// There can only be one open alert per alias. If this is called with an alias which already
// has an alert, it will not be reopened.
export const createAlert = async (alertOpts: {
  alias: string
  message: string
}) => {
  const response = await fetch('https://api.opsgenie.com/v2/alerts', {
    method: 'post',
    body: JSON.stringify({
      message: alertOpts.message,
      alias: alertOpts.alias,
      responders: [{ id: process.env.OPS_GENIE_TEAM, type: 'team' }],
      tags: ['Bedrock-Beta', 'Balance-Low'],
      priority: 'P2',
    }),
    headers: {
      'Content-type': 'application/json',
      Authorization: `GenieKey ${process.env.OPS_GENIE_KEY}`,
    },
  })
  if (!response.ok) {
    console.log(`Error creating alert: ${JSON.stringify(response.body)}`)
  }
}

// Send this with every block. If Ops Genie doesn't get this ping for 10 minutes,
// it will trigger a P3 alert.
export const heartBeat = async () => {
  const response = await fetch(
    `https://api.opsgenie.com/v2/heartbeats/${process.env.OPS_GENIE_HEARTBEAT_NAME}/ping`,
    {
      method: 'get',
      headers: {
        Authorization: `GenieKey ${process.env.OPS_GENIE_KEY}`,
      },
    }
  )
  if (!response.ok) {
    console.log(`Error creating alert: ${JSON.stringify(response.body)}`)
  }
}
