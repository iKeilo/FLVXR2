import { useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";

import type { TunnelGroupNewApiItem } from "@/api/types";
import { createPathTunnel, getPathTunnelDetail, updatePathTunnel } from "@/api";
import { Button } from "@/shadcn-bridge/heroui/button";
import { Input } from "@/shadcn-bridge/heroui/input";
import { Select, SelectItem } from "@/shadcn-bridge/heroui/select";
import { Divider } from "@/shadcn-bridge/heroui/divider";

interface WGPathManagerProps {
  nodes: Array<{ id: number; name: string; status?: number; remark?: string }>;
  groups?: TunnelGroupNewApiItem[];
  pathId?: number | null;
  onCreated?: () => void;
}

export function WGPathManager({
  nodes,
  groups = [],
  pathId,
  onCreated,
}: WGPathManagerProps) {
  const [submitting, setSubmitting] = useState(false);
  const [name, setName] = useState("");
  const [remark, setRemark] = useState("");
  const [entryNodeId, setEntryNodeId] = useState("");
  const [exitNodeId, setExitNodeId] = useState("");
  const [chainNodeIds, setChainNodeIds] = useState<string[]>([]);
  const [tunnelGroupId, setTunnelGroupId] = useState<string>("none");
  const [flow, setFlow] = useState("1");
  const [trafficRatio, setTrafficRatio] = useState("1");
  const [listenStart, setListenStart] = useState("51820");

  const nodeOptions = useMemo(
    () => nodes.filter((node) => Number(node.id) > 0),
    [nodes],
  );
  const selectedNodeIds = useMemo(
    () =>
      [entryNodeId, ...chainNodeIds, exitNodeId]
        .map((id) => Number(id))
        .filter((id) => Number.isFinite(id) && id > 0),
    [entryNodeId, chainNodeIds, exitNodeId],
  );
  const chainPreview = useMemo(() => {
    const nameById = new Map(
      nodeOptions.map((node) => [String(node.id), node.name]),
    );
    const labels = [entryNodeId, ...chainNodeIds, exitNodeId]
      .filter(Boolean)
      .map((id) => nameById.get(id) || id);

    return labels.length > 0 ? labels.join(" -> ") : "入口 -> 出口";
  }, [entryNodeId, chainNodeIds, exitNodeId, nodeOptions]);

  const disabledNodeKeys = (currentValue: string) =>
    selectedNodeIds.map(String).filter((id) => id !== currentValue);

  useEffect(() => {
    if (!pathId) return;
    let cancelled = false;

    getPathTunnelDetail(pathId).then((res) => {
      if (cancelled || res.code !== 0 || !res.data) return;
      const detail = res.data;
      const segments = [...(detail.segments || [])].sort(
        (a, b) => (a.sequence || 0) - (b.sequence || 0),
      );
      const ids =
        segments.length > 0
          ? [
              String(segments[0].fromNodeId),
              ...segments.map((segment) => String(segment.toNodeId)),
            ]
          : [];

      setName(detail.path.name || "");
      setRemark(detail.path.remark || "");
      setTunnelGroupId(
        detail.path.tunnelGroupId ? String(detail.path.tunnelGroupId) : "none",
      );
      setFlow(String(detail.path.flow || 1));
      setTrafficRatio(String(detail.path.trafficRatio || 1));
      setEntryNodeId(ids[0] || "");
      setExitNodeId(ids[ids.length - 1] || "");
      setChainNodeIds(ids.slice(1, -1));
      const portResource = (detail.resources || []).find(
        (item) => item.resourceType === "port" && item.port,
      );

      setListenStart(String(portResource?.port || 51820));
    });

    return () => {
      cancelled = true;
    };
  }, [pathId]);

  const resetForm = () => {
    setName("");
    setRemark("");
    setEntryNodeId("");
    setExitNodeId("");
    setChainNodeIds([]);
    setTunnelGroupId("none");
    setFlow("1");
    setTrafficRatio("1");
    setListenStart("51820");
  };

  const handleCreate = async () => {
    const orderedNodeIds = selectedNodeIds;

    if (!name.trim() || !entryNodeId || !exitNodeId) {
      toast.error("请填写名称，并选择入口和出口节点");
      return;
    }
    if (orderedNodeIds.length < 2) {
      toast.error("WG 隧道至少需要入口和出口两个节点");
      return;
    }
    if (new Set(orderedNodeIds).size !== orderedNodeIds.length) {
      toast.error("同一条 WG 隧道不能重复选择节点");
      return;
    }
    setSubmitting(true);
    try {
      const payload = {
        name: name.trim(),
        transport: "wireguard",
        nodeIds: orderedNodeIds,
        remark: remark.trim(),
        listenStart: Number(listenStart) || 51820,
        tunnelGroupId:
          tunnelGroupId && tunnelGroupId !== "none"
            ? Number(tunnelGroupId)
            : null,
        flow: Number(flow) || 1,
        trafficRatio: Number(trafficRatio) || 1,
      };
      const res = pathId
        ? await updatePathTunnel({ id: pathId, ...payload })
        : await createPathTunnel(payload);

      if (res.code !== 0) {
        toast.error(res.msg || (pathId ? "更新失败" : "创建失败"));
        return;
      }
      if (!pathId) {
        resetForm();
      }
      toast.success(pathId ? "WG 隧道已更新" : "WG 隧道已创建");
      onCreated?.();
    } catch (err: any) {
      toast.error(err?.message || (pathId ? "更新失败" : "创建失败"));
    } finally {
      setSubmitting(false);
    }
  };

  const addChainNode = () => setChainNodeIds((prev) => [...prev, ""]);
  const removeChainNode = (index: number) =>
    setChainNodeIds((prev) => prev.filter((_, i) => i !== index));

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Input
          label="隧道名称"
          placeholder="例如：A-C-D-B WG 隧道"
          value={name}
          variant="bordered"
          onChange={(e) => setName(e.target.value)}
        />
        <Input
          label="备注"
          placeholder="可选"
          value={remark}
          variant="bordered"
          onChange={(e) => setRemark(e.target.value)}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Select
          label="分组"
          selectedKeys={new Set([tunnelGroupId])}
          variant="bordered"
          onSelectionChange={(keys) =>
            setTunnelGroupId(Array.from(keys)[0]?.toString() || "none")
          }
        >
          <SelectItem key="none">未分组</SelectItem>
          {groups.map((group) => (
            <SelectItem key={String(group.id)} textValue={group.name}>
              {group.name}
            </SelectItem>
          ))}
        </Select>
        <Input
          label="起始监听端口"
          type="number"
          value={listenStart}
          variant="bordered"
          onChange={(e) => setListenStart(e.target.value)}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Select
          label="流量计算"
          selectedKeys={new Set([flow])}
          variant="bordered"
          onSelectionChange={(keys) =>
            setFlow(Array.from(keys)[0]?.toString() || "1")
          }
        >
          <SelectItem key="1">单向计算</SelectItem>
          <SelectItem key="2">双向计算</SelectItem>
        </Select>
        <Input
          label="流量倍率"
          min={0.01}
          step="any"
          type="number"
          value={trafficRatio}
          variant="bordered"
          onChange={(e) => setTrafficRatio(e.target.value)}
        />
      </div>

      <Divider />
      <div className="space-y-3">
        <div className="text-sm font-semibold">出入口配置</div>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Select
            disabledKeys={disabledNodeKeys(entryNodeId)}
            label="入口节点"
            selectedKeys={entryNodeId ? new Set([entryNodeId]) : new Set([])}
            variant="bordered"
            onSelectionChange={(keys) =>
              setEntryNodeId(Array.from(keys)[0]?.toString() || "")
            }
          >
            {nodeOptions.map((node) => (
              <SelectItem key={String(node.id)} textValue={node.name}>
                {node.name}
              </SelectItem>
            ))}
          </Select>
          <Select
            disabledKeys={disabledNodeKeys(exitNodeId)}
            label="出口节点"
            selectedKeys={exitNodeId ? new Set([exitNodeId]) : new Set([])}
            variant="bordered"
            onSelectionChange={(keys) =>
              setExitNodeId(Array.from(keys)[0]?.toString() || "")
            }
          >
            {nodeOptions.map((node) => (
              <SelectItem key={String(node.id)} textValue={node.name}>
                {node.name}
              </SelectItem>
            ))}
          </Select>
        </div>
      </div>

      <div className="space-y-3 rounded-2xl border border-default-200 bg-content1/50 p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <div className="text-sm font-semibold">链式配置</div>
            <div className="text-xs text-default-500">
              未添加链式节点时为 A-B，添加后按 A-C-D-B 顺序生成
            </div>
          </div>
          <Button size="sm" variant="flat" onPress={addChainNode}>
            添加链式服务器
          </Button>
        </div>
        {chainNodeIds.length === 0 ? (
          <div className="rounded-xl bg-default-100/70 px-3 py-2 text-sm text-default-500">
            当前链路：{chainPreview}
          </div>
        ) : (
          <div className="space-y-3">
            {chainNodeIds.map((value, index) => (
              <div key={index} className="grid grid-cols-[1fr_auto] gap-2">
                <Select
                  disabledKeys={disabledNodeKeys(value)}
                  label={`第 ${index + 1} 跳链式节点`}
                  selectedKeys={value ? new Set([value]) : new Set([])}
                  variant="bordered"
                  onSelectionChange={(keys) => {
                    const nextValue = Array.from(keys)[0]?.toString() || "";
                    setChainNodeIds((prev) =>
                      prev.map((item, i) => (i === index ? nextValue : item)),
                    );
                  }}
                >
                  {nodeOptions.map((node) => (
                    <SelectItem key={String(node.id)} textValue={node.name}>
                      {node.name}
                    </SelectItem>
                  ))}
                </Select>
                <Button
                  color="danger"
                  className="self-end"
                  variant="flat"
                  onPress={() => removeChainNode(index)}
                >
                  删除
                </Button>
              </div>
            ))}
            <div className="rounded-xl bg-default-100/70 px-3 py-2 text-sm text-default-600">
              当前链路：{chainPreview}
            </div>
          </div>
        )}
      </div>

      <div className="flex justify-end">
        <Button
          color="primary"
          isDisabled={submitting}
          isLoading={submitting}
          onPress={handleCreate}
        >
          {pathId ? "保存 WG 隧道" : "创建 WG 隧道"}
        </Button>
      </div>
    </div>
  );
}
