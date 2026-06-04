import { useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
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
import { ListChecks, PlusCircle } from "lucide-react";
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
import { Progress } from "@/shadcn-bridge/heroui/progress";
import { Select, SelectItem } from "@/shadcn-bridge/heroui/select";
import { Switch } from "@/shadcn-bridge/heroui/switch";
import { getAdminFlag } from "@/utils/session";

const protocols = [
  "vless",
  "hysteria2",
  "trojan",
  "shadowsocks",
  "tuic",
  "socks",
  "http",
];

const vlessFlowOptions = [
  { key: "none", label: "none", value: "" },
  {
    key: "xtls-rprx-vision",
    label: "xtls-rprx-vision",
    value: "xtls-rprx-vision",
  },
  {
    key: "xtls-rprx-vision-udp443",
    label: "xtls-rprx-vision-udp443",
    value: "xtls-rprx-vision-udp443",
  },
];

const fingerprintOptions = [
  "chrome",
  "firefox",
  "safari",
  "ios",
  "android",
  "edge",
  "360",
  "qq",
  "random",
  "randomized",
];

type DeployView = "menu" | "add" | "manage";
type DeployProgressState = {
  active: boolean;
  color: "primary" | "success" | "danger";
  label: string;
  percent: number;
};

const idleDeployProgress: DeployProgressState = {
  active: false,
  color: "primary",
  label: "",
  percent: 0,
};
const DEPLOY_POLL_INTERVAL_MS = 20 * 1000;
const DEPLOY_POLL_MAX_ATTEMPTS = 30;

type NodeInboundSaveResponse = {
  deployStatus?: string;
  revision?: {
    id?: number;
    status?: string;
    errorMessage?: string;
  };
};

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

interface NodeDeployModalProps {
  node: NodeApiItem | null;
  isOpen: boolean;
  onClose: () => void;
}
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

const nullableID = (value: unknown) => {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (value && typeof value === "object") {
    const raw = value as { Int64?: unknown; Valid?: unknown };
    if (raw.Valid === false) {
      return 0;
    }
    const parsed = Number(raw.Int64);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  const parsed = Number(value || 0);
  return Number.isFinite(parsed) ? parsed : 0;
};

const sanitizeOutbound = (raw: Record<string, any>, protocol: string) => {
  const outbound: Record<string, any> = { ...raw };
  if (outbound.tlsTemplate && !outbound.tls) {
    outbound.tls = outbound.tlsTemplate;
  }
  delete outbound.tlsTemplate;
  switch (protocol) {
    case "vless":
      delete outbound.password;
      delete outbound.username;
      delete outbound.method;
      break;
    case "trojan":
    case "hysteria2":
      delete outbound.uuid;
      delete outbound.username;
      delete outbound.method;
      break;
    case "tuic":
      delete outbound.username;
      delete outbound.method;
      break;
    case "shadowsocks":
      delete outbound.uuid;
      delete outbound.username;
      break;
    case "socks":
    case "http":
    case "mixed":
      delete outbound.uuid;
      delete outbound.method;
      break;
  }
  if (!outbound.tls || Object.keys(outbound.tls || {}).length === 0) {
    delete outbound.tls;
  }
  return outbound;
};

const asSingBoxOutbound = (item: NodeDeployedInboundApiItem) => {
  const client = parseClient(item);
  if (client.type && client.server && client.server_port) {
    const outbound = sanitizeOutbound(client, item.protocol);
    return JSON.stringify(outbound, null, 2);
  }
  const outbound: Record<string, unknown> = {
    type: client.type || item.protocol,
    tag: `${item.displayName}-out`,
    server: client.server || item.publishAddr,
    server_port: client.server_port || item.publishPort,
  };
  if (client.uuid) outbound.uuid = client.uuid;
  if (client.password) outbound.password = client.password;
  if (client.flow) outbound.flow = client.flow;
  if (client.network) outbound.network = client.network;
  if (client.packet_encoding) outbound.packet_encoding = client.packet_encoding;
  if (client.tls || client.tlsTemplate) outbound.tls = client.tls || client.tlsTemplate;
  if (client.multiplex) outbound.multiplex = client.multiplex;
  if (client.transport) outbound.transport = client.transport;
  return JSON.stringify(sanitizeOutbound(outbound, item.protocol), null, 2);
};

const asSingBoxClientConfig = (item: NodeDeployedInboundApiItem) =>
  JSON.stringify(
    {
      log: { level: "info" },
      outbounds: [
        JSON.parse(asSingBoxOutbound(item)),
        { type: "direct", tag: "direct" },
      ],
      route: {
        final: JSON.parse(asSingBoxOutbound(item)).tag || item.displayName,
      },
    },
    null,
    2,
  );

const asMihomoProxy = (item: NodeDeployedInboundApiItem) => {
  const client = parseClient(item);
  const tls = client.tls || client.tlsTemplate || {};
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
  const [view, setView] = useState<DeployView>("menu");
  const [inboundSide, setInboundSide] = useState<"server" | "client">("server");
  const [showInboundOptions, setShowInboundOptions] = useState(false);
  const [qrPayload, setQRPayload] = useState("");
  const [manualCopy, setManualCopy] = useState<{
    label: string;
    text: string;
  } | null>(null);
  const [deployProgress, setDeployProgress] =
    useState<DeployProgressState>(idleDeployProgress);
  const isAdmin = getAdminFlag();
  const [form, setForm] = useState({
    id: 0,
    name: "",
    protocol: "vless",
    listenAddr: "::",
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
    setDeployProgress(idleDeployProgress);
    setForm((prev) => ({
      ...prev,
      id: 0,
      name: "",
      listenAddr: "::",
      listenPort: nextPort,
      publishAddr: publishDefault,
      publishPort: nextPort,
      tlsTemplateId: 0,
      inboundOptionsJson: "{}",
      apply: true,
    }));
    setQRPayload("");
  }, [isOpen, node?.id, publishDefault]);

  useEffect(() => {
    if (!deployProgress.active || deployProgress.color !== "primary") return;
    const timer = window.setInterval(() => {
      setDeployProgress((prev) => {
        if (!prev.active || prev.color !== "primary") return prev;
        const cap = prev.percent >= 70 ? 92 : 72;
        const step = prev.percent < 35 ? 6 : 3;
        return { ...prev, percent: Math.min(cap, prev.percent + step) };
      });
    }, 900);
    return () => window.clearInterval(timer);
  }, [deployProgress.active, deployProgress.color]);

  const updateDeployProgress = (
    percent: number,
    label: string,
    color: DeployProgressState["color"] = "primary",
  ) => {
    setDeployProgress({ active: true, color, label, percent });
  };

  const finishDeployProgress = (
    label: string,
    color: DeployProgressState["color"],
    autoHide = color === "success",
  ) => {
    setDeployProgress({ active: true, color, label, percent: 100 });
    if (autoHide) {
      window.setTimeout(() => setDeployProgress(idleDeployProgress), 1200);
    }
  };

  const waitForDeployRevision = async (
    revisionId: number,
    actionLabel: string,
  ) => {
    if (!node?.id || revisionId <= 0) return;
    for (let attempt = 1; attempt <= DEPLOY_POLL_MAX_ATTEMPTS; attempt += 1) {
      updateDeployProgress(
        Math.min(94, 40 + attempt * 3),
        `${actionLabel}: 等待节点执行，${DEPLOY_POLL_INTERVAL_MS / 1000} 秒后第 ${attempt} 次确认`,
      );
      await new Promise((resolve) =>
        window.setTimeout(resolve, DEPLOY_POLL_INTERVAL_MS),
      );
      const detailRes = await getNodeDeployDetail(node.id);
      ensureOK(detailRes);
      setDetail(detailRes.data);
      const revision = (detailRes.data?.revisions || []).find(
        (item) => item.id === revisionId,
      );
      if (!revision) {
        updateDeployProgress(
          Math.min(95, 45 + attempt * 3),
          `${actionLabel}: 已确认面板响应，等待部署版本写入`,
        );
        continue;
      }
      if (revision.status === "deployed") {
        finishDeployProgress(`${actionLabel}: 节点已确认部署完成`, "success");
        return;
      }
      if (revision.status === "failed") {
        throw new Error(revision.errorMessage || "节点部署失败");
      }
      updateDeployProgress(
        Math.min(95, 45 + attempt * 3),
        `${actionLabel}: 第 ${attempt} 次确认，当前状态 ${revision.status || "deploying"}`,
      );
    }
    throw new Error("节点部署仍在执行，已停止自动等待，请稍后在管理入站中查看部署日志");
  };

  const copyText = async (text: string, label: string) => {
    if (!text) {
      toast.error("没有可复制的内容");
      return;
    }
    setManualCopy({ label, text });
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text);
        toast.success(`${label} 已复制，内容已显示`);
        return;
      } else {
        throw new Error("clipboard api unavailable");
      }
    } catch {
      try {
        const textarea = document.createElement("textarea");
        textarea.value = text;
        textarea.setAttribute("readonly", "true");
        textarea.style.position = "fixed";
        textarea.style.left = "-9999px";
        textarea.style.top = "0";
        document.body.appendChild(textarea);
        textarea.focus();
        textarea.select();
        const ok = document.execCommand("copy");
        document.body.removeChild(textarea);
        if (ok) {
          toast.success(`${label} 已复制，内容已显示`);
          return;
        }
      } catch {
        // Fall through to the manual copy dialog.
      }
    }
    toast.error(`${label} 自动复制失败，内容已显示，请手动复制`);
  };

  const updateOption = (key: string, value: string | number) => {
    setForm((prev) => ({
      ...prev,
      inboundOptionsJson: JSON.stringify(
        (() => {
          const options = readOptions(prev.inboundOptionsJson) as Record<
            string,
            any
          >;
          if (value === "" || value == null) {
            delete options[key];
          } else {
            options[key] = value;
          }
          return options;
        })(),
        null,
        2,
      ),
    }));
  };

  const updateNestedOption = (
    parent: string,
    key: string,
    value: string | number | boolean,
  ) => {
    setForm((prev) => ({
      ...prev,
      inboundOptionsJson: JSON.stringify(
        (() => {
          const options = readOptions(prev.inboundOptionsJson) as Record<
            string,
            any
          >;
          const current =
            options[parent] && typeof options[parent] === "object"
              ? options[parent]
              : {};
          const next = { ...current };
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
          return options;
        })(),
        null,
        2,
      ),
    }));
  };

  const replaceNestedOption = (parent: string, value: Record<string, any>) => {
    setForm((prev) => ({
      ...prev,
      inboundOptionsJson: JSON.stringify(
        (() => {
          const options = readOptions(prev.inboundOptionsJson) as Record<
            string,
            any
          >;
          if (Object.keys(value).length === 0) {
            delete options[parent];
          } else {
            options[parent] = { ...value };
          }
          return options;
        })(),
        null,
        2,
      ),
    }));
  };

  const getNestedBool = (parent: string, key: string) => {
    const options = readOptions(form.inboundOptionsJson) as Record<string, any>;
    return Boolean(options[parent]?.[key]);
  };

  const setMultiplexEnabled = (enabled: boolean) => {
    replaceNestedOption(
      "multiplex",
      enabled
        ? {
            enabled: true,
            padding: false,
            protocol: "smux",
            max_connections: 4,
          }
        : {},
    );
  };

  const setTransportEnabled = (enabled: boolean) => {
    replaceNestedOption("transport", enabled ? { type: "tcp" } : {});
  };

  const saveTLS = async () => {
    if (!isAdmin) {
      toast.error("Only administrators can modify TLS templates");
      return;
    }
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
    const actionLabel = form.id > 0 ? "更新入站" : "部署入站";
    setSaving(true);
    updateDeployProgress(8, `${actionLabel}: 校验表单并准备配置`);
    try {
      updateDeployProgress(26, `${actionLabel}: 提交到面板生成 sing-box 配置`);
      const res = await saveNodeDeployInbound({
        ...form,
        nodeId: node.id,
        name: form.name.trim(),
        listenPort: Number(form.listenPort),
        publishPort: Number(form.publishPort || form.listenPort),
        tlsTemplateId: Number(form.tlsTemplateId || 0),
      });
      ensureOK(res);
      const payload = res.data as NodeInboundSaveResponse;
      const revisionId = Number(payload?.revision?.id || 0);
      if (form.apply && revisionId > 0) {
        updateDeployProgress(
          36,
          `${actionLabel}: 已启动后台下发，开始每 ${DEPLOY_POLL_INTERVAL_MS / 1000} 秒确认结果`,
        );
        await waitForDeployRevision(revisionId, actionLabel);
        toast.success(form.id > 0 ? "入站已更新并下发" : "入站已部署");
      } else {
        updateDeployProgress(86, `${actionLabel}: 配置已保存，正在刷新结果`);
        await load();
        finishDeployProgress(`${actionLabel}: 完成`, "success");
        toast.success(form.id > 0 ? "入站已更新" : "入站已保存");
      }
      setForm((prev) => ({
        ...prev,
        id: 0,
        name: "",
        listenPort: 0,
        publishPort: 0,
        apply: true,
      }));
      setView("manage");
    } catch (err: any) {
      finishDeployProgress(
        `${actionLabel}: 失败 - ${err?.message || "请求失败"}`,
        "danger",
        false,
      );
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
      tlsTemplateId: nullableID(item.tlsTemplateId),
      inboundOptionsJson: item.inboundOptionsJson || "{}",
      apply: true,
    });
    setView("add");
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

  const resetInboundForm = () => {
    const port = Math.floor(Math.random() * 40000) + 10000;
    setForm({
      id: 0,
      name: "",
      protocol: "vless",
      listenAddr: "::",
      listenPort: port,
      publishAddr: publishDefault,
      publishPort: port,
      tlsTemplateId: 0,
      inboundOptionsJson: "{}",
      apply: true,
    });
    setView("add");
  };

  const SectionTitle = ({ children }: { children: ReactNode }) => (
    <div className="flex items-center gap-3 py-2 text-xs text-default-500">
      <div className="h-px flex-1 bg-divider" />
      <span className="shrink-0">{children}</span>
      <div className="h-px flex-1 bg-divider" />
    </div>
  );

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

  if (view === "menu") {
    return (
      <Modal isOpen={isOpen} scrollBehavior="inside" size="2xl" onClose={onClose}>
        <ModalContent>
          <ModalBody className="py-8">
            <div className="space-y-8">
              <div>
                <h2 className="text-xl font-semibold text-gray-950 dark:text-gray-50">
                  入站管理
                </h2>
                <p className="mt-2 text-sm text-default-500">
                  欢迎回来。请选择您想要执行的操作以继续管理物流入库流程。
                </p>
              </div>
              {loading ? (
                <div className="rounded-md border border-divider p-8 text-center text-sm text-default-500">
                  正在加载节点部署信息
                </div>
              ) : (
                <div className="space-y-5">
                  <button
                    className="group relative flex min-h-[154px] w-full items-center overflow-hidden rounded-md border border-divider bg-white px-6 py-5 text-left shadow-sm transition hover:border-blue-300 hover:shadow-md dark:bg-gray-950"
                    type="button"
                    onClick={resetInboundForm}
                  >
                    <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-md bg-blue-100 text-blue-600">
                      <PlusCircle className="h-6 w-6" />
                    </div>
                    <div className="ml-5 max-w-[70%]">
                      <div className="text-base font-semibold">添加入站</div>
                      <div className="mt-3 text-sm leading-6 text-default-500">
                        创建新的入站物流配置。定义资源、预期到港时间及初步库存详情，以优化您的入库计划。
                      </div>
                    </div>
                    <PlusCircle className="absolute -right-8 bottom-0 h-32 w-32 text-gray-100 transition group-hover:text-blue-50 dark:text-gray-900" />
                  </button>
                  <button
                    className="group relative flex min-h-[154px] w-full items-center overflow-hidden rounded-md border border-divider bg-white px-6 py-5 text-left shadow-sm transition hover:border-blue-300 hover:shadow-md dark:bg-gray-950"
                    type="button"
                    onClick={() => setView("manage")}
                  >
                    <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-md bg-blue-100 text-slate-700">
                      <ListChecks className="h-6 w-6" />
                    </div>
                    <div className="ml-5 max-w-[70%]">
                      <div className="text-base font-semibold">管理入站</div>
                      <div className="mt-3 text-sm leading-6 text-default-500">
                        跟踪并修改现有的入站记录。查看实时状态、更新物流信息或协调仓库接收队列。
                      </div>
                    </div>
                    <ListChecks className="absolute -right-8 bottom-0 h-32 w-32 text-gray-100 transition group-hover:text-blue-50 dark:text-gray-900" />
                  </button>
                </div>
              )}
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="flat" onPress={onClose}>
              关闭
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    );
  }

  if (view === "add") {
    return (
      <Modal isOpen={isOpen} scrollBehavior="inside" size="4xl" onClose={onClose}>
        <ModalContent>
          <ModalHeader>添加 入站</ModalHeader>
          <ModalBody>
            <div className="space-y-5">
              <div className="grid gap-3 md:grid-cols-2">
                <Select
                  label="类型"
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
                  label="标签"
                  placeholder={generatedName || "vless-43811"}
                  value={form.name}
                  onChange={(e) =>
                    setForm((p) => ({ ...p, name: e.target.value }))
                  }
                />
              </div>

              <div className="grid grid-cols-2 border-b border-divider text-center text-sm">
                <button
                  className={`py-3 font-medium ${
                    inboundSide === "server"
                      ? "border-b-2 border-black text-black dark:border-white dark:text-white"
                      : "text-default-500"
                  }`}
                  type="button"
                  onClick={() => setInboundSide("server")}
                >
                  服务端
                </button>
                <button
                  className={`py-3 font-medium ${
                    inboundSide === "client"
                      ? "border-b-2 border-black text-black dark:border-white dark:text-white"
                      : "text-default-500"
                  }`}
                  type="button"
                  onClick={() => setInboundSide("client")}
                >
                  客户端
                </button>
              </div>

              {inboundSide === "client" ? (
                <div className="space-y-5">
                  <SectionTitle>目标</SectionTitle>
                  <div className="grid gap-3 md:grid-cols-2">
                    <Input
                      label="服务器地址"
                      value={form.publishAddr}
                      onChange={(e) =>
                        setForm((p) => ({ ...p, publishAddr: e.target.value }))
                      }
                    />
                    <Input
                      label="服务器端口"
                      type="number"
                      value={String(form.publishPort || "")}
                      onChange={(e) =>
                        setForm((p) => ({
                          ...p,
                          publishPort: Number(e.target.value),
                        }))
                      }
                    />
                  </div>
                  <SectionTitle>TLS</SectionTitle>
                  <Select
                    label="模板"
                    selectedKeys={new Set([String(form.tlsTemplateId || 0)])}
                    onSelectionChange={(keys) =>
                      setForm((p) => ({
                        ...p,
                        tlsTemplateId: Number(Array.from(keys)[0] || 0),
                      }))
                    }
                  >
                    <SelectItem key="0">无</SelectItem>
                    {tlsTemplates.map((tpl) => (
                      <SelectItem key={String(tpl.id)}>{tpl.name}</SelectItem>
                    ))}
                  </Select>
                  <SectionTitle>{form.protocol}</SectionTitle>
                  {form.protocol === "vless" ? (
                    <Select
                      label="Flow"
                      selectedKeys={
                        new Set([
                          getOption(form.inboundOptionsJson, "flow", "") ||
                            "none",
                        ])
                      }
                      onSelectionChange={(keys) => {
                        const key = String(Array.from(keys)[0] || "none");
                        const option = vlessFlowOptions.find(
                          (item) => item.key === key,
                        );
                        updateOption("flow", option?.value || "");
                      }}
                    >
                      {vlessFlowOptions.map((item) => (
                        <SelectItem key={item.key}>{item.label}</SelectItem>
                      ))}
                    </Select>
                  ) : null}
                  <div className="grid gap-3 md:grid-cols-2">
                    <Select
                      label="网络"
                      selectedKeys={
                        new Set([
                          getOption(form.inboundOptionsJson, "network", "all"),
                        ])
                      }
                      onSelectionChange={(keys) =>
                        updateOption(
                          "network",
                          String(Array.from(keys)[0] || "all") === "all"
                            ? ""
                            : String(Array.from(keys)[0]),
                        )
                      }
                    >
                      <SelectItem key="all">TCP/UDP</SelectItem>
                      <SelectItem key="tcp">TCP</SelectItem>
                      <SelectItem key="udp">UDP</SelectItem>
                    </Select>
                    <Select
                      label="UDP 数据包编码"
                      selectedKeys={
                        new Set([
                          getOption(
                            form.inboundOptionsJson,
                            "packet_encoding",
                            "none",
                          ),
                        ])
                      }
                      onSelectionChange={(keys) => {
                        const value = String(Array.from(keys)[0] || "none");
                        updateOption(
                          "packet_encoding",
                          value === "none" ? "" : value,
                        );
                      }}
                    >
                      <SelectItem key="none">none</SelectItem>
                      <SelectItem key="packetaddr">packetaddr</SelectItem>
                      <SelectItem key="xudp">xudp</SelectItem>
                    </Select>
                  </div>
                  <SectionTitle>出站预览</SectionTitle>
                  <pre className="max-h-64 overflow-auto rounded-md bg-default-100 p-3 text-xs">
                    {JSON.stringify(
                      {
                        type: form.protocol,
                        tag: `${form.name || generatedName || form.protocol}-out`,
                        server: form.publishAddr || publishDefault,
                        server_port: form.publishPort || form.listenPort,
                        flow: (readOptions(form.inboundOptionsJson) as any).flow,
                        tls:
                          form.tlsTemplateId > 0
                            ? tlsTemplates.find(
                                (tpl) => tpl.id === form.tlsTemplateId,
                              )?.name || true
                            : undefined,
                        network: (readOptions(form.inboundOptionsJson) as any)
                          .network,
                        packet_encoding: (readOptions(form.inboundOptionsJson) as any)
                          .packet_encoding,
                        multiplex: (readOptions(form.inboundOptionsJson) as any)
                          .multiplex,
                      },
                      null,
                      2,
                    )}
                  </pre>
                </div>
              ) : (
                <>
              <SectionTitle>监听</SectionTitle>
              <div className="grid items-end gap-3 md:grid-cols-[1fr_0.65fr_auto]">
                <Input
                  label="地址"
                  value={form.listenAddr}
                  onChange={(e) =>
                    setForm((p) => ({ ...p, listenAddr: e.target.value }))
                  }
                />
                <Input
                  label="端口"
                  type="number"
                  value={String(form.listenPort || "")}
                  onChange={(e) =>
                    setForm((p) => ({ ...p, listenPort: Number(e.target.value) }))
                  }
                />
                <Button variant="flat" onPress={() => setShowInboundOptions((v) => !v)}>
                  监听选项
                </Button>
              </div>
              {showInboundOptions ? (
                <Textarea
                  label="高级入站 JSON"
                  minRows={5}
                  value={form.inboundOptionsJson}
                  onChange={(e) =>
                    setForm((p) => ({ ...p, inboundOptionsJson: e.target.value }))
                  }
                />
              ) : null}

              <SectionTitle>传输</SectionTitle>
              <label className="flex w-fit cursor-pointer items-center gap-3 py-1 text-sm">
                <Switch
                  isSelected={Boolean(
                    getNestedOption(form.inboundOptionsJson, "transport", "type"),
                  )}
                  onValueChange={setTransportEnabled}
                />
                <span>
                  启用传输
                </span>
              </label>
              {getNestedOption(form.inboundOptionsJson, "transport", "type") ? (
                <div className="grid gap-3 md:grid-cols-2">
                  <Select
                    label="传输协议"
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
                    label="传输路径"
                    value={getNestedOption(
                      form.inboundOptionsJson,
                      "transport",
                      "path",
                    )}
                    onChange={(e) =>
                      updateNestedOption("transport", "path", e.target.value)
                    }
                  />
                </div>
              ) : null}

              <SectionTitle>用户管理</SectionTitle>
              <div className="rounded-md bg-default-100 px-4 py-3 text-sm">
                <div className="text-xs text-default-500">当前服务器绑定</div>
                <div className="mt-1 break-all font-medium">
                  {detail?.identity?.uuid || "节点身份加载中"}
                </div>
              </div>

              <SectionTitle>TLS</SectionTitle>
              <Select
                label="模板"
                selectedKeys={new Set([String(form.tlsTemplateId || 0)])}
                onSelectionChange={(keys) =>
                  setForm((p) => ({
                    ...p,
                    tlsTemplateId: Number(Array.from(keys)[0] || 0),
                  }))
                }
              >
                <SelectItem key="0">无</SelectItem>
                {tlsTemplates.map((tpl) => (
                  <SelectItem key={String(tpl.id)}>{tpl.name}</SelectItem>
                ))}
              </Select>

              <SectionTitle>多路复用</SectionTitle>
              <label className="flex w-fit cursor-pointer items-center gap-3 py-1 text-sm">
                <Switch
                  isSelected={getNestedBool("multiplex", "enabled")}
                  onValueChange={setMultiplexEnabled}
                />
                <span>
                  启用多路复用
                </span>
              </label>
              {getNestedBool("multiplex", "enabled") ? (
                <div className="grid gap-3 md:grid-cols-2">
                  <Select
                    label="多路复用协议"
                    selectedKeys={
                      new Set([
                        getNestedOption(
                          form.inboundOptionsJson,
                          "multiplex",
                          "protocol",
                          "smux",
                        ),
                      ])
                    }
                    onSelectionChange={(keys) =>
                      updateNestedOption(
                        "multiplex",
                        "protocol",
                        String(Array.from(keys)[0] || "smux"),
                      )
                    }
                  >
                    <SelectItem key="smux">smux</SelectItem>
                    <SelectItem key="yamux">yamux</SelectItem>
                  </Select>
                  <Input
                    label="最大连接数"
                    type="number"
                    value={getNestedOption(
                      form.inboundOptionsJson,
                      "multiplex",
                      "max_connections",
                      "4",
                    )}
                    onChange={(e) =>
                      updateNestedOption(
                        "multiplex",
                        "max_connections",
                        Number(e.target.value || 4),
                      )
                    }
                  />
                </div>
              ) : null}

              <div className="grid gap-3 md:grid-cols-2">
                <Input
                  label="发布地址"
                  value={form.publishAddr}
                  onChange={(e) =>
                    setForm((p) => ({ ...p, publishAddr: e.target.value }))
                  }
                />
                <Input
                  label="发布端口"
                  type="number"
                  value={String(form.publishPort || "")}
                  onChange={(e) =>
                    setForm((p) => ({ ...p, publishPort: Number(e.target.value) }))
                  }
                />
              </div>
                </>
              )}
            </div>
          </ModalBody>
          <ModalFooter>
            {deployProgress.active ? (
              <div className="mr-auto w-full max-w-xl">
                <Progress
                  aria-label={deployProgress.label}
                  color={deployProgress.color}
                  label={deployProgress.label}
                  showValueLabel
                  size="sm"
                  value={deployProgress.percent}
                />
                <div className="mt-2 text-xs text-default-500">
                  首次部署可能会补全 sing-box 核心并重启服务，甲骨文等环境下载较慢时请保持窗口打开。
                </div>
              </div>
            ) : null}
            <Button disabled={saving} variant="bordered" onPress={() => setView("menu")}>
              关闭
            </Button>
            <Button color="primary" isLoading={saving} onPress={saveInbound}>
              {form.id > 0 ? "更新" : "部署"}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    );
  }

  if (view === "manage") {
    return (
      <>
      <Modal isOpen={isOpen} scrollBehavior="inside" size="4xl" onClose={onClose}>
        <ModalContent>
          <ModalHeader>
            <div>
              <div className="text-lg font-semibold">管理入站</div>
              <div className="text-xs text-default-500">{nodeName}</div>
            </div>
          </ModalHeader>
          <ModalBody>
            <div className="grid gap-4 lg:grid-cols-[1fr_0.9fr]">
              <section className="space-y-3">
                {(detail?.inbounds || []).map((item) => (
                  <div
                    key={item.id}
                    className="rounded-md border border-divider bg-white p-4 shadow-sm dark:bg-gray-950"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <div className="text-base font-semibold">
                          {item.displayName}
                        </div>
                        <div className="mt-1 text-xs text-default-500">
                          {item.protocol.toUpperCase()} {item.publishAddr}:{item.publishPort}
                        </div>
                      </div>
                      <Chip size="sm" variant="flat">
                        {item.enabled === 1 ? "已启用" : "已停用"}
                      </Chip>
                    </div>
                    <div className="mt-4 flex flex-wrap gap-2">
                      <Button
                        size="sm"
                        variant="flat"
                        onPress={() => copyText(item.shareUri, "Share URI")}
                      >
                        复制链接
                      </Button>
                      <Button
                        size="sm"
                        variant="flat"
                        onPress={() =>
                          copyText(asSingBoxClientConfig(item), "sing-box client")
                        }
                      >
                        复制客户端
                      </Button>
                      <Button
                        size="sm"
                        variant="flat"
                        onPress={() =>
                          copyText(asSingBoxOutbound(item), "sing-box outbound")
                        }
                      >
                        复制出站
                      </Button>
                      <Button
                        size="sm"
                        variant="flat"
                        onPress={() => setQRPayload(item.shareUri)}
                      >
                        二维码
                      </Button>
                      <Button size="sm" variant="flat" onPress={() => editInbound(item)}>
                        编辑
                      </Button>
                      <Button
                        color="danger"
                        size="sm"
                        variant="flat"
                        onPress={() => deleteInbound(item)}
                      >
                        删除
                      </Button>
                    </div>
                  </div>
                ))}
                {(detail?.inbounds || []).length === 0 ? (
                  <div className="rounded-md border border-dashed border-divider p-8 text-center text-sm text-default-500">
                    暂无已部署入站。
                  </div>
                ) : null}
              </section>

              <aside className="space-y-4">
                <div className="rounded-md border border-divider p-4">
                  <h3 className="mb-3 text-sm font-semibold">节点身份</h3>
                  <div className="space-y-2 break-all text-xs">
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
                    重新生成身份
                  </Button>
                </div>

                {qrPayload ? (
                  <div className="rounded-md border border-divider p-4">
                    <div className="mb-3 flex items-center justify-between">
                      <h3 className="text-sm font-semibold">二维码</h3>
                      <Button size="sm" variant="light" onPress={() => setQRPayload("")}>
                        隐藏
                      </Button>
                    </div>
                    <div className="flex justify-center rounded-md bg-white p-4">
                      <OfflineQRCode value={qrPayload} />
                    </div>
                  </div>
                ) : null}

                <div className="rounded-md border border-divider p-4">
                  <h3 className="mb-3 text-sm font-semibold">配置版本</h3>
                  <div className="space-y-2">
                    {(detail?.revisions || []).slice(0, 5).map((rev) => (
                      <div key={rev.id} className="rounded-md bg-default-100 p-3 text-xs">
                        <div className="flex items-center justify-between gap-2">
                          <span>
                            #{rev.id} - {rev.status} - {rev.coreType}
                          </span>
                          <Button
                            size="sm"
                            variant="light"
                            onPress={() => rollbackConfig(rev.id)}
                          >
                            回滚
                          </Button>
                        </div>
                        <div className="mt-1 truncate text-default-500">
                          {rev.checksum}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="rounded-md border border-divider p-4">
                  <h3 className="mb-3 text-sm font-semibold">部署日志</h3>
                  <div className="space-y-2">
                    {(detail?.logs || []).slice(0, 6).map((log) => (
                      <div key={log.id} className="rounded-md bg-default-100 p-3 text-xs">
                        <div className="flex items-center justify-between gap-2">
                          <span>
                            {log.action} - {log.status}
                          </span>
                          <span className="text-default-500">#{log.revisionId}</span>
                        </div>
                        {log.message ? (
                          <div className="mt-1 line-clamp-2 text-default-500">
                            {log.message}
                          </div>
                        ) : null}
                      </div>
                    ))}
                    {(detail?.logs || []).length === 0 ? (
                      <div className="text-sm text-default-500">暂无部署日志。</div>
                    ) : null}
                  </div>
                </div>
              </aside>
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="bordered" onPress={() => setView("menu")}>
              返回
            </Button>
            <Button variant="flat" onPress={applyConfig}>
              重新下发配置
            </Button>
            <Button color="primary" onPress={resetInboundForm}>
              添加入站
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
      <Modal
        isOpen={Boolean(manualCopy)}
        scrollBehavior="inside"
        size="4xl"
        onClose={() => setManualCopy(null)}
      >
        <ModalContent>
          <ModalHeader>手动复制 {manualCopy?.label || ""}</ModalHeader>
          <ModalBody>
            <textarea
              readOnly
              className="min-h-[260px] w-full resize-y rounded-md border border-divider bg-default-50 p-3 font-mono text-xs"
              value={manualCopy?.text || ""}
              onFocus={(event) => event.currentTarget.select()}
            />
          </ModalBody>
          <ModalFooter>
            <Button variant="bordered" onPress={() => setManualCopy(null)}>
              关闭
            </Button>
            <Button
              color="primary"
              onPress={() =>
                manualCopy ? copyText(manualCopy.text, manualCopy.label) : undefined
              }
            >
              再次复制
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
      </>
    );
  }

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

                <div
                  className={`rounded-lg border border-divider p-4 ${!isAdmin ? "opacity-55 grayscale-[0.15]" : ""}`}
                  title={!isAdmin ? "Only administrators can modify TLS templates" : undefined}
                >
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
                      {fingerprintOptions.map((item) => (
                        <SelectItem key={item}>{item}</SelectItem>
                      ))}
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
                      <Button isDisabled={!isAdmin} size="sm" variant="flat" onPress={applyTLSFormToDraft}>
                        Generate JSON from form
                      </Button>
                      <Button
                        isDisabled={!isAdmin}
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
                    isDisabled={!isAdmin}
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
                              copyText(asSingBoxClientConfig(item), "sing-box client")
                            }
                          >
                            Copy Client
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
                          isDisabled={!isAdmin}
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
