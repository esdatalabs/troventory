// Money helpers: the API deals exclusively in integer minor-unit cents
// (ARCHITECTURE.md §6); these are the two boundary conversions to and from
// the decimal strings a human types into a form.

export function centsToDollarsInput(cents) {
  if (cents == null) return ''
  return (cents / 100).toFixed(2)
}

export function dollarsInputToCents(value) {
  if (value === '' || value == null) return 0
  const amount = Number.parseFloat(value)
  if (Number.isNaN(amount)) return 0
  return Math.round(amount * 100)
}

export function formatMoney(amountCents, currency) {
  if (!currency) return '—'
  const amount = (amountCents / 100).toFixed(2)
  return `${amount} ${currency}`
}
