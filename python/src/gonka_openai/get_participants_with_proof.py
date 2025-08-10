from __future__ import annotations

import binascii
import json
from typing import List
import hashlib
import os

import requests

from .entities import (
    ActiveParticipantWithProof,
    active_participant_with_proof_from_dict,
)
from .utils import Endpoint
from .ics23.cosmos.ics23.v1 import proofs_pb2


class InvalidEpoch(ValueError):
    pass


def _normalize_base_url(url: str) -> str:
    return url[:-1] if url.endswith("/") else url


def get_participants_with_proof(
    base_url: str, epoch: str
) -> List[Endpoint]:
    if not epoch:
        raise InvalidEpoch("epoch must be non-empty")

    # Allow using a local JSON file for testing: base_url starts with file://
    if base_url.startswith("file://"):
        file_path = base_url[len("file://"):]
        if not os.path.isabs(file_path):
            file_path = os.path.abspath(file_path)
        with open(file_path, "r", encoding="utf-8") as f:
            payload = json.load(f)
    else:
        base_url = _normalize_base_url(base_url or "")
        url = f"{base_url}/v1/epochs/{epoch}/participants"

        resp = requests.get(url, headers={"Content-Type": "application/json"}, timeout=30)
        if resp.status_code != 200:
            raise RuntimeError(f"failed to fetch participants with proof: status code {resp.status_code}")

        try:
            payload = resp.json()
        except json.JSONDecodeError as e:
            raise RuntimeError(f"failed to decode response: {e}") from e
    return _process_payload(payload)


def get_participants_with_proof_from_file(file_path: str) -> List[Endpoint]:
    if not os.path.isabs(file_path):
        file_path = os.path.abspath(file_path)
    with open(file_path, "r", encoding="utf-8") as f:
        payload = json.load(f)
    return _process_payload(payload)


def _process_payload(payload: dict) -> List[Endpoint]:
    # Env toggle: if GONKA_VERIFY_PROOF=0, skip ICS23 verification and do light parsing
    verify = os.environ.get("GONKA_VERIFY_PROOF") == "1"

    def _ensure_v1(url: str) -> str:
        if not url:
            return url
        u = url[:-1] if url.endswith('/') else url
        return u if u.endswith('/v1') else f"{u}/v1"

    if not verify:
        participants = (payload or {}).get("active_participants", {}).get("participants", [])
        endpoints: List[Endpoint] = []
        for p in participants:
            url = (p or {}).get("inference_url")
            addr = (p or {}).get("index")
            if not url or not addr:
                continue
            endpoints.append(Endpoint(url=_ensure_v1(url), address=addr))
        return endpoints

    # Full parse + verification
    apwp: ActiveParticipantWithProof = active_participant_with_proof_from_dict(payload)

    try:
        participants_bytes = binascii.unhexlify(apwp.active_participants_bytes)
    except binascii.Error as e:
        raise RuntimeError(f"failed to decode participants bytes: {e}") from e

    # Verify proof against app hash. Minimal Python implementation for our fixed two-op shape.
    proof_ops = apwp.proof_ops.ops
    if not proof_ops:
        proof_ops = _find_proof_ops_in_payload(payload)
    _verify_iavl_proof_against_app_hash(
        app_hash=apwp.block.app_hash,
        proof_ops=proof_ops,
        value=participants_bytes,
    )

    # Map to endpoints
    endpoints: List[Endpoint] = []
    for participant in apwp.active_participants.participants:
        if not participant or not participant.inference_url or not participant.index:
            # Skip invalid entries silently
            continue
        endpoints.append(Endpoint(url=_ensure_v1(participant.inference_url), address=participant.index))

    return endpoints


def _verify_iavl_proof_against_app_hash(app_hash: bytes, proof_ops, value: bytes) -> None:
    if not isinstance(app_hash, (bytes, bytearray)) or len(app_hash) == 0:
        raise RuntimeError("invalid app hash in proof")

    ops = list(proof_ops or [])
    if len(ops) != 2:
        raise RuntimeError(f"expected 2 proof ops, got {len(ops)}")

    # 1) Verify IAVL op
    iavl_op = ops[0]
    if iavl_op.type != "ics23:iavl":
        raise RuntimeError(f"unexpected first proof op type: {iavl_op.type}")

    if not iavl_op.data:
        raise RuntimeError("IAVL proof op has empty data")
    if not isinstance(iavl_op.key, (bytes, bytearray)):
        raise RuntimeError("IAVL proof op has invalid key")

    # Parse and compute store root from IAVL op
    iavl_cp = proofs_pb2.CommitmentProof.FromString(iavl_op.data)
    iavl_exist = _extract_existence(iavl_cp)
    # Ensure the proof binds the provided key/value
    if bytes(iavl_exist.key) != bytes(iavl_op.key):
        raise RuntimeError("IAVL proof key mismatch")
    if bytes(iavl_exist.value) != bytes(value):
        raise RuntimeError("IAVL proof value mismatch")
    store_root = _calculate_root_from_existence(iavl_exist)

    # 2) Verify Simple (multistore) op links store root to app hash
    simple_op = ops[1]
    if simple_op.type != "ics23:simple":
        raise RuntimeError(f"unexpected second proof op type: {simple_op.type}")
    if not simple_op.data:
        raise RuntimeError("simple proof op has empty data")
    if not isinstance(simple_op.key, (bytes, bytearray)):
        raise RuntimeError("simple proof op has invalid key")

    simple_cp = proofs_pb2.CommitmentProof.FromString(simple_op.data)
    simple_exist = _extract_existence(simple_cp)
    # The simple proof proves inclusion of store_root under the provided key into app_hash
    if bytes(simple_exist.key) != bytes(simple_op.key):
        raise RuntimeError("simple proof key mismatch")
    if bytes(simple_exist.value) != bytes(store_root):
        raise RuntimeError("simple proof value (store root) mismatch")
    computed_app_hash = _calculate_root_from_existence(simple_exist)
    if bytes(computed_app_hash) != bytes(app_hash):
        raise RuntimeError("simple proof does not match app hash")


def _find_proof_ops_in_payload(payload: dict):
    from .entities import _from_dict_proof_ops
    return _from_dict_proof_ops(payload).ops


def _extract_existence(cp: proofs_pb2.CommitmentProof) -> proofs_pb2.ExistenceProof:
    # Support only ExistenceProof variant for now
    which = cp.WhichOneof("proof")
    if which != "exist":
        raise RuntimeError(f"unsupported commitment proof type: {which}")
    return cp.exist


def _hash_bytes(op: int, data: bytes) -> bytes:
    # Support minimal set used by Tendermint/IAVL
    if op == proofs_pb2.SHA256:
        return hashlib.sha256(data).digest()
    if op == proofs_pb2.NO_HASH:
        return data
    raise RuntimeError(f"unsupported hash op: {op}")


def _len_prefix(op: int, data: bytes) -> bytes:
    if op == proofs_pb2.NO_PREFIX:
        return b""
    if op == proofs_pb2.VAR_PROTO:
        return _encode_varint(len(data))
    # Add others if needed
    raise RuntimeError(f"unsupported length op: {op}")


def _encode_varint(value: int) -> bytes:
    # Protobuf varint
    out = bytearray()
    v = int(value)
    while True:
        b = v & 0x7F
        v >>= 7
        if v:
            out.append(b | 0x80)
        else:
            out.append(b)
            break
    return bytes(out)


def _apply_leaf_op(leaf: proofs_pb2.LeafOp, key: bytes, value: bytes) -> bytes:
    # Prehash support: handle NO_HASH and SHA256 (common for IAVL)
    if leaf.prehash_key not in (proofs_pb2.NO_HASH, proofs_pb2.SHA256):
        raise RuntimeError(f"unsupported prehash_key: {leaf.prehash_key}")
    if leaf.prehash_value not in (proofs_pb2.NO_HASH, proofs_pb2.SHA256):
        raise RuntimeError(f"unsupported prehash_value: {leaf.prehash_value}")

    hkey = key if leaf.prehash_key == proofs_pb2.NO_HASH else _hash_bytes(leaf.prehash_key, key)
    hval = value if leaf.prehash_value == proofs_pb2.NO_HASH else _hash_bytes(leaf.prehash_value, value)

    # Construct payload: prefix || len(hkey) || hkey || len(hval) || hval
    payload = bytes(leaf.prefix or b"")
    payload += _len_prefix(leaf.length, hkey) + hkey
    payload += _len_prefix(leaf.length, hval) + hval
    return _hash_bytes(leaf.hash, payload)


def _apply_inner_op(inner: proofs_pb2.InnerOp, child: bytes) -> bytes:
    payload = bytes(inner.prefix or b"") + child + bytes(inner.suffix or b"")
    return _hash_bytes(inner.hash, payload)


def _calculate_root_from_existence(ex: proofs_pb2.ExistenceProof) -> bytes:
    cur = _apply_leaf_op(ex.leaf, bytes(ex.key), bytes(ex.value))
    for step in ex.path:
        cur = _apply_inner_op(step, cur)
    return cur


