import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import toast from "react-hot-toast";

import { AnimatedPage } from "@/components/animated-page";
import { SearchBar } from "@/components/search-bar";
import { Button } from "@/shadcn-bridge/heroui/button";
import {
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
} from "@/shadcn-bridge/heroui/table";
import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from "@/shadcn-bridge/heroui/modal";
import { Chip } from "@/shadcn-bridge/heroui/chip";
import { Select, SelectItem } from "@/shadcn-bridge/heroui/select";
import { getAdminOrderList, getAllUsers } from "@/api";
import type { OrderApiItem, UserApiItem } from "@/api/types";
import { PageLoadingState } from "@/components/page-state";

const statusMap: Record<number, { label: string; color: "warning" | "success" | "default" | "danger" }> = {
  0: { label: "待支付", color: "warning" },
  1: { label: "已完成", color: "success" },
  2: { label: "已取消", color: "default" },
  3: { label: "已退款", color: "danger" },
};

const currencyLabel: Record<string, string> = {
  BALANCE: "余额",
  USDT: "USDT",
  YIPAY: "易支付",
};

const statusTabs = [
  { key: "-1", label: "全部" },
  { key: "0", label: "待支付" },
  { key: "1", label: "已完成" },
  { key: "2", label: "已取消" },
  { key: "3", label: "已退款" },
];

export default function AdminOrdersPage() {
  const [loading, setLoading] = useState(true);
  const [refreshingOrders, setRefreshingOrders] = useState(false);
  const isFirstLoad = useRef(true);
  const [orders, setOrders] = useState<OrderApiItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("-1");
  const [keyword, setSearchKeyword] = useState("");
  const [isSearchVisible, setIsSearchVisible] = useState(false);
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [detailOrder, setDetailOrder] = useState<OrderApiItem | null>(null);

  // User filter
  const [users, setUsers] = useState<UserApiItem[]>([]);
  const [userId, setUserId] = useState<string>("all");

  const loadUsers = useCallback(async () => {
    try {
      const res = await getAllUsers({ size: 1000 });
      if (res.code === 0) {
        setUsers(Array.isArray(res.data) ? res.data : []);
      }
    } catch {
      // Silent fail
    }
  }, []);

  const userOptions = useMemo(() => {
    const opts = users.map((u) => ({
      id: u.id,
      name: (u.name && u.name.trim()) || u.user || `#${u.id}`,
    }));
    opts.sort((a, b) => {
      if (a.name === "admin") return -1;
      if (b.name === "admin") return 1;
      return a.name.localeCompare(b.name, "zh-CN");
    });
    return opts;
  }, [users]);

  const loadData = useCallback(async () => {
    if (!isFirstLoad.current) setRefreshingOrders(true);
    try {
      const res = await getAdminOrderList({
        page,
        size: 10,
        status: parseInt(statusFilter),
        keyword: keyword || undefined,
        userId: userId !== "all" ? Number(userId) : undefined,
      });
      if (res.code === 0) {
        setOrders(res.data.list || []);
        setTotal(res.data.total || 0);
      } else {
        toast.error(res.msg || "获取订单列表失败");
      }
    } catch {
      toast.error("获取订单列表失败");
    } finally {
      setLoading(false);
      if (!isFirstLoad.current) setRefreshingOrders(false);
      isFirstLoad.current = false;
    }
  }, [page, statusFilter, keyword, userId]);

  useEffect(() => { loadUsers(); }, [loadUsers]);

  useEffect(() => { loadData(); }, [loadData]);

  const handleViewDetail = (order: OrderApiItem) => {
    setDetailOrder(order);
    setDetailModalOpen(true);
  };

  if (loading && orders.length === 0) return <PageLoadingState message="加载订单中..." />;

  return (
    <AnimatedPage className="px-3 lg:px-6 py-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">账单管理</h1>
        <div className="flex gap-2 items-center">
          <SearchBar
            isVisible={isSearchVisible}
            placeholder="搜索订单号/用户..."
            value={keyword}
            onChange={setSearchKeyword}
            onClose={() => { setIsSearchVisible(false); setSearchKeyword(""); setPage(1); }}
            onOpen={() => setIsSearchVisible(true)}
            width="200px"
          />
          <Select
            //label="用户"
            variant="bordered"
            size="sm"
            className="w-40"
            selectedKeys={userId === "all" ? ["all"] : [userId]}
            onSelectionChange={(keys) => {
              const val = Array.from(keys)[0] as string;
              setUserId(val || "all");
              setPage(1);
            }}
          >
            <SelectItem key="all">全部用户</SelectItem>
            {userOptions.map((u) => (
              <SelectItem key={String(u.id)}>{u.name}</SelectItem>
            ))}
          </Select>
        </div>
      </div>

      {/* Status Tabs */}
      <div className="flex gap-4 border-b border-divider mb-6">
        {statusTabs.map((tab) => (
          <button
            key={tab.key}
            className={`pb-2 text-sm font-medium border-b-2 transition-colors ${statusFilter === tab.key
                ? "border-blue-500 text-blue-600"
                : "border-transparent text-default-500 hover:text-default-700"
              }`}
            onClick={() => { setStatusFilter(tab.key); setPage(1); }}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="relative overflow-hidden rounded-xl border border-divider bg-content1 shadow-md">
        {refreshingOrders && (
          <div className="absolute inset-0 bg-white/60 dark:bg-black/40 z-10 flex items-center justify-center">
            <svg className="animate-spin h-6 w-6 text-primary" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          </div>
        )}
        <Table
          classNames={{
            th: "bg-default-100/50 text-default-600 text-foreground font-semibold text-sm border-b border-divider py-3 uppercase tracking-wider text-left align-middle",
            td: "py-3 border-b border-divider/50 group-data-[last=true]:border-b-0",
            tr: "hover:bg-default-50/50 transition-colors",
          }}
        >
          <TableHeader>
            <TableColumn className="whitespace-nowrap">订单号</TableColumn>
            <TableColumn className="whitespace-nowrap">用户</TableColumn>
            <TableColumn className="whitespace-nowrap">商品</TableColumn>
            <TableColumn className="whitespace-nowrap">金额</TableColumn>
            <TableColumn className="whitespace-nowrap">支付方式</TableColumn>
            <TableColumn className="whitespace-nowrap">状态</TableColumn>
            <TableColumn className="whitespace-nowrap">时间</TableColumn>
            <TableColumn className="whitespace-nowrap">操作</TableColumn>
          </TableHeader>
          <TableBody>
            {orders.length === 0 ? (
              <TableRow><TableCell colSpan={8} className="text-center text-default-400 py-8">暂无订单</TableCell></TableRow>
            ) : orders.map((order) => {
              const st = statusMap[order.status] || { label: "未知", color: "default" };
              return (
                <TableRow key={order.id}>
                  <TableCell className="font-mono text-xs">{order.orderNo}</TableCell>
                  <TableCell>{order.userName}</TableCell>
                  <TableCell>{order.productName}</TableCell>
                  <TableCell className="font-mono">{(order.amount / 100).toFixed(2)} 元</TableCell>
                  <TableCell>{currencyLabel[order.payCurrency] || order.payCurrency}</TableCell>
                  <TableCell>
                    <Chip color={st.color} size="sm">{st.label}</Chip>
                  </TableCell>
                  <TableCell className="text-sm text-default-600">
                    {order.createdAt ? new Date(order.createdAt * 1000).toLocaleString() : "-"}
                  </TableCell>
                  <TableCell>
                    <Button size="sm" variant="flat" onPress={() => handleViewDetail(order)}>详情</Button>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>

      {total > 10 && (
        <div className="flex items-center justify-between mt-4">
          <span className="text-sm text-default-500">共 {total} 条</span>
          <div className="flex gap-2">
            <Button size="sm" variant="flat" isDisabled={page <= 1}
              onPress={() => setPage((p) => Math.max(1, p - 1))}>上一页</Button>
            <span className="flex items-center text-sm text-default-500 px-1">
              {page} / {Math.ceil(total / 10)}
            </span>
            <Button size="sm" variant="flat"
              isDisabled={page >= Math.ceil(total / 10)}
              onPress={() => setPage((p) => p + 1)}>下一页</Button>
          </div>
        </div>
      )}

      <Modal isOpen={detailModalOpen} placement="center" size="2xl"
        onOpenChange={(open) => { if (!open) { setDetailModalOpen(false); setDetailOrder(null); } }}>
        <ModalContent>
          <ModalHeader>订单详情</ModalHeader>
          <ModalBody>
            {detailOrder && (
              <div className="space-y-3 text-sm">
                <div className="flex justify-between"><span className="text-default-500">订单号</span><span className="font-mono">{detailOrder.orderNo}</span></div>
                <div className="flex justify-between"><span className="text-default-500">用户</span><span>{detailOrder.userName}</span></div>
                <div className="flex justify-between"><span className="text-default-500">商品</span><span>{detailOrder.productName}</span></div>
                <div className="flex justify-between"><span className="text-default-500">金额</span><span className="font-mono">{(detailOrder.amount / 100).toFixed(2)} 元</span></div>
                <div className="flex justify-between"><span className="text-default-500">支付方式</span><span>{currencyLabel[detailOrder.payCurrency] || detailOrder.payCurrency}</span></div>
                <div className="flex justify-between"><span className="text-default-500">状态</span><Chip color={statusMap[detailOrder.status]?.color || "default"} size="sm">{statusMap[detailOrder.status]?.label || "未知"}</Chip></div>
                {detailOrder.payTime > 0 && (
                  <div className="flex justify-between"><span className="text-default-500">支付时间</span><span>{new Date(detailOrder.payTime * 1000).toLocaleString()}</span></div>
                )}
                {detailOrder.payAddress && (
                  <div className="flex justify-between"><span className="text-default-500">USDT 地址</span><span className="font-mono text-xs max-w-[200px] break-all text-right">{detailOrder.payAddress}</span></div>
                )}
                {detailOrder.txHash && (
                  <div className="flex justify-between"><span className="text-default-500">交易哈希</span><span className="font-mono text-xs max-w-[200px] break-all text-right">{detailOrder.txHash}</span></div>
                )}
                <div className="flex justify-between"><span className="text-default-500">创建时间</span><span>{detailOrder.createdAt ? new Date(detailOrder.createdAt * 1000).toLocaleString() : "-"}</span></div>
              </div>
            )}
          </ModalBody>
          <ModalFooter>
            <Button variant="flat" onPress={() => { setDetailModalOpen(false); setDetailOrder(null); }}>关闭</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </AnimatedPage>
  );
}
