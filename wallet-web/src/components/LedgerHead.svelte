<script>
  import { t } from '../lib/i18n.js'
  let { last = 'ledger.balance' } = $props()
</script>

<div class="head" aria-hidden="true">
  <span>{t('ledger.date')}</span><span>{t('ledger.desc')}</span><span class="r">{t('ledger.amount')}</span><span class="r">{t(last)}</span>
</div>

<style>
  .head {
    display: grid;
    grid-template-columns: 46px minmax(0, 1fr) 84px 80px;
    gap: 6px;
    padding: 0 0 6px;
    border-bottom: 1px solid var(--fg);
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: var(--muted);
  }
  .r {
    text-align: right;
  }
</style>
