import { useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";
import {
  Copy,
  FileKey2,
  Pencil,
  Plus,
  RefreshCw,
  ShieldCheck,
  Trash2,
} from "lucide-react";

import {
  deleteTLSTemplate,
  generateTLSRealityKeypair,
  generateTLSRealityShortIds,
  getTLSTemplates,
  saveTLSTemplate,
} from "@/api";
import type { NodeTLSTemplateApiItem } from "@/api/types";
import { Button } from "@/shadcn-bridge/heroui/button";
import { Input, Textarea } from "@/shadcn-bridge/heroui/input";
import {
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
} from "@/shadcn-bridge/heroui/modal";
import { Select, SelectItem } from "@/shadcn-bridge/heroui/select";
import { Switch } from "@/shadcn-bridge/heroui/switch";

type TLSType = "tls" | "reality";
type SourceMode = "path" | "content";

interface TLSFormState {
  id?: number;
  name: string;
  type: TLSType;
  sni: string;
  sourceMode: SourceMode;
  certPath: string;
  keyPath: string;
  certContent: string;
  keyContent: string;
  disableSni: boolean;
  insecure: boolean;
  tlsOptions: string;
  acmeEnabled: boolean;
  acmeDomains: string;
  acmeEmail: string;
  echEnabled: boolean;
  echConfig: string;
  handshakeServer: string;
  handshakePort: string;
  privateKey: string;
  publicKey: string;
  shortIds: string;
  fingerprint: string;
}

const emptyForm: TLSFormState = {
  name: "",
  type: "tls",
  sni: "",
  sourceMode: "path",
  certPath: "",
  keyPath: "",
  certContent: "",
  keyContent: "",
  disableSni: false,
  insecure: false,
  tlsOptions: "{}",
  acmeEnabled: false,
  acmeDomains: "",
  acmeEmail: "",
  echEnabled: false,
  echConfig: "",
  handshakeServer: "",
  handshakePort: "443",
  privateKey: "",
  publicKey: "",
  shortIds: "",
  fingerprint: "chrome",
};

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

const ensureOK = <T,>(res: { code: number; msg?: string; data: T }) => {
  if (res.code !== 0) {
    throw new Error(res.msg || "请求失败");
  }

  return res.data;
};

const parseJSON = (raw?: string) => {
  try {
    const parsed = JSON.parse(raw || "{}");

    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
};

const asArrayText = (value: unknown) => {
  if (Array.isArray(value)) {
    return value.join("\n");
  }
  if (typeof value === "string") {
    return value;
  }

  return "";
};

const asCSV = (value: unknown) => {
  if (Array.isArray(value)) {
    return value.join(",");
  }
  if (typeof value === "string") {
    return value;
  }

  return "";
};

const splitLines = (value: string) =>
  value
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean);

const splitCSV = (value: string) =>
  value.split(",").map((item) => item.trim());

const normalizePort = (value: string) => {
  const port = Number(value);

  return Number.isFinite(port) && port > 0 && port <= 65535 ? port : 443;
};

const getTemplateSummary = (item: NodeTLSTemplateApiItem) => {
  const server = parseJSON(item.serverJson) as Record<string, any>;
  const client = parseJSON(item.clientJson) as Record<string, any>;
  const reality = server.reality || client.reality;
  const type = item.type === "reality" || reality ? "reality" : "tls";
  const sni = server.server_name || client.server_name || "-";
  const handshake = reality?.handshake?.server || "-";
  const acme = Boolean(server.acme?.enabled || server.acme?.domain);
  const ech = Boolean(server.ech?.enabled || client.ech?.enabled);

  return { acme, ech, handshake, sni, type };
};

const toForm = (item?: NodeTLSTemplateApiItem): TLSFormState => {
  if (!item) {
    return { ...emptyForm };
  }
  const server = parseJSON(item.serverJson) as Record<string, any>;
  const client = parseJSON(item.clientJson) as Record<string, any>;
  const reality = server.reality || {};
  const acme = server.acme || {};
  const ech = server.ech || client.ech || {};
  const hasContent = Boolean(server.certificate || server.key);
  const extra = { ...server };

  delete extra.enabled;
  delete extra.server_name;
  delete extra.certificate;
  delete extra.certificate_path;
  delete extra.key;
  delete extra.key_path;
  delete extra.acme;
  delete extra.ech;
  delete extra.reality;

  return {
    ...emptyForm,
    id: item.id,
    name: item.name,
    type: item.type === "reality" ? "reality" : "tls",
    sni: server.server_name || client.server_name || "",
    sourceMode: hasContent ? "content" : "path",
    certPath: server.certificate_path || "",
    keyPath: server.key_path || "",
    certContent: asArrayText(server.certificate),
    keyContent: asArrayText(server.key),
    disableSni: Boolean(client.disable_sni),
    insecure: Boolean(client.insecure),
    tlsOptions: JSON.stringify(extra, null, 2),
    acmeEnabled: Boolean(acme.enabled || acme.domain),
    acmeDomains: asCSV(acme.domain),
    acmeEmail: acme.email || "",
    echEnabled: Boolean(ech.enabled),
    echConfig: asArrayText(ech.config || ech.key),
    handshakeServer: reality.handshake?.server || "",
    handshakePort: String(reality.handshake?.server_port || 443),
    privateKey: reality.private_key || "",
    publicKey: client.reality?.public_key || "",
    shortIds: asCSV(reality.short_id),
    fingerprint:
      client.utls?.fingerprint || client.utls_fingerprint || extra.utls?.fingerprint || "chrome",
  };
};

const toPayload = (form: TLSFormState): Partial<NodeTLSTemplateApiItem> => {
  const extra = parseJSON(form.tlsOptions) as Record<string, any>;
  const server: Record<string, any> = {
    ...extra,
    enabled: true,
    server_name: form.sni.trim(),
  };
  const client: Record<string, any> = {
    enabled: true,
    server_name: form.sni.trim(),
    disable_sni: form.disableSni,
    insecure: form.insecure,
  };
  if (form.fingerprint.trim()) {
    client.utls = {
      enabled: true,
      fingerprint: form.fingerprint.trim(),
    };
    client.utls_fingerprint = form.fingerprint.trim();
  }

  if (form.type === "tls") {
    if (form.sourceMode === "content") {
      server.certificate = splitLines(form.certContent);
      server.key = splitLines(form.keyContent);
    } else {
      server.certificate_path = form.certPath.trim();
      server.key_path = form.keyPath.trim();
    }
  } else {
    const realityShortIds = splitCSV(form.shortIds);
    const firstRealityShortId =
      realityShortIds.find((item) => item.trim() !== "") || "";
    server.reality = {
      enabled: true,
      handshake: {
        server: form.handshakeServer.trim(),
        server_port: normalizePort(form.handshakePort),
      },
      private_key: form.privateKey.trim(),
      short_id: realityShortIds,
    };
    client.reality = {
      enabled: true,
      public_key: form.publicKey.trim(),
      short_id: firstRealityShortId,
    };
  }

  if (form.acmeEnabled) {
    server.acme = {
      enabled: true,
      domain: splitCSV(form.acmeDomains).filter(Boolean),
      email: form.acmeEmail.trim(),
    };
  }
  if (form.echEnabled) {
    server.ech = {
      enabled: true,
      config: splitLines(form.echConfig),
    };
    client.ech = {
      enabled: true,
      config: splitLines(form.echConfig),
    };
  }

  return {
    id: form.id,
    name: form.name.trim(),
    type: form.type,
    serverJson: JSON.stringify(server, null, 2),
    clientJson: JSON.stringify(client, null, 2),
  };
};

const TLSPage = () => {
  const [items, setItems] = useState<NodeTLSTemplateApiItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [showOptions, setShowOptions] = useState(false);
  const [form, setForm] = useState<TLSFormState>({ ...emptyForm });

  const stats = useMemo(() => {
    return {
      total: items.length,
      reality: items.filter((item) => getTemplateSummary(item).type === "reality")
        .length,
      used: items.reduce((sum, item) => sum + (item.usageCount || 0), 0),
    };
  }, [items]);

  const loadItems = async () => {
    setLoading(true);
    try {
      const data = ensureOK(await getTLSTemplates());

      setItems(data || []);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "加载 TLS 模板失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadItems();
  }, []);

  const openCreate = () => {
    setForm({ ...emptyForm });
    setShowOptions(false);
    setModalOpen(true);
  };

  const openEdit = (item: NodeTLSTemplateApiItem) => {
    setForm(toForm(item));
    setShowOptions(false);
    setModalOpen(true);
  };

  const patchForm = (patch: Partial<TLSFormState>) => {
    setForm((current) => ({ ...current, ...patch }));
  };

  const handleSave = async () => {
    if (!form.name.trim()) {
      toast.error("请填写 TLS 名称");

      return;
    }
    if (form.type === "reality" && (!form.privateKey || !form.publicKey)) {
      toast.error("Reality 需要先生成或填写公钥和私钥");

      return;
    }
    try {
      setSaving(true);
      ensureOK(await saveTLSTemplate(toPayload(form)));
      toast.success("TLS 模板已保存");
      setModalOpen(false);
      await loadItems();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "保存失败");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (item: NodeTLSTemplateApiItem) => {
    if (!window.confirm(`删除 TLS 模板「${item.name}」？`)) {
      return;
    }
    try {
      ensureOK(await deleteTLSTemplate(item.id));
      toast.success("TLS 模板已删除");
      await loadItems();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "删除失败");
    }
  };

  const handleDuplicate = async (item: NodeTLSTemplateApiItem) => {
    try {
      ensureOK(
        await saveTLSTemplate({
          ...item,
          id: undefined,
          name: `${item.name}-copy`,
        }),
      );
      toast.success("TLS 模板已复制");
      await loadItems();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "复制失败");
    }
  };

  const handleRealityKeypair = async () => {
    try {
      const data = ensureOK(await generateTLSRealityKeypair());

      patchForm({ privateKey: data.privateKey, publicKey: data.publicKey });
      toast.success("Reality 密钥已生成");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "生成密钥失败");
    }
  };

  const handleRealityShortIds = async () => {
    try {
      const data = ensureOK(await generateTLSRealityShortIds());

      patchForm({ shortIds: data.shortIds.join(",") });
      toast.success("Short IDs 已刷新");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "刷新 Short IDs 失败");
    }
  };

  return (
    <div className="min-h-full bg-gray-50 px-4 py-4 text-gray-900 dark:bg-black dark:text-gray-100 sm:px-6">
      <div className="mx-auto flex w-full max-w-7xl flex-col gap-4">
        <div className="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-gray-800 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold">TLS 管理</h1>
            <div className="mt-2 flex flex-wrap gap-2 text-sm text-gray-500 dark:text-gray-400">
              <span>模板 {stats.total}</span>
              <span>Reality {stats.reality}</span>
              <span>入站引用 {stats.used}</span>
            </div>
          </div>
          <Button color="primary" startContent={<Plus className="h-4 w-4" />} onPress={openCreate}>
            添加
          </Button>
        </div>

        {loading ? (
          <div className="flex h-48 items-center justify-center text-sm text-gray-500">
            正在加载 TLS 模板
          </div>
        ) : items.length === 0 ? (
          <div className="flex h-48 items-center justify-center rounded-md border border-dashed border-gray-300 bg-white text-sm text-gray-500 dark:border-gray-800 dark:bg-gray-950">
            暂无 TLS 模板
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
            {items.map((item) => {
              const summary = getTemplateSummary(item);

              return (
                <div
                  key={item.id}
                  className="flex min-h-[190px] flex-col rounded-md border border-gray-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-950"
                >
                  <div className="border-b border-gray-200 px-4 py-3 dark:border-gray-800">
                    <div className="truncate text-lg font-semibold" title={item.name}>
                      {item.name}
                    </div>
                    <div className="truncate text-xs text-gray-500" title={summary.sni}>
                      {summary.type === "reality" ? summary.handshake : summary.sni}
                    </div>
                  </div>
                  <div className="grid flex-1 grid-cols-[92px_1fr] gap-y-2 px-4 py-3 text-sm">
                    <span className="text-gray-500">入站管理</span>
                    <span>{item.usageCount || "-"}</span>
                    <span className="text-gray-500">ACME</span>
                    <span>{summary.acme ? "启用" : "取消"}</span>
                    <span className="text-gray-500">ECH</span>
                    <span>{summary.ech ? "启用" : "取消"}</span>
                    <span className="text-gray-500">Reality</span>
                    <span>{summary.type === "reality" ? "确认" : "取消"}</span>
                  </div>
                  <div className="flex items-center gap-2 border-t border-gray-100 px-3 py-2 dark:border-gray-800">
                    <Button
                      isIconOnly
                      aria-label="编辑 TLS 模板"
                      size="sm"
                      title="编辑"
                      variant="light"
                      onPress={() => openEdit(item)}
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                    <Button
                      isIconOnly
                      aria-label="复制 TLS 模板"
                      size="sm"
                      title="复制"
                      variant="light"
                      onPress={() => handleDuplicate(item)}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                    <Button
                      isIconOnly
                      aria-label="删除 TLS 模板"
                      className="ml-auto text-red-600"
                      size="sm"
                      title="删除"
                      variant="light"
                      onPress={() => handleDelete(item)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      <Modal isOpen={modalOpen} scrollBehavior="inside" size="4xl" onClose={() => setModalOpen(false)}>
        <ModalContent>
          <ModalHeader>{form.id ? "编辑 TLS" : "添加 TLS"}</ModalHeader>
          <ModalBody>
            <div className="flex justify-end">
              <div className="grid w-[220px] grid-cols-2 rounded-md bg-gray-100 p-1 dark:bg-gray-900">
                {(["tls", "reality"] as TLSType[]).map((type) => (
                  <button
                    key={type}
                    className={`h-9 rounded px-3 text-sm font-medium ${
                      form.type === type
                        ? "bg-white text-gray-950 shadow-sm dark:bg-gray-800 dark:text-white"
                        : "text-gray-500"
                    }`}
                    type="button"
                    onClick={() => patchForm({ type })}
                  >
                    {type === "tls" ? "TLS" : "REALITY"}
                  </button>
                ))}
              </div>
            </div>

            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              <Input
                label="名称"
                placeholder="例如 HK Reality"
                value={form.name}
                onChange={(event) => patchForm({ name: event.target.value })}
              />
              <Input
                label="SNI"
                placeholder="example.com"
                value={form.sni}
                onChange={(event) => patchForm({ sni: event.target.value })}
              />
            </div>

            {form.type === "tls" ? (
              <div className="space-y-3">
                <div className="inline-grid grid-cols-2 rounded-md bg-gray-100 p-1 dark:bg-gray-900">
                  {(["path", "content"] as SourceMode[]).map((mode) => (
                    <button
                      key={mode}
                      className={`h-9 rounded px-4 text-sm font-medium ${
                        form.sourceMode === mode
                          ? "bg-white text-gray-950 shadow-sm dark:bg-gray-800 dark:text-white"
                          : "text-gray-500"
                      }`}
                      type="button"
                      onClick={() => patchForm({ sourceMode: mode })}
                    >
                      {mode === "path" ? "使用外部路径" : "使用文件内容"}
                    </button>
                  ))}
                </div>

                {form.sourceMode === "path" ? (
                  <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                    <Input
                      label="证书文件路径"
                      placeholder="/etc/ssl/fullchain.pem"
                      value={form.certPath}
                      onChange={(event) => patchForm({ certPath: event.target.value })}
                    />
                    <Input
                      label="私钥文件路径"
                      placeholder="/etc/ssl/privkey.pem"
                      value={form.keyPath}
                      onChange={(event) => patchForm({ keyPath: event.target.value })}
                    />
                  </div>
                ) : (
                  <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                    <Textarea
                      label="证书内容"
                      minRows={6}
                      value={form.certContent}
                      onChange={(event) => patchForm({ certContent: event.target.value })}
                    />
                    <Textarea
                      label="私钥内容"
                      minRows={6}
                      value={form.keyContent}
                      onChange={(event) => patchForm({ keyContent: event.target.value })}
                    />
                  </div>
                )}
              </div>
            ) : (
              <div className="space-y-3">
                <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                  <Input
                    label="握手服务器"
                    placeholder="www.microsoft.com"
                    value={form.handshakeServer}
                    onChange={(event) => patchForm({ handshakeServer: event.target.value })}
                  />
                  <Input
                    label="服务器端口"
                    placeholder="443"
                    value={form.handshakePort}
                    onChange={(event) => patchForm({ handshakePort: event.target.value })}
                  />
                </div>
                <div className="grid grid-cols-[1fr_auto] gap-3">
                  <Input
                    label="私钥"
                    value={form.privateKey}
                    onChange={(event) => patchForm({ privateKey: event.target.value })}
                  />
                  <Button
                    isIconOnly
                    aria-label="生成 Reality 密钥"
                    className="mt-6"
                    title="生成 Reality 密钥"
                    variant="flat"
                    onPress={handleRealityKeypair}
                  >
                    <FileKey2 className="h-4 w-4" />
                  </Button>
                </div>
                <Input
                  label="公钥"
                  value={form.publicKey}
                  onChange={(event) => patchForm({ publicKey: event.target.value })}
                />
                <div className="grid grid-cols-[1fr_auto] gap-3">
                  <Input
                    label="Short IDs"
                    value={form.shortIds}
                    onChange={(event) => patchForm({ shortIds: event.target.value })}
                  />
                  <Button
                    isIconOnly
                    aria-label="刷新 Short IDs"
                    className="mt-6"
                    title="刷新 Short IDs"
                    variant="flat"
                    onPress={handleRealityShortIds}
                  >
                    <RefreshCw className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            )}

            <div className="grid grid-cols-1 gap-3 border-t border-gray-200 pt-4 dark:border-gray-800 md:grid-cols-2">
              <Select
                label="Fingerprint"
                selectedKeys={new Set([form.fingerprint || "chrome"])}
                onSelectionChange={(keys) =>
                  patchForm({
                    fingerprint: String(Array.from(keys)[0] || "chrome"),
                  })
                }
              >
                {fingerprintOptions.map((item) => (
                  <SelectItem key={item}>{item}</SelectItem>
                ))}
              </Select>
              <label className="flex items-center gap-3 text-sm">
                <Switch
                  isSelected={form.disableSni}
                  onValueChange={(value) => patchForm({ disableSni: value })}
                />
                禁用 SNI
              </label>
              <label className="flex items-center gap-3 text-sm">
                <Switch
                  isSelected={form.insecure}
                  onValueChange={(value) => patchForm({ insecure: value })}
                />
                允许不安全
              </label>
            </div>

            <div className="space-y-3 border-t border-gray-200 pt-4 dark:border-gray-800">
              <button
                className="flex items-center gap-2 text-sm font-medium text-blue-600"
                type="button"
                onClick={() => setShowOptions((value) => !value)}
              >
                <ShieldCheck className="h-4 w-4" />
                TLS 选项
              </button>
              {showOptions ? (
                <Textarea
                  label="高级 TLS JSON"
                  minRows={5}
                  value={form.tlsOptions}
                  onChange={(event) => patchForm({ tlsOptions: event.target.value })}
                />
              ) : null}
            </div>

            <div className="space-y-3 border-t border-gray-200 pt-4 dark:border-gray-800">
              <label className="flex items-center gap-3 text-sm font-medium">
                <Switch
                  isSelected={form.acmeEnabled}
                  onValueChange={(value) => patchForm({ acmeEnabled: value })}
                />
                ACME
              </label>
              {form.acmeEnabled ? (
                <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                  <Input
                    label="域名"
                    placeholder="example.com,*.example.com"
                    value={form.acmeDomains}
                    onChange={(event) => patchForm({ acmeDomains: event.target.value })}
                  />
                  <Input
                    label="邮箱"
                    placeholder="admin@example.com"
                    value={form.acmeEmail}
                    onChange={(event) => patchForm({ acmeEmail: event.target.value })}
                  />
                </div>
              ) : null}
            </div>

            <div className="space-y-3 border-t border-gray-200 pt-4 dark:border-gray-800">
              <label className="flex items-center gap-3 text-sm font-medium">
                <Switch
                  isSelected={form.echEnabled}
                  onValueChange={(value) => patchForm({ echEnabled: value })}
                />
                ECH
              </label>
              {form.echEnabled ? (
                <Textarea
                  label="ECH 配置"
                  minRows={3}
                  value={form.echConfig}
                  onChange={(event) => patchForm({ echConfig: event.target.value })}
                />
              ) : null}
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="bordered" onPress={() => setModalOpen(false)}>
              关闭
            </Button>
            <Button color="primary" isLoading={saving} onPress={handleSave}>
              保存
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </div>
  );
};

export default TLSPage;
