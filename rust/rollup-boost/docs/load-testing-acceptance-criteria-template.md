## Overview

This document serves as a generic acceptance criteria checklist template to be run through before deploying network upgrades through Rollup Boost. Copy this checklist and fill it out with the appropriate data for your specific chain before upgrading your network..

## **Environment**
- **Network:** [Specify the network]
- **RPC Endpoint:** [Specify the blockchain network RPC URL]
- **Contender Load Testing Scenarios:**
- **Block Builder Version:** [Specify the deployed version]
- **OP Stack Version:** [GitHub link to release]
- **Test Period:** [Specify the test period]

## **Acceptance Criteria Checklist**
### **1. Overall Network Performance**

🎯 **Peak Gas Utilization (GPS)**
- **Description:** Measures the highest observed gas consumption per second in the network.
- **Expected:** 5 million GPS
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

🎯 **Peak Transactions per Second (TPS)**
- **Description:** Measures the maximum number of transactions successfully processed per second.
- **Expected:** [Expected TPS]
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

🎯 **Average Block Time**
- **Description:** Time taken between consecutive block proposals.
- **Expected:** 1 second
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

---

### **2. Chain Operator Performance Metrics**

***RPC Ingress Metrics (Transaction Handling)***

🎯 **Peak Burst Transactions per Second (TPS)**
- **Description:** Measures the highest rate at which transactions are submitted to the RPC within a short burst.
- **Expected:** 1,000 TPS
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

🎯 **Sustained Transactions per Second (TPS)**
- **Description:** Measures the average transaction ingress rate over a sustained 20-minute period.
- **Expected:** 200 TPS
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

***RPC Latency Metrics***

🎯 **Average RPC Response Latency**
- **Description:** Measures the typical response time for RPC requests.
- **Expected:** ≤ 60 ms
- **Measured:** [Insert measured value]
-**Supporting Data:** [Link to results]

🎯 **Max RPC Response Latency**
- **Description:** Measures the worst-case response time for RPC requests.
- **Expected:** ≤ 120 ms
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

***RPC Egress (Block Builder Communication)***

🎯 **Average Block Builder Response Latency**
- **Expected:** ≤ 60 ms
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

🎯 **Max Block Builder Response Latency**
- **Expected:** ≤ 100 ms
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

***Sequencer & Rollup Boost Health Checks***

🎯 **Sequencer Resource Utilization (CPU, Memory, Disk, Network)**
- **Expected:** No resource exhaustion or anomalies.
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

🎯 **Rollup Boost Resource Utilization (CPU, Memory, Disk, Network)**
- **Expected:** No resource exhaustion or anomalies.
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

🎯 **RPC Endpoint Stability**
- **Expected:** No downtime or unexpected crashes.
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

🎯 **Non-Transaction RPC Method Stability**
- **Description:** Ensures non-transaction-related RPC methods function without failure.
- **Expected:** No failures.
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

---

### **3. Block Builder Performance Metrics**

🎯 **Missed Block Rate**
- **Description:** Percentage of expected blocks that were not produced.
- **Expected:** ≤ 10%
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

🎯 **End-to-End Block Build Latency**

**Description:** Time taken from transaction ingress to finalized block production.
- **Expected:** ≤ 100 ms
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

🎯 **Block Submission Response Latency**
- **Expected:** ≤ 100 ms
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

🎯 **Block Builder Health Checks (CPU, Memory, Disk, Network)**
- **Expected:** No resource exhaustion or anomalies.
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

---

### **4. System-Wide Reliability Checks**

🎯 **100% Transaction Forwarding Accuracy**
- **Description:** Ensures all transactions sent to the system are correctly received and logged.
- **Expected:** 100% transaction acceptance.
- **Measured:** [Insert measured value]
- **Supporting Data:** [Link to results]

🎯 **Observability: Full Transaction Traceability**
- **Description:** Verifies that transactions can be traced end-to-end across all infrastructure components.
- **Expected:** Given a random transaction hash during load testing, timestamps for each step in the transaction pipeline can be observed.
- **Measured:**
   - **Client Submission Timestamp:** [Measured result]
   - **Chain Operator Ingress Timestamp:** [Measured result]
   - **Block Builder Ingress Timestamp:** [Measured result]
   - **Block Submission Timestamp:** [Measured result]

---

### **Final Decision**

✅ **All criteria met?**
- **Decision:** [Yes/No]
- **Date:** [Specify the test period]

**Notes & Recommendations**

•	[Add any additional observations, issues, or follow-up actions]