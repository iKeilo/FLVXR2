import { useCallback, useEffect, useMemo, useState } from "react";
import toast from "react-hot-toast";
import {
  AlertCircle,
  CheckCircle2,
  RefreshCw,
  UploadCloud,
} from "lucide-react";

import { Button } from "@/shadcn-bridge/heroui/button";
import {
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
} from "@/shadcn-bridge/heroui/modal";
import { Spinner } from "@/shadcn-bridge/heroui/spinner";
import {
  checkSystemUpgrade,
  runSystemUpgrade,
  type ReleaseChannel,
} from "@/api";
import type { SystemUpgradeCheckApiData } from "@/api/types";
import {
  UPDATE_CHANNEL_CHANGED_EVENT,
  getUpdateReleaseChannel,
  setUpdateReleaseChannel,
} from "@/utils/version-update";

interface SidebarUpdateButtonProps {
  collapsed: boolean;
  fallbackVersion: string;
  isAdmin: boolean;
}

type UpgradeCheckState = {
  data: SystemUpgradeCheckApiData | null;
  error: string | null;
  checkedAt: number | null;
};

const formatVersion = (value?: string | null): string => {
  const version = (value || "").trim();

  return version || "dev";
};

export function SidebarUpdateButton({
  collapsed,
  fallbackVersion,
  isAdmin,
}: SidebarUpdateButtonProps) {
  const [channel, setChannel] = useState<ReleaseChannel>(
    getUpdateReleaseChannel(),
  );
  const [modalOpen, setModalOpen] = useState(false);
  const [checking, setChecking] = useState(false);
  const [upgrading, setUpgrading] = useState(false);
  const [state, setState] = useState<UpgradeCheckState>({
    data: null,
    error: null,
    checkedAt: null,
  });

  const currentVersion = formatVersion(
    state.data?.currentVersion || fallbackVersion,
  );
  const latestVersion = formatVersion(state.data?.latestVersion);
  const hasUpdate = !!state.data?.hasUpdate;
  const canUpgrade =
    !!state.data?.capability?.capable &&
    !!state.data?.latestVersion &&
    !checking &&
    !upgrading;
  const updateButtonDisabled = !canUpgrade || !hasUpdate;

  const capabilityReasons = useMemo(() => {
    return state.data?.capability?.reasons?.filter(Boolean) || [];
  }, [state.data?.capability?.reasons]);

  const refreshUpgradeStatus = useCallback(
    async (options?: { toastOnError?: boolean }) => {
      setChecking(true);
      try {
        const res = await checkSystemUpgrade(channel);

        if (res.code !== 0 || !res.data) {
          const message = res.msg || "检查更新失败";

          setState((prev) => ({
            ...prev,
            error: message,
            checkedAt: Date.now(),
          }));
          if (options?.toastOnError) {
            toast.error(message);
          }

          return null;
        }

        setState({
          data: res.data,
          error: null,
          checkedAt: Date.now(),
        });

        return res.data;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "检查更新失败，请稍后重试";

        setState((prev) => ({
          ...prev,
          error: message,
          checkedAt: Date.now(),
        }));
        if (options?.toastOnError) {
          toast.error(message);
        }

        return null;
      } finally {
        setChecking(false);
      }
    },
    [channel],
  );

  useEffect(() => {
    if (!isAdmin) {
      return;
    }

    void refreshUpgradeStatus();
  }, [isAdmin, refreshUpgradeStatus]);

  useEffect(() => {
    const syncChannel = () => {
      setChannel(getUpdateReleaseChannel());
    };

    window.addEventListener(UPDATE_CHANNEL_CHANGED_EVENT, syncChannel);
    window.addEventListener("storage", syncChannel);

    return () => {
      window.removeEventListener(UPDATE_CHANNEL_CHANGED_EVENT, syncChannel);
      window.removeEventListener("storage", syncChannel);
    };
  }, []);

  const openUpgradeModal = async () => {
    setModalOpen(true);
    await refreshUpgradeStatus({ toastOnError: true });
  };

  const handleChannelChange = async (nextChannel: ReleaseChannel) => {
    if (channel === nextChannel) {
      return;
    }
    setUpdateReleaseChannel(nextChannel);
    setChannel(nextChannel);
  };

  useEffect(() => {
    if (modalOpen) {
      void refreshUpgradeStatus({ toastOnError: true });
    }
  }, [channel, modalOpen, refreshUpgradeStatus]);

  const handleUpgrade = async () => {
    const targetVersion = state.data?.latestVersion;

    if (!targetVersion) {
      toast.error("未获取到可用的目标版本");

      return;
    }
    if (!state.data?.capability?.capable) {
      toast.error(
        capabilityReasons.length > 0
          ? capabilityReasons.join("; ")
          : "当前环境不支持面板自升级",
      );

      return;
    }

    setUpgrading(true);
    try {
      const res = await runSystemUpgrade(targetVersion, channel);

      if (res.code === 0) {
        toast.success(res.data?.message || "升级已触发，面板将自动重启");
        setModalOpen(false);
        setTimeout(() => {
          window.location.reload();
        }, 60000);
      } else {
        toast.error(res.msg || "升级失败");
        setUpgrading(false);
      }
    } catch (err) {
      toast.error(
        "升级失败：" + (err instanceof Error ? err.message : "未知错误"),
      );
      setUpgrading(false);
    }
  };

  if (!isAdmin) {
    return null;
  }

  return (
    <>
      <button
        className={[
          "relative flex items-center justify-center overflow-hidden rounded-xl border transition-all",
          "border-white/60 bg-white/55 text-gray-700 shadow-sm backdrop-blur-xl",
          "hover:bg-white/75 hover:text-primary-600 dark:border-white/10 dark:bg-white/10 dark:text-gray-100 dark:hover:bg-white/15",
          collapsed ? "h-9 w-9 px-0" : "h-9 min-w-0 flex-1 px-2.5",
        ].join(" ")}
        title={
          hasUpdate
            ? `发现新版本 ${latestVersion}`
            : `当前版本 ${currentVersion}`
        }
        type="button"
        onClick={openUpgradeModal}
      >
        {hasUpdate && (
          <span className="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-red-500 shadow-[0_0_0_3px_rgba(239,68,68,0.18)]" />
        )}
        {checking && !state.checkedAt ? (
          <Spinner size="sm" />
        ) : collapsed ? (
          <UploadCloud className="h-4 w-4" />
        ) : (
          <span className="flex min-w-0 items-center gap-1.5 text-[11px] font-semibold leading-none">
            <UploadCloud className="h-3.5 w-3.5 shrink-0" />
            <span className="truncate">v{currentVersion}</span>
          </span>
        )}
      </button>

      <Modal
        backdrop="blur"
        classNames={{
          base: "!w-[calc(100%-32px)] !mx-auto sm:!w-full rounded-2xl overflow-hidden",
        }}
        isOpen={modalOpen}
        placement="center"
        scrollBehavior="outside"
        size="md"
        onOpenChange={setModalOpen}
      >
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader className="flex flex-col gap-1">
                <h2 className="text-xl font-bold">版本更新</h2>
                <p className="text-xs font-normal text-default-500">
                  打开窗口后会重新检查当前版本和最新 Release。
                </p>
              </ModalHeader>
              <ModalBody>
                {checking && !state.data ? (
                  <div className="flex items-center justify-center gap-3 py-10 text-sm text-default-500">
                    <Spinner size="sm" />
                    正在检查版本...
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div className="grid grid-cols-2 gap-3">
                      <div className="rounded-2xl border border-white/60 bg-white/55 p-3 dark:border-white/10 dark:bg-white/10">
                        <p className="text-xs text-default-500">当前版本</p>
                        <p className="mt-1 truncate text-lg font-bold">
                          v{currentVersion}
                        </p>
                      </div>
                      <div className="rounded-2xl border border-white/60 bg-white/55 p-3 dark:border-white/10 dark:bg-white/10">
                        <p className="text-xs text-default-500">最新版本</p>
                        <p className="mt-1 truncate text-lg font-bold text-primary">
                          {state.data?.latestVersion
                            ? `v${latestVersion}`
                            : "未获取"}
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center justify-between gap-3 rounded-2xl border border-white/60 bg-white/50 p-3 dark:border-white/10 dark:bg-white/10">
                      <div>
                        <p className="text-sm font-semibold">更新通道</p>
                        <p className="text-xs text-default-500">
                          稳定版用于正式发布，开发版用于测试标签。
                        </p>
                      </div>
                      <div className="flex shrink-0 rounded-xl bg-gray-100 p-1 dark:bg-white/10">
                        <button
                          className={`rounded-lg px-2.5 py-1 text-xs font-semibold transition ${
                            channel === "stable"
                              ? "bg-primary text-primary-foreground shadow-sm"
                              : "text-default-500"
                          }`}
                          type="button"
                          onClick={() => void handleChannelChange("stable")}
                        >
                          稳定版
                        </button>
                        <button
                          className={`rounded-lg px-2.5 py-1 text-xs font-semibold transition ${
                            channel === "dev"
                              ? "bg-primary text-primary-foreground shadow-sm"
                              : "text-default-500"
                          }`}
                          type="button"
                          onClick={() => void handleChannelChange("dev")}
                        >
                          开发版
                        </button>
                      </div>
                    </div>

                    <div className="rounded-2xl border border-white/60 bg-white/50 p-3 text-sm dark:border-white/10 dark:bg-white/10">
                      {state.error ? (
                        <div className="flex gap-2 text-danger">
                          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
                          <span>{state.error}</span>
                        </div>
                      ) : hasUpdate ? (
                        <div className="flex gap-2 text-primary">
                          <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" />
                          <span>
                            发现新版本 v{latestVersion}，确认后将更新到该版本。
                          </span>
                        </div>
                      ) : (
                        <div className="flex gap-2 text-default-500">
                          <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" />
                          <span>当前已经是该通道的最新版本。</span>
                        </div>
                      )}

                      {capabilityReasons.length > 0 && (
                        <div className="mt-3 rounded-xl bg-danger-500/10 px-3 py-2 text-xs text-danger-600 dark:text-danger-300">
                          {capabilityReasons.join("; ")}
                        </div>
                      )}
                    </div>

                    <div className="grid grid-cols-1 gap-y-1.5 pl-4 text-xs text-danger-600/80 dark:text-danger-400/80">
                      <p className="list-item">
                        升级会替换 compose 并更新 FLUX_VERSION
                      </p>
                      <p className="list-item">升级过程中面板会短暂不可用</p>
                      <p className="list-item">失败时会尝试自动回滚旧配置</p>
                    </div>
                  </div>
                )}
              </ModalBody>
              <ModalFooter className="flex gap-2">
                <Button isDisabled={upgrading} variant="flat" onPress={onClose}>
                  取消
                </Button>
                <Button
                  isIconOnly
                  aria-label="重新检查"
                  isDisabled={checking || upgrading}
                  variant="flat"
                  onPress={() =>
                    void refreshUpgradeStatus({ toastOnError: true })
                  }
                >
                  <RefreshCw
                    className={`h-4 w-4 ${checking ? "animate-spin" : ""}`}
                  />
                </Button>
                <Button
                  color="primary"
                  isDisabled={updateButtonDisabled}
                  isLoading={upgrading}
                  onPress={handleUpgrade}
                >
                  {hasUpdate ? `更新到 v${latestVersion}` : "无需更新"}
                </Button>
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>
    </>
  );
}
