import { ref, computed } from "vue";
import {
  wechatLogin,
  getMemberMe,
  updateMemberProfile,
  uploadMemberAvatar,
  getMemberAddresses,
  createMemberAddress,
  updateMemberAddress,
  deleteMemberAddress,
  setDefaultMemberAddress,
  type MemberInfo,
  type AddressResponse,
  type AddressCreateRequest,
  type AddressUpdateRequest,
} from "@/api";
import { storage, resolveImageUrl } from "@/utils";
import { getStoredToken } from "./request";

const TOKEN_KEY = "token";
const MEMBER_KEY = "member_info";

// 全局响应式单例状态
const token = ref<string>(getStoredToken());
const member = ref<MemberInfo | null>(storage.get<MemberInfo>(MEMBER_KEY));
const addresses = ref<AddressResponse[]>([]);
const loading = ref<boolean>(false);
const error = ref<string | null>(null);

const isLoggedIn = computed<boolean>(() => {
  return !!token.value && !!member.value;
});

/**
 * 规范化会员头像地址
 */
function normalizeMember(info: MemberInfo): MemberInfo {
  if (info && info.avatar_url) {
    info.avatar_url = resolveImageUrl(info.avatar_url);
  }
  return info;
}

export function useMember() {
  /**
   * 凭微信 Code 登录并同步会员资料
   */
  async function loginWithCode(code: string) {
    loading.value = true;
    error.value = null;
    try {
      const res = await wechatLogin(code);
      if (res && res.code === 1 && res.data) {
        const { token: jwtToken, member: memberData } = res.data;
        token.value = jwtToken;
        member.value = normalizeMember(memberData);

        // 统一持久化存储
        uni.setStorageSync(TOKEN_KEY, jwtToken);
        storage.set(TOKEN_KEY, jwtToken);
        storage.set(MEMBER_KEY, member.value);

        console.log("【会员认证成功】:", res.data);

        // 异步静默拉取地址列表
        fetchAddresses().catch(() => {});

        return res.data;
      }
      throw new Error(res.message || "登录失败");
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "登录失败";
      error.value = msg;
      console.warn("会员登录失败:", msg);
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 刷新当前会员资料 (GET /api/miniapp/member/me)
   */
  async function refreshMember(): Promise<MemberInfo | null> {
    if (!token.value) return null;
    try {
      const res = await getMemberMe();
      if (res && res.code === 1 && res.data) {
        member.value = normalizeMember(res.data);
        storage.set(MEMBER_KEY, member.value);
        return member.value;
      }
      return member.value;
    } catch (err) {
      console.warn("获取会员资料失败:", err);
      return member.value;
    }
  }

  /**
   * 更新会员昵称 (POST /api/miniapp/member/profile)
   */
  async function updateNickname(nickname: string): Promise<MemberInfo | null> {
    if (!token.value) return null;
    loading.value = true;
    try {
      const res = await updateMemberProfile({ nickname });
      if (res && res.code === 1 && res.data) {
        member.value = normalizeMember(res.data);
        storage.set(MEMBER_KEY, member.value);
        return member.value;
      }
      return member.value;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 上传并更新会员头像 (POST /api/miniapp/member/avatar)
   */
  async function updateAvatar(filePath: string) {
    if (!token.value) return null;
    loading.value = true;
    try {
      const res = await uploadMemberAvatar(filePath);
      if (res && res.code === 1 && res.data) {
        if (member.value) {
          member.value.avatar_id = res.data.avatar_id;
          member.value.avatar_url = resolveImageUrl(res.data.avatar_url);
          storage.set(MEMBER_KEY, member.value);
        }
        return res.data;
      }
      return null;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 获取收货地址列表 (GET /api/miniapp/member/addresses)
   */
  async function fetchAddresses(): Promise<AddressResponse[]> {
    if (!token.value) {
      addresses.value = [];
      return [];
    }
    try {
      const res = await getMemberAddresses();
      if (res && res.code === 1 && res.data) {
        addresses.value = res.data;
        return res.data;
      }
      return addresses.value;
    } catch (err) {
      console.warn("拉取地址列表失败:", err);
      return addresses.value;
    }
  }

  /**
   * 新增收货地址 (POST /api/miniapp/member/addresses/create)
   */
  async function addAddress(req: AddressCreateRequest) {
    const res = await createMemberAddress(req);
    await fetchAddresses();
    return res;
  }

  /**
   * 更新收货地址 (POST /api/miniapp/member/addresses/update)
   */
  async function editAddress(req: AddressUpdateRequest) {
    const res = await updateMemberAddress(req);
    await fetchAddresses();
    return res;
  }

  /**
   * 删除收货地址 (POST /api/miniapp/member/addresses/delete)
   */
  async function removeAddress(id: number) {
    const res = await deleteMemberAddress(id);
    await fetchAddresses();
    return res;
  }

  /**
   * 设为默认收货地址 (POST /api/miniapp/member/addresses/default)
   */
  async function makeDefaultAddress(id: number) {
    const res = await setDefaultMemberAddress(id);
    await fetchAddresses();
    return res;
  }

  /**
   * 退出登录
   */
  function logout() {
    token.value = "";
    member.value = null;
    addresses.value = [];
    storage.remove(TOKEN_KEY);
    storage.remove(MEMBER_KEY);
    uni.removeStorageSync(TOKEN_KEY);
  }

  return {
    token,
    member,
    addresses,
    loading,
    error,
    isLoggedIn,
    loginWithCode,
    refreshMember,
    updateNickname,
    updateAvatar,
    fetchAddresses,
    addAddress,
    editAddress,
    removeAddress,
    makeDefaultAddress,
    logout,
  };
}
