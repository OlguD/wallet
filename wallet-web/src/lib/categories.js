import { I } from './icons.js'

// Arayüzde gösterilen kategoriler (backend'deki anahtarlarla aynı).
export const EXPENSE_CATS = [
  { id: 'groceries', icon: I.cart, tint: '#D7EEE2' },
  { id: 'bills', icon: I.bolt, tint: '#E5DDF2' },
  { id: 'transport', icon: I.car, tint: '#DCE7F5' },
  { id: 'food', icon: I.coffee, tint: '#FBEBC8' },
  { id: 'rent', icon: I.home, tint: '#FADFCF' },
  { id: 'subscription', icon: I.repeat, tint: '#E1EEDC' },
  { id: 'gift', icon: I.gift, tint: '#F6DCE6' },
  { id: 'other', icon: I.dots, tint: '#E6E2D8' },
]

export const INCOME_CATS = [
  { id: 'salary', icon: I.income, tint: '#D7EEE2' },
  { id: 'extra', icon: I.plus, tint: '#FBEBC8' },
  { id: 'gift', icon: I.gift, tint: '#F6DCE6' },
  { id: 'other', icon: I.dots, tint: '#E6E2D8' },
]

const all = {
  groceries: EXPENSE_CATS[0], bills: EXPENSE_CATS[1], transport: EXPENSE_CATS[2], food: EXPENSE_CATS[3],
  rent: EXPENSE_CATS[4], subscription: EXPENSE_CATS[5], gift: EXPENSE_CATS[6], other: EXPENSE_CATS[7],
  salary: INCOME_CATS[0], extra: INCOME_CATS[1],
  health: { id: 'health', icon: I.heart, tint: '#F9E0E0' },
  shopping: { id: 'shopping', icon: I.bag, tint: '#E8E1F7' },
}

export const catsFor = (type) => (type === 'income' ? INCOME_CATS : EXPENSE_CATS)

/** İşlemin ikon/renk bilgisi; hesaplaşma ve kategorisiz işlemler dahil. */
export function catInfo(tx) {
  if (tx?.category && all[tx.category]) return all[tx.category]
  if (tx?.description?.startsWith('Hesaplasma')) return { id: 'settlement', icon: I.swap, tint: '#E6E2D8' }
  return tx?.type === 'income' ? { id: 'none', icon: I.arrowDown, tint: '#D7EEE2' } : { id: 'none', icon: I.dots, tint: '#E6E2D8' }
}

// C temasında hesap kartı renkleri (sırayla).
export const POCKET_COLORS = [
  { bg: '#0F5C55', fg: '#FFFFFF' },
  { bg: '#4B2A7A', fg: '#FFFFFF' },
  { bg: '#F2B441', fg: '#1E1A12' },
  { bg: '#2B4BDB', fg: '#FFFFFF' },
  { bg: '#B4441B', fg: '#FFFFFF' },
  { bg: '#2F6B3A', fg: '#FFFFFF' },
]
export const pocketColor = (i) => POCKET_COLORS[i % POCKET_COLORS.length]

// Birikim hedefi simgeleri (backend'deki listeyle aynı).
export const GOAL_ICONS = ['target', 'car', 'home', 'plane', 'phone', 'gift', 'bag', 'heart', 'school']
export const goalIcon = (g) => I[GOAL_ICONS.includes(g?.icon) ? g.icon : 'target']
