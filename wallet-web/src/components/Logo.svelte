<script>
  // Cüzdan işareti: kart + para + cep. Renkler temadan gelir.
  let { size = 34, drop = true } = $props()
</script>

<svg class:cz-drop={drop} width={size} height={size} viewBox="0 0 64 64" role="img" aria-label="Cüzdan" style="overflow: visible; flex: 0 0 auto">
  <rect x="10" y="13" width="44" height="41" rx="11" fill="var(--logo-plate)" />
  <circle class="cz-coin" cx="32" cy="31" r="8.5" fill="var(--logo-coin)" />
  <path d="M10 31 Q32 43 54 31 V43 A11 11 0 0 1 43 54 H21 A11 11 0 0 1 10 43 Z" fill="var(--logo-front)" />
</svg>
