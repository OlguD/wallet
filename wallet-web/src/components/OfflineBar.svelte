<script>
  // Üstte küçük durum hapı: çevrimdışıyken ve kuyrukta bekleyen değişiklik varken görünür.
  import { fly } from 'svelte/transition'
  import Icon from './Icon.svelte'
  import { I } from '../lib/icons.js'
  import { t } from '../lib/i18n.js'
  import { net, flush } from '../lib/offline.svelte.js'

  const show = $derived(!net.online || net.pending > 0)
</script>

{#if show}
  <button type="button" class="pill" class:off={!net.online} transition:fly={{ y: -30, duration: 250 }} onclick={() => flush()} aria-live="polite">
    <Icon d={net.online ? I.repeat : I.x} size={14} stroke={2.4} />
    <span>
      {#if !net.online}{t('off.banner')}{#if net.pending} · {t('off.pending_n', { n: net.pending })}{/if}
      {:else if net.syncing}{t('off.syncing')}
      {:else}{t('off.pending_n', { n: net.pending })} · {t('off.retry')}{/if}
    </span>
  </button>
{/if}

<style>
  .pill {
    position: fixed;
    z-index: 35;
    top: calc(env(safe-area-inset-top, 0px) + 6px);
    left: 50%;
    transform: translateX(-50%);
    max-width: calc(100% - 32px);
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
    border-radius: 16px;
    border: none;
    font-size: 12px;
    font-weight: 600;
    background: #e8a33d;
    color: #1e1a12;
    box-shadow: 0 6px 18px rgba(0, 0, 0, 0.2);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .pill.off {
    background: #3a3a40;
    color: #f4f4f5;
  }
</style>
