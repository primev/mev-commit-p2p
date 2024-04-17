package keyexchange

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto/ecies"
	keyexchangepb "github.com/primevprotocol/mev-commit/gen/go/keyexchange/v1"
	"github.com/primevprotocol/mev-commit/pkg/keykeeper"
	"github.com/primevprotocol/mev-commit/pkg/p2p"
	"github.com/primevprotocol/mev-commit/pkg/signer"
	"github.com/primevprotocol/mev-commit/pkg/topology"
	"google.golang.org/protobuf/proto"
)

func New(
	topo Topology,
	streamer p2p.Streamer,
	keyKeeper keykeeper.KeyKeeper,
	logger *slog.Logger,
	signer signer.Signer,
) *KeyExchange {
	return &KeyExchange{
		topo:              topo,
		streamer:          streamer,
		keyKeeper:         keyKeeper,
		logger:            logger,
		signer:            signer,
	}
}

func (ke *KeyExchange) timestampMessageStream() p2p.StreamDesc {
	return p2p.StreamDesc{
		Name:    ProtocolName,
		Version: ProtocolVersion,
		Handler: ke.handleTimestampMessage,
	}
}

func (ke *KeyExchange) Streams() []p2p.StreamDesc {
	return []p2p.StreamDesc{ke.timestampMessageStream()}
}

func (ke *KeyExchange) SendTimestampMessage() error {
	providers, err := ke.getProviders()
	if err != nil {
		ke.logger.Error("getting providers", "error", err)
		return ErrNoProvidersAvailable
	}

	encryptedKeys, timestampMessage, err := ke.prepareMessages(providers)
	if err != nil {
		return err
	}

	if err := ke.distributeMessages(providers, encryptedKeys, timestampMessage); err != nil {
		return err
	}

	return nil
}

func (ke *KeyExchange) getProviders() ([]p2p.Peer, error) {
	providers := ke.topo.GetPeers(topology.Query{Type: p2p.PeerTypeProvider})
	if len(providers) == 0 {
		return nil, ErrNoProvidersAvailable
	}
	return providers, nil
}

func (ke *KeyExchange) prepareMessages(providers []p2p.Peer) ([][]byte, []byte, error) {
	bidderKK, ok := ke.keyKeeper.(*keykeeper.BidderKeyKeeper)
	if !ok {
		return nil, nil, fmt.Errorf("keyKeeper is not of type BidderKeyKeeper")
	}

	var encryptedKeys [][]byte
	for _, provider := range providers {
		encryptedKey, err := ecies.Encrypt(rand.Reader, provider.Keys.PKEPublicKey, bidderKK.AESKey, nil, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("error encrypting key for provider %s: %w", provider.EthAddress, err)
		}
		encryptedKeys = append(encryptedKeys, encryptedKey)
	}

	timestampMessage := fmt.Sprintf("mev-commit bidder %s setup %d", bidderKK.KeySigner.GetAddress(), time.Now().Unix())
	encryptedTimestampMessage, err := keykeeper.EncryptWithAESGCM(bidderKK.AESKey, []byte(timestampMessage))
	if err != nil {
		return nil, nil, fmt.Errorf("error encrypting timestamp message: %w", err)
	}

	return encryptedKeys, encryptedTimestampMessage, nil
}

func (ke *KeyExchange) distributeMessages(providers []p2p.Peer, encryptedKeys [][]byte, timestampMessage []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ekmWithSignature, err := ke.createSignedMessage(encryptedKeys, timestampMessage)
	if err != nil {
		return fmt.Errorf("error creating signed message: %w", err)
	}

	var wg sync.WaitGroup
	errorsChan := make(chan error, len(providers))

	for _, provider := range providers {
		wg.Add(1)
		go func(provider p2p.Peer) {
			defer wg.Done()
			if err := ke.sendMessageToProvider(ctx, provider, ekmWithSignature); err != nil {
				errorsChan <- err
				ke.logger.Error("error sending message to provider", "provider", provider.EthAddress, "error", err)
			}
		}(provider)
	}

	wg.Wait()
	close(errorsChan)

	if len(errorsChan) > 0 {
		return fmt.Errorf("errors occurred while distributing messages")
	}

	return nil
}

func (ke *KeyExchange) createSignedMessage(encryptedKeys [][]byte, timestampMessage []byte) (*keyexchangepb.EKMWithSignature, error) {
	message := keyexchangepb.EncryptedKeysMessage{
		EncryptedKeys:    encryptedKeys,
		TimestampMessage: timestampMessage,
	}

	messageBytes, err := proto.Marshal(&message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	hashedMessage := hashData(messageBytes)

	bidderKK := ke.keyKeeper.(*keykeeper.BidderKeyKeeper)

	signature, err := bidderKK.KeySigner.SignHash(hashedMessage.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	ekmWithSignature := &keyexchangepb.EKMWithSignature{
		Message:   messageBytes,
		Signature: signature,
	}

	return ekmWithSignature, nil
}

func (ke *KeyExchange) sendMessageToProvider(ctx context.Context, provider p2p.Peer, ekmWithSignature *keyexchangepb.EKMWithSignature) error {
	stream, err := ke.streamer.NewStream(
		ctx,
		provider,
		nil,
		ke.timestampMessageStream(),
	)
	if err != nil {
		return fmt.Errorf("failed to create new stream to provider %s: %w", provider.EthAddress, err)
	}
	defer stream.Close()

	err = stream.WriteMsg(ctx, ekmWithSignature)
	if err != nil {
		_ = stream.Reset()
		return fmt.Errorf("failed to send message to provider %s: %w", provider.EthAddress, err)
	}

	return nil
}

func (ke *KeyExchange) handleTimestampMessage(ctx context.Context, peer p2p.Peer, stream p2p.Stream) error {
	ekmWithSignature, err := ke.readAndVerifyMessage(ctx, peer, stream)
	if err != nil {
		return fmt.Errorf("read and verify message failed: %w", err)
	}

	message, aesKey, err := ke.decryptMessage(ekmWithSignature)
	if err != nil {
		return fmt.Errorf("decrypt message failed: %w", err)
	}

	if err := ke.validateAndProcessTimestamp(message); err != nil {
		return fmt.Errorf("validate and process timestamp failed: %w", err)
	}

	ke.keyKeeper.(*keykeeper.ProviderKeyKeeper).SetAESKey(peer.EthAddress, aesKey)

	ke.logger.Info("successfully processed timestamp message", "peer", peer.EthAddress, "key", aesKey)

	return nil
}

func (ke *KeyExchange) readAndVerifyMessage(ctx context.Context, peer p2p.Peer, stream p2p.Stream) (*keyexchangepb.EKMWithSignature, error) {
	if peer.Type != p2p.PeerTypeBidder {
		return nil, ErrInvalidBidderTypeForMessage
	}

	ekmWithSignature := new(keyexchangepb.EKMWithSignature)

	err := stream.ReadMsg(ctx, ekmWithSignature)
	if err != nil {
		return nil, err
	}

	err = ke.verifySignature(peer, ekmWithSignature)
	if err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	return ekmWithSignature, nil
}

func (ke *KeyExchange) verifySignature(peer p2p.Peer, ekm *keyexchangepb.EKMWithSignature) error {
	verified, ethAddress, err := ke.signer.Verify(ekm.Signature, ekm.Message)
	if err != nil {
		return errors.Join(err, ErrSignatureVerificationFailed)
	}

	if !verified {
		return ErrSignatureVerificationFailed
	}

	if !bytes.Equal(peer.EthAddress.Bytes(), ethAddress.Bytes()) {
		return ErrObservedAddressMismatch
	}

	return nil
}

func (ke *KeyExchange) decryptMessage(ekmWithSignature *keyexchangepb.EKMWithSignature) ([]byte, []byte, error) {
	var (
		aesKey    []byte
		decrypted bool
		err       error
		message   keyexchangepb.EncryptedKeysMessage
	)

	err = proto.Unmarshal(ekmWithSignature.Message, &message)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	providerKK := ke.keyKeeper.(*keykeeper.ProviderKeyKeeper)

	for i := 0; i < len(message.EncryptedKeys); i++ {
		aesKey, err = providerKK.DecryptWithECIES(message.EncryptedKeys[i])
		if err == nil {
			decrypted = true
			break // Successfully decrypted AES key, stop trying further keys
		}
	}

	if !decrypted {
		return nil, nil, fmt.Errorf("none of the AES keys could be decrypted")
	}

	encryptedMessage := message.TimestampMessage
	decryptedMessage, err := keykeeper.DecryptWithAESGCM(aesKey, encryptedMessage)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt message: %w", err)
	}

	return decryptedMessage, aesKey, nil
}

func (ke *KeyExchange) validateAndProcessTimestamp(message []byte) error {
	_, timestamp, err := parseTimestampMessage(string(message))
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	if !isTimestampRecent(timestamp) {
		return fmt.Errorf("the timestamp is more than 1 minute old")
	}

	return nil
}