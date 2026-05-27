import { useState, useEffect, useCallback } from "react";
import toast from "react-hot-toast";

import { AnimatedPage } from "@/components/animated-page";
import { SearchBar } from "@/components/search-bar";
import { Button } from "@/shadcn-bridge/heroui/button";
import { Input } from "@/shadcn-bridge/heroui/input";
import { Textarea } from "@/shadcn-bridge/heroui/input";
import { Select, SelectItem } from "@/shadcn-bridge/heroui/select";
import { Checkbox } from "@/shadcn-bridge/heroui/checkbox";
import { Switch } from "@/shadcn-bridge/heroui/switch";
import {
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
} from "@/shadcn-bridge/heroui/table";
import { Chip } from "@/shadcn-bridge/heroui/chip";
import { Card, CardBody } from "@/shadcn-bridge/heroui/card";
import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from "@/shadcn-bridge/heroui/modal";
import {
  getProductList,
  createProduct,
  updateProduct,
  deleteProduct,
  getPackageList,
  createPackage,
  updatePackage,
  deletePackage,
  getPackageDetail,
  getTunnelGroupList,
} from "@/api";
import type {
  ProductApiItem,
  SubscriptionPackageApiItem,
  TunnelGroupApiItem,
} from "@/api/types";
import { PageLoadingState } from "@/components/page-state";
import { useLocalStorageState } from "@/hooks/use-local-storage-state";

const productTypeOptions = [
  { value: "recharge", label: "余额充值" },
  { value: "traffic", label: "流量包" },
  { value: "time", label: "时长续费" },
];

const typeBadgeColor: Record<string, "primary" | "warning" | "success"> = {
  recharge: "warning",
  traffic: "primary",
  time: "success",
};

interface ProductForm {
  id?: number;
  name: string;
  description: string;
  type: string;
  priceYuan: string;
  value: number;
  sortOrder: number;
  status: number;
}

const defaultProductForm: ProductForm = {
  name: "",
  description: "",
  type: "traffic",
  priceYuan: "0",
  value: 0,
  sortOrder: 0,
  status: 1,
};

const typeUnit: Record<string, string> = {
  recharge: "元",
  traffic: "GB",
  time: "天",
};

interface PackageForm {
  id?: number;
  name: string;
  description: string;
  priceYuan: string;
  validityDays: number;
  trafficLimit: number;
  portCount: number;
  speedLimit: number;
  maxRules: number;
  maxConnections: number;
  maxIPAccess: number;
  autoRenew: boolean;
  enabled: boolean;
  shopVisible: boolean;
  sortOrder: number;
  tunnelGroupIds: number[];
}

const defaultPackageForm: PackageForm = {
  name: "",
  description: "",
  priceYuan: "0",
  validityDays: 30,
  trafficLimit: 0,
  portCount: 0,
  speedLimit: 0,
  maxRules: 0,
  maxConnections: 0,
  maxIPAccess: 0,
  autoRenew: false,
  enabled: true,
  shopVisible: true,
  sortOrder: 0,
  tunnelGroupIds: [],
};

export default function AdminProductsPage() {
  const [tab, setTab] = useState<"products" | "packages">("products");

  // ── Product state ──
  const [productsLoading, setProductsLoading] = useState(true);
  const [products, setProducts] = useState<ProductApiItem[]>([]);
  const [searchKeyword, setSearchKeyword] = useLocalStorageState("admin-products-search", "");
  const [isSearchVisible, setIsSearchVisible] = useState(false);
  const [productModalOpen, setProductModalOpen] = useState(false);
  const [productDeleteModalOpen, setProductDeleteModalOpen] = useState(false);
  const [isProductEdit, setIsProductEdit] = useState(false);
  const [productForm, setProductForm] = useState<ProductForm>({ ...defaultProductForm });
  const [productToDelete, setProductToDelete] = useState<ProductApiItem | null>(null);
  const [productSubmitLoading, setProductSubmitLoading] = useState(false);

  // ── Package state ──
  const [packagesLoading, setPackagesLoading] = useState(true);
  const [pkgList, setPkgList] = useState<SubscriptionPackageApiItem[]>([]);
  const [tunnelGroups, setTunnelGroups] = useState<TunnelGroupApiItem[]>([]);
  const [pkgModalOpen, setPkgModalOpen] = useState(false);
  const [pkgDeleteModalOpen, setPkgDeleteModalOpen] = useState(false);
  const [isPkgEdit, setIsPkgEdit] = useState(false);
  const [pkgForm, setPkgForm] = useState<PackageForm>({ ...defaultPackageForm });
  const [pkgToDelete, setPkgToDelete] = useState<SubscriptionPackageApiItem | null>(null);
  const [pkgSubmitLoading, setPkgSubmitLoading] = useState(false);
  const [pkgModalLoading, setPkgModalLoading] = useState(false);

  // ── Load products ──
  const loadProducts = useCallback(async () => {
    setProductsLoading(true);
    try {
      const res = await getProductList();
      if (res.code === 0) {
        setProducts(Array.isArray(res.data) ? res.data : []);
      } else {
        toast.error(res.msg || "获取商品列表失败");
      }
    } catch {
      toast.error("获取商品列表失败");
    } finally {
      setProductsLoading(false);
    }
  }, []);

  useEffect(() => { loadProducts(); }, [loadProducts]);

  // ── Load packages ──
  const loadPackages = useCallback(async () => {
    setPackagesLoading(true);
    try {
      const [pkgRes, tgRes] = await Promise.all([
        getPackageList(),
        getTunnelGroupList(),
      ]);
      if (pkgRes.code === 0) {
        setPkgList(Array.isArray(pkgRes.data) ? pkgRes.data : []);
      } else {
        toast.error(pkgRes.msg || "获取套餐列表失败");
      }
      if (tgRes.code === 0) {
        setTunnelGroups(Array.isArray(tgRes.data) ? tgRes.data : []);
      }
    } catch {
      toast.error("获取套餐列表失败");
    } finally {
      setPackagesLoading(false);
    }
  }, []);

  // ── Product handlers ──
  const filtered = products.filter((p) =>
    !searchKeyword || p.name?.toLowerCase().includes(searchKeyword.toLowerCase())
  );

  const activeCount = products.filter((p) => p.status === 1).length;
  const typeCounts: Record<string, number> = {};
  for (const p of products) {
    typeCounts[p.type] = (typeCounts[p.type] || 0) + 1;
  }

  const handleProductAdd = () => {
    setProductForm({ ...defaultProductForm });
    setIsProductEdit(false);
    setProductModalOpen(true);
  };

  const handleProductEdit = (item: ProductApiItem) => {
    setProductForm({
      id: item.id,
      name: item.name,
      description: item.description || "",
      type: item.type,
      priceYuan: (item.price / 100).toFixed(2),
      value: item.value,
      sortOrder: item.sortOrder || 0,
      status: item.status,
    });
    setIsProductEdit(true);
    setProductModalOpen(true);
  };

  const handleProductSubmit = async () => {
    if (!productForm.name.trim()) {
      toast.error("商品名称不能为空");
      return;
    }
    setProductSubmitLoading(true);
    try {
      const priceFen = Math.round(parseFloat(productForm.priceYuan || "0") * 100);
      const data = {
        id: productForm.id,
        name: productForm.name,
        description: productForm.description,
        type: productForm.type,
        price: priceFen,
        value: productForm.value,
        sortOrder: productForm.sortOrder,
        status: productForm.status,
      };
      const res = isProductEdit ? await updateProduct(data) : await createProduct(data);
      if (res.code === 0) {
        toast.success(isProductEdit ? "更新成功" : "创建成功");
        setProductModalOpen(false);
        loadProducts();
      } else {
        toast.error(res.msg || "操作失败");
      }
    } catch {
      toast.error("网络错误");
    } finally {
      setProductSubmitLoading(false);
    }
  };

  const handleProductDelete = (item: ProductApiItem) => {
    setProductToDelete(item);
    setProductDeleteModalOpen(true);
  };

  const confirmProductDelete = async () => {
    if (!productToDelete) return;
    try {
      const res = await deleteProduct(productToDelete.id);
      if (res.code === 0) {
        toast.success("已删除");
        setProductDeleteModalOpen(false);
        setProductToDelete(null);
        loadProducts();
      } else {
        toast.error(res.msg || "删除失败");
      }
    } catch {
      toast.error("网络错误");
    }
  };

  // ── Package handlers ──
  const handlePkgAdd = () => {
    setPkgForm({ ...defaultPackageForm });
    setIsPkgEdit(false);
    setPkgModalOpen(true);
  };

  const handlePkgEdit = async (item: SubscriptionPackageApiItem) => {
    setIsPkgEdit(true);
    setPkgModalLoading(true);
    setPkgModalOpen(true);
    try {
      const res = await getPackageDetail(item.id);
      if (res.code === 0) {
        const p = res.data.package;
        setPkgForm({
          id: p.id,
          name: p.name,
          description: p.description || "",
          priceYuan: (p.price / 100).toFixed(2),
          validityDays: p.validityDays,
          trafficLimit: p.trafficLimit,
          portCount: p.portCount,
          speedLimit: p.speedLimit,
          maxRules: p.maxRules,
          maxConnections: p.maxConnections,
          maxIPAccess: p.maxIPAccess,
          autoRenew: p.autoRenew === 1,
          enabled: p.enabled === 1,
          shopVisible: p.shopVisible === 1,
          sortOrder: p.sortOrder,
          tunnelGroupIds: res.data.tunnelGroupIds || [],
        });
      } else {
        toast.error(res.msg || "获取套餐详情失败");
        setPkgModalOpen(false);
      }
    } catch {
      toast.error("网络错误");
      setPkgModalOpen(false);
    } finally {
      setPkgModalLoading(false);
    }
  };

  const handlePkgSubmit = async () => {
    if (!pkgForm.name.trim()) {
      toast.error("套餐名称不能为空");
      return;
    }
    setPkgSubmitLoading(true);
    try {
      const priceFen = Math.round(parseFloat(pkgForm.priceYuan || "0") * 100);
      const data = {
        id: pkgForm.id,
        name: pkgForm.name,
        description: pkgForm.description,
        price: priceFen,
        validityDays: pkgForm.validityDays,
        trafficLimit: pkgForm.trafficLimit,
        portCount: pkgForm.portCount,
        speedLimit: pkgForm.speedLimit,
        maxRules: pkgForm.maxRules,
        maxConnections: pkgForm.maxConnections,
        maxIPAccess: pkgForm.maxIPAccess,
        autoRenew: pkgForm.autoRenew ? 1 : 0,
        enabled: pkgForm.enabled ? 1 : 0,
        shopVisible: pkgForm.shopVisible ? 1 : 0,
        sortOrder: pkgForm.sortOrder,
        tunnelGroupIds: pkgForm.tunnelGroupIds,
      };
      const res = isPkgEdit ? await updatePackage(data) : await createPackage(data);
      if (res.code === 0) {
        toast.success(isPkgEdit ? "更新成功" : "创建成功");
        setPkgModalOpen(false);
        loadPackages();
      } else {
        toast.error(res.msg || "操作失败");
      }
    } catch {
      toast.error("网络错误");
    } finally {
      setPkgSubmitLoading(false);
    }
  };

  const handlePkgDelete = (item: SubscriptionPackageApiItem) => {
    setPkgToDelete(item);
    setPkgDeleteModalOpen(true);
  };

  const confirmPkgDelete = async () => {
    if (!pkgToDelete) return;
    try {
      const res = await deletePackage(pkgToDelete.id);
      if (res.code === 0) {
        toast.success("已删除");
        setPkgDeleteModalOpen(false);
        setPkgToDelete(null);
        loadPackages();
      } else {
        toast.error(res.msg || "删除失败");
      }
    } catch {
      toast.error("网络错误");
    }
  };

  const toggleTunnelGroup = (id: number) => {
    setPkgForm((prev) => {
      const current = prev.tunnelGroupIds;
      if (current.includes(id)) {
        return { ...prev, tunnelGroupIds: current.filter((v) => v !== id) };
      }
      return { ...prev, tunnelGroupIds: [...current, id] };
    });
  };

  // ── Render ──
  return (
    <AnimatedPage className="px-3 lg:px-6 py-8">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold">商品管理</h1>
      </div>

      {/* Tab bar */}
      <div className="flex gap-1 mb-6 border-b border-divider pb-0">
        <button
          className={`px-4 py-2 text-sm font-medium rounded-t-lg border border-b-0 transition-colors ${
            tab === "products"
              ? "bg-content1 border-divider text-foreground -mb-px"
              : "border-transparent text-default-400 hover:text-foreground"
          }`}
          onClick={() => setTab("products")}
        >
          充值商品
        </button>
        <button
          className={`px-4 py-2 text-sm font-medium rounded-t-lg border border-b-0 transition-colors ${
            tab === "packages"
              ? "bg-content1 border-divider text-foreground -mb-px"
              : "border-transparent text-default-400 hover:text-foreground"
          }`}
          onClick={() => setTab("packages")}
        >
          套餐管理
        </button>
      </div>

      {/* ──── Products Tab ──── */}
      {tab === "products" && (
        <>
          {productsLoading ? (
            <PageLoadingState message="加载商品中..." />
          ) : (
            <>
              <div className="flex items-center justify-between mb-4">
                <div />
                <div className="flex gap-2">
                  <SearchBar
                    isVisible={isSearchVisible}
                    placeholder="搜索商品..."
                    value={searchKeyword}
                    onChange={setSearchKeyword}
                    onClose={() => { setIsSearchVisible(false); setSearchKeyword(""); }}
                    onOpen={() => setIsSearchVisible(true)}
                  />
                  <Button color="primary" size="sm" variant="flat" onPress={handleProductAdd}>
                    新增商品
                  </Button>
                </div>
              </div>

              <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 lg:gap-4 mb-6">
                <Card className="border border-gray-200 dark:border-default-200 shadow-md hover:shadow-lg transition-shadow">
                  <CardBody className="p-3 lg:p-4">
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-xs text-default-500">商品总数</span>
                      <div className="p-1.5 rounded-lg bg-blue-100 dark:bg-blue-500/20">
                        <svg className="w-4 h-4 text-blue-600" fill="currentColor" viewBox="0 0 20 20">
                          <path d="M3 3a1 1 0 000 2h16a1 1 0 100-2H3z" />
                          <path fillRule="evenodd" d="M3 7a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
                          <path fillRule="evenodd" d="M3 11a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
                          <path fillRule="evenodd" d="M3 15a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clipRule="evenodd" />
                        </svg>
                      </div>
                    </div>
                    <p className="text-xl font-bold text-foreground">{products.length}</p>
                  </CardBody>
                </Card>
                <Card className="border border-gray-200 dark:border-default-200 shadow-md hover:shadow-lg transition-shadow">
                  <CardBody className="p-3 lg:p-4">
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-xs text-default-500">上架</span>
                      <div className="p-1.5 rounded-lg bg-green-100 dark:bg-green-500/20">
                        <svg className="w-4 h-4 text-green-600" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                        </svg>
                      </div>
                    </div>
                    <p className="text-xl font-bold text-green-600">{activeCount}</p>
                  </CardBody>
                </Card>
                <Card className="border border-gray-200 dark:border-default-200 shadow-md hover:shadow-lg transition-shadow">
                  <CardBody className="p-3 lg:p-4">
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-xs text-default-500">下架</span>
                      <div className="p-1.5 rounded-lg bg-red-100 dark:bg-red-500/20">
                        <svg className="w-4 h-4 text-red-600" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                        </svg>
                      </div>
                    </div>
                    <p className="text-xl font-bold text-gray-400">{products.length - activeCount}</p>
                  </CardBody>
                </Card>
                <Card className="border border-gray-200 dark:border-default-200 shadow-md hover:shadow-lg transition-shadow">
                  <CardBody className="p-3 lg:p-4">
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-xs text-default-500">类型分布</span>
                      <div className="p-1.5 rounded-lg bg-purple-100 dark:bg-purple-500/20">
                        <svg className="w-4 h-4 text-purple-600" fill="currentColor" viewBox="0 0 20 20">
                          <path d="M2 11a1 1 0 011-1h2a1 1 0 011 1v5a1 1 0 01-1 1H3a1 1 0 01-1-1v-5zM8 7a1 1 0 011-1h2a1 1 0 011 1v9a1 1 0 01-1 1H9a1 1 0 01-1-1V7zM14 4a1 1 0 011-1h2a1 1 0 011 1v12a1 1 0 01-1 1h-2a1 1 0 01-1-1V4z" />
                        </svg>
                      </div>
                    </div>
                    <div className="text-sm mt-2">
                      {productTypeOptions.map((opt) => (
                        <span key={opt.value} className="mr-3">
                          {opt.label}
                          <span className="ml-1 font-semibold text-foreground">{typeCounts[opt.value] || 0}</span>
                        </span>
                      ))}
                    </div>
                  </CardBody>
                </Card>
              </div>

              <div className="overflow-hidden rounded-xl border border-divider bg-content1 shadow-md">
                <Table classNames={{ th: "bg-default-100/50 text-default-600 text-foreground font-semibold text-sm border-b border-divider py-3 uppercase tracking-wider text-left align-middle", td: "py-3 border-b border-divider/50 group-data-[last=true]:border-b-0", tr: "hover:bg-default-50/50 transition-colors" }}>
                  <TableHeader>
                    <TableColumn className="whitespace-nowrap">名称</TableColumn>
                    <TableColumn className="whitespace-nowrap">类型</TableColumn>
                    <TableColumn className="whitespace-nowrap">价格</TableColumn>
                    <TableColumn className="whitespace-nowrap">价值</TableColumn>
                    <TableColumn className="whitespace-nowrap">排序</TableColumn>
                    <TableColumn className="whitespace-nowrap">状态</TableColumn>
                    <TableColumn className="whitespace-nowrap">操作</TableColumn>
                  </TableHeader>
                  <TableBody>
                  {filtered.map((item) => {
                    const typeLabel = productTypeOptions.find((t) => t.value === item.type)?.label || item.type;
                    return (
                      <TableRow key={item.id}>
                        <TableCell>
                          <div className="font-medium">{item.name}</div>
                          {item.description && (
                            <div className="text-xs text-gray-400 truncate max-w-40">{item.description}</div>
                          )}
                        </TableCell>
                        <TableCell>
                          <Chip color={typeBadgeColor[item.type] || "default"} size="sm" variant="flat">
                            {typeLabel}
                          </Chip>
                        </TableCell>
                        <TableCell className="font-mono">{(item.price / 100).toFixed(2)} 元</TableCell>
                        <TableCell>{item.value} {typeUnit[item.type] || ""}</TableCell>
                        <TableCell>{item.sortOrder}</TableCell>
                        <TableCell>
                          <Chip color={item.status === 1 ? "success" : "default"} size="sm">
                            {item.status === 1 ? "上架" : "下架"}
                          </Chip>
                        </TableCell>
                        <TableCell>
                          <div className="flex gap-2">
                            <Button size="sm" variant="flat" onPress={() => handleProductEdit(item)}>编辑</Button>
                            <Button size="sm" color="danger" variant="flat" onPress={() => handleProductDelete(item)}>删除</Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>

              <Modal isOpen={productModalOpen} placement="center" size="2xl"
                onOpenChange={(open) => { if (!open) setProductModalOpen(false); }}>
                <ModalContent>
                  <ModalHeader>{isProductEdit ? "编辑商品" : "新增商品"}</ModalHeader>
                  <ModalBody className="space-y-4">
                    <Input label="商品名称" value={productForm.name} variant="bordered"
                      onChange={(e) => setProductForm((p) => ({ ...p, name: e.target.value }))} />
                    <div className="space-y-1">
                      <label className="text-sm text-gray-400 text-foreground">说明</label>
                      <Textarea value={productForm.description} variant="bordered" className="w-full min-h-20"
                        onChange={(e) => setProductForm((p) => ({ ...p, description: e.target.value }))} />
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <Select label="类型" variant="bordered"
                        selectedKeys={[productForm.type]}
                        onSelectionChange={(keys) => {
                          const val = Array.from(keys)[0] as string;
                          if (val) setProductForm((p) => ({ ...p, type: val }));
                        }}>
                        {productTypeOptions.map((opt) => (
                          <SelectItem key={opt.value}>{opt.label}</SelectItem>
                        ))}
                      </Select>
                      <Input label="价格 (元)" type="number" step="0.01" min="0" value={productForm.priceYuan} variant="bordered"
                        onChange={(e) => setProductForm((p) => ({ ...p, priceYuan: e.target.value }))} />
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <Input label="价值" type="number" min="0" value={String(productForm.value)} variant="bordered"
                        onChange={(e) => setProductForm((p) => ({ ...p, value: parseInt(e.target.value) || 0 }))} />
                      <Input label="排序" type="number" min="0" value={String(productForm.sortOrder)} variant="bordered"
                        onChange={(e) => setProductForm((p) => ({ ...p, sortOrder: parseInt(e.target.value) || 0 }))} />
                    </div>
                    <Select label="状态" variant="bordered"
                      selectedKeys={[String(productForm.status)]}
                      onSelectionChange={(keys) => {
                        const val = Array.from(keys)[0] as string;
                        if (val) setProductForm((p) => ({ ...p, status: parseInt(val) }));
                      }}>
                      <SelectItem key="1">上架</SelectItem>
                      <SelectItem key="0">下架</SelectItem>
                    </Select>
                    <div className="text-xs text-gray-400">
                      {productForm.type === "recharge" ? "价值单位：分（充值到余额）" :
                       productForm.type === "traffic" ? "价值单位：GB（增加流量）" :
                       "价值单位：天（延长有效期）"}
                    </div>
                  </ModalBody>
                  <ModalFooter>
                    <Button variant="flat" onPress={() => setProductModalOpen(false)}>取消</Button>
                    <Button color="primary" isLoading={productSubmitLoading} onPress={handleProductSubmit}>确定</Button>
                  </ModalFooter>
                </ModalContent>
              </Modal>

              <Modal isOpen={productDeleteModalOpen} placement="center"
                onOpenChange={(open) => { if (!open) { setProductDeleteModalOpen(false); setProductToDelete(null); } }}>
                <ModalContent>
                  <ModalHeader>确认删除</ModalHeader>
                  <ModalBody>
                    确定要删除商品"{productToDelete?.name}"吗？
                  </ModalBody>
                  <ModalFooter>
                    <Button variant="flat" onPress={() => { setProductDeleteModalOpen(false); setProductToDelete(null); }}>取消</Button>
                    <Button color="danger" onPress={confirmProductDelete}>删除</Button>
                  </ModalFooter>
                </ModalContent>
              </Modal>
            </>
          )}
        </>
      )}

      {/* ──── Packages Tab ──── */}
      {tab === "packages" && (
        <>
          {packagesLoading ? (
            <PageLoadingState message="加载套餐中..." />
          ) : (
            <>
              <div className="flex items-center justify-end mb-4">
                <Button color="primary" size="sm" variant="flat" onPress={handlePkgAdd}>
                  新增套餐
                </Button>
              </div>

              <div className="overflow-hidden rounded-xl border border-divider bg-content1 shadow-md">
                <Table classNames={{ th: "bg-default-100/50 text-default-600 text-foreground font-semibold text-sm border-b border-divider py-3 uppercase tracking-wider text-left align-middle", td: "py-3 border-b border-divider/50 group-data-[last=true]:border-b-0", tr: "hover:bg-default-50/50 transition-colors" }}>
                  <TableHeader>
                    <TableColumn className="whitespace-nowrap">名称</TableColumn>
                    <TableColumn className="whitespace-nowrap">价格</TableColumn>
                    <TableColumn className="whitespace-nowrap">有效期</TableColumn>
                    <TableColumn className="whitespace-nowrap">流量</TableColumn>
                    <TableColumn className="whitespace-nowrap">端口</TableColumn>
                    <TableColumn className="whitespace-nowrap">限速</TableColumn>
                    <TableColumn className="whitespace-nowrap">排序</TableColumn>
                    <TableColumn className="whitespace-nowrap">启用</TableColumn>
                    <TableColumn className="whitespace-nowrap">上架</TableColumn>
                    <TableColumn className="whitespace-nowrap">自动续费</TableColumn>
                    <TableColumn className="whitespace-nowrap">操作</TableColumn>
                  </TableHeader>
                  <TableBody>
                    {pkgList.map((item) => (
                      <TableRow key={item.id}>
                        <TableCell>
                          <div className="font-medium">{item.name}</div>
                          {item.description && (
                            <div className="text-xs text-gray-400 truncate max-w-32">{item.description}</div>
                          )}
                        </TableCell>
                        <TableCell className="font-mono">{(item.price / 100).toFixed(2)} 元</TableCell>
                        <TableCell>{item.validityDays} 天</TableCell>
                        <TableCell>{item.trafficLimit > 0 ? `${item.trafficLimit} GB` : "不限"}</TableCell>
                        <TableCell>{item.portCount > 0 ? item.portCount : "不限"}</TableCell>
                        <TableCell>{item.speedLimit > 0 ? `${item.speedLimit} Mbps` : "不限"}</TableCell>
                        <TableCell>{item.sortOrder}</TableCell>
                        <TableCell>
                          <Chip color={item.enabled === 1 ? "success" : "default"} size="sm" variant="flat">
                            {item.enabled === 1 ? "是" : "否"}
                          </Chip>
                        </TableCell>
                        <TableCell>
                          <Chip color={item.shopVisible === 1 ? "primary" : "default"} size="sm" variant="flat">
                            {item.shopVisible === 1 ? "是" : "否"}
                          </Chip>
                        </TableCell>
                        <TableCell>
                          <Chip color={item.autoRenew === 1 ? "warning" : "default"} size="sm" variant="flat">
                            {item.autoRenew === 1 ? "开" : "关"}
                          </Chip>
                        </TableCell>
                        <TableCell>
                          <div className="flex gap-2">
                            <Button size="sm" variant="flat" onPress={() => handlePkgEdit(item)}>编辑</Button>
                            <Button size="sm" color="danger" variant="flat" onPress={() => handlePkgDelete(item)}>删除</Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              {/* Package Form Modal */}
              <Modal isOpen={pkgModalOpen} placement="center" size="2xl" scrollBehavior="inside"
                onOpenChange={(open) => { if (!open) { setPkgModalOpen(false); } }}>
                <ModalContent>
                  <ModalHeader>{isPkgEdit ? "编辑套餐" : "新增套餐"}</ModalHeader>
                  {pkgModalLoading ? (
                    <ModalBody><PageLoadingState message="加载套餐详情..." /></ModalBody>
                  ) : (
                    <>
                      <ModalBody className="space-y-4">
                        <Input label="套餐名称" value={pkgForm.name} variant="bordered"
                          onChange={(e) => setPkgForm((p) => ({ ...p, name: e.target.value }))} />
                        <div className="space-y-1">
                          <label className="text-sm text-gray-400 text-foreground">说明</label>
                          <Textarea value={pkgForm.description} variant="bordered" className="w-full min-h-20"
                            onChange={(e) => setPkgForm((p) => ({ ...p, description: e.target.value }))} />
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                          <Input label="价格 (元)" type="number" step="0.01" min="0" value={pkgForm.priceYuan} variant="bordered"
                            onChange={(e) => setPkgForm((p) => ({ ...p, priceYuan: e.target.value }))} />
                          <Input label="有效期 (天)" type="number" min="1" value={String(pkgForm.validityDays)} variant="bordered"
                            onChange={(e) => setPkgForm((p) => ({ ...p, validityDays: parseInt(e.target.value) || 0 }))} />
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                          <Input label="流量 (GB, 0=不限)" type="number" min="0" value={String(pkgForm.trafficLimit)} variant="bordered"
                            onChange={(e) => setPkgForm((p) => ({ ...p, trafficLimit: parseInt(e.target.value) || 0 }))} />
                          <Input label="端口数 (0=不限)" type="number" min="0" value={String(pkgForm.portCount)} variant="bordered"
                            onChange={(e) => setPkgForm((p) => ({ ...p, portCount: parseInt(e.target.value) || 0 }))} />
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                          <Input label="限速 (Mbps, 0=不限)" type="number" min="0" value={String(pkgForm.speedLimit)} variant="bordered"
                            onChange={(e) => setPkgForm((p) => ({ ...p, speedLimit: parseInt(e.target.value) || 0 }))} />
                          <Input label="规则数 (0=不限)" type="number" min="0" value={String(pkgForm.maxRules)} variant="bordered"
                            onChange={(e) => setPkgForm((p) => ({ ...p, maxRules: parseInt(e.target.value) || 0 }))} />
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                          <Input label="最大连接数 (0=不限)" type="number" min="0" value={String(pkgForm.maxConnections)} variant="bordered"
                            onChange={(e) => setPkgForm((p) => ({ ...p, maxConnections: parseInt(e.target.value) || 0 }))} />
                          <Input label="IP 限制 (0=不限)" type="number" min="0" value={String(pkgForm.maxIPAccess)} variant="bordered"
                            onChange={(e) => setPkgForm((p) => ({ ...p, maxIPAccess: parseInt(e.target.value) || 0 }))} />
                        </div>
                        <Input label="排序" type="number" min="0" value={String(pkgForm.sortOrder)} variant="bordered"
                          onChange={(e) => setPkgForm((p) => ({ ...p, sortOrder: parseInt(e.target.value) || 0 }))} />
                        <div className="flex flex-wrap gap-4">
                          <Switch
                            isSelected={pkgForm.enabled}
                            onValueChange={(v) => setPkgForm((p) => ({ ...p, enabled: v }))}
                          >
                            启用
                          </Switch>
                          <Switch
                            isSelected={pkgForm.shopVisible}
                            onValueChange={(v) => setPkgForm((p) => ({ ...p, shopVisible: v }))}
                          >
                            商城可见
                          </Switch>
                          <Switch
                            isSelected={pkgForm.autoRenew}
                            onValueChange={(v) => setPkgForm((p) => ({ ...p, autoRenew: v }))}
                          >
                            自动续费
                          </Switch>
                        </div>
                        <div className="space-y-2">
                          <label className="text-sm text-gray-400 text-foreground">关联隧道分组</label>
                          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-40 overflow-y-auto border border-divider rounded-lg p-3">
                            {tunnelGroups.length === 0 && (
                              <span className="text-xs text-gray-400 col-span-full">暂无隧道分组</span>
                            )}
                            {tunnelGroups.map((tg) => (
                              <Checkbox
                                key={tg.id}
                                isSelected={pkgForm.tunnelGroupIds.includes(tg.id)}
                                onValueChange={() => toggleTunnelGroup(tg.id)}
                              >
                                {tg.name}
                              </Checkbox>
                            ))}
                          </div>
                        </div>
                      </ModalBody>
                      <ModalFooter>
                        <Button variant="flat" onPress={() => setPkgModalOpen(false)}>取消</Button>
                        <Button color="primary" isLoading={pkgSubmitLoading} onPress={handlePkgSubmit}>确定</Button>
                      </ModalFooter>
                    </>
                  )}
                </ModalContent>
              </Modal>

              <Modal isOpen={pkgDeleteModalOpen} placement="center"
                onOpenChange={(open) => { if (!open) { setPkgDeleteModalOpen(false); setPkgToDelete(null); } }}>
                <ModalContent>
                  <ModalHeader>确认删除</ModalHeader>
                  <ModalBody>
                    确定要删除套餐"{pkgToDelete?.name}"吗？
                  </ModalBody>
                  <ModalFooter>
                    <Button variant="flat" onPress={() => { setPkgDeleteModalOpen(false); setPkgToDelete(null); }}>取消</Button>
                    <Button color="danger" onPress={confirmPkgDelete}>删除</Button>
                  </ModalFooter>
                </ModalContent>
              </Modal>
            </>
          )}
        </>
      )}
    </AnimatedPage>
  );
}
