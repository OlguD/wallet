<script>
  import { fly } from 'svelte/transition'
  import { app } from '../lib/store.svelte.js'
</script>

{#if app.toast}
  {#key app.toast.id}
    <div class="toast" class:err={app.toast.kind === 'err'} role="status" transition:fly={{ y: -30, duration: 260 }}>
      {app.toast.text}
    </div>
  {/key}
{/if}

<style>
  .toast {
    position: fixed;
    z-index: 90;
    left: 50%;
    top: calc(env(safe-area-inset-top, 0px) + 10px);
    transform: translateX(-50%);
    max-width: calc(100% - 40px);
    padding: 12px 18px;
    border-radius: 22px;
    background: var(--fg);
    color: var(--bg);
    font-size: 15px;
    font-weight: 600;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.25);
    text-align: center;
  }
  .toast.err {
    background: var(--danger);
    color: #fff;
  }
  :global([data-theme='b']) .toast {
    border-radius: 4px;
  }
</style>
