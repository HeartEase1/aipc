import type {PaymentOrder} from '@/types/payment'
export function bonusOrderCSV(orders:PaymentOrder[]):string {
 const escape=(v:unknown)=>{const text=String(v??'');return '"'+(/^(?:\s*[=+@-]|[\t\r\n])/.test(text)?"'":"")+text.replace(/"/g,'""')+'"'}
 const rows:unknown[][]=[['订单号','状态','活动','币种','充值面额','实付','折扣','换算倍率','赠送比例%','本金余额','预计赠送余额','已赠送余额','合计到账余额','已收回赠送','返利说明']]
 for(const order of orders){const b=order.pricing?.bonus;const gift=b?.credited?Number(b.expected):0;rows.push([order.out_trade_no,order.status,b?.title,order.currency,order.original_amount,order.pay_amount,order.discount_amount,b?.multiplier,b?.percent,order.amount,b?.preview ?? b?.expected,gift,order.amount+gift,b?.refunded,b?.reason])}
 return '\uFEFF'+rows.map(row=>row.map(escape).join(',')).join('\r\n')
}
export function downloadBonusOrderCSV(orders:PaymentOrder[]){const url=URL.createObjectURL(new Blob([bonusOrderCSV(orders)],{type:'text/csv;charset=utf-8'}));const a=document.createElement('a');a.href=url;a.download='recharge-orders.csv';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000)}
