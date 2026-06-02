import { useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";

import {
  applyNodeDeployConfig,
  deleteNodeDeployInbound,
  deleteNodeTLSTemplate,
  getNodeDeployDetail,
  getNodeTLSTemplates,
  regenerateNodeIdentity,
  rollbackNodeDeployConfig,
  saveNodeDeployInbound,
  saveNodeTLSTemplate,
} from "@/api";
import type {
  NodeApiItem,
  NodeDeployDetailApiItem,
  NodeDeployedInboundApiItem,
  NodeTLSTemplateApiItem,
} from "@/api/types";
import { Button } from "@/shadcn-bridge/heroui/button";
import { Chip } from "@/shadcn-bridge/heroui/chip";
import { Input, Textarea } from "@/shadcn-bridge/heroui/input";
import {
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
} from "@/shadcn-bridge/heroui/modal";
import { Select, SelectItem } from "@/shadcn-bridge/heroui/select";

const protocols = [
  "vless",
  "hysteria2",
  "trojan",
  "shadowsocks",
  "tuic",
  "socks",
  "http",
];

interface NodeDeployModalProps {
  node: NodeApiItem | null;
  isOpen: boolean;
  onClose: () => void;
}
const defaultTLS: Partial<NodeTLSTemplateApiItem> = {
  name: "",
  type: "tls",
  serverJson: JSON.stringify({ enabled: true, server_name: "" }, null, 2),
  clientJson: JSON.stringify(
    { enabled: true, server_name: "", insecure: false },
    null,
    2,
  ),
};

const defaultTLSForm = {
  sni: "",
  fingerprint: "chrome",
  certFile: "",
  keyFile: "",
  certContent: "",
  keyContent: "",
  insecure: false,
  realityHandshakeServer: "",
  realityHandshakePort: "443",
  realityPrivateKey: "",
  realityPublicKey: "",
  realityShortId: "",
};

const ensureOK = (res: { code: number; msg?: string }) => {
  if (res.code !== 0) {
    throw new Error(res.msg || "Request failed");
  }
};

const readOptions = (raw: string) => {
  try {
    const parsed = JSON.parse(raw || "{}");
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
};

const getOption = (raw: string, key: string, fallback = "") => {
  const options = readOptions(raw) as Record<string, any>;
  const value = options[key];
  return value == null ? fallback : String(value);
};

const getNestedOption = (
  raw: string,
  parent: string,
  key: string,
  fallback = "",
) => {
  const options = readOptions(raw) as Record<string, any>;
  const value = options[parent]?.[key];
  return value == null ? fallback : String(value);
};

const parseClient = (item: NodeDeployedInboundApiItem) => {
  try {
    return JSON.parse(item.clientConfigJson || "{}");
  } catch {
    return {};
  }
};

const asSingBoxOutbound = (item: NodeDeployedInboundApiItem) => {
  const client = parseClient(item);
  const outbound: Record<string, unknown> = {
    type: client.type || item.protocol,
    tag: `${item.displayName}-out`,
    server: client.server || item.publishAddr,
    server_port: client.server_port || item.publishPort,
  };
  if (client.uuid) outbound.uuid = client.uuid;
  if (client.password) outbound.password = client.password;
  if (client.tlsTemplate) outbound.tls = client.tlsTemplate;
  return JSON.stringify(outbound, null, 2);
};

const asMihomoProxy = (item: NodeDeployedInboundApiItem) => {
  const client = parseClient(item);
  const tls = client.tlsTemplate || {};
  const type = item.protocol === "shadowsocks" ? "ss" : item.protocol;
  const lines = [
    `name: ${item.displayName}`,
    `type: ${type}`,
    `server: ${client.server || item.publishAddr}`,
    `port: ${client.server_port || item.publishPort}`,
  ];

  if (item.protocol === "vless") lines.push(`uuid: ${client.uuid}`);
  if (item.protocol === "trojan") lines.push(`password: ${client.password}`);
  if (item.protocol === "hysteria2") lines.push(`password: ${client.password}`);
  if (item.protocol === "tuic") {
    lines.push(`uuid: ${client.uuid}`);
    lines.push(`password: ${client.password}`);
  }
  if (item.protocol === "shadowsocks") {
    lines.push("cipher: 2022-blake3-aes-128-gcm");
    lines.push(`password: ${client.password}`);
  }
  if (tls && Object.keys(tls).length > 0) {
    lines.push("tls: true");
    if (tls.server_name) lines.push(`servername: ${tls.server_name}`);
    if (tls.insecure) lines.push("skip-cert-verify: true");
    if (tls.reality?.enabled) {
      lines.push("reality-opts:");
      if (tls.reality.public_key) {
        lines.push(`  public-key: ${tls.reality.public_key}`);
      }
      if (tls.reality.short_id) {
        lines.push(`  short-id: ${tls.reality.short_id}`);
      }
    }
  }
  return lines.join("\n");
};

const qrAlignmentPatternPositions: number[][] = [
  [],
  [],
  [6, 18],
  [6, 22],
  [6, 26],
  [6, 30],
  [6, 34],
  [6, 22, 38],
  [6, 24, 42],
  [6, 26, 46],
  [6, 28, 50],
  [6, 30, 54],
  [6, 32, 58],
  [6, 34, 62],
  [6, 26, 46, 66],
  [6, 26, 48, 70],
  [6, 26, 50, 74],
  [6, 30, 54, 78],
  [6, 30, 56, 82],
  [6, 30, 58, 86],
  [6, 34, 62, 90],
];

const qrBlocksL: number[][] = [
  [],
  [1, 26, 19],
  [1, 44, 34],
  [1, 70, 55],
  [1, 100, 80],
  [1, 134, 108],
  [2, 86, 68],
  [2, 98, 78],
  [2, 121, 97],
  [2, 146, 116],
  [2, 86, 68, 2, 87, 69],
  [4, 101, 81],
  [2, 116, 92, 2, 117, 93],
  [4, 133, 107],
  [3, 145, 115, 1, 146, 116],
  [5, 109, 87, 1, 110, 88],
  [5, 122, 98, 1, 123, 99],
  [1, 135, 107, 5, 136, 108],
  [5, 150, 120, 1, 151, 121],
  [3, 141, 113, 4, 142, 114],
  [3, 135, 107, 5, 136, 108],
];

const qrGalois = (() => {
  const exp = new Array<number>(512).fill(0);
  const log = new Array<number>(256).fill(0);
  let x = 1;
  for (let i = 0; i < 255; i++) {
    exp[i] = x;
    log[x] = i;
    x <<= 1;
    if (x & 0x100) x ^= 0x11d;
  }
  for (let i = 255; i < 512; i++) exp[i] = exp[i - 255];
  return { exp, log };
})();

const qrGFMul = (a: number, b: number) =>
  a === 0 || b === 0 ? 0 : qrGalois.exp[qrGalois.log[a] + qrGalois.log[b]];

const qrPolyMul = (a: number[], b: number[]) => {
  const out = new Array<number>(a.length + b.length - 1).fill(0);
  for (let i = 0; i < a.length; i++) {
    for (let j = 0; j < b.length; j++) out[i + j] ^= qrGFMul(a[i], b[j]);
  }
  return out;
};

const qrECGenerator = (degree: number) => {
  let poly = [1];
  for (let i = 0; i < degree; i++) poly = qrPolyMul(poly, [1, qrGalois.exp[i]]);
  return poly;
};

const qrReedSolomon = (data: number[], ecCount: number) => {
  const generator = qrECGenerator(ecCount);
  const result = new Array<number>(ecCount).fill(0);
  for (const value of data) {
    const factor = value ^ result.shift()!;
    result.push(0);
    for (let i = 0; i < ecCount; i++) result[i] ^= qrGFMul(generator[i + 1], factor);
  }
  return result;
};

const qrBCH = (data: number, poly: number) => {
  let value = data;
  const polyLen = Math.floor(Math.log2(poly)) + 1;
  value <<= polyLen - 1;
  while (Math.floor(Math.log2(value)) + 1 >= polyLen) {
    value ^= poly << (Math.floor(Math.log2(value)) + 1 - polyLen);
  }
  return value;
};

const qrAppendBits = (bits: number[], value: number, length: number) => {
  for (let i = length - 1; i >= 0; i--) bits.push((value >>> i) & 1);
};

const createQRMatrix = (text: string) => {
  const bytes = Array.from(new TextEncoder().encode(text));
  let version = 1;
  let dataCodewords = 0;
  for (; version < qrBlocksL.length; version++) {
    const blocks = qrBlocksL[version];
    dataCodewords = 0;
    for (let i = 0; i < blocks.length; i += 3) dataCodewords += blocks[i] * blocks[i + 2];
    const lengthBits = version <= 9 ? 8 : 16;
    if (4 + lengthBits + bytes.length * 8 <= dataCodewords * 8) break;
  }
  if (version >= qrBlocksL.length) return null;

  const size = version * 4 + 17;
  const matrix = Array.from({ length: size }, () => new Array<boolean>(size).fill(false));
  const reserved = Array.from({ length: size }, () => new Array<boolean>(size).fill(false));
  const set = (r: number, c: number, dark: boolean, reserve = true) => {
    if (r < 0 || c < 0 || r >= size || c >= size) return;
    matrix[r][c] = dark;
    if (reserve) reserved[r][c] = true;
  };

  const finder = (r: number, c: number) => {
    for (let y = -1; y <= 7; y++) {
      for (let x = -1; x <= 7; x++) {
        const dark =
          x >= 0 &&
          x <= 6 &&
          y >= 0 &&
          y <= 6 &&
          (x === 0 || x === 6 || y === 0 || y === 6 || (x >= 2 && x <= 4 && y >= 2 && y <= 4));
        set(r + y, c + x, dark);
      }
    }
  };
  finder(0, 0);
  finder(0, size - 7);
  finder(size - 7, 0);

  for (let i = 8; i < size - 8; i++) {
    set(6, i, i % 2 === 0);
    set(i, 6, i % 2 === 0);
  }

  for (const r of qrAlignmentPatternPositions[version]) {
    for (const c of qrAlignmentPatternPositions[version]) {
      if (reserved[r][c]) continue;
      for (let y = -2; y <= 2; y++) {
        for (let x = -2; x <= 2; x++) set(r + y, c + x, Math.max(Math.abs(x), Math.abs(y)) !== 1);
      }
    }
  }

  for (let i = 0; i < 9; i++) {
    if (i !== 6) {
      set(8, i, false);
      set(i, 8, false);
    }
  }
  for (let i = 0; i < 8; i++) {
    set(size - 1 - i, 8, false);
    set(8, size - 1 - i, false);
  }
  set(size - 8, 8, true);

  if (version >= 7) {
    const versionBits = (version << 12) | qrBCH(version, 0x1f25);
    for (let i = 0; i < 18; i++) {
      const bit = ((versionBits >>> i) & 1) === 1;
      set(Math.floor(i / 3), size - 11 + (i % 3), bit);
      set(size - 11 + (i % 3), Math.floor(i / 3), bit);
    }
  }

  const bits: number[] = [];
  qrAppendBits(bits, 0b0100, 4);
  qrAppendBits(bits, bytes.length, version <= 9 ? 8 : 16);
  for (const b of bytes) qrAppendBits(bits, b, 8);
  qrAppendBits(bits, 0, Math.min(4, dataCodewords * 8 - bits.length));
  while (bits.length % 8 !== 0) bits.push(0);
  const data: number[] = [];
  for (let i = 0; i < bits.length; i += 8) data.push(bits.slice(i, i + 8).reduce((a, b) => (a << 1) | b, 0));
  for (let pad = 0; data.length < dataCodewords; pad ^= 1) data.push(pad ? 0x11 : 0xec);

  const dataBlocks: number[][] = [];
  const ecBlocks: number[][] = [];
  let offset = 0;
  const blocks = qrBlocksL[version];
  for (let i = 0; i < blocks.length; i += 3) {
    for (let j = 0; j < blocks[i]; j++) {
      const block = data.slice(offset, offset + blocks[i + 2]);
      offset += blocks[i + 2];
      dataBlocks.push(block);
      ecBlocks.push(qrReedSolomon(block, blocks[i + 1] - blocks[i + 2]));
    }
  }
  const codewords: number[] = [];
  for (let i = 0; i < Math.max(...dataBlocks.map((b) => b.length)); i++) for (const b of dataBlocks) if (i < b.length) codewords.push(b[i]);
  for (let i = 0; i < Math.max(...ecBlocks.map((b) => b.length)); i++) for (const b of ecBlocks) if (i < b.length) codewords.push(b[i]);

  const dataBits: number[] = [];
  for (const cw of codewords) qrAppendBits(dataBits, cw, 8);
  let bitIndex = 0;
  let upward = true;
  for (let col = size - 1; col > 0; col -= 2) {
    if (col === 6) col--;
    for (let i = 0; i < size; i++) {
      const row = upward ? size - 1 - i : i;
      for (let dx = 0; dx < 2; dx++) {
        const c = col - dx;
        if (reserved[row][c]) continue;
        const bit = bitIndex < dataBits.length ? dataBits[bitIndex++] === 1 : false;
        matrix[row][c] = bit !== ((row + c) % 2 === 0);
      }
    }
    upward = !upward;
  }

  const formatBits = ((((1 << 3) | 0) << 10) | qrBCH((1 << 3) | 0, 0x537)) ^ 0x5412;
  const coords1 = [[8, 0], [8, 1], [8, 2], [8, 3], [8, 4], [8, 5], [8, 7], [8, 8], [7, 8], [5, 8], [4, 8], [3, 8], [2, 8], [1, 8], [0, 8]];
  const coords2 = [[size - 1, 8], [size - 2, 8], [size - 3, 8], [size - 4, 8], [size - 5, 8], [size - 6, 8], [size - 7, 8], [8, size - 8], [8, size - 7], [8, size - 6], [8, size - 5], [8, size - 4], [8, size - 3], [8, size - 2], [8, size - 1]];
  coords1.forEach(([r, c], i) => set(r, c, ((formatBits >>> i) & 1) === 1));
  coords2.forEach(([r, c], i) => set(r, c, ((formatBits >>> i) & 1) === 1));
  return matrix;
};

function OfflineQRCode({ value }: { value: string }) {
  const matrix = useMemo(() => createQRMatrix(value), [value]);
  if (!matrix) {
    return <div className="rounded-md border border-warning-300 bg-warning-50 p-3 text-xs text-warning-700">QR payload is too long for the offline renderer. Copy the QR payload instead.</div>;
  }
  const quiet = 4;
  const size = matrix.length + quiet * 2;
  const cells: JSX.Element[] = [];
  matrix.forEach((row, r) =>
    row.forEach((dark, c) => {
      if (dark) cells.push(<rect key={`${r}-${c}`} height="1" width="1" x={c + quiet} y={r + quiet} />);
    }),
  );
  return (
    <svg aria-label="Offline QR code" className="h-52 w-52 rounded bg-white p-2" shapeRendering="crispEdges" viewBox={`0 0 ${size} ${size}`}>
      <rect fill="white" height={size} width={size} x="0" y="0" />
      <g fill="black">{cells}</g>
    </svg>
  );
}

export function NodeDeployModal({
  node,
  isOpen,
  onClose,
}: NodeDeployModalProps) {
  const [detail, setDetail] = useState<NodeDeployDetailApiItem | null>(null);
  const [tlsTemplates, setTlsTemplates] = useState<NodeTLSTemplateApiItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [tlsDraft, setTlsDraft] =
    useState<Partial<NodeTLSTemplateApiItem>>(defaultTLS);
  const [tlsForm, setTLSForm] = useState(defaultTLSForm);
  const [qrPayload, setQRPayload] = useState("");
  const [form, setForm] = useState({
    id: 0,
    name: "",
    protocol: "vless",
    listenAddr: "127.0.0.1",
    listenPort: 0,
    publishAddr: "",
    publishPort: 0,
    tlsTemplateId: 0,
    inboundOptionsJson: "{}",
    apply: true,
  });

  const activeRevision = detail?.revisions?.[0];
  const nodeName = String(node?.name || "");
  const publishDefault = String(
    (node as any)?.serverIpV4 ||
      (node as any)?.serverIp ||
      (node as any)?.serverIpV6 ||
      (node as any)?.ip ||
      "",
  );

  const generatedName = useMemo(() => {
    if (!nodeName) return "";
    return `${nodeName}-${form.protocol.toUpperCase()}`;
  }, [nodeName, form.protocol]);

  const load = async () => {
    if (!node?.id) return;
    setLoading(true);
    try {
      const [detailRes, tlsRes] = await Promise.all([
        getNodeDeployDetail(node.id),
        getNodeTLSTemplates(),
      ]);
      ensureOK(detailRes);
      ensureOK(tlsRes);
      setDetail(detailRes.data);
      setTlsTemplates(tlsRes.data || []);
      setForm((prev) => {
        const port = prev.listenPort || Math.floor(Math.random() * 40000) + 10000;
        return {
          ...prev,
          publishAddr: prev.publishAddr || publishDefault,
          listenPort: port,
          publishPort: prev.publishPort || port,
        };
      });
    } catch (err: any) {
      toast.error(err?.message || "Failed to load node deployment");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen) void load();
  }, [isOpen, node?.id]);

  useEffect(() => {
    if (!isOpen) return;
    const nextPort = Math.floor(Math.random() * 40000) + 10000;
    setForm((prev) => ({
      ...prev,
      id: 0,
      name: "",
      listenAddr: "127.0.0.1",
      listenPort: nextPort,
      publishAddr: publishDefault,
      publishPort: nextPort,
      tlsTemplateId: 0,
      inboundOptionsJson: "{}",
      apply: true,
    }));
    setQRPayload("");
  }, [isOpen, node?.id, publishDefault]);

  const copyText = async (text: string, label: string) => {
    if (!text) {
      toast.error("Nothing to copy");
      return;
    }
    await navigator.clipboard.writeText(text);
    toast.success(`${label} copied`);
  };

  const updateOption = (key: string, value: string | number) => {
    const options = readOptions(form.inboundOptionsJson) as Record<string, any>;
    if (value === "" || value == null) {
      delete options[key];
    } else {
      options[key] = value;
    }
    setForm((prev) => ({
      ...prev,
      inboundOptionsJson: JSON.stringify(options, null, 2),
    }));
  };

  const updateNestedOption = (
    parent: string,
    key: string,
    value: string | number,
  ) => {
    const options = readOptions(form.inboundOptionsJson) as Record<string, any>;
    const next =
      options[parent] && typeof options[parent] === "object"
        ? options[parent]
        : {};
    if (value === "" || value == null) {
      delete next[key];
    } else {
      next[key] = value;
    }
    if (Object.keys(next).length === 0) {
      delete options[parent];
    } else {
      options[parent] = next;
    }
    setForm((prev) => ({
      ...prev,
      inboundOptionsJson: JSON.stringify(options, null, 2),
    }));
  };

  const saveTLS = async () => {
    if (!tlsDraft.name?.trim()) {
      toast.error("TLS template name is required");
      return;
    }
    try {
      ensureOK(await saveNodeTLSTemplate(tlsDraft));
      toast.success("TLS template saved");
      setTlsDraft(defaultTLS);
      const res = await getNodeTLSTemplates();
      ensureOK(res);
      setTlsTemplates(res.data || []);
    } catch (err: any) {
      toast.error(err?.message || "Failed to save TLS template");
    }
  };

  const saveInbound = async () => {
    if (!node?.id) return;
    setSaving(true);
    try {
      const res = await saveNodeDeployInbound({
        ...form,
        nodeId: node.id,
        name: form.name.trim(),
        listenPort: Number(form.listenPort),
        publishPort: Number(form.publishPort || form.listenPort),
        tlsTemplateId: Number(form.tlsTemplateId || 0),
      });
      ensureOK(res);
      toast.success(form.apply ? "Inbound saved and deployed" : "Inbound saved");
      await load();
      setForm((prev) => ({
        ...prev,
        id: 0,
        name: "",
        listenPort: 0,
        publishPort: 0,
      }));
    } catch (err: any) {
      toast.error(err?.message || "Failed to save inbound");
    } finally {
      setSaving(false);
    }
  };

  const editInbound = (item: NodeDeployedInboundApiItem) => {
    setForm({
      id: item.id,
      name: item.displayName,
      protocol: item.protocol,
      listenAddr: item.listenAddr,
      listenPort: item.listenPort,
      publishAddr: item.publishAddr,
      publishPort: item.publishPort,
      tlsTemplateId: Number(item.tlsTemplateId || 0),
      inboundOptionsJson: item.inboundOptionsJson || "{}",
      apply: true,
    });
  };

  const deleteInbound = async (item: NodeDeployedInboundApiItem) => {
    if (!confirm(`Delete ${item.displayName}?`)) return;
    try {
      ensureOK(await deleteNodeDeployInbound(item.id));
      toast.success("Inbound deleted and config redeployed");
      await load();
    } catch (err: any) {
      toast.error(err?.message || "Delete failed");
    }
  };

  const applyConfig = async () => {
    if (!node?.id) return;
    try {
      ensureOK(await applyNodeDeployConfig(node.id));
      toast.success("Config redeployed");
      await load();
    } catch (err: any) {
      toast.error(err?.message || "Deploy failed");
    }
  };

  const rollbackConfig = async (revisionId: number) => {
    if (!node?.id) return;
    if (!confirm("Rollback to this config revision?")) return;
    try {
      ensureOK(await rollbackNodeDeployConfig(node.id, revisionId));
      toast.success("Config rolled back");
      await load();
    } catch (err: any) {
      toast.error(err?.message || "Rollback failed");
    }
  };

  const regenerateIdentity = async () => {
    if (!node?.id) return;
    if (!confirm("Regenerate this node identity? Existing links will change.")) {
      return;
    }
    try {
      ensureOK(await regenerateNodeIdentity(node.id));
      toast.success("Node identity regenerated");
      await load();
    } catch (err: any) {
      toast.error(err?.message || "Regenerate failed");
    }
  };

  const applyTLSFormToDraft = () => {
    const type = String(tlsDraft.type || "tls");
    const server: Record<string, any> = { enabled: true };
    const client: Record<string, any> = { enabled: true };

    if (tlsForm.sni) {
      server.server_name = tlsForm.sni;
      client.server_name = tlsForm.sni;
    }
    if (tlsForm.fingerprint) {
      client.utls = { enabled: true, fingerprint: tlsForm.fingerprint };
      client.utls_fingerprint = tlsForm.fingerprint;
    }
    if (tlsForm.certFile) server.certificate_path = tlsForm.certFile;
    if (tlsForm.keyFile) server.key_path = tlsForm.keyFile;
    if (tlsForm.certContent) server.certificate = tlsForm.certContent;
    if (tlsForm.keyContent) server.key = tlsForm.keyContent;
    if (tlsForm.insecure) client.insecure = true;

    if (type === "reality") {
      server.reality = {
        enabled: true,
        handshake: {
          server: tlsForm.realityHandshakeServer,
          server_port: Number(tlsForm.realityHandshakePort || 443),
        },
        private_key: tlsForm.realityPrivateKey,
        short_id: tlsForm.realityShortId
          ? tlsForm.realityShortId.split(",").map((v) => v.trim()).filter(Boolean)
          : undefined,
      };
      client.reality = {
        enabled: true,
        public_key: tlsForm.realityPublicKey,
        short_id: tlsForm.realityShortId.split(",")[0]?.trim() || "",
      };
    }

    setTlsDraft((prev) => ({
      ...prev,
      serverJson: JSON.stringify(server, null, 2),
      clientJson: JSON.stringify(client, null, 2),
    }));
  };

  return (
    <Modal isOpen={isOpen} scrollBehavior="inside" size="4xl" onClose={onClose}>
      <ModalContent>
        <ModalHeader>
          <div>
            <div className="text-lg font-semibold">Node Deploy</div>
            <div className="text-xs text-default-500">{nodeName}</div>
          </div>
        </ModalHeader>
        <ModalBody>
          {loading ? (
            <div className="py-10 text-center text-default-500">Loading...</div>
          ) : (
            <div className="grid gap-4 lg:grid-cols-[1.15fr_0.85fr]">
              <section className="space-y-4">
                <div className="rounded-lg border border-divider p-4">
                  <div className="mb-3 flex items-center justify-between">
                    <h3 className="text-sm font-semibold">Deploy Inbound</h3>
                    <Chip size="sm" variant="flat">
                      Default: {generatedName}
                    </Chip>
                  </div>
                  <div className="grid gap-3 md:grid-cols-2">
                    <Input
                      label="Name"
                      placeholder={generatedName}
                      value={form.name}
                      onChange={(e) =>
                        setForm((p) => ({ ...p, name: e.target.value }))
                      }
                    />
                    <Select
                      label="Protocol"
                      selectedKeys={new Set([form.protocol])}
                      onSelectionChange={(keys) =>
                        setForm((p) => ({
                          ...p,
                          protocol: String(Array.from(keys)[0] || "vless"),
                        }))
                      }
                    >
                      {protocols.map((p) => (
                        <SelectItem key={p}>{p.toUpperCase()}</SelectItem>
                      ))}
                    </Select>
                    <Input
                      label="Listen address"
                      value={form.listenAddr}
                      onChange={(e) =>
                        setForm((p) => ({ ...p, listenAddr: e.target.value }))
                      }
                    />
                    <Input
                      label="Listen port"
                      type="number"
                      value={String(form.listenPort || "")}
                      onChange={(e) =>
                        setForm((p) => ({ ...p, listenPort: Number(e.target.value) }))
                      }
                    />
                    <Input
                      label="Publish address"
                      value={form.publishAddr}
                      onChange={(e) =>
                        setForm((p) => ({ ...p, publishAddr: e.target.value }))
                      }
                    />
                    <Input
                      label="Publish port"
                      type="number"
                      value={String(form.publishPort || "")}
                      onChange={(e) =>
                        setForm((p) => ({ ...p, publishPort: Number(e.target.value) }))
                      }
                    />
                    <Select
                      label="TLS template"
                      selectedKeys={new Set([String(form.tlsTemplateId || 0)])}
                      onSelectionChange={(keys) =>
                        setForm((p) => ({
                          ...p,
                          tlsTemplateId: Number(Array.from(keys)[0] || 0),
                        }))
                      }
                    >
                      <SelectItem key="0">No TLS</SelectItem>
                      {tlsTemplates.map((tpl) => (
                        <SelectItem key={String(tpl.id)}>{tpl.name}</SelectItem>
                      ))}
                    </Select>
                  </div>

                  <div className="mt-3 rounded-md border border-divider/70 p-3">
                    <div className="mb-3 text-xs font-semibold text-default-600">
                      Protocol fields
                    </div>
                    <div className="grid gap-3 md:grid-cols-2">
                      {form.protocol === "vless" && (
                        <>
                          <Input
                            label="Flow"
                            placeholder="xtls-rprx-vision"
                            value={getOption(form.inboundOptionsJson, "flow")}
                            onChange={(e) => updateOption("flow", e.target.value)}
                          />
                          <Select
                            label="Transport"
                            selectedKeys={
                              new Set([
                                getNestedOption(
                                  form.inboundOptionsJson,
                                  "transport",
                                  "type",
                                  "tcp",
                                ),
                              ])
                            }
                            onSelectionChange={(keys) =>
                              updateNestedOption(
                                "transport",
                                "type",
                                String(Array.from(keys)[0] || "tcp"),
                              )
                            }
                          >
                            <SelectItem key="tcp">TCP</SelectItem>
                            <SelectItem key="ws">WebSocket</SelectItem>
                            <SelectItem key="http">HTTP</SelectItem>
                            <SelectItem key="grpc">gRPC</SelectItem>
                          </Select>
                          <Input
                            label="Transport path"
                            value={getNestedOption(
                              form.inboundOptionsJson,
                              "transport",
                              "path",
                            )}
                            onChange={(e) =>
                              updateNestedOption("transport", "path", e.target.value)
                            }
                          />
                        </>
                      )}
                      {form.protocol === "hysteria2" && (
                        <>
                          <Input
                            label="Up Mbps"
                            type="number"
                            value={getOption(
                              form.inboundOptionsJson,
                              "up_mbps",
                              "100",
                            )}
                            onChange={(e) =>
                              updateOption("up_mbps", Number(e.target.value))
                            }
                          />
                          <Input
                            label="Down Mbps"
                            type="number"
                            value={getOption(
                              form.inboundOptionsJson,
                              "down_mbps",
                              "100",
                            )}
                            onChange={(e) =>
                              updateOption("down_mbps", Number(e.target.value))
                            }
                          />
                          <Input
                            label="Obfs password"
                            value={getNestedOption(
                              form.inboundOptionsJson,
                              "obfs",
                              "password",
                            )}
                            onChange={(e) =>
                              updateNestedOption("obfs", "password", e.target.value)
                            }
                          />
                        </>
                      )}
                      {form.protocol === "shadowsocks" && (
                        <Select
                          label="Method"
                          selectedKeys={
                            new Set([
                              getOption(
                                form.inboundOptionsJson,
                                "method",
                                "2022-blake3-aes-128-gcm",
                              ),
                            ])
                          }
                          onSelectionChange={(keys) =>
                            updateOption(
                              "method",
                              String(
                                Array.from(keys)[0] ||
                                  "2022-blake3-aes-128-gcm",
                              ),
                            )
                          }
                        >
                          <SelectItem key="2022-blake3-aes-128-gcm">
                            2022-blake3-aes-128-gcm
                          </SelectItem>
                          <SelectItem key="2022-blake3-aes-256-gcm">
                            2022-blake3-aes-256-gcm
                          </SelectItem>
                          <SelectItem key="aes-128-gcm">aes-128-gcm</SelectItem>
                          <SelectItem key="aes-256-gcm">aes-256-gcm</SelectItem>
                          <SelectItem key="chacha20-ietf-poly1305">
                            chacha20-ietf-poly1305
                          </SelectItem>
                        </Select>
                      )}
                      {form.protocol === "tuic" && (
                        <Select
                          label="Congestion control"
                          selectedKeys={
                            new Set([
                              getOption(
                                form.inboundOptionsJson,
                                "congestion_control",
                                "cubic",
                              ),
                            ])
                          }
                          onSelectionChange={(keys) =>
                            updateOption(
                              "congestion_control",
                              String(Array.from(keys)[0] || "cubic"),
                            )
                          }
                        >
                          <SelectItem key="cubic">cubic</SelectItem>
                          <SelectItem key="bbr">bbr</SelectItem>
                          <SelectItem key="new_reno">new_reno</SelectItem>
                        </Select>
                      )}
                      {["trojan", "socks", "http"].includes(form.protocol) && (
                        <div className="text-xs text-default-500">
                          This protocol uses the node-bound credential by default.
                        </div>
                      )}
                    </div>
                  </div>

                  <Textarea
                    className="mt-3"
                    label="Advanced options JSON"
                    minRows={5}
                    value={form.inboundOptionsJson}
                    onChange={(e) =>
                      setForm((p) => ({ ...p, inboundOptionsJson: e.target.value }))
                    }
                  />
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Button color="primary" isLoading={saving} onPress={saveInbound}>
                      Save and Deploy
                    </Button>
                    <Button
                      variant="flat"
                      onPress={() => setForm((p) => ({ ...p, apply: !p.apply }))}
                    >
                      {form.apply ? "Mode: save + deploy" : "Mode: save only"}
                    </Button>
                    <Button variant="flat" onPress={applyConfig}>
                      Redeploy Config
                    </Button>
                  </div>
                </div>

                <div className="rounded-lg border border-divider p-4">
                  <h3 className="mb-3 text-sm font-semibold">TLS Template</h3>
                  <div className="grid gap-3 md:grid-cols-2">
                    <Input
                      label="Name"
                      value={tlsDraft.name || ""}
                      onChange={(e) =>
                        setTlsDraft((p) => ({ ...p, name: e.target.value }))
                      }
                    />
                    <Select
                      label="Type"
                      selectedKeys={new Set([tlsDraft.type || "tls"])}
                      onSelectionChange={(keys) =>
                        setTlsDraft((p) => ({
                          ...p,
                          type: String(Array.from(keys)[0] || "tls"),
                        }))
                      }
                    >
                      <SelectItem key="tls">TLS</SelectItem>
                      <SelectItem key="reality">Reality</SelectItem>
                    </Select>
                  </div>
                  <div className="mt-3 rounded-md border border-divider/70 p-3">
                    <div className="mb-3 text-xs font-semibold text-default-600">
                      TLS form
                    </div>
                    <div className="grid gap-3 md:grid-cols-2">
                      <Input
                        label="SNI / server name"
                        value={tlsForm.sni}
                        onChange={(e) =>
                          setTLSForm((p) => ({ ...p, sni: e.target.value }))
                        }
                      />
                      <Select
                        label="Fingerprint"
                        selectedKeys={new Set([tlsForm.fingerprint])}
                        onSelectionChange={(keys) =>
                          setTLSForm((p) => ({
                            ...p,
                            fingerprint: String(Array.from(keys)[0] || "chrome"),
                          }))
                        }
                      >
                        <SelectItem key="chrome">Chrome</SelectItem>
                        <SelectItem key="firefox">Firefox</SelectItem>
                        <SelectItem key="safari">Safari</SelectItem>
                        <SelectItem key="edge">Edge</SelectItem>
                        <SelectItem key="random">Random</SelectItem>
                      </Select>
                      <Input
                        label="Certificate file"
                        value={tlsForm.certFile}
                        onChange={(e) =>
                          setTLSForm((p) => ({ ...p, certFile: e.target.value }))
                        }
                      />
                      <Input
                        label="Private key file"
                        value={tlsForm.keyFile}
                        onChange={(e) =>
                          setTLSForm((p) => ({ ...p, keyFile: e.target.value }))
                        }
                      />
                      {tlsDraft.type === "reality" && (
                        <>
                          <Input
                            label="Reality handshake server"
                            value={tlsForm.realityHandshakeServer}
                            onChange={(e) =>
                              setTLSForm((p) => ({
                                ...p,
                                realityHandshakeServer: e.target.value,
                              }))
                            }
                          />
                          <Input
                            label="Reality handshake port"
                            type="number"
                            value={tlsForm.realityHandshakePort}
                            onChange={(e) =>
                              setTLSForm((p) => ({
                                ...p,
                                realityHandshakePort: e.target.value,
                              }))
                            }
                          />
                          <Input
                            label="Reality private key"
                            value={tlsForm.realityPrivateKey}
                            onChange={(e) =>
                              setTLSForm((p) => ({
                                ...p,
                                realityPrivateKey: e.target.value,
                              }))
                            }
                          />
                          <Input
                            label="Reality public key"
                            value={tlsForm.realityPublicKey}
                            onChange={(e) =>
                              setTLSForm((p) => ({
                                ...p,
                                realityPublicKey: e.target.value,
                              }))
                            }
                          />
                          <Input
                            label="Reality short IDs"
                            value={tlsForm.realityShortId}
                            onChange={(e) =>
                              setTLSForm((p) => ({
                                ...p,
                                realityShortId: e.target.value,
                              }))
                            }
                          />
                        </>
                      )}
                    </div>
                    <div className="mt-3 grid gap-3 md:grid-cols-2">
                      <Textarea
                        label="Certificate content"
                        minRows={3}
                        value={tlsForm.certContent}
                        onChange={(e) =>
                          setTLSForm((p) => ({ ...p, certContent: e.target.value }))
                        }
                      />
                      <Textarea
                        label="Private key content"
                        minRows={3}
                        value={tlsForm.keyContent}
                        onChange={(e) =>
                          setTLSForm((p) => ({ ...p, keyContent: e.target.value }))
                        }
                      />
                    </div>
                    <div className="mt-3 flex gap-2">
                      <Button size="sm" variant="flat" onPress={applyTLSFormToDraft}>
                        Generate JSON from form
                      </Button>
                      <Button
                        size="sm"
                        variant="flat"
                        onPress={() =>
                          setTLSForm((p) => ({ ...p, insecure: !p.insecure }))
                        }
                      >
                        {tlsForm.insecure ? "Insecure: on" : "Insecure: off"}
                      </Button>
                    </div>
                  </div>
                  <Textarea
                    className="mt-3"
                    label="Server JSON"
                    minRows={4}
                    value={tlsDraft.serverJson || ""}
                    onChange={(e) =>
                      setTlsDraft((p) => ({ ...p, serverJson: e.target.value }))
                    }
                  />
                  <Textarea
                    className="mt-3"
                    label="Client JSON"
                    minRows={4}
                    value={tlsDraft.clientJson || ""}
                    onChange={(e) =>
                      setTlsDraft((p) => ({ ...p, clientJson: e.target.value }))
                    }
                  />
                  <Button
                    className="mt-3"
                    color="secondary"
                    variant="flat"
                    onPress={saveTLS}
                  >
                    Save TLS Template
                  </Button>
                </div>
              </section>

              <aside className="space-y-4">
                <div className="rounded-lg border border-divider p-4">
                  <h3 className="mb-3 text-sm font-semibold">Node Identity</h3>
                  <div className="space-y-2 text-xs">
                    <div>UUID: {detail?.identity?.uuid}</div>
                    <div>Hysteria2: {detail?.identity?.hysteria2Password}</div>
                    <div>Reality Short ID: {detail?.identity?.realityShortId}</div>
                  </div>
                  <Button
                    className="mt-3"
                    color="warning"
                    size="sm"
                    variant="flat"
                    onPress={regenerateIdentity}
                  >
                    Regenerate identity
                  </Button>
                </div>

                <div className="rounded-lg border border-divider p-4">
                  <h3 className="mb-3 text-sm font-semibold">Deployed Inbounds</h3>
                  <div className="space-y-3">
                    {(detail?.inbounds || []).map((item) => (
                      <div
                        key={item.id}
                        className="rounded-md border border-divider/70 p-3"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div>
                            <div className="text-sm font-medium">
                              {item.displayName}
                            </div>
                            <div className="text-xs text-default-500">
                              {item.protocol} {item.publishAddr}:{item.publishPort}
                            </div>
                          </div>
                          <Chip size="sm" variant="flat">
                            {item.enabled === 1 ? "Enabled" : "Disabled"}
                          </Chip>
                        </div>
                        <div className="mt-3 flex flex-wrap gap-2">
                          <Button
                            size="sm"
                            variant="flat"
                            onPress={() => copyText(item.shareUri, "Share URI")}
                          >
                            Copy URI
                          </Button>
                          <Button
                            size="sm"
                            variant="flat"
                            onPress={() =>
                              copyText(item.clientConfigJson, "Client JSON")
                            }
                          >
                            Copy JSON
                          </Button>
                          <Button
                            size="sm"
                            variant="flat"
                            onPress={() =>
                              copyText(asSingBoxOutbound(item), "sing-box outbound")
                            }
                          >
                            Copy Outbound
                          </Button>
                          <Button
                            size="sm"
                            variant="flat"
                            onPress={() =>
                              copyText(asMihomoProxy(item), "Mihomo proxy")
                            }
                          >
                            Copy Mihomo
                          </Button>
                          <Button
                            size="sm"
                            variant="flat"
                            onPress={() => copyText(item.shareUri, "QR payload")}
                          >
                            Copy QR payload
                          </Button>
                          <Button
                            size="sm"
                            variant="flat"
                            onPress={() => setQRPayload(item.shareUri)}
                          >
                            Show QR
                          </Button>
                          <Button
                            size="sm"
                            variant="flat"
                            onPress={() => editInbound(item)}
                          >
                            Edit
                          </Button>
                          <Button
                            color="danger"
                            size="sm"
                            variant="flat"
                            onPress={() => deleteInbound(item)}
                          >
                            Delete
                          </Button>
                        </div>
                      </div>
                    ))}
                    {(detail?.inbounds || []).length === 0 && (
                      <div className="text-sm text-default-500">
                        No deployed inbound yet.
                      </div>
                    )}
                  </div>
                </div>

                <div className="rounded-lg border border-divider p-4">
                  <h3 className="mb-3 text-sm font-semibold">Config Preview</h3>
                  <pre className="max-h-72 overflow-auto rounded-md bg-default-100 p-3 text-xs">
                    {activeRevision?.configJson || "Save an inbound to render config."}
                  </pre>
                </div>

                {qrPayload && (
                  <div className="rounded-lg border border-divider p-4">
                    <div className="mb-3 flex items-center justify-between">
                      <h3 className="text-sm font-semibold">QR Preview</h3>
                      <Button size="sm" variant="light" onPress={() => setQRPayload("")}>
                        Hide
                      </Button>
                    </div>
                    <div className="flex justify-center rounded-md bg-white p-4">
                      <OfflineQRCode value={qrPayload} />
                    </div>
                    <div className="mt-2 break-all text-xs text-default-500">
                      {qrPayload}
                    </div>
                  </div>
                )}

                <div className="rounded-lg border border-divider p-4">
                  <h3 className="mb-3 text-sm font-semibold">Config Revisions</h3>
                  <div className="space-y-2">
                    {(detail?.revisions || []).slice(0, 5).map((rev) => (
                      <div
                        key={rev.id}
                        className="rounded-md bg-default-100 px-3 py-2 text-xs"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <span>
                            #{rev.id} - {rev.status} - {rev.coreType}
                          </span>
                          <Button
                            size="sm"
                            variant="light"
                            onPress={() => rollbackConfig(rev.id)}
                          >
                            Rollback
                          </Button>
                        </div>
                        <div className="mt-1 truncate text-default-500">
                          {rev.checksum}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="rounded-lg border border-divider p-4">
                  <h3 className="mb-3 text-sm font-semibold">Deploy Logs</h3>
                  <div className="space-y-2">
                    {(detail?.logs || []).slice(0, 6).map((log) => (
                      <div
                        key={log.id}
                        className="rounded-md bg-default-100 px-3 py-2 text-xs"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <span>
                            {log.action} - {log.status}
                          </span>
                          <span className="text-default-500">
                            #{log.revisionId}
                          </span>
                        </div>
                        {log.message && (
                          <div className="mt-1 line-clamp-2 text-default-500">
                            {log.message}
                          </div>
                        )}
                      </div>
                    ))}
                    {(detail?.logs || []).length === 0 && (
                      <div className="text-sm text-default-500">
                        No deploy logs yet.
                      </div>
                    )}
                  </div>
                </div>

                <div className="rounded-lg border border-divider p-4">
                  <h3 className="mb-3 text-sm font-semibold">TLS Templates</h3>
                  <div className="space-y-2">
                    {tlsTemplates.map((tpl) => (
                      <div
                        key={tpl.id}
                        className="flex items-center justify-between rounded-md bg-default-100 px-3 py-2 text-sm"
                      >
                        <span>
                          {tpl.name} - {tpl.type}
                        </span>
                        <Button
                          color="danger"
                          size="sm"
                          variant="light"
                          onPress={async () => {
                            ensureOK(await deleteNodeTLSTemplate(tpl.id));
                            const res = await getNodeTLSTemplates();
                            ensureOK(res);
                            setTlsTemplates(res.data || []);
                          }}
                        >
                          Delete
                        </Button>
                      </div>
                    ))}
                  </div>
                </div>
              </aside>
            </div>
          )}
        </ModalBody>
        <ModalFooter>
          <Button variant="flat" onPress={onClose}>
            Close
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
