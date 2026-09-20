import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  cleanRegionName,
  matchRegionName,
  matchRegionFromTree,
  resolveRegionCodes,
  findMatchingAddress,
  parseAddressText,
  setCachedRegionTree,
} from '../src/utils/addressParse.ts';
import {
  setWechatAddressDraft,
  consumeWechatAddressDraft,
  clearWechatAddressDraft,
} from '../src/composables/useWechatAddressImport.ts';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const regionsJsonPath = path.resolve(__dirname, '../../admin-server/data/region/regions.json');
const regionsData = JSON.parse(fs.readFileSync(regionsJsonPath, 'utf-8'));

console.log('--- Starting WeChat Address & Real Region Matching Tests ---');

// ----------------------------------------------------
// 测试 1: 真实区划树精准逐级匹配（覆盖直辖市、普通城市、省直辖市、同名区县与旧区划）
// ----------------------------------------------------
console.log('Test 1: 真实区划树匹配 (深圳、杭州、北京、省直辖、同名区县、旧区划)');
{
  setCachedRegionTree(regionsData);

  // 1. 深圳南山区
  const szRes = matchRegionFromTree(regionsData, '广东省', '深圳市', '南山区');
  assert.equal(szRes.matched, true, '深圳南山区应匹配成功');
  assert.equal(szRes.province_code, '440000');
  assert.equal(szRes.city_code, '440300');
  assert.equal(szRes.district_code, '440305');
  assert.equal(szRes.district, '南山区');

  // 2. 杭州西湖区
  const hzRes = matchRegionFromTree(regionsData, '浙江省', '杭州市', '西湖区');
  assert.equal(hzRes.matched, true, '杭州西湖区应匹配成功');
  assert.equal(hzRes.province_code, '330000');
  assert.equal(hzRes.city_code, '330100');
  assert.equal(hzRes.district_code, '330106');
  assert.equal(hzRes.district, '西湖区');

  // 3. 北京海淀区 (直辖市: 微信返回 provinceName="北京市", cityName="北京市", countyName="海淀区")
  const bjRes = matchRegionFromTree(regionsData, '北京市', '北京市', '海淀区');
  assert.equal(bjRes.matched, true, '北京海淀区应匹配成功');
  assert.equal(bjRes.province_code, '110000');
  assert.equal(bjRes.city_code, '110100'); // 市辖区
  assert.equal(bjRes.district_code, '110108'); // 海淀区
  assert.equal(bjRes.district, '海淀区');

  // 4. 省直辖县级市 - 河南济源市 (province="河南省", cityName="济源市", countyName="济源市")
  const jyRes = matchRegionFromTree(regionsData, '河南省', '济源市', '济源市');
  assert.equal(jyRes.matched, true, '河南济源市应匹配成功');
  assert.equal(jyRes.province_code, '410000');
  assert.equal(jyRes.city_code, '419000'); // 省直辖县级行政区划
  assert.equal(jyRes.district_code, '419001'); // 济源市

  // 5. 省直辖县级市 - 湖北仙桃市 (province="湖北省", cityName="仙桃市", countyName="仙桃市")
  const xtRes = matchRegionFromTree(regionsData, '湖北省', '仙桃市', '仙桃市');
  assert.equal(xtRes.matched, true, '湖北仙桃市应匹配成功');
  assert.equal(xtRes.province_code, '420000');
  assert.equal(xtRes.city_code, '429000');
  assert.equal(xtRes.district_code, '429004'); // 仙桃市

  // 6. 同名区县（南京鼓楼区 vs 开封鼓楼区）
  const njGulou = matchRegionFromTree(regionsData, '江苏省', '南京市', '鼓楼区');
  assert.equal(njGulou.matched, true);
  assert.equal(njGulou.district_code, '320106', '南京鼓楼区代码应为 320106');

  const kfGulou = matchRegionFromTree(regionsData, '河南省', '开封市', '鼓楼区');
  assert.equal(kfGulou.matched, true);
  assert.equal(kfGulou.district_code, '410204', '开封鼓楼区代码应为 410204');
  assert.notEqual(njGulou.district_code, kfGulou.district_code, '同名区县在不同城市代码应不同');

  // 7. 旧区划或不存在区划（如已撤销的“崇文区”或虚构“火星区”）
  const oldRes = matchRegionFromTree(regionsData, '北京市', '北京市', '崇文区');
  assert.equal(oldRes.matched, false, '已撤销或不存在区划应判定匹配失败');
  assert.equal(oldRes.district_code, '', '匹配失败严禁猜测区划编码');

  const fakeRes = matchRegionFromTree(regionsData, '广东省', '深圳市', '火星区');
  assert.equal(fakeRes.matched, false);
  assert.equal(fakeRes.district_code, '');

  console.log('-> Test 1 PASSED');
}

// ----------------------------------------------------
// 测试 2: resolveRegionCodes 严禁虚构回退与真实解析
// ----------------------------------------------------
console.log('Test 2: resolveRegionCodes 行为校验');
{
  // A. 完整已知编码直接沿用
  const res1 = resolveRegionCodes('广东省', '深圳市', '南山区', {
    province_code: '440000',
    city_code: '440300',
    district_code: '440305',
  });
  assert.equal(res1.matched, true);
  assert.equal(res1.district_code, '440305');

  // B. 传入名称从真实树解析
  const res2 = resolveRegionCodes('浙江省', '杭州市', '西湖区', undefined, regionsData);
  assert.equal(res2.matched, true);
  assert.equal(res2.province_code, '330000');
  assert.equal(res2.city_code, '330100');
  assert.equal(res2.district_code, '330106');

  // C. 查无区划时必须返回 matched: false 且 code 为空，绝不返回 0100/0101
  const res3 = resolveRegionCodes('未知省', '未知市', '未知区', undefined, regionsData);
  assert.equal(res3.matched, false);
  assert.equal(res3.province_code, '');
  assert.equal(res3.city_code, '');
  assert.equal(res3.district_code, '');
  assert.notEqual(res3.city_code, '110100', '严禁默认北京或硬编码 0100');

  console.log('-> Test 2 PASSED');
}

// ----------------------------------------------------
// 测试 3: 街道与详细地址防重复拼接与名称清洗
// ----------------------------------------------------
console.log('Test 3: 街道与详细地址拼接防重复');
{
  // 1. detailInfo 未包含 streetName，正常前置拼接
  const street1 = '粤海街道';
  const detail1 = '科技南一路28号';
  let final1 = detail1;
  if (street1 && !final1.startsWith(street1)) {
    final1 = `${street1}${final1}`;
  }
  assert.equal(final1, '粤海街道科技南一路28号');

  // 2. detailInfo 已经包含 streetName，避免重复拼接为“粤海街道粤海街道...”
  const street2 = '粤海街道';
  const detail2 = '粤海街道科技南一路28号';
  let final2 = detail2;
  if (street2 && !final2.startsWith(street2)) {
    final2 = `${street2}${final2}`;
  }
  assert.equal(final2, '粤海街道科技南一路28号');

  // 3. cleanRegionName 测试
  assert.equal(cleanRegionName('广东省'), '广东');
  assert.equal(cleanRegionName('新疆维吾尔自治区'), '新疆');
  assert.equal(cleanRegionName('西湖区'), '西湖');
  assert.equal(cleanRegionName('延边朝鲜族自治州'), '延边朝鲜族');
  assert.equal(matchRegionName('延边', '延边朝鲜族自治州'), true);

  console.log('-> Test 3 PASSED');
}

// ----------------------------------------------------
// 测试 4: 单客户端已有地址比对去重防重 (findMatchingAddress)
// ----------------------------------------------------
console.log('Test 4: findMatchingAddress 客户端防重检索');
{
  const addressList = [
    {
      id: 1,
      name: '张三',
      mobile: '13800138000',
      province: '广东省',
      province_code: '440000',
      city: '深圳市',
      city_code: '440300',
      district: '南山区',
      district_code: '440305',
      address: '粤海街道科技园1号',
      is_default: 1,
    },
    {
      id: 2,
      name: '李四',
      mobile: '13900139000',
      province: '北京市',
      province_code: '110000',
      city: '北京市',
      city_code: '110100',
      district: '海淀区',
      district_code: '110108',
      address: '中关村南大街1号',
      is_default: 0,
    },
  ];

  // 1. 完全相同地址（带首尾空格） -> 精准命中已有项
  const match1 = findMatchingAddress(addressList, {
    name: ' 张三 ',
    mobile: '13800138000',
    province_code: '440000',
    city_code: '440300',
    district_code: '440305',
    address: '粤海街道科技园1号 ',
  });
  assert.ok(match1, '应命中已有完全相同的收货地址');
  assert.equal(match1.id, 1);

  // 2. 手机号不同 -> 未命中
  const match2 = findMatchingAddress(addressList, {
    name: '张三',
    mobile: '13800138001',
    province_code: '440000',
    city_code: '440300',
    district_code: '440305',
    address: '粤海街道科技园1号',
  });
  assert.equal(match2, undefined, '手机号不同不应视为重复');

  // 3. 详细地址不同 -> 未命中
  const match3 = findMatchingAddress(addressList, {
    name: '张三',
    mobile: '13800138000',
    province_code: '440000',
    city_code: '440300',
    district_code: '440305',
    address: '粤海街道科技园2号',
  });
  assert.equal(match3, undefined, '门牌号不同不应视为重复');

  console.log('-> Test 4 PASSED');
}

// ----------------------------------------------------
// 测试 5: 会话内内存草稿流转与账号安全隔离
// ----------------------------------------------------
console.log('Test 5: 会话草稿内存管理与跨账号隔离');
{
  clearWechatAddressDraft();

  // 1. 用户 A 导入草稿
  setWechatAddressDraft({
    userId: 1001,
    name: '王五',
    mobile: '13700137000',
    province: '广东省',
    province_code: '440000',
    city: '深圳市',
    city_code: '440300',
    district: '南山区',
    district_code: '440305',
    address: '科兴科学园',
    postal_code: '518000',
    regionMatched: true,
  });

  // 2. 用户 B 尝试消费该草稿 -> 跨账号安全拦截并自动销毁
  const userBDraft = consumeWechatAddressDraft(1002);
  assert.equal(userBDraft, null, '不同账号尝试消费草稿应被拦截并清空');

  // 3. 此时草稿已被清理，用户 A 再次读取也是 null
  const userADraft = consumeWechatAddressDraft(1001);
  assert.equal(userADraft, null, '被作废的跨账号草稿应已清理');

  // 4. 重新为用户 A 存入草稿
  setWechatAddressDraft({
    userId: 1001,
    name: '王五',
    mobile: '13700137000',
    province: '广东省',
    province_code: '440000',
    city: '深圳市',
    city_code: '440300',
    district: '南山区',
    district_code: '440305',
    address: '科兴科学园',
    postal_code: '518000',
    regionMatched: true,
  });

  // 5. 用户 A 正常读取草稿 -> 成功获取且立即一次性消费（阅后即焚）
  const consumed = consumeWechatAddressDraft(1001);
  assert.ok(consumed, '同账号应成功消费草稿');
  assert.equal(consumed.name, '王五');

  // 再次消费已为空
  assert.equal(consumeWechatAddressDraft(1001), null, '草稿应阅后即焚');

  console.log('-> Test 5 PASSED');
}

console.log('ALL WECHAT ADDRESS & REGION MATCHING TESTS PASSED SUCCESSFULLY!');
process.exit(0);
