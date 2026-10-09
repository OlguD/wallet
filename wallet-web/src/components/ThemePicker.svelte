<script>
  import Icon from './Icon.svelte'
  import { I } from '../lib/icons.js'
  import { t } from '../lib/i18n.js'
  import { prefs, setTheme } from '../lib/prefs.svelte.js'

  let { compact = false } = $props()
  const themes = [
    { id: 'a', bg: '#0B0B0D', fg: '#F4F4F5', accent: '#C6F432', font: "'Geist', sans-serif", weight: 600, plate: '#2E2E35', front: '#F4F4F5' },
    { id: 'b', bg: '#F4EFE6', fg: '#1C1A17', accent: '#C2410C', font: "'Newsreader', serif", weight: 500, plate: '#D8CDBB', front: '#1C1A17' },
    { id: 'c', bg: '#F2F0EA', fg: '#1E1A12', accent: '#F2B441', font: "'Bricolage Grotesque', sans-serif", weight: 800, plate: '#0F5C55', front: '#1E1A12' },
  ]
</script>

<div class="themes" class:compact>
  {#each themes as th}
    <button type="button" class="theme" aria-pressed={prefs.theme === th.id} onclick={() => setTheme(th.id)}
      style="background: {th.bg}; color: {th.fg}">
      <svg width={compact ? 26 : 30} height={compact ? 26 : 30} viewBox="0 0 64 64" aria-hidden="true">
        <rect x="10" y="13" width="44" height="41" rx="11" fill={th.plate} />
        <circle cx="32" cy="31" r="8.5" fill={th.id === 'c' ? '#F2B441' : th.accent} />
        <path d="M10 31 Q32 43 54 31 V43 A11 11 0 0 1 43 54 H21 A11 11 0 0 1 10 43 Z" fill={th.front} />
      </svg>
      {#if !compact}<span class="sample" style="font-family: {th.font}; font-weight: {th.weight}">38.798<small>,00</small></span>{/if}
      <span class="name">{t('theme.' + th.id)}</span>
      {#if !compact}<span class="desc">{t('theme.' + th.id + '.desc')}</span>{/if}
      {#if prefs.theme === th.id}<span class="tick" style="background: {th.id === 'b' ? th.accent : th.id === 'c' ? '#0F5C55' : th.accent}; color: {th.id === 'a' ? '#0B0B0D' : '#fff'}"><Icon d={I.check} size={14} stroke={3} /></span>{/if}
    </button>
  {/each}
</div>

<style>
  .themes {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
  }
  .theme {
    position: relative;
    border: 1px solid rgba(128, 128, 128, 0.25);
    border-radius: 18px;
    padding: 12px;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
    text-align: left;
    min-height: 150px;
    transition: transform 0.2s cubic-bezier(0.34, 1.5, 0.5, 1), box-shadow 0.2s;
  }
  .compact .theme {
    min-height: 0;
    flex-direction: row;
    align-items: center;
    padding: 8px 10px;
    border-radius: 14px;
  }
  .theme[aria-pressed='true'] {
    box-shadow: 0 0 0 2px var(--bg), 0 0 0 4px var(--fg);
    transform: translateY(-2px);
  }
  .sample {
    font-size: 22px;
    letter-spacing: -0.03em;
    margin-top: auto;
  }
  .sample small {
    font-size: 13px;
    opacity: 0.55;
  }
  .name {
    font-size: 14px;
    font-weight: 700;
  }
  .compact .name {
    font-size: 13px;
  }
  .desc {
    font-size: 11px;
    opacity: 0.7;
    line-height: 1.3;
  }
  .tick {
    position: absolute;
    top: 8px;
    right: 8px;
    width: 22px;
    height: 22px;
    border-radius: 11px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .compact .tick {
    top: -6px;
    right: -6px;
  }
</style>
