## Overview

The preconfirmation package creates a simple system where two types of users, referred to as users and providers, can exchange bid requests and confirmations over a peer-to-peer network. Users use the SendBid function to send bids and wait for confirmations from providers. Providers use the handleBid function to receive bids, check them, and send back confirmations if the bids are valid. 

### Diagram
![](preconf-mc.png)
