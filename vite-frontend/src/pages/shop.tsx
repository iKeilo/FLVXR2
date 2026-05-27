import { useState, useEffect, useCallback } from "react";
import toast from "react-hot-toast";

import { AnimatedPage } from "@/components/animated-page";
import { Button } from "@/shadcn-bridge/heroui/button";
import { Card, CardBody, CardHeader } from "@/shadcn-bridge/heroui/card";
import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from "@/shadcn-bridge/heroui/modal";
import { Chip } from "@/shadcn-bridge/heroui/chip";
import { getProductList, createOrder, payOrder, getPaymentConfigs, getPackageList, createPackageOrder } from "@/api";
import type { ProductApiItem, PaymentChannelItem, SubscriptionPackageApiItem } from "@/api/types";
import { PageLoadingState } from "@/components/page-state";

export default function ShopPage() {
  const [loading, setLoading] = useState(true);
  const [products, setProducts] = useState<ProductApiItem[]>([]);
  const [packages, setPackages] = useState<SubscriptionPackageApiItem[]>([]);
  const [payChannels, setPayChannels] = useState<PaymentChannelItem[]>([]);
  const [buyModalOpen, setBuyModalOpen] = useState(false);
  const [selectedProduct, setSelectedProduct] = useState<ProductApiItem | null>(null);
  const [selectedPackage, setSelectedPackage] = useState<SubscriptionPackageApiItem | null>(null);
  const [selectedCurrency, setSelectedCurrency] = useState("BALANCE");
  const [submitting, setSubmitting] = useState(false);
  const [payResult, setPayResult] = useState<{ payUrl: string; payAddress: string; payAmount: string; orderNo: string } | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [prodRes, payRes, pkgRes] = await Promise.all([
        getProductList(),
        getPaymentConfigs(),
        getPackageList(),
      ]);
      if (prodRes.code === 0) {
        setProducts(Array.isArray(prodRes.data) ? prodRes.data : []);
      }
      if (payRes.code === 0) {
        setPayChannels(Array.isArray(payRes.data) ? payRes.data : []);
      }
      if (pkgRes.code === 0) {
        setPackages(Array.isArray(pkgRes.data) ? pkgRes.data.filter((p: SubscriptionPackageApiItem) => p.shopVisible === 1 && p.enabled === 1) : []);
      }
    } catch {
      toast.error("加载失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const handleBuy = (product: ProductApiItem) => {
    setSelectedProduct(product);
    setSelectedPackage(null);
    setSelectedCurrency("BALANCE");
    setPayResult(null);
    setBuyModalOpen(true);
  };

  const handleBuyPackage = (pkg: SubscriptionPackageApiItem) => {
    setSelectedPackage(pkg);
    setSelectedProduct(null);
    setSelectedCurrency("BALANCE");
    setPayResult(null);
    setBuyModalOpen(true);
  };

  const availableChannels = [
    { channel: "BALANCE", label: "余额支付", desc: "使用账户余额" },
    ...payChannels
      .filter((c) => c.enabled)
      .map((c) => ({
        channel: c.channel,
        label: c.channel === "USDT" ? "USDT (TRC-20)" : "易支付 (支付宝/微信)",
        desc: c.channel === "USDT" ? "加密货币支付" : "扫码支付",
      })),
  ];

  const handleSubmitOrder = async () => {
    if (!selectedProduct && !selectedPackage) return;
    setSubmitting(true);
    try {
      const res = selectedPackage
        ? await createPackageOrder({
            packageId: selectedPackage.id,
            payCurrency: selectedCurrency,
          })
        : await createOrder({
            productId: selectedProduct!.id,
            payCurrency: selectedCurrency,
          });
      if (res.code !== 0) {
        if (res.code === 1001) {
          toast.error("余额不足，请选择其他支付方式或联系管理员充值");
        } else {
          toast.error(res.msg || "下单失败");
        }
        setSubmitting(false);
        return;
      }

      if (selectedCurrency === "BALANCE") {
        toast.success("购买成功");
        setBuyModalOpen(false);
        loadData();
        return;
      }

      const payRes = await payOrder(res.data.orderId);
      if (payRes.code === 0) {
        setPayResult(payRes.data);
      } else {
        toast.error(payRes.msg || "获取支付信息失败");
      }
    } catch {
      toast.error("网络错误");
    } finally {
      setSubmitting(false);
    }
  };

  const productGroups = [
    { type: "recharge", label: "余额充值" },
    { type: "traffic", label: "流量包" },
    { type: "time", label: "时长续费" },
  ];

  if (loading) return <PageLoadingState message="加载商品中..." />;

  return (
    <AnimatedPage className="px-3 lg:px-6 py-8">
      <h1 className="text-2xl font-bold mb-6">商城</h1>

      {productGroups.map((group) => {
        const items = products.filter((p) => p.type === group.type && p.status === 1);
        if (items.length === 0) return null;
        return (
          <div key={group.type} className="mb-8">
            <h2 className="text-lg font-semibold mb-3">{group.label}</h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {items.map((product) => (
                <Card key={product.id}>
                  <CardHeader>
                    <div className="flex justify-between items-center w-full">
                      <span className="font-medium">{product.name}</span>
                      <Chip color="primary" size="sm">
                        {(product.price / 100).toFixed(2)} 元
                      </Chip>
                    </div>
                  </CardHeader>
                  <CardBody>
                    <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
                      {product.description || (product.type === "recharge"
                        ? `充值 ${product.value} 分`
                        : product.type === "traffic"
                          ? `增加 ${product.value} GB 流量`
                          : `延长 ${product.value} 天有效期`)}
                    </p>
                    <Button color="primary" className="w-full" onPress={() => handleBuy(product)}>
                      立即购买
                    </Button>
                  </CardBody>
                </Card>
              ))}
            </div>
          </div>
        );
      })}

      {packages.length > 0 && (
        <div key="packages" className="mb-8">
          <h2 className="text-lg font-semibold mb-3">套餐</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {packages.map((pkg) => (
              <Card key={pkg.id}>
                <CardHeader>
                  <div className="flex justify-between items-center w-full">
                    <span className="font-medium">{pkg.name}</span>
                    <Chip color="secondary" size="sm">
                      {(pkg.price / 100).toFixed(2)} 元
                    </Chip>
                  </div>
                </CardHeader>
                <CardBody>
                  <div className="text-sm text-gray-500 dark:text-gray-400 mb-4 space-y-1">
                    {pkg.description && <p>{pkg.description}</p>}
                    <p>有效期: {pkg.validityDays} 天</p>
                    <p>流量: {pkg.trafficLimit > 0 ? `${pkg.trafficLimit} GB` : "不限"}</p>
                    <p>端口: {pkg.portCount > 0 ? pkg.portCount : "不限"}</p>
                    {pkg.speedLimit > 0 && <p>限速: {pkg.speedLimit} Mbps</p>}
                  </div>
                  <Button color="secondary" className="w-full" onPress={() => handleBuyPackage(pkg)}>
                    立即购买
                  </Button>
                </CardBody>
              </Card>
            ))}
          </div>
        </div>
      )}

      {products.filter((p) => p.status === 1).length === 0 && packages.length === 0 && (
        <div className="text-center text-gray-400 py-20">暂无上架商品</div>
      )}

      <Modal isOpen={buyModalOpen} placement="center" size="2xl"
        onOpenChange={(open) => { if (!open) { setBuyModalOpen(false); setPayResult(null); setSelectedPackage(null); } }}>
        <ModalContent>
          <ModalHeader>
            {payResult ? "去支付" : "确认购买"}
          </ModalHeader>
          <ModalBody>
            {payResult ? (
              <div className="space-y-4">
                <p className="text-sm text-gray-500">
                  订单号: {payResult.orderNo}
                </p>
                {payResult.payUrl ? (
                  <div>
                    <p className="mb-2">点击下方按钮跳转支付：</p>
                    <Button
                      color="primary"
                      className="w-full"
                      onPress={() => window.open(payResult.payUrl, "_blank")}
                    >
                      前去支付
                    </Button>
                  </div>
                ) : null}
                {payResult.payAddress ? (
                  <div>
                    <p className="mb-2">请向以下地址转账 USDT (TRC-20)：</p>
                    <div className="bg-gray-100 dark:bg-gray-800 p-3 rounded text-sm break-all font-mono">
                      {payResult.payAddress}
                    </div>
                    {payResult.payAmount ? (
                      <p className="mt-2 text-sm">
                        金额: <strong>{payResult.payAmount} USDT</strong>
                      </p>
                    ) : null}
                  </div>
                ) : null}
                <p className="text-xs text-gray-400">
                  支付完成后请前往"我的订单"页面查看状态
                </p>
              </div>
            ) : (
              <>
              <p className="mb-4">
                商品: <strong>{selectedProduct?.name || selectedPackage?.name}</strong>
              </p>
              <p className="mb-4">
                价格: <strong>{(selectedProduct?.price ?? selectedPackage?.price ?? 0) / 100} 元</strong>
              </p>
                <div className="space-y-2">
                  <p className="text-sm font-medium">选择支付方式：</p>
                  {availableChannels.map((ch) => (
                    <label
                      key={ch.channel}
                      className={`flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${
                        selectedCurrency === ch.channel
                          ? "border-primary bg-primary/5"
                          : "border-gray-200 dark:border-gray-700"
                      }`}
                      onClick={() => setSelectedCurrency(ch.channel)}
                    >
                      <input
                        type="radio"
                        name="payCurrency"
                        value={ch.channel}
                        checked={selectedCurrency === ch.channel}
                        onChange={() => setSelectedCurrency(ch.channel)}
                        className="accent-primary"
                      />
                      <div>
                        <p className="font-medium">{ch.label}</p>
                        <p className="text-xs text-gray-400">{ch.desc}</p>
                      </div>
                    </label>
                  ))}
                </div>
              </>
            )}
          </ModalBody>
          <ModalFooter>
            {payResult ? (
              <Button variant="flat" onPress={() => { setBuyModalOpen(false); setPayResult(null); setSelectedPackage(null); }}>
                关闭
              </Button>
            ) : (
              <>
                <Button variant="flat" onPress={() => { setBuyModalOpen(false); setPayResult(null); setSelectedPackage(null); }}>取消</Button>
                <Button color="primary" isLoading={submitting} onPress={handleSubmitOrder}>
                  确认支付
                </Button>
              </>
            )}
          </ModalFooter>
        </ModalContent>
      </Modal>
    </AnimatedPage>
  );
}
