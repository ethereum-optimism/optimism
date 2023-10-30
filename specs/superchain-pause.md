# Superchain Pause

The whole damn superchain can be paused.

WIP

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Pausability flow

The simplest way to modify the current system to meet the pause spec is:

```mermaid
flowchart TD
StandardBridge --> L1CrossDomainMessenger
L1ERC721Bridge --> L1CrossDomainMessenger
L1CrossDomainMessenger --> OptimismPortal
OptimismPortal --> SystemConfig
SystemConfig --> SuperchainConfig
