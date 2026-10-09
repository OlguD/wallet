<script>
  let { d, size = 20, stroke = 1.9, ...rest } = $props()
</script>

<svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width={stroke}
  stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" {...rest}><path {d} /></svg>
