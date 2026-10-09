<script>
  import { fade } from 'svelte/transition'
  const letters = 'cüzdan'.split('')
</script>

<div class="splash" out:fade={{ duration: 280 }}>
  <div class="mark">
    <div class="glow"></div>
    <svg width="132" height="132" viewBox="0 0 64 64" role="img" aria-label="Cüzdan">
      <rect class="back" x="10" y="13" width="44" height="41" rx="11" fill="var(--logo-plate)" />
      <circle class="coin" cx="32" cy="31" r="8.5" fill="var(--logo-coin)" />
      <path class="front" d="M10 31 Q32 43 54 31 V43 A11 11 0 0 1 43 54 H21 A11 11 0 0 1 10 43 Z" fill="var(--logo-front)" />
    </svg>
  </div>
  <div class="word" aria-hidden="true">
    {#each letters as ch, i}<span style="animation-delay: {(0.36 + i * 0.045).toFixed(3)}s">{ch}</span>{/each}
  </div>
</div>

<style>
  .splash {
    position: fixed;
    inset: 0;
    z-index: 100;
    background: var(--bg);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 26px;
    font-family: 'Bricolage Grotesque', sans-serif;
  }
  .mark {
    position: relative;
    width: 132px;
    height: 132px;
  }
  svg {
    overflow: visible;
    position: relative;
  }
  .glow {
    position: absolute;
    left: 34px;
    top: 32px;
    width: 64px;
    height: 64px;
    border-radius: 32px;
    background: var(--logo-coin);
    filter: blur(18px);
    opacity: 0;
    animation: glow 1.6s ease-out both;
  }
  .back,
  .coin,
  .front {
    transform-box: fill-box;
  }
  .back {
    transform-origin: 50% 60%;
    animation: back 1.4s cubic-bezier(0.2, 0.8, 0.2, 1) both;
  }
  .coin {
    animation: coin 1.4s cubic-bezier(0.3, 0.7, 0.3, 1) both;
  }
  .front {
    transform-origin: 50% 100%;
    animation: front 1.4s cubic-bezier(0.2, 0.8, 0.2, 1) both;
  }
  .word {
    display: flex;
    font-size: 52px;
    font-weight: 800;
    letter-spacing: -0.04em;
    line-height: 1;
    color: var(--fg);
  }
  .word span {
    display: inline-block;
    animation: letter 0.5s cubic-bezier(0.2, 0.8, 0.2, 1) both;
  }
  @keyframes back {
    0% { transform: scale(0.3); opacity: 0; }
    25% { transform: scale(1.07); opacity: 1; }
    38% { transform: scale(1); }
  }
  @keyframes coin {
    0%, 18% { transform: translateY(-52px) rotate(-30deg); opacity: 0; }
    30% { opacity: 1; }
    50% { transform: translateY(6px) rotate(0); }
    63% { transform: translateY(-3px); }
    75%, 100% { transform: translateY(0); opacity: 1; }
  }
  @keyframes front {
    0%, 8% { transform: translateY(16px); opacity: 0; }
    30% { transform: translateY(0); opacity: 1; }
    58% { transform: scaleY(0.88) scaleX(1.03); }
    72% { transform: scaleY(1.04) scaleX(0.99); }
    84%, 100% { transform: none; }
  }
  @keyframes letter {
    from { transform: translateY(18px); opacity: 0; }
    to { transform: none; opacity: 1; }
  }
  @keyframes glow {
    0%, 40% { transform: scale(0.4); opacity: 0; }
    50% { opacity: 0.55; }
    100% { transform: scale(1.6); opacity: 0; }
  }
</style>
