## Overview

The preconfirmation package creates a simple system where two types of users, referred to as searchers and builders, can exchange bid requests and confirmations over a peer-to-peer network. Searchers use the SendBid function to send bids and wait for confirmations from builders. Builders use the handleBid function to receive bids, check them, and send back confirmations if the bids are valid. 

### Diagram
![](preconf-mc.png)