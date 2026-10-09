<script>
  import Icon from './Icon.svelte'
  import { I } from '../lib/icons.js'
  import { decimalSep } from '../lib/money.js'
  import { prefs } from '../lib/prefs.svelte.js'

  let { onkey } = $props()
  const keys = ['1', '2', '3', '4', '5', '6', '7', '8', '9', 'dec', '0', 'del']

  function press(k) {
    navigator.vibrate?.(8)
    onkey(k)
  }
</script>

<div class="keypad keypad-{prefs.theme}">
  {#each keys as k}
    <button type="button" class="key" aria-label={k === 'del' ? 'Sil' : k === 'dec' ? decimalSep() : k} onclick={() => press(k)}>
      {#if k === 'del'}
        <Icon d={I.del} size={25} stroke={1.7} />
      {:else}
        {k === 'dec' ? decimalSep() : k}
      {/if}
    </button>
  {/each}
</div>

<style>
  .keypad {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    flex: 0 0 auto;
  }
  .key {
    border: none;
    background: transparent;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--fg);
    user-select: none;
    -webkit-user-select: none;
  }
  .keypad-a .key {
    height: 54px;
    font-size: 28px;
    font-weight: 400;
    border-radius: 27px;
    transition: background-color 0.1s ease, transform 0.12s ease;
  }
  .keypad-a .key:active {
    background-color: #1f1f24;
    transform: scale(0.92);
  }
  .keypad-b {
    border-top: 1px solid var(--line);
    border-left: 1px solid var(--line);
  }
  .keypad-b .key {
    height: 54px;
    border-right: 1px solid var(--line);
    border-bottom: 1px solid var(--line);
    font-family: 'Newsreader', serif;
    font-size: 27px;
    transition: background-color 0.12s ease;
  }
  .keypad-b .key:active {
    background-color: #e9e1d3;
  }
  .keypad-c {
    gap: 6px;
  }
  .keypad-c .key {
    height: 46px;
    border-radius: 14px;
    background: #fff;
    font-family: 'Bricolage Grotesque', sans-serif;
    font-size: 23px;
    font-weight: 700;
    transition: transform 0.12s ease, background-color 0.12s ease;
  }
  .keypad-c .key:active {
    transform: scale(0.9);
    background-color: #e8e4da;
  }
  @media (max-height: 700px) {
    .keypad-a .key,
    .keypad-b .key {
      height: 46px;
    }
    .keypad-c .key {
      height: 40px;
    }
  }
</style>
