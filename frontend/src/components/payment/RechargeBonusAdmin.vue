<template>
 <section class="card mt-6 space-y-4 p-5">
  <div class="flex items-center justify-between"><div><h2 class="text-lg font-semibold">充值赠送</h2><p class="text-sm text-gray-500">按充值面额赠送，可与折扣叠加；获赠订单整笔不参与邀请返利。</p></div><button class="btn btn-primary" @click="edit()">创建活动</button></div>
  <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
  <div class="rounded-lg bg-gray-50 p-3 text-xs dark:bg-dark-800">海报 {{ stats.count }} 张 · {{ (stats.size_bytes / 1048576).toFixed(2) }} MB <span :class="stats.failed ? 'text-red-600 font-semibold' : ''"> · 清理失败 {{ stats.failed }} 张</span><p class="mt-1">未绑定或已替换海报 24 小时后回收，活动结束 30 天后回收。暂停或草稿引用仍保留。存储桶开启版本保留时，请配置非当前版本过期规则。</p></div>
  <div class="overflow-x-auto"><table class="w-full text-left text-sm"><thead><tr><th class="p-2">活动</th><th>时间</th><th>赠送方式</th><th>状态</th><th>操作</th></tr></thead><tbody>
   <tr v-for="c in campaigns" :key="c.id" class="border-t dark:border-dark-700"><td class="p-2">{{ c.title }}<p class="text-xs text-gray-500">{{ c.settlement_currency }} · {{ c.tiers.length }} 档</p></td><td>{{ date(c.starts_at) }}<br />{{ date(c.ends_at) }}<p class="text-xs">{{ c.timezone }}</p></td><td>{{ frequencies[c.frequency] }}</td><td>{{ c.enabled ? '已启用' : '已停用' }}</td><td><button class="btn btn-secondary btn-sm" @click="edit(c)">编辑</button><button class="btn btn-secondary btn-sm ml-2" @click="removeTarget=c">删除</button></td></tr>
   <tr v-if="!campaigns.length"><td colspan="5" class="p-6 text-center text-gray-500">尚无赠送活动</td></tr>
  </tbody></table></div>
  <BaseDialog :show="open" title="充值赠送活动" width="wide" @close="!busy && (open=false)">
   <form id="bonus-form" class="space-y-4" @submit.prevent="save">
    <label class="block">标题<input v-model.trim="form.title" required maxlength="160" class="input mt-1" /></label>
    <label class="block">文案<textarea v-model="form.copy" maxlength="4000" rows="3" class="input mt-1" /></label>
    <div class="grid gap-4 sm:grid-cols-2"><label>币种<input v-model="form.settlement_currency" required :disabled="!!id" class="input mt-1" maxlength="3" /></label><label>赠送方式<select v-model="form.frequency" class="input mt-1"><option v-for="(label,key) in frequencies" :key="key" :value="key">{{ label }}</option></select></label></div>
    <div class="grid gap-4 sm:grid-cols-2"><label>开始时间<input v-model="start" type="datetime-local" required class="input mt-1" /></label><label>结束时间<input v-model="end" type="datetime-local" required class="input mt-1" /></label></div>
    <p class="text-xs text-gray-500">上述时间使用浏览器时区 {{ browserTimezone }}；以下活动时区用于每日领取与弹窗日期。</p>
    <label class="block">活动时区<input v-model="form.timezone" required class="input mt-1" /></label>
    <div><p class="font-medium">金额档位</p><p class="text-xs text-gray-500">超过下限、包含上限；上限留空表示不限。空档不赠送。比例最高 100%。</p>
     <div v-for="(tier,index) in form.tiers" :key="index" class="mt-2 grid grid-cols-[1fr_1fr_1fr_auto] gap-2">
      <label class="text-xs">超过<input v-model="tier.min_amount" type="number" min="0" step="0.001" required class="input" /></label>
      <label class="text-xs">至（含）<input v-model="tier.max_amount" type="number" min="0" step="0.001" class="input" placeholder="不限" /></label>
      <label class="text-xs">赠送 %<input v-model="tier.bonus_percent" type="number" min="0.0001" max="100" step="0.0001" required class="input" /></label>
      <button type="button" class="btn btn-secondary self-end" :disabled="form.tiers.length===1" @click="form.tiers.splice(index,1)">移除</button>
     </div><button type="button" class="btn btn-secondary mt-2" :disabled="form.tiers.length>=50" @click="form.tiers.push({min_amount:'0',max_amount:null,bonus_percent:'5'})">添加档位</button>
    </div>
    <p class="rounded-lg bg-amber-50 p-3 text-sm text-amber-900 dark:bg-amber-950/30 dark:text-amber-200">充值 100 元，到账倍率 1:10，赠送 10%：本金 1000 + 赠送 100 = 1100 余额。首笔以成功付款确认为准；退款按比例收回赠送，不恢复次数。</p>
    <label class="block">活动海报（可选）<input type="file" accept="image/png,image/jpeg,image/webp" class="mt-1 block w-full text-sm" :disabled="busy||uploading" @change="upload" /></label>
    <p class="text-xs text-gray-500">PNG / JPEG / WebP，最多 5 MB、8192 像素边长及 2400 万像素；复用图片存储配置。</p>
    <img v-if="posterURL" :src="posterURL" alt="海报预览" class="max-h-48 rounded-lg" />
    <button v-if="form.poster_id" type="button" class="btn btn-secondary" @click="form.poster_id=null;posterURL=''">移除海报</button>
    <label class="flex gap-2"><input v-model="form.enabled" type="checkbox" />启用活动</label>
    <p v-if="formError" role="alert" class="text-sm text-red-600">{{ formError }}</p>
   </form>
   <template #footer><button class="btn btn-secondary" :disabled="busy" @click="open=false">取消</button><button form="bonus-form" type="submit" class="btn btn-primary" :disabled="busy||uploading">{{ uploading ? '上传中…' : '保存' }}</button></template>
  </BaseDialog>
  <ConfirmDialog :show="!!removeTarget" title="删除活动" message="删除后新订单不再参与。已有有效订单仍按快照处理，历史账务保留，海报按生命周期回收。" danger @cancel="removeTarget=null" @confirm="remove" />
  <TotpStepUpDialog :controller="stepUp" />
 </section>
</template>
<script setup lang="ts">
import {onMounted,ref} from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'
import {useStepUp,isStepUpCancelled} from '@/composables/useStepUp'
import {rechargeBonusAPI,type RechargeBonusCampaign} from '@/api/admin/balanceMarketing'
const stepUp=useStepUp()
const frequencies={every_payment:'每笔赠送',daily_first:'每日首笔符合条件',campaign_first:'活动期间首笔符合条件'}
const campaigns=ref<RechargeBonusCampaign[]>([]), open=ref(false),busy=ref(false),uploading=ref(false),error=ref(''),formError=ref(''),id=ref<number>(),posterURL=ref(''),start=ref(''),end=ref('')
const stats=ref({count:0,size_bytes:0,failed:0}),removeTarget=ref<RechargeBonusCampaign|null>(null)
const empty=():Omit<RechargeBonusCampaign,'id'>=>({title:'',copy:'',poster_id:null,settlement_currency:'CNY',enabled:false,frequency:'every_payment',starts_at:'',ends_at:'',timezone:'Asia/Shanghai',tiers:[{min_amount:'0',max_amount:'100',bonus_percent:'5'},{min_amount:'100',max_amount:'500',bonus_percent:'10'}]})
const form=ref(empty()),browserTimezone=Intl.DateTimeFormat().resolvedOptions().timeZone
const message=(e:unknown)=>(e as {message?:string})?.message||'操作失败'
const date=(v:string)=>new Date(v).toLocaleString()
const local=(v:string)=>{const d=new Date(v);return new Date(d.getTime()-d.getTimezoneOffset()*60000).toISOString().slice(0,16)}
async function load(){try{const [c,s]=await Promise.all([rechargeBonusAPI.list(),rechargeBonusAPI.stats()]);campaigns.value=c.data;stats.value=s.data;error.value=''}catch(e){error.value=message(e)}}
function edit(c?:RechargeBonusCampaign){form.value=c?JSON.parse(JSON.stringify(c)):empty();id.value=c?.id;start.value=c?local(c.starts_at):'';end.value=c?local(c.ends_at):'';posterURL.value=c?.poster_url||'';formError.value='';open.value=true}
async function save(){busy.value=true;try{const data={...form.value,starts_at:new Date(start.value).toISOString(),ends_at:new Date(end.value).toISOString(),tiers:form.value.tiers.map(t=>({min_amount:String(t.min_amount),max_amount:t.max_amount==null||String(t.max_amount)===''?null:String(t.max_amount),bonus_percent:String(t.bonus_percent)}))};await stepUp.run(()=>rechargeBonusAPI.save(data,id.value));open.value=false;await load()}catch(e){if(!isStepUpCancelled(e))formError.value=message(e)}finally{busy.value=false}}
async function upload(event:Event){const input=event.target as HTMLInputElement;const file=input.files?.[0];if(!file)return;if(file.size>5*1024*1024){formError.value='图片不能超过 5 MB';return}uploading.value=true;try{const result=await stepUp.run(()=>rechargeBonusAPI.upload(file));form.value.poster_id=result.data.id;posterURL.value=result.data.url;formError.value=''}catch(e){if(!isStepUpCancelled(e))formError.value=message(e)}finally{uploading.value=false;input.value=''}}
async function remove(){if(!removeTarget.value)return;busy.value=true;try{const target=removeTarget.value.id;await stepUp.run(()=>rechargeBonusAPI.delete(target));removeTarget.value=null;await load()}catch(e){if(!isStepUpCancelled(e))error.value=message(e)}finally{busy.value=false}}
onMounted(load)
</script>
