<script>
  // Dövizli ödemede hesaba işlenecek tutar. Kurdan önerilir (Sun Döviz, TransferSheet
  // ile aynı mantık); elle değiştirilirse öneri durur.
  //  out: hesaptan çıkan → ödeme dövizini almanın maliyeti (hesap → döviz kuru)
  //  in:  hesaba giren → gelen dövizin hesap para birimindeki karşılığı
  import { t } from '../lib/i18n.js'
  import { rate, convert, fmtRate } from '../lib/fx.js'
  import { kurusToKeypad, decimalSep, symbol } from '../lib/money.js'

  let { amount = 0, currency, accountCurrency, direction = 'out', value = $bindable('') } = $props()
  let touched = $state(false)

  const fxRate = $derived(rate(currency, accountCurrency))

  $effect(() => {
    if (touched) return
    let k = null
    if (direction === 'in') k = convert(amount, currency, accountCurrency)
    else {
      const r = rate(accountCurrency, currency)
      k = r ? Math.round(amount / r) : null
    }
    value = k ? kurusToKeypad(k).replace('.', decimalSep()) : ''
  })
</script>

<label class="field">
  <span>{t(direction === 'in' ? 'pay.acc_in' : 'pay.acc_out', { c: symbol(accountCurrency) })}</span>
  <input class="input num" bind:value inputmode="decimal" placeholder="0" oninput={() => (touched = true)} />
  {#if fxRate}<small class="hint">{t('tr.rate', { from: currency, to: accountCurrency, v: fmtRate(fxRate) })}</small>{/if}
</label>
