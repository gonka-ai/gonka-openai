package gonkaopenai

import (
	cryptotypes "github.com/cometbft/cometbft/proto/tendermint/crypto"
	comettypes "github.com/cometbft/cometbft/types"
)

type ExcludedParticipant struct {
	Address string `json:"address"`
}

type ActiveParticipantWithProof struct {
	ActiveParticipants      ActiveParticipants    `json:"active_participants"`
	Addresses               []string              `json:"addresses"`
	ActiveParticipantsBytes string                `json:"active_participants_bytes"`
	ProofOps                *cryptotypes.ProofOps `json:"proof_ops"`
	Validators              []*Validator          `json:"validators"`
	Block                   *comettypes.Block     `json:"block"`
	ExcludedParticipants    []ExcludedParticipant `json:"excluded_participants"`
}

type Validator struct {
	Address          string `json:"address"`
	PubKey           string `json:"pub_key"`
	VotingPower      int64  `json:"voting_power"`
	ProposerPriority int64  `json:"proposer_priority"`
}

// TODO: import as dependency from inference-chain
type ActiveParticipants struct {
	Participants         []*ActiveParticipant `protobuf:"bytes,1,rep,name=participants,proto3" json:"participants,omitempty"`
	EpochGroupId         uint64               `protobuf:"varint,2,opt,name=epoch_group_id,json=epochGroupId,proto3" json:"epoch_group_id,omitempty"`
	PocStartBlockHeight  int64                `protobuf:"varint,3,opt,name=poc_start_block_height,json=pocStartBlockHeight,proto3" json:"poc_start_block_height,omitempty"`
	EffectiveBlockHeight int64                `protobuf:"varint,4,opt,name=effective_block_height,json=effectiveBlockHeight,proto3" json:"effective_block_height,omitempty"`
	CreatedAtBlockHeight int64                `protobuf:"varint,5,opt,name=created_at_block_height,json=createdAtBlockHeight,proto3" json:"created_at_block_height,omitempty"`
	EpochId              uint64               `protobuf:"varint,6,opt,name=epoch_id,json=epochId,proto3" json:"epoch_id,omitempty"`
}

type ActiveParticipant struct {
	Index        string      `json:"index,omitempty"`
	ValidatorKey string      `json:"validator_key,omitempty"`
	Weight       int64       `json:"weight,omitempty"`
	InferenceUrl string      `json:"inference_url,omitempty"`
	Models       []string    `json:"models,omitempty"`
	Seed         *RandomSeed `json:"seed,omitempty"`
}

type RandomSeed struct {
	Participant string `protobuf:"bytes,1,opt,name=participant,proto3" json:"participant,omitempty"`
	BlockHeight int64  `protobuf:"varint,2,opt,name=block_height,json=blockHeight,proto3" json:"block_height,omitempty"`
	Signature   string `protobuf:"bytes,3,opt,name=signature,proto3" json:"signature,omitempty"`
}
