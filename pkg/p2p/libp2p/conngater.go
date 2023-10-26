package libp2p

import (
	"math/big"

	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/register"
)

type connectionAllowance int

const (
	Undecided connectionAllowance = iota
	DenyUnresolvedAddress
	DenyBadRegisterCall
	DenyBlockedPeer
	DenyNotEnoughStake
	DenyUserToUser
	Accept
)

//var connectionAllowanceStrings = map[connectionAllowance]string{
//	Undecided:              "Undecided",
//	DenyUnresolvedAddress:  "DenyUnresolvedAddress",
//	DenyBadRegisterCall:    "DenyBadRegisterCall",
//	DenyBlockedPeer:        "DenyBlockedPeer",
//	DenyNotEnoughStake:     "DenyNotEnoughStake",
//	DenyUserToUser: "DenyUserToUser",
//	Accept:                 "Allow",
//}

func (c connectionAllowance) isDeny() bool {
	return !(c == Accept || c == Undecided)
}

// make sure the connections are between provider<>provider, provider<>user!
type ConnectionGater interface {
	// InterceptPeerDial intercepts peer dialing
	InterceptPeerDial(p peer.ID) (allow bool)
	// InterceptAddrDial intercepts address dialing
	InterceptAddrDial(peer.ID, multiaddr.Multiaddr) (allow bool)
	// InterceptAccept intercepts connection acceptance
	InterceptAccept(network.ConnMultiaddrs) (allow bool)
	// InterceptSecured intercepts secured connection
	InterceptSecured(network.Direction, peer.ID, network.ConnMultiaddrs) (allow bool)
	// InterceptUpgraded intercepts upgraded connection
	InterceptUpgraded(network.Conn) (allow bool, reason control.DisconnectReason)
}

type connectionGater struct {
	register     register.Register
	selfType     p2p.PeerType
	minimumStake *big.Int
	metrics      *metrics
}

// newConnectionGater creates a new instance of ConnectionGater
func newConnectionGater(register register.Register, selfType p2p.PeerType, minimumStake *big.Int, metrics *metrics) ConnectionGater {
	return &connectionGater{
		register:     register,
		selfType:     selfType,
		minimumStake: minimumStake,
		metrics:      metrics,
	}
}

// checkPeerTrusted determines the trust status of a peer
func (cg *connectionGater) checkPeerTrusted(p peer.ID) connectionAllowance {
	// TODO: Implement the logic to determine whether the peer is trusted or not
	return Undecided
}

// TODO @iowar blocker implementation: consult the team
// checkPeerBlocked checks if a peer is blocked and returns the appropriate connection allowance status
func (cg *connectionGater) checkPeerBlocked(p peer.ID) connectionAllowance {
	//	// check if the peer is in the list of blocked peers, and deny the connection if found
	//	for _, peerID := range cg.blocker.list() {
	//		if p == peerID {
	//			return DenyBlockedPeer
	//		}
	//	}
	//
	// if the peer is not in the blocked list, allow the connection
	return Accept
}

// checkPeerStake checks if a peer has enough stake and returns the appropriate
// connection allowance status
func (cg *connectionGater) checkPeerStake(p peer.ID) connectionAllowance {
	// get eth address
	ethAddress, err := GetEthAddressFromPeerID(p)
	if err != nil {
		return DenyUnresolvedAddress
	}

	// get stake
	stake, err := cg.register.GetStake(ethAddress)
	if err != nil {
		return DenyBadRegisterCall
	}

	enoughStake := stake.Cmp(cg.minimumStake) >= 0

	// possible s<>s connection
	// ! s<>s connection
	if (cg.selfType == p2p.PeerTypeUser) && !enoughStake {
		return DenyUserToUser
	}

	// Reject potential s<>s connections and accept the remaining requests,
	// allowing authentication during the handshake phase
	return Accept
}

// resolveConnectionAllowance resolves the connection allowance based on trusted and blocked statuses
func resolveConnectionAllowance(
	trustedStatus connectionAllowance,
	blockedStatus connectionAllowance,
) connectionAllowance {
	// if the peer's trusted status is 'Undecided', resolve the connection allowance based on the blocked status
	if trustedStatus == Undecided {
		return blockedStatus
	}
	return trustedStatus
}

// checks if a peer is allowed to dial/accept
func (cg *connectionGater) checkAllowedPeer(p peer.ID) connectionAllowance {
	return resolveConnectionAllowance(cg.checkPeerTrusted(p), cg.checkPeerBlocked(p))
}

// InterceptPeerDial intercepts the process of dialing a peer
//
// all peer dialing attempts are allowed
func (cg *connectionGater) InterceptPeerDial(p peer.ID) bool {
	allowance := cg.checkAllowedPeer(p)
	if allowance.isDeny() {
		return false
	}

	return !cg.checkPeerStake(p).isDeny()
}

// InterceptAddrDial intercepts the process of dialing an address
//
// all address dialing attempts are allowed
// TODO rate limiter
func (cg *connectionGater) InterceptAddrDial(p peer.ID, addr multiaddr.Multiaddr) bool {
	allowance := cg.checkAllowedPeer(p)
	return !allowance.isDeny()
}

// InterceptAccept intercepts the process of accepting a connection
//
// all connection acceptance attempts are allowed
func (cg *connectionGater) InterceptAccept(connMultiaddrs network.ConnMultiaddrs) bool {
	return true
}

// InterceptSecured intercepts a secured connection, regardless of its direction (inbound/outbound)
func (cg *connectionGater) InterceptSecured(dir network.Direction, p peer.ID, connMultiaddrs network.ConnMultiaddrs) bool {
	allowance := cg.checkAllowedPeer(p)
	if allowance.isDeny() {
		cg.metrics.RejectedConnectionCount.Inc()
		return false
	}

	// note: we are indifferent to the direction (inbound/outbound)
	// if you want to manipulate (inbound/outbound) connections, make the change
	// note:if it is desired to not establish a connection with a peer,
	// ensure that it is rejected in both incoming and outgoing connections
	if dir == network.DirInbound {
		return cg.validateInboundConnection(p, connMultiaddrs)
	} else {
		return cg.validateOutboundConnection(p, connMultiaddrs)
	}
}

// InterceptUpgraded intercepts the process of upgrading a connection
//
// all connection upgrade attempts are allowed
func (cg *connectionGater) InterceptUpgraded(conn network.Conn) (bool, control.DisconnectReason) {
	return true, control.DisconnectReason(0)
}

// validateInboundConnection validates an inbound connection by extracting its
// public key and performing stake validation if the validation succeeds and
// the peer's stake is greater than minimal stake, the connection is allowed
// otherwise, the connection is rejected
func (cg *connectionGater) validateInboundConnection(p peer.ID, connMultiaddrs network.ConnMultiaddrs) bool {
	allowance := cg.checkPeerStake(p)
	if allowance.isDeny() {
		cg.metrics.RejectedConnectionCount.Inc()
	}

	return !allowance.isDeny()
}

// validateOutboundConnection validates an outbound connection by extracting
// its public key and performing stake validation if the validation succeeds
// and the peer's stake is greater than minimal stake, the connection is
// allowed otherwise, the connection is rejected
func (cg *connectionGater) validateOutboundConnection(p peer.ID, connMultiaddrs network.ConnMultiaddrs) bool {
	allowance := cg.checkPeerStake(p)
	if allowance.isDeny() {
		cg.metrics.RejectedConnectionCount.Inc()
	}

	return !allowance.isDeny()
}
