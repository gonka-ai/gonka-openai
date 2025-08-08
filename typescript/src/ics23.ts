import { createHash } from 'node:crypto';

// This module provides a minimal ICS23 verifier for our fixed proof shape.

// Lazy import to avoid heavy generated code for now; we parse the parts we need.
// In a follow-up, replace with protobufjs-generated static code for cosmos/ics23/v1/proofs.proto

export interface ProofOp {
  type: string;
  key: Uint8Array;
  data: Uint8Array;
}

function sha256(buf: Uint8Array): Uint8Array {
  return new Uint8Array(createHash('sha256').update(Buffer.from(buf)).digest());
}

function concat(...parts: Uint8Array[]): Uint8Array {
  const len = parts.reduce((a, b) => a + b.length, 0);
  const out = new Uint8Array(len);
  let offset = 0;
  for (const p of parts) {
    out.set(p, offset);
    offset += p.length;
  }
  return out;
}

function varintEncode(value: number): Uint8Array {
  // protobuf varint
  let v = value >>> 0;
  const out: number[] = [];
  while (v >= 0x80) {
    out.push((v & 0x7f) | 0x80);
    v >>>= 7;
  }
  out.push(v);
  return new Uint8Array(out);
}

// Minimal ExistenceProof decode: we only need leaf, key, value, path
interface LeafOp {
  hash: number; // 1 = SHA256
  prehashKey: number; // 0 = NO_HASH
  prehashValue: number; // 1 = SHA256
  length: number; // 1 = VAR_PROTO
  prefix: Uint8Array;
}

interface InnerOp {
  hash: number; // 1 = SHA256
  prefix: Uint8Array;
  suffix: Uint8Array;
}

interface ExistenceProof {
  key: Uint8Array;
  value: Uint8Array;
  leaf: LeafOp;
  path: InnerOp[];
}

class BinReader {
  buf: Uint8Array;
  pos: number;
  len: number;
  constructor(buf: Uint8Array) {
    this.buf = buf;
    this.pos = 0;
    this.len = buf.length;
  }
  uint32(): number {
    // protobuf varint decode
    let value = 0;
    let shift = 0;
    while (true) {
      if (this.pos >= this.len) throw new Error('varint overflow');
      const b = this.buf[this.pos++];
      value |= (b & 0x7f) << shift;
      if ((b & 0x80) === 0) break;
      shift += 7;
      if (shift > 35) throw new Error('varint too long');
    }
    return value >>> 0;
  }
  int32(): number {
    const u = this.uint32();
    return u | 0;
  }
  bytes(): Uint8Array {
    const len = this.uint32();
    const start = this.pos;
    const end = start + len;
    if (end > this.len) throw new Error('bytes out of range');
    this.pos = end;
    return this.buf.subarray(start, end);
  }
  skipType(wireType: number): void {
    switch (wireType) {
      case 0: // varint
        this.uint32();
        return;
      case 1: // 64-bit
        this.pos += 8;
        return;
      case 2: { // length-delimited
        const len = this.uint32();
        this.pos += len;
        return;
      }
      case 5: // 32-bit
        this.pos += 4;
        return;
      default:
        throw new Error(`unsupported wire type: ${wireType}`);
    }
  }
}

function decodeCommitmentProof(data: Uint8Array): ExistenceProof {
  // We expect proof.oneof = exist
  // Using protobufjs Reader with known field numbers per cosmos/ics23/v1/proofs.proto
  const r = new BinReader(data);
  // oneof proof -> field 1 = exist (ExistenceProof)
  let existBytes: Uint8Array | null = null;
  while (r.pos < r.len) {
    const tag = r.uint32();
    const field = tag >>> 3;
    if (field === 1) {
      const len = r.uint32();
      const start = r.pos;
      const end = start + len;
      if (end > r.len) throw new Error('exist proof out of range');
      existBytes = r.buf.subarray(start, end);
      r.pos = end;
    } else {
      // skip
      if ((tag & 7) === 2) {
        const len = r.uint32();
        r.pos += len;
      } else {
        // varint or fixed; consume conservatively
        r.skipType(tag & 7);
      }
    }
  }
  if (!existBytes) throw new Error('unsupported proof type');

  // Decode ExistenceProof
  const er = new BinReader(existBytes);
  const ex: ExistenceProof = { key: new Uint8Array(), value: new Uint8Array(), leaf: { hash: 0, prehashKey: 0, prehashValue: 0, length: 0, prefix: new Uint8Array() }, path: [] };
  while (er.pos < er.len) {
    const tag = er.uint32();
    const field = tag >>> 3;
    if (field === 1) {
      ex.key = er.bytes();
    } else if (field === 2) {
      ex.value = er.bytes();
    } else if (field === 3) {
      // LeafOp
      const len = er.uint32();
      const start = er.pos;
      const end = start + len;
      if (end > er.len) throw new Error('leaf out of range');
      const lr = new BinReader(er.buf.subarray(start, end));
      er.pos = end;
      const leaf: LeafOp = { hash: 0, prehashKey: 0, prehashValue: 0, length: 0, prefix: new Uint8Array() };
      while (lr.pos < lr.len) {
        const t = lr.uint32();
        const f = t >>> 3;
        if (f === 1) leaf.hash = lr.int32();
        else if (f === 2) leaf.prehashKey = lr.int32();
        else if (f === 3) leaf.prehashValue = lr.int32();
        else if (f === 4) leaf.length = lr.int32();
        else if (f === 5) leaf.prefix = lr.bytes();
        else lr.skipType(t & 7);
      }
      ex.leaf = leaf;
    } else if (field === 4) {
      // InnerOp repeated
      const len = er.uint32();
      const start = er.pos;
      const end = start + len;
      if (end > er.len) throw new Error('inner out of range');
      const ir = new BinReader(er.buf.subarray(start, end));
      er.pos = end;
      const inner: InnerOp = { hash: 0, prefix: new Uint8Array(), suffix: new Uint8Array() };
      while (ir.pos < ir.len) {
        const t = ir.uint32();
        const f = t >>> 3;
        if (f === 1) inner.hash = ir.int32();
        else if (f === 2) inner.prefix = ir.bytes();
        else if (f === 3) inner.suffix = ir.bytes();
        else ir.skipType(t & 7);
      }
      ex.path.push(inner);
    } else {
      er.skipType(tag & 7);
    }
  }
  return ex;
}

function applyLeafOp(leaf: LeafOp, key: Uint8Array, value: Uint8Array): Uint8Array {
  const hkey = leaf.prehashKey === 0 ? key : sha256(key);
  const hval = leaf.prehashValue === 0 ? value : sha256(value);
  const payload = concat(
    leaf.prefix || new Uint8Array(),
    varintEncode(hkey.length), hkey,
    varintEncode(hval.length), hval,
  );
  return sha256(payload);
}

function applyInnerOp(inner: InnerOp, child: Uint8Array): Uint8Array {
  const payload = concat(inner.prefix || new Uint8Array(), child, inner.suffix || new Uint8Array());
  return sha256(payload);
}

export function verifyIcs23(appHash: Uint8Array, proofOps: ProofOp[], value: Uint8Array): void {
  if (!appHash?.length) throw new Error('invalid app hash');
  if (!Array.isArray(proofOps) || proofOps.length !== 2) throw new Error('expected 2 proof ops');

  // IAVL
  const iavl = proofOps[0];
  if (iavl.type !== 'ics23:iavl') throw new Error(`unexpected first proof op type: ${iavl.type}`);
  const iavlCp = decodeCommitmentProof(iavl.data);
  if (Buffer.compare(Buffer.from(iavlCp.key), Buffer.from(iavl.key)) !== 0) throw new Error('IAVL proof key mismatch');
  if (Buffer.compare(Buffer.from(iavlCp.value), Buffer.from(value)) !== 0) throw new Error('IAVL proof value mismatch');
  let cur = applyLeafOp(iavlCp.leaf, iavlCp.key, iavlCp.value);
  for (const step of iavlCp.path) cur = applyInnerOp(step, cur);
  const storeRoot = cur;

  // Simple (multistore)
  const simple = proofOps[1];
  if (simple.type !== 'ics23:simple') throw new Error(`unexpected second proof op type: ${simple.type}`);
  const simpleCp = decodeCommitmentProof(simple.data);
  if (Buffer.compare(Buffer.from(simpleCp.key), Buffer.from(simple.key)) !== 0) throw new Error('simple proof key mismatch');
  if (Buffer.compare(Buffer.from(simpleCp.value), Buffer.from(storeRoot)) !== 0) throw new Error('simple proof store root mismatch');
  let root = applyLeafOp(simpleCp.leaf, simpleCp.key, simpleCp.value);
  for (const step of simpleCp.path) root = applyInnerOp(step, root);

  if (Buffer.compare(Buffer.from(root), Buffer.from(appHash)) !== 0) {
    throw new Error('simple proof does not match app hash');
  }
}


