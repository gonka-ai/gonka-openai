from __future__ import annotations

from dataclasses import dataclass, field
from typing import List, Optional, Any, Dict, Union


@dataclass
class RandomSeed:
    participant: Optional[str] = None
    block_height: Optional[int] = None
    signature: Optional[str] = None


@dataclass
class ActiveParticipant:
    index: Optional[str] = None
    validator_key: Optional[str] = None
    weight: Optional[int] = None
    inference_url: Optional[str] = None
    models: List[str] = field(default_factory=list)
    seed: Optional[RandomSeed] = None


@dataclass
class ActiveParticipants:
    participants: List[ActiveParticipant] = field(default_factory=list)
    epoch_group_id: Optional[int] = None
    poc_start_block_height: Optional[int] = None
    effective_block_height: Optional[int] = None
    created_at_block_height: Optional[int] = None
    epoch_id: Optional[int] = None


@dataclass
class Validator:
    address: Optional[str] = None
    pub_key: Optional[str] = None
    voting_power: Optional[int] = None
    proposer_priority: Optional[int] = None


@dataclass
class ProofOp:
    type: str
    key: bytes
    data: bytes


@dataclass
class ProofOps:
    ops: List[ProofOp]


@dataclass
class Block:
    app_hash: bytes


@dataclass
class ActiveParticipantWithProof:
    active_participants: ActiveParticipants
    addresses: List[str]
    active_participants_bytes: str
    proof_ops: ProofOps
    validators: List[Validator]
    block: Block


def _from_dict_proof_ops(d: Any) -> ProofOps:
    def is_op_dict(x: Any) -> bool:
        if not isinstance(x, dict):
            return False
        keys = {k.lower() for k in x.keys()}
        return ("type" in keys) and ("data" in keys)

    def find_ops_list(obj: Any) -> List[Dict[str, Any]]:
        # Direct list of ops
        if isinstance(obj, list):
            if obj and all(isinstance(e, dict) for e in obj) and any(is_op_dict(e) for e in obj):
                return obj  # looks like a list of ops
            # search inside
            for e in obj:
                res = find_ops_list(e)
                if res:
                    return res
            return []
        # Dict: try common keys, then recurse into values
        if isinstance(obj, dict):
            for key in ("ops", "Ops", "proof_ops", "proofOps", "ProofOps"):
                val = obj.get(key)
                if isinstance(val, list) and val:
                    if any(is_op_dict(e) for e in val):
                        return val
            # search nested dict values
            for v in obj.values():
                res = find_ops_list(v)
                if res:
                    return res
        return []

    ops_in = find_ops_list(d) or []
    ops: List[ProofOp] = []
    for op in ops_in:
        # cometbft json encodes bytes as base64 strings
        t = op.get("type") if "type" in op else op.get("Type", "")
        k_raw = op.get("key") if "key" in op else op.get("Key", "")
        d_raw = op.get("data") if "data" in op else op.get("Data", "")
        if isinstance(k_raw, str):
            import base64

            key_bytes = base64.b64decode(k_raw)
        else:
            key_bytes = bytes(k_raw or b"")
        if isinstance(d_raw, str):
            import base64

            data_bytes = base64.b64decode(d_raw)
        else:
            data_bytes = bytes(d_raw or b"")
        ops.append(ProofOp(type=t, key=key_bytes, data=data_bytes))
    return ProofOps(ops=ops)


def active_participant_with_proof_from_dict(d: Dict[str, Any]) -> ActiveParticipantWithProof:
    ap_dict = d.get("active_participants", {}) or {}
    parts_in = ap_dict.get("participants", []) or []
    participants: List[ActiveParticipant] = []
    for p in parts_in:
        participants.append(
            ActiveParticipant(
                index=p.get("index"),
                validator_key=p.get("validator_key"),
                weight=p.get("weight"),
                inference_url=p.get("inference_url"),
                models=p.get("models", []) or [],
                seed=(
                    RandomSeed(
                        participant=p.get("seed", {}).get("participant"),
                        block_height=p.get("seed", {}).get("block_height"),
                        signature=p.get("seed", {}).get("signature"),
                    )
                    if p.get("seed")
                    else None
                ),
            )
        )

    active_participants = ActiveParticipants(
        participants=participants,
        epoch_group_id=ap_dict.get("epoch_group_id"),
        poc_start_block_height=ap_dict.get("poc_start_block_height"),
        effective_block_height=ap_dict.get("effective_block_height"),
        created_at_block_height=ap_dict.get("created_at_block_height"),
        epoch_id=ap_dict.get("epoch_id"),
    )

    import base64, binascii

    def _extract_app_hash(block_obj: Union[Dict[str, Any], List[Any], None]) -> bytes:
        if not block_obj:
            return b""
        # If it's a dict, try direct and header nesting
        if isinstance(block_obj, dict):
            val = block_obj.get("app_hash")
            if val is None and isinstance(block_obj.get("header"), dict):
                val = block_obj["header"].get("app_hash")
            if isinstance(val, str) and val:
                # Detect hex first (common for app_hash), else treat as base64
                v = val.strip()
                is_hex = all(c in "0123456789abcdefABCDEF" for c in v) and (len(v) % 2 == 0)
                if is_hex:
                    try:
                        return bytes.fromhex(v)
                    except ValueError:
                        pass
                try:
                    return base64.b64decode(v)
                except binascii.Error:
                    return b""
            return b""
        # If it's a list, scan for dicts containing header/app_hash
        if isinstance(block_obj, list):
            for item in block_obj:
                if isinstance(item, dict):
                    ah = _extract_app_hash(item)
                    if ah:
                        return ah
        return b""

    block = Block(app_hash=_extract_app_hash(d.get("block")))

    validators_in = d.get("validators", []) or []
    validators = [
        Validator(
            address=v.get("address"),
            pub_key=v.get("pub_key"),
            voting_power=v.get("voting_power"),
            proposer_priority=v.get("proposer_priority"),
        )
        for v in validators_in
    ]

    # proof_ops could be nested or raw list in top-level response
    proof_ops_src: Any = d.get("proof_ops") or d.get("proofOps") or d.get("ProofOps") or {}

    return ActiveParticipantWithProof(
        active_participants=active_participants,
        addresses=d.get("addresses", []) or [],
        active_participants_bytes=d.get("active_participants_bytes", "") or "",
        proof_ops=_from_dict_proof_ops(proof_ops_src),
        validators=validators,
        block=block,
    )


