<script>
  import Sheet from './Sheet.svelte'
  import Icon from './Icon.svelte'
  import { I } from '../lib/icons.js'

  // options: [{ value, label, sub?, icon?, tint? }]
  let { title, options, selected, onpick, onclose } = $props()
</script>

<Sheet {onclose} top={Math.max(120, 560 - options.length * 60)} label={title}>
  <h2 class="h-section" style="padding: 4px 0">{title}</h2>
  <div class="list">
    {#each options as o}
      <button type="button" class="row" aria-pressed={o.value === selected} onclick={() => { onpick(o.value); onclose() }}>
        {#if o.icon}
          <span class="ico" style={o.tint ? `background: ${o.tint}; color: #1E1A12` : ''}><Icon d={o.icon} size={18} /></span>
        {/if}
        <span class="grow">
          <span class="title">{o.label}</span>
          {#if o.sub}<span class="sub">{o.sub}</span>{/if}
        </span>
        {#if o.value === selected}<Icon d={I.check} size={20} stroke={2.4} />{/if}
      </button>
    {/each}
  </div>
</Sheet>
